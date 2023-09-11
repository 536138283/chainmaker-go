package scheduler

import (
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
	txsMap *sync.Map
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
func NewSenderCollection(
	txBatch []*commonPb.Transaction,
	snapshot protocol.Snapshot,
	txAddressMap *sync.Map,
	log protocol.Logger) *SenderCollection {
	return &SenderCollection{
		txsMap: getSenderTxCollection(txBatch, snapshot, txAddressMap, log),
	}
}

// getSenderTxCollection split txs in txBatch by sender account
func getSenderTxCollection(
	txBatch []*commonPb.Transaction,
	snapshot protocol.Snapshot,
	txAddressMap *sync.Map,
	log protocol.Logger) *sync.Map {
	txCollectionMap := new(sync.Map)

	var err error
	chainCfg := snapshot.GetLastChainConfig()

	for _, tx := range txBatch {
		// get the public key from tx
		pk, err2 := getPkFromTx(tx, snapshot)
		if err2 != nil {
			log.Errorf("getPkFromTx failed: err = %v", err)
			continue
		}

		// convert the public key to `ZX` or `CM` or `EVM` address
		address, err2 := publicKeyToAddress(pk, chainCfg)
		if err2 != nil {
			log.Error("publicKeyToAddress failed: err = %v", err)
			continue
		}

		txAddressMap.Store(tx.Payload.TxId, address)
		txCollection, loaded := txCollectionMap.LoadOrStore(address, &TxCollection{
			publicKey:      pk,
			accountBalance: int64(0),
			totalGasUsed:   int64(0),
			txs:            make([]*commonPb.Transaction, 0),
		})

		collection, ok := txCollection.(*TxCollection)
		if !ok {
			log.Error("get collection failed")
			continue
		}

		collection.txs = append(collection.txs, tx)

		if !loaded {
			txCollectionMap.Store(address, collection)
		}
	}

	txCollectionMap.Range(func(key, value interface{}) bool {
		senderAddress, ok := key.(string)
		if !ok {
			log.Warnf("get sender address fail")
		}

		txCollection, ok := value.(*TxCollection)
		if !ok {
			log.Warnf("get tx collection fail")
		}

		// get the account balance from snapshot
		txCollection.accountBalance, err = getAccountBalanceFromSnapshot(senderAddress, snapshot, log)
		if err != nil {
			errMsg := fmt.Sprintf("get account balance failed: err = %v", err)
			log.Error(errMsg)
			for _, tx := range txCollection.txs {
				tx.Result = &commonPb.Result{
					Code: commonPb.TxStatusCode_CONTRACT_FAIL,
					ContractResult: &commonPb.ContractResult{
						Code:    uint32(1),
						Result:  nil,
						Message: errMsg,
						GasUsed: uint64(0),
					},
					RwSetHash: nil,
					Message:   errMsg,
				}
			}
		}
		return true
	})

	return txCollectionMap
}

// Clear clear addr in txs map
func (s *SenderCollection) Clear() {
	s.txsMap.Range(func(key, value interface{}) bool {
		s.txsMap.Delete(key)
		return true
	})
}

func getAddressFromTx(tx *commonPb.Transaction, snapshot protocol.Snapshot) (string, error) {
	chainConfig := snapshot.GetLastChainConfig()
	pk, err := getPkFromTx(tx, snapshot)
	if err != nil {
		return "", err
	}
	address, err := publicKeyToAddress(pk, chainConfig)
	if err != nil {
		return "", err
	}
	return address, nil
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
