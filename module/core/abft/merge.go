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

	if len(m.txBatchIDList) == 1 {
		m.block.Txs = baseTxBatch.Txs
		m.block.Dag = baseTxBatch.Dag
		for _, tx := range baseTxBatch.Txs {
			m.allTxsMap[tx.Header.TxId] = tx
		}
		m.rwSetMap = baseRWSetMap
		return nil
	}

	repeatTxMap := m.prepare()
	baseWriteTable := getBaseWriteTable(baseRWSetMap)
	for i := 1; i < len(m.txBatchIDList); i++ {
		txBatchID := m.txBatchIDList[i]
		txBatch := m.txBatchInfo[txBatchID].txBatch
		repeatedTxIndexs := repeatTxMap[txBatchID]
		rwSetMap := m.txBatchInfo[txBatchID].rwSetMap

		// get the releated tx for repeated tx
		conflictRepeatedTxMap := findReliantTxForRepeatedTx(
			repeatedTxIndexs,
			txBatch.Txs,
			rwSetMap)

		// merge txBatch(Txs and RWSetMap)
		m.doMerge(
			baseTxBatch,
			txBatch,
			baseRWSetMap,
			rwSetMap,
			baseWriteTable,
			repeatedTxIndexs,
			conflictRepeatedTxMap)
	}

	// rebuild dag for new RWSetMap
	dag := m.buildDAG(baseTxBatch, baseRWSetMap)

	// edit block
	m.block.Txs = baseTxBatch.Txs
	m.block.Dag = dag

	// set rwSetMap
	m.rwSetMap = baseRWSetMap

	return nil
}

// 拿到重复交易(key=>BatchID, value=>重复交易的下标)
func (m *Merger) prepare() map[string][]int {

	// record the deleted & repeated transaction(BatchID->deleted transaction 's position)
	repeatTrans := make(map[string][]int)
	for _, txBatchID := range m.txBatchIDList {
		if info, ok := m.txBatchInfo[txBatchID]; ok {
			txs := info.txBatch.Txs
			for i, _ := range txs {
				txID := txs[i].Header.TxId
				if _, ok := m.allTxsMap[txID]; !ok {
					m.allTxsMap[txID] = txs[i]
				} else {
					repeatTrans[txBatchID] = append(repeatTrans[txBatchID], i)
				}
			}
		}
	}
	return repeatTrans
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
	baseWriteTable map[string]struct{},
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
			for _, txRead := range rwSet.TxReads {
				finalKey := constructKey(txRead.ContractName, txRead.Key)
				// check if RWSet conflict
				if ifNeedModify(
					finalKey,
					index,
					baseWriteTable,
					failTxWriteTable,
					reliantTxIndexMap) {

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
	finalKey string,
	index int,
	baseWriteTable,
	failTxWriteTable map[string]struct{},
	reliantTxIndexMap map[int]struct{}) bool {

	_, ok1 := baseWriteTable[finalKey]
	_, ok2 := failTxWriteTable[finalKey]
	_, ok3 := reliantTxIndexMap[index]

	if ok1 || ok2 || ok3 {
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

func findReliantTxForRepeatedTx(repeatedTxIndexs []int, txs []*commonpb.Transaction, rwSetMap map[string]*commonpb.TxRWSet) map[int][]int {
	reliantTxMap := make(map[int][]int)
	for _, repeatedTxIndex := range repeatedTxIndexs {
		repeatedTx := txs[repeatedTxIndex]
		repeatedTxRWSet := rwSetMap[repeatedTx.Header.TxId]

		// get repeated tx 's write table
		repeatedTxWriteTable := getWriteTable(repeatedTxRWSet)

		getReliantTxMap(txs, rwSetMap, repeatedTxWriteTable, reliantTxMap, repeatedTxIndex)
	}

	return reliantTxMap
}

func getReliantTxMap(
	txs []*commonpb.Transaction,
	rwSetMap map[string]*commonpb.TxRWSet,
	repeatedTxWriteTable map[string]struct{},
	reliantTxMap map[int][]int, index int) {

	for j := index + 1; j <= len(txs)-1; j++ {
		tx := txs[j]
		txId := tx.Header.TxId
		rwSet := rwSetMap[txId]
		for _, txRead := range rwSet.TxReads {
			finalKey := constructKey(txRead.ContractName, txRead.Key)
			// check if RWSet conflict
			if _, ok := repeatedTxWriteTable[finalKey]; ok {
				reliantTxMap[index] = append(reliantTxMap[index], j)
			}
		}
	}
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
