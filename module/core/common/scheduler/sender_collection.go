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
	txsMap         *sync.Map
	txAddressCache map[string]string
	specialTxTable []*commonPb.Transaction
}

// TxCollection tx collection struct
type TxCollection struct {
	// public key to generate address
	publicKey crypto.PublicKey
	// balance of the address saved at SenderCollection
	accountBalance int64
	// total gas added each tx
	totalGasUsed int64
	txs          []*commonPb.Transaction

	// Mutex for synchronizing concurrent access to accountBalance
	mu sync.Mutex
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
	snapshot protocol.Snapshot,
	txAddressMap map[string]string) *SenderCollection {

	txCollectionMap, specialTxTable := ts.getSenderTxCollection(txBatch, snapshot, txAddressMap)
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
	txAddressCache map[string]string) (*sync.Map, []*commonPb.Transaction) {

	txCollectionMap := new(sync.Map)
	var specialTxList []*commonPb.Transaction

	for _, tx := range txBatch {
		pk, address, err := ts.getTxPayerPkAndAddress(txAddressCache, tx, snapshot)
		if err != nil {
			ts.log.Errorf("Scheduler getTxPayerAddress failed, err = %v", err)
			continue
		}

		var txCollection *TxCollection
		txCollectionValue, exists := txCollectionMap.Load(address)
		if !exists {
			balance, err := getAccountBalanceFromSnapshot(address, snapshot, ts.log)
			if err != nil {
				ts.log.Error("get balance failed, err = %v", err)
				continue
			}
			txCollection = &TxCollection{
				publicKey:      pk,
				accountBalance: balance,
				totalGasUsed:   int64(0),
				txs:            make([]*commonPb.Transaction, 0),
			}
			txCollectionMap.Store(address, txCollection)
		} else {
			var ok bool
			txCollection, ok = txCollectionValue.(*TxCollection)
			if !ok {
				ts.log.Errorf("load TxCollection failed")
				continue
			}
		}

		txCollection.totalGasUsed += int64(tx.Payload.Limit.GasLimit)
		if txCollection.totalGasUsed > txCollection.accountBalance {
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
	s.txsMap.Range(func(key, value interface{}) bool {
		s.txsMap.Delete(key)
		return true
	})
}

func (s *SenderCollection) resetTotalGasUsed() {
	s.txsMap.Range(func(key, value interface{}) bool {
		collection, ok := value.(*TxCollection)
		if ok {
			collection.totalGasUsed = 0
		}
		return true
	})
}

func (s *SenderCollection) chargeGasInSenderCollection(
	tx *commonPb.Transaction, txResult *commonPb.Result) (uint64, error) {

	address, exist := s.txAddressCache[tx.Payload.TxId]
	if !exist {
		return 0, fmt.Errorf("cannot find account balance for %v", tx.Payload.TxId)
	}
	txs, exist := s.txsMap.Load(address)
	if !exist {
		return 0, fmt.Errorf("cannot find account balance for %v", tx.Payload.TxId)
	}
	senderTxs, ok := txs.(*TxCollection)
	if !ok {
		return 0, fmt.Errorf("cannot find TxCollection for %v", tx.Payload.TxId)
	}

	senderTxs.mu.Lock()
	defer senderTxs.mu.Unlock()
	gasUsed := txResult.ContractResult.GasUsed
	if senderTxs.totalGasUsed+int64(gasUsed) > senderTxs.accountBalance {
		if gasUsed > 0 {
			gasAvailable := senderTxs.accountBalance - senderTxs.totalGasUsed
			if gasAvailable < 0 {
				gasAvailable = 0
			}
			senderTxs.totalGasUsed = senderTxs.accountBalance
			return uint64(gasAvailable), fmt.Errorf("account balance is not enough for tx: %v", tx.Payload.TxId)
		}
	}

	senderTxs.totalGasUsed += int64(gasUsed)
	return gasUsed, nil
}

func (s *SenderCollection) checkBalanceInSenderCollection(
	tx *commonPb.Transaction) error {

	address, exist := s.txAddressCache[tx.Payload.TxId]
	if !exist {
		return fmt.Errorf("cannot find account balance for %v", tx.Payload.TxId)
	}
	txs, exist := s.txsMap.Load(address)
	if !exist {
		return fmt.Errorf("cannot find account balance for %v", tx.Payload.TxId)
	}

	senderTxs, ok := txs.(*TxCollection)
	if !ok {
		return fmt.Errorf("cannot find TxCollection for %v", tx.Payload.TxId)
	}

	if senderTxs.totalGasUsed >= senderTxs.accountBalance {
		return errors.New("account balance is not enough")
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
