/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"chainmaker.org/chainmaker-go/common/bitmap"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/hex"
	"sync"
)

type Merger struct {
	rwSetMap      map[string]*commonpb.TxRWSet
	lock          sync.Mutex
	txBatchInfo   map[string]*TxBatchInfo // key -> BatchId
	baseTxBatchID string
	allTxsMap     map[string]*commonpb.Transaction // record all transaction(txId->Transaction)
	log           *logger.CMLogger                 // logger
}

type TxBatchInfo struct {
	txBatch  *commonpb.Block
	rwSetMap map[string]*commonpb.TxRWSet // key->txId
}

func NewMerger() *Merger {
	return &Merger{
		lock:        sync.Mutex{},
		txBatchInfo: make(map[string]*TxBatchInfo),
		allTxsMap:   make(map[string]*commonpb.Transaction),
	}
}

func (m *Merger) Merge(block *commonpb.Block, txBatchIDList []string) error {
	baseTxBatch := m.txBatchInfo[m.baseTxBatchID].txBatch
	baseRWSetMap := m.txBatchInfo[m.baseTxBatchID].rwSetMap
	// init baseRWSetMap if empty
	if baseRWSetMap == nil {
		baseRWSetMap = make(map[string]*commonpb.TxRWSet)
	}
	baseWriteTable := getBaseWriteTable(baseRWSetMap)

	// record baseTxBatch 's tx
	for _, tx := range baseTxBatch.Txs {
		m.allTxsMap[tx.Header.TxId] = tx
	}

	// merge Tx start with the second txBatch
	for i := 1; i < len(txBatchIDList); i++ {
		// get txBatch 's info
		txBatchID := txBatchIDList[i]
		txBatch := m.txBatchInfo[txBatchID].txBatch
		rwSetMap := m.txBatchInfo[txBatchID].rwSetMap

		// merge txBatch(Txs and RWSetMap)
		m.doMerge(
			baseTxBatch,
			txBatch,
			baseRWSetMap,
			rwSetMap,
			baseWriteTable)

		// rebuild dag for new RWSetMap
		baseTxBatch.Dag = m.buildDAG(baseTxBatch, baseRWSetMap)
	}

	// edit block
	block.Txs = baseTxBatch.Txs
	block.Dag = baseTxBatch.Dag

	// set rwSetMap
	m.rwSetMap = baseRWSetMap

	return nil
}

func (m *Merger) doMerge(
	baseTxBatch,
	txBatch *commonpb.Block,
	baseRWSetMap,
	rwSetMap map[string]*commonpb.TxRWSet,
	baseWriteTable map[string]struct{}) {

	failTxWriteTable := make(map[string]struct{})
	repeatTx := 0
	for _, tx := range txBatch.Txs {
		txId := tx.Header.TxId
		rwSet := rwSetMap[txId]
		// discard repeat tx
		if _, ok := m.allTxsMap[txId]; ok {
			repeatTx++
			if tx.Result.Code == commonpb.TxStatusCode_SUCCESS {
				updateWriteTable(failTxWriteTable, rwSet)
			}
			continue
		}

		if tx.Result.Code == commonpb.TxStatusCode_SUCCESS {
			if ifConflict(rwSet, baseWriteTable, failTxWriteTable) {
				// modify conflict tx
				modifyTxResult(tx)

				updateWriteTable(failTxWriteTable, rwSet)
				rwSet = modifyTxRWSet(txId)
			}
		}

		// merge RWSetMap
		baseRWSetMap[txId] = rwSet

		// merge Tx
		baseTxBatch.Txs = append(baseTxBatch.Txs, tx)

		// update allTxsMap
		m.allTxsMap[txId] = tx
	}
	m.log.Debugf("branchId[%s], repeatTx[%d]", hex.EncodeToString(txBatch.Header.BlockHash), repeatTx)
}

func (m *Merger) getRepeatTx(txBatchID string) ([]int, map[string]struct{}) {

	// record the deleted & repeated transaction(BatchID->deleted transaction 's position)
	repeatTxIndexs := make([]int, 0)
	repeatTxIDMap := make(map[string]struct{})

	if info, ok := m.txBatchInfo[txBatchID]; ok {
		txs := info.txBatch.Txs
		for i, _ := range txs {
			txID := txs[i].Header.TxId

			// set all Transaction to a Map(txId=>tx)
			if _, ok := m.allTxsMap[txID]; !ok {
				m.allTxsMap[txID] = txs[i]
			} else {
				repeatTxIndexs = append(repeatTxIndexs, i)
				repeatTxIDMap[txID] = struct{}{}
			}
		}
	}

	return repeatTxIndexs, repeatTxIDMap
}

func (m *Merger) buildDAG(txBatch *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) *commonpb.DAG {
	m.lock.Lock()
	defer m.lock.Unlock()

	txCount := len(txBatch.Txs)
	m.log.Debugf("start building DAG for block %d with %d txs", txBatch.Header.BlockHeight, txCount)
	txRWSetTable := make([]*commonpb.TxRWSet, 0)
	for _, tx := range txBatch.Txs {
		txRWSetTable = append(txRWSetTable, rwSetMap[tx.Header.TxId])
	}

	// build read-write bitmap for all transactions
	readBitmaps, writeBitmaps := buildRWBitmaps(txCount, txRWSetTable)
	cumulativeReadBitmap, cumulativeWriteBitmap := buildCumulativeBitmap(readBitmaps, writeBitmaps)

	dag := &commonpb.DAG{}
	if txCount == 0 {
		return dag
	}

	dag.Vertexes = make([]*commonpb.DAG_Neighbor, txCount)

	// build DAG base on read and write bitmaps
	// reachMap describes reachability from tx i to tx j in DAG.
	// For example, if the DAG is tx3 -> tx2 -> tx1 -> begin, the reachMap is
	// 		tx1		tx2		tx3
	// tx1	0		0		0
	// tx2	1		0		0
	// tx3	1		1		0
	reachMap := make([]*bitmap.Bitmap, txCount)
	for i := 0; i < txCount; i++ {
		// 1、get read and write bitmap for tx i
		readBitmapForI := readBitmaps[i]
		writeBitmapForI := writeBitmaps[i]

		// directReach is used to build DAG
		// reach is used to save reachability we have already known
		directReachFromI := &bitmap.Bitmap{}
		reachFromI := &bitmap.Bitmap{}
		reachFromI.Set(i)

		if i > 0 && fastConflicted(readBitmapForI, writeBitmapForI, cumulativeReadBitmap[i-1], cumulativeWriteBitmap[i-1]) {
			// check reachability one by one, then build table
			buildReach(i, reachFromI, readBitmaps, writeBitmaps, readBitmapForI, writeBitmapForI, directReachFromI, reachMap)
		}
		reachMap[i] = reachFromI

		// build DAG based on directReach bitmap
		dag.Vertexes[i] = &commonpb.DAG_Neighbor{
			Neighbors: make([]int32, 0, 16),
		}
		for _, j := range directReachFromI.Pos1() {
			dag.Vertexes[i].Neighbors = append(dag.Vertexes[i].Neighbors, int32(j))
		}
	}
	m.log.Debugf("build DAG for block %d finished", txBatch.Header.BlockHeight)
	return dag

}

func ifConflict(rwSet *commonpb.TxRWSet, writeTable, failTxWriteTable map[string]struct{}) bool {
	return isWRConflict(rwSet, writeTable) || isWRConflict(rwSet, failTxWriteTable)
}

func getRepeatTxIndexFromBaseBatch(baseTxBatch *commonpb.Block, repeatTxMap map[string]struct{}) map[string]int {
	repeatTxIndexInBaseBatch := make(map[string]int, 0)
	for index, tx := range baseTxBatch.Txs {
		txId := tx.Header.TxId
		if _, ok := repeatTxMap[txId]; ok {
			repeatTxIndexInBaseBatch[txId] = index
		}
	}
	return repeatTxIndexInBaseBatch
}

func constructKey(contractName string, key []byte) string {
	return contractName + string(key)
}

func getBaseWriteTable(rwSetMap map[string]*commonpb.TxRWSet) map[string]struct{} {
	writeTable := make(map[string]struct{})
	for _, rwSet := range rwSetMap {
		for _, txWrite := range rwSet.TxWrites {
			finalKey := constructKey(txWrite.ContractName, txWrite.Key)
			writeTable[finalKey] = struct{}{}
		}
	}
	return writeTable
}

func updateWriteTable(writeTable map[string]struct{}, rwSet *commonpb.TxRWSet) {
	for _, txWrite := range rwSet.TxWrites {
		finalKey := constructKey(txWrite.ContractName, txWrite.Key)
		writeTable[finalKey] = struct{}{}
	}
}

func getWriteTable(rwSet *commonpb.TxRWSet) map[string]struct{} {
	writeTable := make(map[string]struct{})
	for _, txWrite := range rwSet.TxWrites {
		finalKey := constructKey(txWrite.ContractName, txWrite.Key)
		writeTable[finalKey] = struct{}{}
	}
	return writeTable
}

func isWRConflict(rwSet *commonpb.TxRWSet, writeTable map[string]struct{}) bool {
	for _, txRead := range rwSet.TxReads {
		finalKey := constructKey(txRead.ContractName, txRead.Key)
		// check if RWSet conflict
		if _, ok := writeTable[finalKey]; ok {
			return true
		}
	}
	return false
}

func isWWConflict(rwSet *commonpb.TxRWSet, writeTable map[string]struct{}) bool {
	for _, txWrite := range rwSet.TxWrites {
		finalKey := constructKey(txWrite.ContractName, txWrite.Key)
		// check if RWSet conflict
		if _, ok := writeTable[finalKey]; ok {
			return true
		}
	}
	return false
}

func modifyTxResult(tx *commonpb.Transaction) {
	tx.Result = &commonpb.Result{
		Code: commonpb.TxStatusCode_CONTRACT_FAIL,
		ContractResult: &commonpb.ContractResult{
			Code:    commonpb.ContractResultCode_FAIL,
			Result:  nil,
			Message: "Transaction conflict",
		},
		RwSetHash: nil,
	}
}

func modifyTxRWSet(txId string) *commonpb.TxRWSet {
	return &commonpb.TxRWSet{
		TxId:     txId,
		TxReads:  make([]*commonpb.TxRead, 0),
		TxWrites: make([]*commonpb.TxWrite, 0),
	}
}

func buildRWBitmaps(txCount int, txRWSetTable []*commonpb.TxRWSet) ([]*bitmap.Bitmap, []*bitmap.Bitmap) {
	dictIndex := 0
	readBitmap := make([]*bitmap.Bitmap, txCount)
	writeBitmap := make([]*bitmap.Bitmap, txCount)
	keyDict := make(map[string]int, 1024)
	for i := 0; i < txCount; i++ {
		readTableItemForI := txRWSetTable[i].TxReads
		writeTableItemForI := txRWSetTable[i].TxWrites

		readBitmap[i] = &bitmap.Bitmap{}
		for _, keyForI := range readTableItemForI {
			if existIndex, ok := keyDict[string(keyForI.Key)]; !ok {
				keyDict[string(keyForI.Key)] = dictIndex
				readBitmap[i].Set(dictIndex)
				dictIndex++
			} else {
				readBitmap[i].Set(existIndex)
			}
		}

		writeBitmap[i] = &bitmap.Bitmap{}
		for _, keyForI := range writeTableItemForI {
			if existIndex, ok := keyDict[string(keyForI.Key)]; !ok {
				keyDict[string(keyForI.Key)] = dictIndex
				writeBitmap[i].Set(dictIndex)
				dictIndex++
			} else {
				writeBitmap[i].Set(existIndex)
			}
		}
	}
	return readBitmap, writeBitmap
}

func buildCumulativeBitmap(readBitmap []*bitmap.Bitmap, writeBitmap []*bitmap.Bitmap) ([]*bitmap.Bitmap, []*bitmap.Bitmap) {
	cumulativeReadBitmap := make([]*bitmap.Bitmap, len(readBitmap))
	cumulativeWriteBitmap := make([]*bitmap.Bitmap, len(writeBitmap))

	for i, b := range readBitmap {
		cumulativeReadBitmap[i] = b.Clone()
		if i > 0 {
			cumulativeReadBitmap[i].Or(cumulativeReadBitmap[i-1])
		}
	}
	for i, b := range writeBitmap {
		cumulativeWriteBitmap[i] = b.Clone()
		if i > 0 {
			cumulativeWriteBitmap[i].Or(cumulativeWriteBitmap[i-1])
		}
	}
	return cumulativeReadBitmap, cumulativeWriteBitmap
}

// fast conflict cases: I read & J write; I write & J read; I write & J write
func fastConflicted(readBitmapForI, writeBitmapForI, cumulativeReadBitmap, cumulativeWriteBitmap *bitmap.Bitmap) bool {
	if readBitmapForI.InterExist(cumulativeWriteBitmap) || writeBitmapForI.InterExist(cumulativeWriteBitmap) || writeBitmapForI.InterExist(cumulativeReadBitmap) {
		return true
	}
	return false
}

func buildReach(i int, reachFromI *bitmap.Bitmap,
	readBitmaps []*bitmap.Bitmap, writeBitmaps []*bitmap.Bitmap,
	readBitmapForI *bitmap.Bitmap, writeBitmapForI *bitmap.Bitmap,
	directReachFromI *bitmap.Bitmap, reachMap []*bitmap.Bitmap) {

	for j := i - 1; j >= 0; j-- {
		if reachFromI.Has(j) {
			continue
		}

		readBitmapForJ := readBitmaps[j]
		writeBitmapForJ := writeBitmaps[j]
		if conflicted(readBitmapForI, writeBitmapForI, readBitmapForJ, writeBitmapForJ) {
			directReachFromI.Set(j)
			reachFromI.Or(reachMap[j])
		}
	}
}

// Conflict cases: I read & J write; I write & J read; I write & J write
func conflicted(readBitmapForI, writeBitmapForI, readBitmapForJ, writeBitmapForJ *bitmap.Bitmap) bool {
	if readBitmapForI.InterExist(writeBitmapForJ) || writeBitmapForI.InterExist(writeBitmapForJ) || writeBitmapForI.InterExist(readBitmapForJ) {
		return true
	}
	return false
}
