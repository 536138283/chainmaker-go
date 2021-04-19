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
	txBatchInfo   map[string]*TxBatchInfo // key -> BrtchId
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
	// set base TxBrtch Id
	m.baseTxBatchID = m.txBatchIDList[0]
	baseTxBrtch := m.txBatchInfo[m.baseTxBatchID].txBatch
	baseRWSetMap := m.txBatchInfo[m.baseTxBatchID].rwSetMap
	//baseWriteTable := getWriteTable(baseRWSetMap)

	if len(m.txBatchIDList) == 1 {
		m.block.Txs = baseTxBrtch.Txs
		m.block.Dag = baseTxBrtch.Dag

		for _, tx := range baseTxBrtch.Txs {
			m.allTxsMap[tx.Header.TxId] = tx
		}

		m.rwSetMap = baseRWSetMap

		return nil
	}

	repeatTxMap, baseTxTable := m.prepare()
	for i := 0; i < len(m.txBatchIDList); i++ {
		txBrtchID := m.txBatchIDList[i]
		txBrtch := m.txBatchInfo[txBrtchID].txBatch
		repeatedTxIndexs := repeatTxMap[txBrtchID]
		rwSetMap := m.txBatchInfo[txBrtchID].rwSetMap

		// 将依赖该重复交易集中的交易的执行结果置为失败
		conflictRepeatedTxMap := make(map[int][]int)
		// get the releated tx for repeated tx
		findReleatedTxForRepeatedTx(
			repeatedTxIndexs,
			txBrtch.Txs,
			rwSetMap)

		// reWrite releated Tx 's result
		m.modifyRleatedTxResult(
			conflictRepeatedTxMap,
			txBrtch,
			baseTxBrtch,
			baseTxTable,
			txBrtchID,
			baseRWSetMap,
			rwSetMap)

		// delete repeated Txs
		deleteRepeatedTx(
			repeatedTxIndexs,
			rwSetMap,
			txBrtch)

		// merge txBrtch(Txs and RWSetMap)
		mergeTxsAndRWSetMap(
			baseTxBrtch,
			txBrtch,
			baseRWSetMap,
			rwSetMap)
	}

	dag := m.buildDAG(baseTxBrtch, baseRWSetMap)

	// edit block
	m.block.Txs = baseTxBrtch.Txs
	m.block.Dag = dag

	// set rwSetMap
	m.rwSetMap = baseRWSetMap

	return nil
}

func mergeTxsAndRWSetMap(baseTxBrtch, txBrtch *commonpb.Block, baseRWSetMap, rwSetMap map[string]*commonpb.TxRWSet) {
	// merge Txs
	baseTxBrtch.Txs = append(baseTxBrtch.Txs, txBrtch.Txs...)

	// merge RWSetMap
	for txId, rwSet := range rwSetMap {
		baseRWSetMap[txId] = rwSet
	}
}

func constructKey(contractName string, key []byte) string {
	return contractName + string(key)
}

// 拿到重复交易(key=>BrtchID, value=>重复交易的下标)以及第一个批次的交易(key=>txId, value=>交易的下标)的数据集
func (m *Merger) prepare() (map[string][]int, map[string]int) {

	// record the deleted & repeated transaction(BrtchID->deleted transaction 's position)
	repeatTrans := make(map[string][]int)
	baseTxIndexTable := make(map[string]int)
	for _, txBrtchID := range m.txBatchIDList {
		if info, ok := m.txBatchInfo[txBrtchID]; ok {
			txs := info.txBatch.Txs
			for i, _ := range txs {
				txID := txs[i].Header.TxId
				if txBrtchID == m.baseTxBatchID {
					baseTxIndexTable[txID] = i
				}
				if _, ok := m.allTxsMap[txID]; !ok {
					m.allTxsMap[txID] = txs[i]
				} else {
					repeatTrans[txBrtchID] = append(repeatTrans[txBrtchID], i)
				}
			}
		}
	}
	return repeatTrans, baseTxIndexTable
}

func getWriteTable(rwSet *commonpb.TxRWSet) map[string]struct{} {
	writeTable := make(map[string]struct{})
	for _, txWrite := range rwSet.TxWrites {
		finalKey := constructKey(txWrite.ContractName, txWrite.Key)
		writeTable[finalKey] = struct{}{}
	}
	return writeTable
}

func deleteRepeatedTx(repeatedTxIndexs []int, rwSetMap map[string]*commonpb.TxRWSet, txBrtch *commonpb.Block) {
	for _, repeatedTxIndex := range repeatedTxIndexs {
		// delete repeated tx from rwSetMap
		delete(rwSetMap, txBrtch.Txs[repeatedTxIndex].Header.TxId)

		// delete repeated tx from Txs
		deleteRepeatedTxFromTxs(repeatedTxIndex, txBrtch)
	}
}

func deleteRepeatedTxFromTxs(repeatedTxIndex int, txBrtch *commonpb.Block) {
	switch repeatedTxIndex {
	case 0:
		txBrtch.Txs = txBrtch.Txs[repeatedTxIndex+1:]
	case len(txBrtch.Txs):
		txBrtch.Txs = txBrtch.Txs[:repeatedTxIndex]
	default:
		txBrtch.Txs = append(txBrtch.Txs[:repeatedTxIndex], txBrtch.Txs[repeatedTxIndex+1:]...)
	}
}

func (m *Merger) modifyRleatedTxResult(
	conflictRepeatedTxMap map[int][]int,
	txBrtch, baseTxBrtch *commonpb.Block,
	baseTxTable map[string]int, BrtchID string,
	baseRWSetMap, rwSetMap map[string]*commonpb.TxRWSet) {

	for repeatedTxIndex, releatedTxIndexs := range conflictRepeatedTxMap {
		txId := txBrtch.Txs[repeatedTxIndex].Header.TxId
		if ok := ifNeedModify(
			txBrtch,
			baseTxBrtch,
			repeatedTxIndex,
			baseTxTable[txId],
			baseRWSetMap,
			rwSetMap); ok {

			for _, releatedTxIndex := range releatedTxIndexs {
				releatedTx := txBrtch.Txs[releatedTxIndex]
				releatedTxId := releatedTx.Header.TxId
				modifyTxResult(releatedTx)
				modifyTxRWSet(m.txBatchInfo[BrtchID].rwSetMap, releatedTxId)
			}
		}
	}
}

func findReleatedTxForRepeatedTx(repeatedTxIndexs []int, txs []*commonpb.Transaction, rwSetMap map[string]*commonpb.TxRWSet) map[int][]int {
	releatedTxMap := make(map[int][]int)
	for _, repeatedTxIndex := range repeatedTxIndexs {
		repeatedTx := txs[repeatedTxIndex]
		repeatedTxRWSet := rwSetMap[repeatedTx.Header.TxId]

		// get repeated tx 's write table
		repeatedTxWriteTable := getWriteTable(repeatedTxRWSet)

		getReleatedTxMap(txs, rwSetMap, repeatedTxWriteTable, releatedTxMap, repeatedTxIndex)
	}

	return releatedTxMap
}

func getReleatedTxMap(
	txs []*commonpb.Transaction,
	rwSetMap map[string]*commonpb.TxRWSet,
	repeatedTxWriteTable map[string]struct{},
	releatedTxMap map[int][]int, index int) {

	for j := index + 1; j <= len(txs)-1; j++ {
		tx := txs[j]
		txId := tx.Header.TxId
		rwSet := rwSetMap[txId]
		for _, txRead := range rwSet.TxReads {
			finalKey := constructKey(txRead.ContractName, txRead.Key)
			// check if RWSet conflict
			if _, ok := repeatedTxWriteTable[finalKey]; ok {
				releatedTxMap[index] = append(releatedTxMap[index], j)
			}
		}
	}
}

func ifNeedModify(
	txBrtch, baseTxBrtch *commonpb.Block,
	repeatedTxIndex, baseTxIndex int,
	baseRWSetMap, rwSetMap map[string]*commonpb.TxRWSet) bool {

	writeTable := make(map[string]struct{})

	// get the write table from repeated tx 's neighbors(from ownTxBrtch)
	for _, neighbor := range txBrtch.Dag.Vertexes[repeatedTxIndex].Neighbors {
		tx := txBrtch.Txs[int(neighbor)]
		writeTable = getWriteTable(rwSetMap[tx.Header.TxId])
	}

	// get the write table from repeated tx 's neighbors(from baseTxBrtch)
	for _, neighbor := range baseTxBrtch.Dag.Vertexes[baseTxIndex].Neighbors {
		tx := baseTxBrtch.Txs[int(neighbor)]
		writeTable = getWriteTable(baseRWSetMap[tx.Header.TxId])
	}

	tx := txBrtch.Txs[repeatedTxIndex]
	txId := tx.Header.TxId
	rwSet := rwSetMap[txId]
	for _, txRead := range rwSet.TxReads {
		finalKey := constructKey(txRead.ContractName, txRead.Key)
		// check if RWSet conflict
		if _, ok := writeTable[finalKey]; ok {
			return true
		}
	}

	return false
}

func getWriteTableFor() {

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

func (m *Merger) buildDAG(txBrtch *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) *commonpb.DAG {
	m.lock.Lock()
	defer m.lock.Unlock()

	txCount := len(txBrtch.Txs)
	log.Debugf("start building DAG for block %d with %d txs", m.block.Header.BlockHeight, txCount)

	txRWSetTable := make([]*commonpb.TxRWSet, txCount)
	for i, tx := range txBrtch.Txs {
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
