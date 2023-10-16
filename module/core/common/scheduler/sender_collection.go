package scheduler

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/vm-native/v3/accountmgr"

	"chainmaker.org/chainmaker/common/v3/crypto"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/protocol/v3"
)

// SenderCollection contains:
// key: address
// value: tx collection will address's other data
type SenderCollection struct {
	txsMap         map[string]*TxCollection
	txAddressCache map[string]string
	specialTxTable []*commonPb.Transaction
}

// TxCollection tx collection struct
type TxCollection struct {
	// public key to generate address
	publicKey crypto.PublicKey
	// balance of the address saved at SenderCollection
	accountBalance uint64
	// total gas added each tx
	totalGasUsed uint64
	txs          []*commonPb.Transaction

	// Mutex for synchronizing concurrent access to accountBalance
	mu sync.Mutex
}

func (g *TxCollection) addTxGas(gas uint64) error {
	totalGasUsed := g.totalGasUsed + gas
	if totalGasUsed < g.totalGasUsed {
		return errors.New("add tx gas in TxCollection overflow")
	}

	g.totalGasUsed = totalGasUsed
	return nil
}

func (g *TxCollection) String() string {
	pubKeyStr, _ := g.publicKey.String()
	return fmt.Sprintf(
		"\nTxsGroup{ \n\tpublicKey: %s, \n\taccountBalance: %v, \n\ttotalGasUsed: %v, \n\ttxs: [%d items] }",
		pubKeyStr, g.accountBalance, g.totalGasUsed, len(g.txs))
}

// NewSenderCollection new sender collection
func (ts *TxScheduler) NewSenderCollection(
	txBatch []*commonPb.Transaction,
	snapshot protocol.Snapshot) *SenderCollection {

	blockVersion := snapshot.GetLastChainConfig().GetBlockVersion()
	txAddressMap := make(map[string]string, len(txBatch))
	txCollectionMap, specialTxTable := ts.getSenderTxCollection(txBatch, snapshot, txAddressMap, blockVersion)
	return &SenderCollection{
		txsMap:         txCollectionMap,
		txAddressCache: txAddressMap,
		specialTxTable: specialTxTable,
	}
}

// getSenderTxCollection split txs in txBatch by sender account
func (ts *TxScheduler) getSenderTxCollection(
	txBatch []*commonPb.Transaction,
	snapshot protocol.Snapshot,
	txAddressCache map[string]string,
	blockVersion uint32) (map[string]*TxCollection, []*commonPb.Transaction) {

	txCollectionMap := make(map[string]*TxCollection)
	var specialTxList []*commonPb.Transaction

	for _, tx := range txBatch {
		pk, address, err := ts.getTxPayerPkAndAddress(txAddressCache, tx, snapshot)
		if err != nil {
			ts.log.Errorf("Scheduler getTxPayerAddress failed, err = %v", err)
			continue
		}

		txCollection, exists := txCollectionMap[address]
		if !exists {
			balance, err := getAccountBalanceFromSnapshot(address, snapshot, ts.log)
			if err != nil {
				ts.log.Error("get balance failed, err = %v", err)
				continue
			}
			txCollection = &TxCollection{
				publicKey:      pk,
				accountBalance: uint64(balance),
				totalGasUsed:   0,
				txs:            make([]*commonPb.Transaction, 0),
			}
			txCollectionMap[address] = txCollection
		}

		if tx.Payload.Limit != nil {
			if err := txCollection.addTxGas(tx.Payload.Limit.GasLimit); err != nil {
				continue
			}
		}

		txNeedChargeGas := ts.checkNativeFilter(
			blockVersion,
			tx.GetPayload().ContractName,
			tx.GetPayload().Method,
			tx, snapshot)

		if txCollection.totalGasUsed > txCollection.accountBalance && txNeedChargeGas {
			specialTxList = append(specialTxList, tx)
		} else {
			txCollection.txs = append(txCollection.txs, tx)
		}
	}

	return txCollectionMap, specialTxList
}

func (ts *TxScheduler) getTxPayerPkAndAddress(
	txAddressCache map[string]string,
	tx *commonPb.Transaction,
	snapshot protocol.Snapshot) (crypto.PublicKey, string, error) {

	pk, address, err := ts.getSenderPkAndAddress(tx, snapshot)
	if err != nil {
		return nil, "", fmt.Errorf("get sender pk failed: err = %v", err)
	}

	txAddressCache[tx.Payload.TxId] = address
	return pk, address, nil
}

// Clear clear addr in txs map
func (s *SenderCollection) Clear() {
	s.txsMap = make(map[string]*TxCollection)
}

func (s *SenderCollection) resetTotalGasUsed() {
	for _, txCollection := range s.txsMap {
		txCollection.totalGasUsed = 0
	}
}

func (s *SenderCollection) getParallelTxsNum() int {
	num := 0
	for _, txCollection := range s.txsMap {
		num += len(txCollection.txs)
	}

	return num
}

func (s *SenderCollection) chargeGasInSenderCollection(
	tx *commonPb.Transaction, txResult *commonPb.Result) (uint64, error) {

	// 处理不需要扣费的交易
	if tx.Payload.Limit == nil {
		return 0, nil
	}

	address, exist := s.txAddressCache[tx.Payload.TxId]
	if !exist {
		return 0, fmt.Errorf("cannot find account balance for %v", tx.Payload.TxId)
	}
	senderTxs, exist := s.txsMap[address]
	if !exist {
		return 0, fmt.Errorf("cannot find payer address for %v", tx.Payload.TxId)
	}

	senderTxs.mu.Lock()
	defer senderTxs.mu.Unlock()
	gasUsed := txResult.ContractResult.GasUsed

	// gasUsed 超过 gasLimit
	if gasUsed > tx.Payload.Limit.GasLimit {
		gasCharged := tx.Payload.Limit.GasLimit
		err := senderTxs.addTxGas(tx.Payload.Limit.GasLimit)
		if err != nil {
			return gasCharged, fmt.Errorf(
				"totalGasUsed is overflow after add gas limit of tx `%v`", tx.Payload.TxId)
		}
		return gasCharged, fmt.Errorf("gasUsed(%v) is greater than gasLimit(%v) for tx `%v`",
			txResult.ContractResult.GasUsed, tx.Payload.Limit.GasLimit, tx.Payload.TxId)
	}

	// overflow checking
	totalGasUsed := senderTxs.totalGasUsed + gasUsed
	if totalGasUsed < senderTxs.totalGasUsed {
		return gasUsed, fmt.Errorf(
			"totalGasUsed is overflow after add gas used of tx `%v`", tx.Payload.TxId)
	}

	if totalGasUsed > senderTxs.accountBalance {
		if gasUsed > 0 {
			gasAvailable := senderTxs.accountBalance - senderTxs.totalGasUsed
			if gasAvailable < 0 {
				gasAvailable = 0
			}
			senderTxs.totalGasUsed = senderTxs.accountBalance
			return gasAvailable, fmt.Errorf("account balance remains %v, is not enough for tx: %v",
				gasAvailable, tx.Payload.TxId)
		}
	}

	senderTxs.totalGasUsed += totalGasUsed
	return gasUsed, nil
}

func (s *SenderCollection) checkBalanceInSenderCollection(
	tx *commonPb.Transaction) error {

	// 处理不需要扣费的交易
	if tx.Payload.Limit == nil {
		return nil
	}

	address, exist := s.txAddressCache[tx.Payload.TxId]
	if !exist {
		return fmt.Errorf("cannot find account balance for %v", tx.Payload.TxId)
	}
	senderTxs, exist := s.txsMap[address]
	if !exist {
		return fmt.Errorf("cannot find account balance for %v", tx.Payload.TxId)
	}

	if senderTxs.totalGasUsed >= senderTxs.accountBalance {
		return fmt.Errorf("account balance is not enough for tx `%v`", tx.Payload.TxId)
	}

	if tx.Payload.Limit != nil {
		// overflow checking
		totalGasUsed := senderTxs.totalGasUsed + tx.Payload.Limit.GasLimit
		if totalGasUsed < senderTxs.totalGasUsed {
			return fmt.Errorf("totalGasUsed is overflow after add tx `%v`", tx.Payload.TxId)
		}
		if totalGasUsed > senderTxs.accountBalance {
			return fmt.Errorf("account balance is not enough for tx `%v` gas limit", tx.Payload.TxId)
		}
	}

	return nil
}

// getAccountBalanceFromSnapshot get account balance from snapshot
func getAccountBalanceFromSnapshot(
	address string, snapshot protocol.Snapshot, log protocol.Logger) (int64, error) {
	chainConfig := snapshot.GetLastChainConfig()
	blockVersion := chainConfig.GetBlockVersion()
	log.Debugf("address = %v, blockVersion = %v", address, blockVersion)

	if blockVersion < blockVersion2310 {
		return getAccountBalanceFromSnapshot2300(address, snapshot, log)
	}

	return getAccountBalanceFromSnapshot2310(address, snapshot, log)
}

// getAccountBalanceFromSnapshot2300 get account balance from snapshot for 2300 version
func getAccountBalanceFromSnapshot2300(
	address string, snapshot protocol.Snapshot, log protocol.Logger) (int64, error) {

	var err error
	var balance int64
	balanceData, err := snapshot.GetKey(-1,
		syscontract.SystemContract_ACCOUNT_MANAGER.String(),
		[]byte(accountmgr.AccountPrefix+address))
	if err != nil {
		return -1, err
	}

	if len(balanceData) == 0 {
		balance = int64(0)
	} else {
		balance, err = strconv.ParseInt(string(balanceData), 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return balance, nil
}

// getAccountBalanceFromSnapshot2310 get account balance from snapshot for 2310 version
func getAccountBalanceFromSnapshot2310(
	address string, snapshot protocol.Snapshot, log protocol.Logger) (int64, error) {
	var err error
	var balance int64
	var frozen bool

	// 查询账户的余额
	balanceData, err := snapshot.GetKey(-1,
		syscontract.SystemContract_ACCOUNT_MANAGER.String(),
		[]byte(accountmgr.AccountPrefix+address))
	if err != nil {
		return -1, err
	}

	if len(balanceData) == 0 {
		balance = int64(0)
	} else {
		balance, err = strconv.ParseInt(string(balanceData), 10, 64)
		if err != nil {
			return 0, err
		}
	}

	// 查询账户的状态
	frozenData, err := snapshot.GetKey(-1,
		syscontract.SystemContract_ACCOUNT_MANAGER.String(),
		[]byte(accountmgr.FrozenPrefix+address))
	if err != nil {
		return -1, err
	}

	if len(frozenData) == 0 {
		frozen = false
	} else {
		if string(frozenData) == "0" {
			frozen = false
		} else if string(frozenData) == "1" {
			frozen = true
		}
	}
	log.Debugf("balance = %v, freeze = %v", balance, frozen)

	if frozen {
		return 0, fmt.Errorf("account `%s` has been locked", address)
	}

	return balance, nil
}
