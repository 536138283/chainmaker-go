/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"bytes"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"fmt"
	"github.com/prometheus/common/log"
	"sync"
)

func RearrangeRWSet(block *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) []*commonpb.TxRWSet {
	rwSet := make([]*commonpb.TxRWSet, 0)
	if rwSetMap == nil {
		return rwSet
	}
	for _, tx := range block.Txs {
		if set, ok := rwSetMap[tx.Header.TxId]; ok {
			rwSet = append(rwSet, set)
		}
	}
	return rwSet
}

// IsTxHashValid, to check if transaction hash is valid
func isTxHashValid(tx *commonpb.Transaction, txInPool *commonpb.Transaction, hashType string) error {
	poolTxRawHash, err := utils.CalcTxRequestHash(hashType, txInPool)
	if err != nil {
		return fmt.Errorf("calc pool txhash error (tx:%s), %s", tx.Header.TxId, err.Error())
	}
	txRawHash, err := utils.CalcTxRequestHash(hashType, tx)
	if err != nil {
		return fmt.Errorf("calc req txhash error (tx:%s), %s", tx.Header.TxId, err.Error())
	}
	// check if tx equals with tx in pool
	if !bytes.Equal(txRawHash, poolTxRawHash) {
		return fmt.Errorf("txhash (tx:%s) expect %x, got %x", tx.Header.TxId, poolTxRawHash, txRawHash)
	}
	return nil
}

// VerifyTxResult, to check if transaction result is valid,
// compare result simulate in this node with executed in other node
func VerifyTxResult(tx *commonpb.Transaction, result *commonpb.Result, hashType string) error {
	// verify if result is equal
	txResultHash, err := utils.CalcTxResultHash(hashType, tx.Result)
	if err != nil {
		return fmt.Errorf("calc tx result (tx:%s), %s)", tx.Header.TxId, err.Error())
	}
	resultHash, err := utils.CalcTxResultHash(hashType, result)
	if err != nil {
		return fmt.Errorf("calc tx result (tx:%s), %s)", tx.Header.TxId, err.Error())
	}
	if !bytes.Equal(txResultHash, resultHash) {
		return fmt.Errorf("tx result (tx:%s) expect %x, got %x", tx.Header.TxId, txResultHash, resultHash)
	}
	return nil
}

// IsTxRWSetValid, to check if transaction read write set is valid
func IsTxRWSetValid(block *commonpb.Block, tx *commonpb.Transaction, rwSet *commonpb.TxRWSet, result *commonpb.Result,
	rwsetHash []byte) error {
	if rwSet == nil || result == nil {
		return fmt.Errorf("txresult, rwset == nil (tx:%s)",
			block.Header.BlockHeight, block.Header.BlockHash, tx.Header.TxId)
	}
	if !bytes.Equal(tx.Result.RwSetHash, rwsetHash) {
		return fmt.Errorf("tx rwset (tx:%s) expect %x, got %x", tx.Header.TxId, tx.Result.RwSetHash, rwsetHash)
	}
	return nil
}

type verifyBlockBatch struct {
	txs       []*commonpb.Transaction
	newAddTxs []*commonpb.Transaction
	txHash    [][]byte
}

// verifyStat, statistic for verify steps
type verifyStat struct {
	totalCount  int
	dbLasts     int64
	sigLasts    int64
	othersLasts int64
	sigCount    int
}

type verifyTxBase struct {
	block       *commonpb.Block
	txRWSetMap  map[string]*commonpb.TxRWSet
	txResultMap map[string]*commonpb.Result
	log         *logger.CMLogger
	store       protocol.BlockchainStore
	txPool      protocol.TxPool
	ac          protocol.AccessControlProvider
	hashType    string
	chainId     string
}

// VerifyTxs verify transactions in block
// include if transaction is double spent, transaction signature
func VerifyTxs(base *verifyTxBase) (txHashes [][]byte, txNewAdd []*commonpb.Transaction, errTxs []*commonpb.Transaction, err error) {

	verifyBatchs := utils.DispatchTxVerifyTask(base.block.Txs)
	resultTasks := make(map[int]verifyBlockBatch)
	stats := make(map[int]*verifyStat)
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	waitCount := len(verifyBatchs)
	wg.Add(waitCount)
	txIds := utils.GetTxIds(base.block.Txs)
	txsRet, txsHeightRet := base.txPool.GetTxsByTxIds(txIds)

	startTicker := utils.CurrentTimeMillisSeconds()
	for i := 0; i < waitCount; i++ {
		index := i
		go func() {
			defer wg.Done()
			txs := verifyBatchs[index]
			stat := &verifyStat{
				totalCount: len(txs),
			}
			txHashes, newAddTxs, err := verifyTx(base, txs, stat, txsRet, txsHeightRet)
			if err != nil {
				return
			}
			resultMu.Lock()
			defer resultMu.Unlock()
			resultTasks[index] = verifyBlockBatch{
				txs:       txs,
				txHash:    txHashes,
				newAddTxs: newAddTxs,
			}
			stats[index] = stat
		}()
	}
	wg.Wait()
	concurrentLasts := utils.CurrentTimeMillisSeconds() - startTicker
	txHashes, txNewAdd, errTxs, err = txVerifyResultsMerge(resultTasks, verifyBatchs, errTxs, txHashes, txNewAdd)

	if err != nil {
		return txHashes, txNewAdd, errTxs, err
	}
	for i, stat := range stats {
		if stat != nil {
			log.Debugf("verify stat (index:%d,sigcount:%d/%d,db:%d,sig:%d,other:%d,total:%d)",
				i, stat.sigCount, stat.totalCount, stat.dbLasts, stat.sigLasts, stat.othersLasts, concurrentLasts)
		}
	}

	return txHashes, txNewAdd, nil, nil
}

func verifyTx(base *verifyTxBase, txs []*commonpb.Transaction, stat *verifyStat,
	txsRet map[string]*commonpb.Transaction, txsHeightRet map[string]int64) ([][]byte, []*commonpb.Transaction, error) {
	txHashes := make([][]byte, 0)
	newAddTxs := make([]*commonpb.Transaction, 0) // tx that verified and not in txpool, need to be added to txpool
	for _, tx := range txs {
		if err := validateTx(base, tx, stat, txsRet, txsHeightRet, newAddTxs); err != nil {
			return nil, nil, err
		}

		startOthersTicker := utils.CurrentTimeMillisSeconds()
		rwSet := base.txRWSetMap[tx.Header.TxId]
		result := base.txResultMap[tx.Header.TxId]
		rwsetHash, err := utils.CalcRWSetHash(base.hashType, rwSet)
		if err != nil {
			log.Warnf("calc rwset hash error (tx:%s), %s", tx.Header.TxId, err)
			return nil, nil, err
		}
		if err := IsTxRWSetValid(base.block, tx, rwSet, result, rwsetHash); err != nil {
			return nil, nil, err
		}
		result.RwSetHash = rwsetHash
		// verify if rwset hash is equal
		if err := VerifyTxResult(tx, result, base.hashType); err != nil {
			return nil, nil, err
		}
		hash, err := utils.CalcTxHash(base.hashType, tx)
		if err != nil {
			log.Warnf("calc txhash error (tx:%s), %s", tx.Header.TxId, err)
			return nil, nil, err
		}
		txHashes = append(txHashes, hash)
		stat.othersLasts += utils.CurrentTimeMillisSeconds() - startOthersTicker
	}
	return txHashes, newAddTxs, nil
}

func txVerifyResultsMerge(resultTasks map[int]verifyBlockBatch,
	verifyBatchs map[int][]*commonpb.Transaction, errTxs []*commonpb.Transaction, txHashes [][]byte,
	txNewAdd []*commonpb.Transaction) ([][]byte, []*commonpb.Transaction, []*commonpb.Transaction, error) {
	if len(resultTasks) < len(verifyBatchs) {
		return nil, nil, errTxs, fmt.Errorf("tx verify error, batch num mismatch, received: %d,expected:%d", len(resultTasks), len(verifyBatchs))
	}
	for i := 0; i < len(resultTasks); i++ {
		batch := resultTasks[i]
		if len(batch.txs) != len(batch.txHash) {
			return nil, nil, errTxs, fmt.Errorf("tx verify error, txs in batch mismatch, received: %d, expected:%d", len(batch.txHash), len(batch.txs))
		}
		txHashes = append(txHashes, batch.txHash...)
		txNewAdd = append(txNewAdd, batch.newAddTxs...)

	}
	return txHashes, txNewAdd, nil, nil
}

func validateTx(base *verifyTxBase, tx *commonpb.Transaction, stat *verifyStat,
	txsRet map[string]*commonpb.Transaction, txsHeightRet map[string]int64,
	newAddTxs []*commonpb.Transaction, ) error {
	txInPool, existTx := txsRet[tx.Header.TxId]
	//blockHeight := txsHeightRet[tx.Header.TxId]
	if existTx {
		//if consensuspb.ConsensusType_HOTSTUFF == consensusType && blockHeight != block.Header.BlockHeight && blockHeight > 0 {
		//	err := fmt.Errorf("tx duplicate in pending (tx:%s), txInPoolHeight:%d, txInBlockHeight:%d",
		//		tx.Header.TxId, blockHeight, block.Header.BlockHeight)
		//	return err
		//}
		if err := isTxHashValid(tx, txInPool, base.hashType); err != nil {
			return err
		}
		return nil
	}
	startDBTicker := utils.CurrentTimeMillisSeconds()
	isExist, err := base.store.TxExists(tx.Header.TxId)
	stat.dbLasts += utils.CurrentTimeMillisSeconds() - startDBTicker
	if err != nil || isExist {
		err = fmt.Errorf("tx duplicate in DB (tx:%s)", tx.Header.TxId)
		return err
	}
	stat.sigCount++
	startSigTicker := utils.CurrentTimeMillisSeconds()
	// if tx in txpool, means tx has already validated. tx noIt in txpool, need to validate.
	if err := utils.VerifyTxWithoutPayload(tx, base.chainId, base.ac); err != nil {
		err = fmt.Errorf("acl error (tx:%s), %s", tx.Header.TxId, err.Error())
		return err
	}
	stat.sigLasts += utils.CurrentTimeMillisSeconds() - startSigTicker
	// tx valid and put into txpool
	newAddTxs = append(newAddTxs, tx)

	return nil
}
