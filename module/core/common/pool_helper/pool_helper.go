package pool_helper

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
)

type TxPoolHelper interface {
	// ReGenTxBatchesWithRetryTxs Generate new batches by retryTxs and return batchIds of new batches for batch txPool,
	// then, put new batches into the pendingCache of pool
	// and retry old batches retrieved by the batchIds into the queue of pool.
	ReGenTxBatchesWithRetryTxs(blockHeight uint64, batchIds []string, retryTxs []*common.Transaction) (newBatchIds []string)
	// ReGenTxBatchesWithRemoveTxs Remove removeTxs in batches that retrieved by the batchIds
	// to create new batches for batch txPool and return batchIds of new batches
	// then put new batches into the pendingCache of pool
	// and delete old batches retrieved by the batchIds in pool.
	ReGenTxBatchesWithRemoveTxs(blockHeight uint64, batchIds []string, removeTxs []*common.Transaction) (newBatchIds []string)
	// RemoveTxsInTxBatches Remove removeTxs in batches that retrieved by the batchIds
	// to create new batches for batch txPool.
	// then, put new batches into the queue of pool
	// and delete old batches retrieved by the batchIds in pool.
	RemoveTxsInTxBatches(batchIds []string, removeTxs []*common.Transaction)
	// GetAllTxsByTxIds Retrieve all transactions by the txIds from single or normal txPool synchronously.
	// if there are some transactions lacked, it need to obtain them by height from the proposer.
	// if txPool get all transactions before timeout return txsRet, otherwise, return error.
	GetAllTxsByTxIds(txIds []string, proposerId string, height uint64, timeoutMs int) (txsRet map[string]*common.Transaction, err error)
	// GetAllTxsByBatchIds Retrieve all transactions by the batchIds from batch txPool synchronously.
	// if there are some batches lacked, it need to obtain them by height from the proposer.
	// if txPool get all batches before timeout return txsRet, otherwise, return error.
	GetAllTxsByBatchIds(batchIds []string, proposerId string, height uint64, timeoutMs int) (txsRet []*common.Transaction, err error)
	// AddTxsToPendingCache These transactions will be added to single or normal txPool to avoid the transactions
	// are fetched again and re-filled into the new block. Because of the chain confirmation
	// rule in the HotStuff consensus algorithm.
	AddTxsToPendingCache(txs []*common.Transaction, blockHeight uint64)
	// AddTxBatchesToPendingCache These transactions will be added to batch txPool to avoid the transactions
	// are fetched again and re-filled into the new block. Because of the chain confirmation
	// rule in the HotStuff consensus algorithm.
	AddTxBatchesToPendingCache(batchIds []string, blockHeight uint64)
	// RetryAndRemoveTxs Process transactions within multiple proposed blocks at the same height to
	// ensure that these transactions are not lost for single or normal txPool
	// re-add valid transactions which that are not on local node.
	// remove transactions in the commit block.
	RetryAndRemoveTxs(retryTxs []*common.Transaction, removeTxs []*common.Transaction)
	// RetryAndRemoveTxBatches Process batches within multiple proposed blocks at the same height to
	// ensure that these batches are not lost for batch txPool.
	// re-add valid batches to the queue of pool.
	// remove batches in the commit block.
	RetryAndRemoveTxBatches(retryBatchIds []string, removeBatchIds []string)
}
