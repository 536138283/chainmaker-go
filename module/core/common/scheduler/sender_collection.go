package scheduler

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/crypto"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

// SenderCollection contains:
// key: address
// value: tx collection will address's other data
type SenderCollection struct {
	txsMap map[string]*TxCollection
}

type TxCollection struct {
	// public key to generate address
	publicKey      crypto.PublicKey
	// balance of the address saved at SenderCollection
	accountBalance int64
	// total gas added each tx
	totalGasUsed   int64
	txs            []*commonPb.Transaction
}

func (g *TxCollection) String() string {
	pubKeyStr, _ := g.publicKey.String()
	return fmt.Sprintf(
		"\nTxsGroup{ \n\tpublicKey: %s, \n\taccountBalance: %v, \n\ttotalGasUsed: %v, \n\ttxs: [%d items] }",
		pubKeyStr, g.accountBalance, g.totalGasUsed, len(g.txs))
}

func NewSenderCollection(
	txBatch []*commonPb.Transaction,
	snapshot protocol.Snapshot,
	log protocol.Logger) *SenderCollection {
	return &SenderCollection{
		txsMap: getSenderTxCollection(txBatch, snapshot, log),
	}
}

func getSenderTxCollection(
	txBatch []*commonPb.Transaction,
	snapshot protocol.Snapshot,
	log protocol.Logger) map[string]*TxCollection {
	txCollectionMap := make(map[string]*TxCollection)

	for _, tx := range txBatch {
		pk, err := getPkFromTx(tx, snapshot)
		if err != nil {
			log.Errorf("getPkFromTx failed: err = %v", err)
			continue
		}

		address, err := publicKeyToAddress(pk)
		if err != nil {
			log.Error("publicKeyToAddress failed: err = %v", err)
			continue
		}

		txCollection, exists := txCollectionMap[address]
		if !exists {
			txCollection = &TxCollection{
				publicKey:      pk,
				accountBalance: int64(0),
				totalGasUsed:   int64(0),
				txs:            make([]*commonPb.Transaction, 0),
			}
			txCollectionMap[address] = txCollection
		}
		txCollection.txs = append(txCollection.txs, tx)
	}

	var err error
	for senderAddress, txCollection := range txCollectionMap {
		txCollection.accountBalance, err = getAccountBalanceFromSnapshot(senderAddress, snapshot)
		if err != nil {
			log.Error("getAccountBalanceFromSnapshot failed: err = %v", err)
		}
	}

	return txCollectionMap
}

func (s SenderCollection) Clear() {
	for addr := range s.txsMap {
		delete(s.txsMap, addr)
	}
}
