/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package commiter

//
//import (
//	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
//	"fmt"
//	//"github.com/gogo/protobuf/proto"
//	"testing"
//
//	"chainmaker.org/chainmaker-go/core/cache"
//	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
//)
//
//var (
//	contractName = "testContract"
//	mode1        = "NoRepeatTx_NoConflic"
//	mode2        = "NoRepeatTx_HasConflic"
//	mode3        = "HasRepeatTx_NoConflic"
//	mode4        = "HasRepeatTx_HasConflic"
//)
//
//type testHelper struct {
//	testType string
//	block    *commonpb.Block
//	rwSetMap map[string]*commonpb.TxRWSet
//}
//
//func newVerifyHelper(block *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet, testType string) *testHelper {
//	return &testHelper{
//		testType: testType,
//		block:    block,
//		rwSetMap: rwSetMap,
//	}
//}
//
//func (vh *testHelper) checkMergeResults(neighborMap map[int][]int32, txNum int, successTxs map[string]struct{}) error {
//	block := vh.block
//	rwSetMap := vh.rwSetMap
//
//	if len(block.Txs) != txNum {
//		return fmt.Errorf("merge tx fail; wanted :%v, got :%v", txNum, len(block.Txs))
//	}
//
//	vertexesNum := 0
//	for i, v := range block.Dag.Vertexes {
//		if neighbor, ok := neighborMap[i]; ok {
//			if len(neighbor) != len(v.Neighbors) {
//				return fmt.Errorf("merger dag fail, wanted neighbor: %v, got: %v", neighbor, v.Neighbors)
//			}
//			vertexesNum++
//		}
//	}
//
//	if vertexesNum != len(neighborMap) {
//		return fmt.Errorf("merger dag fail, wanted dag: %v, got: %v", neighborMap, block.Dag)
//	}
//
//	txMap := make(map[string]struct{})
//	for _, v := range block.Txs {
//		if _, ok := txMap[v.Payload.TxId]; !ok {
//			txMap[v.Payload.TxId] = struct{}{}
//		} else {
//			return fmt.Errorf("merge tx fail, final Txs has the repeated tx.")
//		}
//	}
//
//	if len(rwSetMap) != txNum {
//		return fmt.Errorf("merge rwSetMap fail; wanted :%v, got :%v", txNum, len(rwSetMap))
//	}
//	for _, rwSet := range rwSetMap {
//		if _, ok := successTxs[rwSet.TxId]; ok {
//			err := vh.ifRWSetEmpty(rwSet)
//			if err != nil {
//				return err
//			}
//		} else {
//			if rwSet.TxId == "" {
//				return fmt.Errorf("merge rwSet fail, txId is empty")
//			}
//
//			if len(rwSet.TxReads) != 0 {
//				return fmt.Errorf("merge rwSet fail, TxReads should be empty")
//			}
//
//			if len(rwSet.TxWrites) != 0 {
//				return fmt.Errorf("merge rwSet fail, TxWrites should be empty")
//			}
//		}
//
//	}
//	return nil
//}
//
//func (vh *testHelper) ifRWSetEmpty(rwSet *commonpb.TxRWSet) error {
//	if rwSet.TxId == "" {
//		return fmt.Errorf("merge rwSet fail, txId is empty")
//	}
//
//	for _, writes := range rwSet.TxWrites {
//		if writes.ContractName == "" ||
//			len(writes.Key) == 0 ||
//			len(writes.Value) == 0 {
//			return fmt.Errorf("merge rwSet fail, TxWrites is empty")
//		}
//	}
//
//	for _, reads := range rwSet.TxReads {
//		if reads.ContractName == "" ||
//			len(reads.Key) == 0 ||
//			len(reads.Value) == 0 {
//			return fmt.Errorf("merge rwSet fail, TxReads is empty")
//		}
//	}
//
//	return nil
//}
//
//func TestMerger_Merge(t *testing.T) {
//
//	branchID1 := []byte("a")
//	branchID2 := []byte("b")
//	branchID3 := []byte("c")
//	branchID4 := []byte("d")
//
//	cach1, txList1 := addTxBatch_NoRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4)
//	neighborMap1 := make(map[int][]int32)
//	neighborMap1 = map[int][]int32{
//		0: []int32{}, 1: []int32{}, 2: []int32{}, 3: []int32{},
//		4: []int32{}, 5: []int32{}, 6: []int32{}, 7: []int32{},
//	}
//	successTxMap1 := make(map[string]struct{})
//	successTxMap1 = map[string]struct{}{
//		txList1[0].Payload.TxId: {}, txList1[1].Payload.TxId: {},
//		txList1[2].Payload.TxId: {}, txList1[3].Payload.TxId: {},
//		txList1[4].Payload.TxId: {}, txList1[5].Payload.TxId: {},
//		txList1[6].Payload.TxId: {}, txList1[7].Payload.TxId: {},
//	}
//	//
//	cach2, txList2 := addTxBatch_NoRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4)
//	neighborMap2 := make(map[int][]int32)
//	neighborMap2 = map[int][]int32{
//		0: []int32{}, 1: []int32{0}, 2: []int32{1}, 3: []int32{},
//		4: []int32{}, 5: []int32{}, 6: []int32{}, 7: []int32{5},
//		8: []int32{7}, 9: []int32{}, 10: []int32{9}, 11: []int32{},
//	}
//	successTxMap2 := make(map[string]struct{})
//	successTxMap2 = map[string]struct{}{
//		txList2[0].Payload.TxId: {}, txList2[1].Payload.TxId: {},
//		txList2[2].Payload.TxId: {}, txList2[3].Payload.TxId: {},
//		txList2[4].Payload.TxId: {}, txList2[5].Payload.TxId: {},
//		txList2[6].Payload.TxId: {}, txList2[7].Payload.TxId: {},
//		txList2[8].Payload.TxId: {}, txList2[9].Payload.TxId: {},
//		txList2[10].Payload.TxId: {}, txList2[11].Payload.TxId: {},
//	}
//
//	cach3, txList3 := addTxBatch_HasRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4)
//	neighborMap3 := make(map[int][]int32)
//	neighborMap3 = map[int][]int32{
//		0: []int32{}, 1: []int32{}, 2: []int32{}, 3: []int32{},
//		4: []int32{}, 5: []int32{}, 6: []int32{}, 7: []int32{},
//	}
//	successTxMap3 := make(map[string]struct{})
//	successTxMap3 = map[string]struct{}{
//		txList3[0].Payload.TxId: {}, txList3[1].Payload.TxId: {},
//		txList3[2].Payload.TxId: {}, txList3[3].Payload.TxId: {},
//		txList3[4].Payload.TxId: {}, txList3[5].Payload.TxId: {},
//		txList3[6].Payload.TxId: {}, txList3[7].Payload.TxId: {},
//	}
//
//	cach4, txList4 := addTxBatch_HasRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4)
//	neighborMap4 := make(map[int][]int32)
//	neighborMap4 = map[int][]int32{
//		0: []int32{}, 1: []int32{}, 2: []int32{}, 3: []int32{},
//		4: []int32{}, 5: []int32{}, 6: []int32{}, 7: []int32{},
//		8: []int32{}, 9: []int32{}, 10: []int32{}, 11: []int32{},
//	}
//	successTxMap4 := make(map[string]struct{})
//	successTxMap4 = map[string]struct{}{
//		txList4[0].Payload.TxId: {}, txList4[1].Payload.TxId: {},
//		txList4[2].Payload.TxId: {}, txList4[3].Payload.TxId: {},
//		txList4[6].Payload.TxId: {}, txList4[7].Payload.TxId: {},
//		txList4[9].Payload.TxId: {}, txList4[10].Payload.TxId: {},
//	}
//
//	cach5, txList5 := addTxBatch_HasRepeatTx_HasConflic_2(branchID1, branchID2, branchID3, branchID4)
//	neighborMap5 := make(map[int][]int32)
//	neighborMap5 = map[int][]int32{0: []int32{}, 1: []int32{}, 2: []int32{1}, 3: []int32{}, 4: []int32{},
//		5: []int32{}, 6: []int32{}, 7: []int32{}, 8: []int32{}, 9: []int32{}, 10: []int32{}, 11: []int32{},
//	}
//	successTxMap5 := make(map[string]struct{})
//	successTxMap5 = map[string]struct{}{
//		txList5[0].Payload.TxId: {}, txList5[1].Payload.TxId: {},
//		txList5[2].Payload.TxId: {}, txList5[3].Payload.TxId: {},
//		txList5[6].Payload.TxId: {}, txList5[7].Payload.TxId: {},
//		txList5[9].Payload.TxId: {}, txList5[10].Payload.TxId: {},
//	}
//
//	tests := []struct {
//		cach        *cache.AbftCache
//		testType    string
//		neighborMap map[int][]int32
//		txNum       int
//		successTxs  map[string]struct{}
//	}{
//		{cach1, mode1, neighborMap1, 8, successTxMap1},
//		{cach2, mode2, neighborMap2, 12, successTxMap2},
//		{cach3, mode3, neighborMap3, 8, successTxMap3},
//		{cach4, mode4, neighborMap4, 12, successTxMap4},
//		{cach5, mode4, neighborMap5, 13, successTxMap5},
//	}
//
//	for _, v := range tests {
//		m := NewMerger()
//		c := &Committer{
//			merger:        m,
//			retryList:     nil,
//			abftCache:     v.cach,
//			txBatchIDList: make([]string, 0),
//		}
//
//		txBatchHash := [][]byte{branchID3, branchID2, branchID4, branchID1}
//		c.prepare(txBatchHash)
//		c.sortTxBatchID()
//
//		block := CreateNewTestBlock(3)
//
//		c.merger.baseTxBatchID = c.txBatchIDList[0]
//		if err := c.merger.Merge(block, c.txBatchIDList); err != nil {
//			panic(err)
//		}
//
//		vh := newVerifyHelper(block, c.merger.rwSetMap, v.testType)
//		err := vh.checkMergeResults(v.neighborMap, v.txNum, v.successTxs)
//		fmt.Printf("test mode: %s, result: %v \n", v.testType, err == nil)
//		if err != nil {
//			fmt.Printf("fail reason: %s \n", err.Error())
//		}
//
//		fmt.Println("rwSetMap:", c.merger.rwSetMap)
//
//	}
//
//}
//
//func getTxsForMerge() []*commonpb.Transaction {
//	contractId := &commonpb.Contract{
//		Name:    contractName,
//		Version: "1",
//		RuntimeType:     commonpb.RuntimeType_WASMER,
//	}
//	parameters := make(map[string]string, 8)
//	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
//	tx1 := newTx("a0000000000000000000000000000001", contractId, parameters)
//	tx2 := newTx("a0000000000000000000000000000002", contractId, parameters)
//	tx3 := newTx("a0000000000000000000000000000003", contractId, parameters)
//	tx4 := newTx("a0000000000000000000000000000004", contractId, parameters)
//	tx5 := newTx("a0000000000000000000000000000005", contractId, parameters)
//	tx6 := newTx("a0000000000000000000000000000006", contractId, parameters)
//	tx7 := newTx("a0000000000000000000000000000007", contractId, parameters)
//	tx8 := newTx("a0000000000000000000000000000008", contractId, parameters)
//	tx9 := newTx("a0000000000000000000000000000009", contractId, parameters)
//	tx10 := newTx("a0000000000000000000000000000010", contractId, parameters)
//	tx11 := newTx("a00000000000000000000000000000011", contractId, parameters)
//	tx12 := newTx("a00000000000000000000000000000012", contractId, parameters)
//
//	txList := []*commonpb.Transaction{tx0, tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12}
//
//	return txList
//}
//
///**
//branch0:[tx0:{read: key1,  write: key2},  tx1:{read: key3,  wirte key4}]
//branch1:[tx2:{read: key5,  write: key6},  tx3:{read: key7,  wirte key8}]
//branch2:[tx4:{read: key9,  write: key10}, tx5:{read: key11, wirte key12}]
//branch3:[tx6:{read: key13, write: key14}, tx7:{read: key15, wirte key16}]
//*/
//func addTxBatch_NoRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4 []byte) (*cache.AbftCache, []*commonpb.Transaction) {
//	txList := getTxsForMerge()
//	tx0 := txList[0]
//	tx1 := txList[1]
//	tx2 := txList[2]
//	tx3 := txList[3]
//	tx4 := txList[4]
//	tx5 := txList[5]
//	tx6 := txList[6]
//	tx7 := txList[7]
//
//	hc := cache.NewAbftCache()
//	m := NewMerger()
//	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap0[tx0.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx0.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K1"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx1.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx1.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K3"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash0 := branchID1
//	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1})
//	b0.Dag = m.buildDAG(b0, rwSetMap0)
//	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
//	//hc.SetProposedTxBatch(b0, rwSetMap0)
//
//	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap1[tx2.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx2.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K6"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash1 := branchID2
//	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx2, tx3})
//	b1.Dag = m.buildDAG(b1, rwSetMap1)
//	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
//	//hc.SetProposedTxBatch(b1, rwSetMap1)
//
//	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap2[tx4.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx4.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K9"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K10"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx5.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx5.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K11"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//	}
//
//	hash2 := branchID3
//	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx4, tx5})
//	b2.Dag = m.buildDAG(b2, rwSetMap2)
//	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
//	//hc.SetProposedTxBatch(b2, rwSetMap2)
//
//	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap3[tx6.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx6.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K13"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx7.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx7.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K15"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K16"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash3 := branchID4
//	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx6, tx7})
//	b3.Dag = m.buildDAG(b3, rwSetMap3)
//	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
//	//hc.SetProposedTxBatch(b3, rwSetMap3)
//
//	return hc, txList
//}
//
///**
//branch0:[tx0:{read: key1,  write: key2},  tx1:{read: key2,  wirte key3},  tx8:{read: key3,   wirte key4}]
//branch1:[tx2:{read: key2,  write: key3},  tx3:{read: key4,  wirte key5},  tx9:{read: key6,   wirte key6}]
//branch2:[tx4:{read: key7,  write: key8},  tx5:{read: key6,  wirte key9},  tx10:{read: key9,  wirte key10}]
//branch3:[tx6:{read: key11, write: key12}, tx7:{read: key12, wirte key13}, tx11:{read: key4,  wirte key14}]
//*/
//func addTxBatch_NoRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4 []byte) (*cache.AbftCache, []*commonpb.Transaction) {
//	txList := getTxsForMerge()
//	tx0 := txList[0]
//	tx1 := txList[1]
//	tx2 := txList[2]
//	tx3 := txList[3]
//	tx4 := txList[4]
//	tx5 := txList[5]
//	tx6 := txList[6]
//	tx7 := txList[7]
//	tx8 := txList[8]
//	tx9 := txList[9]
//	tx10 := txList[10]
//	tx11 := txList[11]
//
//	hc := cache.NewAbftCache()
//	m := NewMerger()
//	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap0[tx0.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx0.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K1"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx1.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx1.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K3"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx8.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx8.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K3"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash0 := branchID1
//	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1, tx8})
//	b0.Dag = m.buildDAG(b0, rwSetMap0)
//	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
//	//hc.SetProposedTxBatch(b0, rwSetMap0)
//
//	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap1[tx2.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx2.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx9.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx9.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K6"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K6"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash1 := branchID2
//	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx2, tx3, tx9})
//	b1.Dag = m.buildDAG(b1, rwSetMap1)
//	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
//	//hc.SetProposedTxBatch(b1, rwSetMap1)
//
//	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap2[tx4.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx4.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx5.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx5.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K6"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K9"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx10.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx10.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K9"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K10"),
//			Value:        []byte("V"),
//		}},
//	}
//
//	hash2 := branchID3
//	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx4, tx5, tx10})
//	b2.Dag = m.buildDAG(b2, rwSetMap2)
//	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
//	//hc.SetProposedTxBatch(b2, rwSetMap2)
//
//	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap3[tx6.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx6.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K11"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx7.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx7.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K13"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx11.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx11.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash3 := branchID4
//	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx6, tx7, tx11})
//	b3.Dag = m.buildDAG(b3, rwSetMap3)
//	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
//	//hc.SetProposedTxBatch(b3, rwSetMap3)
//
//	return hc, txList
//}
//
///**
//branch0:[tx0:{read: key1,  write: key2},  tx1:{read: key3,  wirte key4}]
//branch1:[tx1:{read: key3,  write: key4},  tx2:{read: key5,  wirte key6},  tx3:{read: key7,  wirte key8}]
//branch2:[tx4:{read: key9,  write: key10}, tx5:{read: key11, wirte key12}, tx3:{read: key7,  wirte key8}]
//branch3:[tx6:{read: key13, write: key14}, tx7:{read: key15, wirte key16}, tx5:{read: key11, wirte key12}, tx3:{read: key7,  wirte key8}]
//*/
//func addTxBatch_HasRepeatTx_NoConflic(branchID1, branchID2, branchID3, branchID4 []byte) (*cache.AbftCache, []*commonpb.Transaction) {
//	txList := getTxs()
//	tx0 := txList[0]
//	tx1 := txList[1]
//	tx2 := txList[2]
//	tx3 := txList[3]
//	tx4 := txList[4]
//	tx5 := txList[5]
//	tx6 := txList[6]
//	tx7 := txList[7]
//
//	hc := cache.NewAbftCache()
//	m := NewMerger()
//	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap0[tx0.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx0.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K1"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx1.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx1.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K3"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash0 := branchID1
//	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1})
//	b0.Dag = m.buildDAG(b0, rwSetMap0)
//	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
//	//hc.SetProposedTxBatch(b0, rwSetMap0)
//
//	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap1[tx1.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx1.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K3"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx2.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx2.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K6"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash1 := branchID2
//	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx1, tx2, tx3})
//	b1.Dag = m.buildDAG(b1, rwSetMap1)
//	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
//	//hc.SetProposedTxBatch(b1, rwSetMap1)
//
//	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap2[tx4.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx4.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K9"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K10"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx5.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx5.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K11"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//
//	hash2 := branchID3
//	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx4, tx3, tx5})
//	b2.Dag = m.buildDAG(b2, rwSetMap2)
//	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
//	//hc.SetProposedTxBatch(b2, rwSetMap2)
//
//	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap3[tx6.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx6.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K13"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx7.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx7.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K15"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K16"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx5.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx5.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K11"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash3 := branchID4
//	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx6, tx7, tx5, tx3})
//	b3.Dag = m.buildDAG(b3, rwSetMap3)
//	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
//	//hc.SetProposedTxBatch(b3, rwSetMap3)
//
//	return hc, txList
//}
//
///**
//branch0:[tx0:{read: key1,  write: key2},  tx1:{read: key2,  wirte key3},  tx2:{read: key5,   wirte key6},  tx3:{read: key7,  wirte key8}]
//branch1:[tx3:{read: key7,  wirte key8},  tx4:{read: key8,  wirte key9},  tx5:{read: key9,   wirte key10},  tx6:{read: key11, write: key12}]
//branch2:[tx7:{read: key13,  write: key14},  tx3:{read: key7,  wirte key8},  tx8:{read: key8,  wirte key15}, tx9:{read: key16, write: key17}]
//branch3:[tx7:{read: key13, write: key14}, tx10:{read: key14, wirte key18}, tx6:{read: key11,  wirte key12}, tx11:{read: key12, write: key19}]
//*/
//func addTxBatch_HasRepeatTx_HasConflic(branchID1, branchID2, branchID3, branchID4 []byte) (*cache.AbftCache, []*commonpb.Transaction) {
//	txList := getTxsForMerge()
//	tx0 := txList[0]
//	tx1 := txList[1]
//	tx2 := txList[2]
//	tx3 := txList[3]
//	tx4 := txList[4]
//	tx5 := txList[5]
//	tx6 := txList[6]
//	tx7 := txList[7]
//	tx8 := txList[8]
//	tx9 := txList[9]
//	tx10 := txList[10]
//	tx11 := txList[11]
//
//	hc := cache.NewAbftCache()
//	m := NewMerger()
//	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap0[tx0.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx0.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K1"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx1.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx1.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K3"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx2.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx2.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K6"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash0 := branchID1
//	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1, tx2, tx3})
//	b0.Dag = m.buildDAG(b0, rwSetMap0)
//	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
//	//hc.SetProposedTxBatch(b0, rwSetMap0)
//
//	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap1[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx4.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx4.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K9"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx5.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx5.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K9"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K10"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx6.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx6.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K11"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash1 := branchID2
//	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx3, tx4, tx5, tx6})
//	b1.Dag = m.buildDAG(b1, rwSetMap1)
//	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
//	//hc.SetProposedTxBatch(b1, rwSetMap1)
//
//	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap2[tx7.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx7.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K13"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx8.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx8.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K15"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx9.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx9.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K16"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K17"),
//			Value:        []byte("V"),
//		}},
//	}
//
//	hash2 := branchID3
//	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx7, tx3, tx8, tx9})
//	b2.Dag = m.buildDAG(b2, rwSetMap2)
//	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
//	//hc.SetProposedTxBatch(b2, rwSetMap2)
//
//	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap3[tx7.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx7.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K13"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx10.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx10.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K18"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx6.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx6.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K11"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx11.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx11.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K19"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash3 := branchID4
//	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx7, tx10, tx6, tx11})
//	b3.Dag = m.buildDAG(b3, rwSetMap3)
//	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
//	//hc.SetProposedTxBatch(b3, rwSetMap3)
//
//	return hc, txList
//}
//
//func addTxBatch_HasRepeatTx_HasConflic_2(branchID1, branchID2, branchID3, branchID4 []byte) (*cache.AbftCache, []*commonpb.Transaction) {
//	txList := getTxsForMerge()
//	tx0 := txList[0]
//	tx1 := txList[1]
//	tx2 := txList[2]
//	tx3 := txList[3]
//	tx4 := txList[4]
//	tx5 := txList[5]
//	tx6 := txList[6]
//	tx7 := txList[7]
//	tx8 := txList[8]
//	tx9 := txList[9]
//	tx10 := txList[10]
//	tx11 := txList[11]
//	tx12 := txList[12]
//
//	hc := cache.NewAbftCache()
//	m := NewMerger()
//	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap0[tx0.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx0.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K1"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx1.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx1.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K3"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx2.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx2.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap0[tx3.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx3.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K7"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash0 := branchID1
//	b0 := createBatch(hash0, 3, []*commonpb.Transaction{tx0, tx1, tx2, tx3})
//	b0.Dag = m.buildDAG(b0, rwSetMap0)
//	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
//	//hc.SetProposedTxBatch(b0, rwSetMap0)
//
//	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap1[tx2.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx2.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx4.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx4.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx5.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx5.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K9"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap1[tx6.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx6.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K11"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K12"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash1 := branchID2
//	b1 := createBatch(hash1, 3, []*commonpb.Transaction{tx2, tx4, tx5, tx6})
//	b1.Dag = m.buildDAG(b1, rwSetMap1)
//	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
//	//hc.SetProposedTxBatch(b1, rwSetMap1)
//
//	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap2[tx7.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx7.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K13"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx4.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx4.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K5"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx8.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx8.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K8"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K15"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap2[tx9.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx9.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K16"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K17"),
//			Value:        []byte("V"),
//		}},
//	}
//
//	hash2 := branchID3
//	b2 := createBatch(hash2, 3, []*commonpb.Transaction{tx7, tx4, tx8, tx9})
//	b2.Dag = m.buildDAG(b2, rwSetMap2)
//	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
//	//hc.SetProposedTxBatch(b2, rwSetMap2)
//
//	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap3[tx7.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx7.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K13"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx10.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx10.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K14"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K18"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx11.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx11.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K4"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K19"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwSetMap3[tx12.Payload.TxId] = &commonpb.TxRWSet{
//		TxId: tx12.Payload.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractName,
//			Key:          []byte("K19"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractName,
//			Key:          []byte("K20"),
//			Value:        []byte("V"),
//		}},
//	}
//	hash3 := branchID4
//	b3 := createBatch(hash3, 3, []*commonpb.Transaction{tx7, tx10, tx11, tx12})
//	b3.Dag = m.buildDAG(b3, rwSetMap3)
//	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
//	//hc.SetProposedTxBatch(b3, rwSetMap3)
//
//	return hc, txList
//}
//
//func CreateNewTestBlock(height uint64) *commonpb.Block {
//	var hash = []byte("0123456789")
//	var version = uint32(1)
//	var block = &commonpb.Block{
//		Header: &commonpb.BlockHeader{
//			ChainId:        "Chain1",
//			BlockHeight:    height,
//			PreBlockHash:   hash,
//			BlockHash:      hash,
//			PreConfHeight:  0,
//			BlockVersion:   version,
//			DagHash:        hash,
//			RwSetRoot:      hash,
//			TxRoot:         hash,
//			BlockTimestamp: 0,
//			Proposer:       &acPb.Member{MemberInfo: []byte("User1")},
//			ConsensusArgs:  nil,
//			TxCount:        1,
//			Signature:      []byte(""),
//		},
//		Dag: &commonpb.DAG{
//			Vertexes: nil,
//		},
//		Txs: nil,
//	}
//	tx := CreateNewTestTx()
//	txs := make([]*commonpb.Transaction, 1)
//	txs[0] = tx
//	block.Txs = txs
//	return block
//}
//
//func CreateNewTestTx() *commonpb.Transaction {
//	return &commonpb.Transaction{
//		Payload: &commonpb.Payload{
//			ChainId: "",
//			TxId:    "",
//		},
//		Sender: &commonpb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
//			Signature: []byte("sign1"),
//		},
//		Result: &commonpb.Result{
//			Code: commonpb.TxStatusCode_SUCCESS,
//			ContractResult: &commonpb.ContractResult{
//				Result: []byte("ok"),
//			},
//		},
//	}
//}
//
//func newTx(txId string, contractName string, parameterMap map[string]string) *commonpb.Transaction {
//
//	var parameters []*commonpb.KeyValuePair
//	for key, value := range parameterMap {
//		parameters = append(parameters, &commonpb.KeyValuePair{
//			Key:   key,
//			Value: []byte(value),
//		})
//	}
//	//payload := &commonpb.Payload{
//	//	ContractName: contractName,
//	//	Method:       "method",
//	//	Parameters:   parameters,
//	//}
//	//payloadBytes, _ := proto.Marshal(payload)
//	return &commonpb.Transaction{
//		Payload: &commonpb.Payload{
//			ChainId:        "",
//			TxType:         commonpb.TxType_QUERY_CONTRACT,
//			TxId:           txId,
//		},
//		Sender: &commonpb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
//			Signature: []byte("sign1"),
//		},
//		Result:           &commonpb.Result{
//			Code:           commonpb.TxStatusCode_SUCCESS,
//			ContractResult: nil,
//			RwSetHash:      nil,
//		},
//	}
//}
