/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"chainmaker.org/chainmaker-go/common/bitmap"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"github.com/prometheus/common/log"
	"sync"
)

type Merger struct {
	block         *commonpb.Block // the commited block
	rwSetMap      map[string]*commonpb.TxRWSet
	lock          sync.Mutex
	txBatchInfo   map[string]*TxBatchInfo // key -> BatchId
	txBatchIDList []string
	baseTxBatchID string
	allTxsMap     map[string]*commonpb.Transaction // record all transaction(txId->Transaction)
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

func (m *Merger) Merge() error {
	// set base TxBatch Id
	m.baseTxBatchID = m.txBatchIDList[0]
	baseTxBatch := m.txBatchInfo[m.baseTxBatchID].txBatch
	baseRWSetMap := m.txBatchInfo[m.baseTxBatchID].rwSetMap
	baseWriteTable := getBaseWriteTable(baseRWSetMap)

	// record baseTxBatch 's tx
	for _, tx := range baseTxBatch.Txs {
		m.allTxsMap[tx.Header.TxId] = tx
	}

	// merge Tx start with the second txBatch
	for i := 1; i < len(m.txBatchIDList); i++ {
		txBatchID := m.txBatchIDList[i]
		txBatch := m.txBatchInfo[txBatchID].txBatch

		// get repeat Tx compare to the baseTxBatch
		repeatTxsMap, repeatTxIDMap := m.getRepeatTx(txBatchID)
		repeatedTxIndexs := repeatTxsMap[txBatchID]
		rwSetMap := m.txBatchInfo[txBatchID].rwSetMap

		repeatTxIndexFromBaseBatch := getRepeatTxIndexFromBaseBatch(baseTxBatch, repeatTxIDMap)

		conflictRepeatedTxMap := findReliantTxForRepeatedTx(
			repeatedTxIndexs,
			txBatch.Txs,
			rwSetMap, baseRWSetMap, repeatTxIndexFromBaseBatch, txBatch, baseTxBatch)

		// merge txBatch(Txs and RWSetMap)
		m.doMerge(
			baseTxBatch,
			txBatch,
			baseRWSetMap,
			rwSetMap,
			baseWriteTable,
			repeatedTxIndexs,
			conflictRepeatedTxMap)

		// rebuild dag for new RWSetMap
		baseTxBatch.Dag = m.buildDAG(baseTxBatch, baseRWSetMap)
	}

	// edit block
	m.block.Txs = baseTxBatch.Txs
	m.block.Dag = baseTxBatch.Dag

	// set rwSetMap
	m.rwSetMap = baseRWSetMap

	return nil
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

func (m *Merger) getRepeatTx(txBatchID string) (map[string][]int, map[string]struct{}) {

	// record the deleted & repeated transaction(BatchID->deleted transaction 's position)
	repeatTxs := make(map[string][]int)
	repeatTxIDMap := make(map[string]struct{})

	if info, ok := m.txBatchInfo[txBatchID]; ok {
		txs := info.txBatch.Txs
		for i, _ := range txs {
			txID := txs[i].Header.TxId

			// set all Transaction to a Map(txId=>tx)
			if _, ok := m.allTxsMap[txID]; !ok {
				m.allTxsMap[txID] = txs[i]
			} else {
				repeatTxs[txBatchID] = append(repeatTxs[txBatchID], i)
				repeatTxIDMap[txID] = struct{}{}
			}
		}
	}

	return repeatTxs, repeatTxIDMap
}

func (m *Merger) buildDAG(txBatch *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) *commonpb.DAG {
	m.lock.Lock()
	defer m.lock.Unlock()

	txCount := len(txBatch.Txs)
	log.Debugf("start building DAG for block %d with %d txs", m.block.Header.BlockHeight, txCount)

	txRWSetTable := make([]*commonpb.TxRWSet, txCount)
	for i, tx := range txBatch.Txs {
		txRWSetTable[i] = rwSetMap[tx.Header.TxId]
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
	log.Debugf("build DAG for block %d finished", m.block.Header.BlockHeight)
	return dag

}

func (m *Merger) doMerge(
	baseTxBatch,
	txBatch *commonpb.Block,
	baseRWSetMap,
	rwSetMap map[string]*commonpb.TxRWSet,
	baseWriteTable map[string]map[string]struct{},
	repeatedTxIndexs []int,
	conflictRepeatedTxMap map[int][]int) {

	repeatedTxIndexMap := getRepeatedTxIndexMap(repeatedTxIndexs)
	reliantTxIndexMap := getReliantTxIndexMap(conflictRepeatedTxMap)
	failTxWriteTable := make(map[string]struct{})
	newWriteTable := make(map[string]struct{})
	for index, tx := range txBatch.Txs {
		// merge
		if _, ok := repeatedTxIndexMap[index]; !ok {
			txId := tx.Header.TxId
			rwSet := rwSetMap[txId]

			//range read & panduan if

			for _, txRead := range rwSet.TxReads {
				finalKey := constructKey(txRead.ContractName, txRead.Key)
				// check if RWSet conflict
				if ifNeedModify(
					index,
					finalKey,
					baseWriteTable,
					failTxWriteTable,
					reliantTxIndexMap, txBatch) {

					// modify conflict tx
					modifyTxResult(tx)

					updateWriteTable(failTxWriteTable, rwSet)
					rwSet = getEmptyRWSet(txId)
				} else {
					updateWriteTable(newWriteTable, rwSet)
				}
			}
			// merge RWSetMap
			baseRWSetMap[txId] = rwSet

			// merge Tx
			baseTxBatch.Txs = append(baseTxBatch.Txs, tx)
		}
	}
}
func ifNeedModify(
	index int,
	finalKey string,
	baseWriteTable map[string]map[string]struct{},
	failTxWriteTable map[string]struct{},
	reliantTxIndexMap map[int]struct{}, txBatch *commonpb.Block) bool {

	if txIdMap, ok := baseWriteTable[finalKey]; ok {
		if len(txBatch.Dag.Vertexes[index].Neighbors) == 0 {
			return true
		}

		for _, neighbor := range txBatch.Dag.Vertexes[index].Neighbors {
			neighborTxId := txBatch.Txs[int(neighbor)].Header.TxId
			if _, ok = txIdMap[neighborTxId]; !ok {
				return true
			}
		}
	}

	if _, ok := failTxWriteTable[finalKey]; ok {
		return true
	}
	if _, ok := reliantTxIndexMap[index]; ok {
		return true
	}

	return false
}

func getReliantTxIndexMap(conflictRepeatedTxMap map[int][]int) map[int]struct{} {
	reliantTxIndexMap := make(map[int]struct{})
	for _, reliantTxIndexs := range conflictRepeatedTxMap {
		for _, reliantTxIndex := range reliantTxIndexs {
			reliantTxIndexMap[reliantTxIndex] = struct{}{}
		}
	}
	return reliantTxIndexMap
}

func getRepeatedTxIndexMap(repeatedTxIndexs []int) map[int]struct{} {
	repeatedTxIndexMap := make(map[int]struct{})
	for _, index := range repeatedTxIndexs {
		repeatedTxIndexMap[index] = struct{}{}
	}
	return repeatedTxIndexMap
}

func constructKey(contractName string, key []byte) string {
	return contractName + string(key)
}

func getBaseWriteTable(rwSetMap map[string]*commonpb.TxRWSet) map[string]map[string]struct{} {
	writeTable := make(map[string]map[string]struct{})
	for _, rwSet := range rwSetMap {
		for _, txWrite := range rwSet.TxWrites {
			finalKey := constructKey(txWrite.ContractName, txWrite.Key)
			txIdMap := make(map[string]struct{})
			txIdMap[rwSet.TxId] = struct{}{}
			writeTable[finalKey] = txIdMap
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
func findReliantTxForRepeatedTx(
	repeatedTxIndexs []int,
	txs []*commonpb.Transaction,
	rwSetMap, baseRWSetMap map[string]*commonpb.TxRWSet,
	repeatedTxIndexsInBaseBatch map[string]int, txBatch, baseTxBatch *commonpb.Block) map[int][]int {

	reliantTxMap := make(map[int][]int)
	for _, repeatedTxIndex := range repeatedTxIndexs {
		repeatedTx := txs[repeatedTxIndex]
		repeatedTxRWSet := rwSetMap[repeatedTx.Header.TxId]

		// get repeated tx 's write table
		repeatedTxWriteTable := getWriteTable(repeatedTxRWSet)

		getReliantTxMap(
			repeatedTxIndex,
			repeatedTx.Header.TxId,
			txs, rwSetMap, baseRWSetMap,
			repeatedTxWriteTable, reliantTxMap,
			repeatedTxIndexsInBaseBatch, txBatch, baseTxBatch)
	}

	return reliantTxMap
}

func getReliantTxMap(
	index int, repeatedTxId string,
	txs []*commonpb.Transaction, rwSetMap,
	baseRWSetMap map[string]*commonpb.TxRWSet,
	repeatedTxWriteTable map[string]struct{},
	reliantTxMap map[int][]int,
	repeatedTxIndexsInBaseBatch map[string]int, txBatch, baseTxBatch *commonpb.Block) {

	if isSpecialRepeatTx(
		index, repeatedTxId,
		rwSetMap, baseRWSetMap,
		repeatedTxIndexsInBaseBatch, txBatch, baseTxBatch) {
		return
	}

	for j := index + 1; j <= len(txs)-1; j++ {
		tx := txs[j]
		txId := tx.Header.TxId
		rwSet := rwSetMap[txId]
		if isRWConflict(rwSet, repeatedTxWriteTable) {
			reliantTxMap[index] = append(reliantTxMap[index], j)
		}
	}
}

func isSpecialRepeatTx(
	index int, repeatedTxId string,
	rwSetMap, baseRWSetMap map[string]*commonpb.TxRWSet,
	repeatedTxIndexsInBaseBatch map[string]int,
	txBatch, baseTxBatch *commonpb.Block) bool {

	baseRepeatTxIndex := repeatedTxIndexsInBaseBatch[repeatedTxId]
	// if releated In base
	baseNeighborWriteTable := getNeighborWriteTable(baseTxBatch, baseRWSetMap, baseRepeatTxIndex)
	if isRWConflict(baseRWSetMap[repeatedTxId], baseNeighborWriteTable) {
		return false
	}
	// if releatrd in self batch
	neighborWriteTable := getNeighborWriteTable(txBatch, rwSetMap, index)
	if isRWConflict(rwSetMap[repeatedTxId], neighborWriteTable) {
		return false
	}
	baseRepeatTxWriteTable := getWriteTable(baseRWSetMap[repeatedTxId])
	if isExitsWWReliantTx(baseRepeatTxIndex+1, len(baseTxBatch.Txs), baseRepeatTxWriteTable, baseTxBatch.Txs, baseRWSetMap) {
		return false
	}
	return true
}

func isExitsWWReliantTx(
	index, baseTxBatchLen int,
	writeTable map[string]struct{},
	txs []*commonpb.Transaction, rwSetMap map[string]*commonpb.TxRWSet) bool {
	for i := index; i <= baseTxBatchLen-1; i++ {
		tx := txs[i]
		txId := tx.Header.TxId
		rwSet := rwSetMap[txId]
		if isWWConflict(rwSet, writeTable) {
			return true
		}
	}
	return false
}

func getNeighborWriteTable(txBatch *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet, index int) map[string]struct{} {
	neighborWriteTable := make(map[string]struct{})
	for _, neighbor := range txBatch.Dag.Vertexes[index].Neighbors {
		neighborTxId := txBatch.Txs[int(neighbor)].Header.TxId
		neighborWriteTable = getWriteTable(rwSetMap[neighborTxId])
	}
	return neighborWriteTable
}

func isRWConflict(rwSet *commonpb.TxRWSet, writeTable map[string]struct{}) bool {
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
			Message: "Transaction conflic",
		},
		RwSetHash: nil,
	}
}

func modifyTxRWSet(rwSetMap map[string]*commonpb.TxRWSet, txId string) {
	rwSetMap[txId] = &commonpb.TxRWSet{
		TxId:     txId,
		TxReads:  nil,
		TxWrites: nil,
	}
}

func getEmptyRWSet(txId string) *commonpb.TxRWSet {
	return &commonpb.TxRWSet{
		TxId:     txId,
		TxReads:  nil,
		TxWrites: nil,
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
