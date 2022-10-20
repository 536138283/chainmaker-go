/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package commiter

//
//import (
//	"chainmaker.org/chainmaker-go/core/cache"
//	commonpb "chainmaker.org/chainmaker/pb-go/common"
//	"encoding/hex"
//	"fmt"
//	"github.com/gogo/protobuf/proto"
//	"github.com/stretchr/testify/require"
//	"testing"
//)
//
//// sort check(whether the order is identical)
//func TestCommitter_Commit_SortBranIDList(t *testing.T) {
//
//	branchID1 := hex.EncodeToString([]byte("0123456789"))
//	branchID2 := hex.EncodeToString([]byte("0213456789"))
//	branchID3 := hex.EncodeToString([]byte("0987654321"))
//	branchID4 := hex.EncodeToString([]byte("0123456789"))
//
//	branchIDList1 := []string{branchID1, branchID2, branchID3, branchID4}
//	branchIDList2 := []string{branchID2, branchID3, branchID4, branchID1}
//	branchIDList3 := []string{branchID3, branchID2, branchID1, branchID4}
//	branchIDList4 := []string{branchID4, branchID1, branchID3, branchID2}
//	branchIDList5 := []string{branchID4}
//	branchIDList6 := []string{}
//
//	tests := []struct {
//		branchIDList []string
//	}{
//		{branchIDList1},
//		{branchIDList2},
//		{branchIDList3},
//		{branchIDList4},
//		{branchIDList5},
//		{branchIDList6},
//	}
//
//	baseBranchIDOrder := make([]string, 0)
//	for i, tt := range tests {
//		commiter := &Committer{
//			txBatchIDList: tt.branchIDList,
//		}
//
//		commiter.sortTxBatchID()
//		fmt.Printf("i: %v, branchList: %s \n", i, commiter.txBatchIDList)
//
//		if i == 0 {
//			baseBranchIDOrder = commiter.txBatchIDList
//		} else if i <= 3 {
//			for j, v := range baseBranchIDOrder {
//				if commiter.txBatchIDList[j] != v {
//					require.Equal(t, v, commiter.txBatchIDList[j], "sort fail")
//				}
//			}
//		}
//	}
//}
//
//func TestCommitter_Commit_getRetryListAfterABA(t *testing.T) {
//	branchID1 := []byte("a")
//	branchID2 := []byte("b")
//	branchID3 := []byte("c")
//	branchID4 := []byte("d")
//	branchID5 := []byte("e")
//	branchID6 := []byte("f")
//
//	cach := addTxBatch(branchID1, branchID2, branchID3, branchID4, branchID5, branchID6)
//
//	m := NewMerger()
//	c := &Committer{
//		merger:        m,
//		retryList:     nil,
//		abftCache:     cach,
//		txBatchIDList: make([]string, 0),
//	}
//
//	txBatchHash := [][]byte{branchID3, branchID2, branchID5, branchID4}
//	c.prepare(txBatchHash)
//	fmt.Println("sort before", c.txBatchIDList)
//
//	c.sortTxBatchID()
//	fmt.Println("sort after:", c.txBatchIDList)
//
//	txList := getTxs()
//	allTrans := make(map[string]*commonpb.Transaction)
//	allTrans[txList[0].Payload.TxId] = txList[0]
//	allTrans[txList[1].Payload.TxId] = txList[1]
//	allTrans[txList[2].Payload.TxId] = txList[2]
//	allTrans[txList[3].Payload.TxId] = txList[3]
//	allTrans[txList[4].Payload.TxId] = txList[4]
//	//allTrans[txList[5].Header.TxId] = txList[5]
//	//allTrans[txList[6].Header.TxId] = txList[6]
//	allTrans[txList[7].Payload.TxId] = txList[7]
//
//	c.merger.allTxsMap = allTrans
//
//	c.handleABAFailTxs()
//
//	for txId, _ := range c.merger.allTxsMap {
//		fmt.Printf("allTrans-txId:%s\n", txId)
//	}
//
//	for _, v := range c.retryList {
//		fmt.Println("retryList:", v.Payload.TxId)
//	}
//
//}
//
//func newTx(txId string, contractName string, parameterMap map[string]string) *commonpb.Transaction {
//
//	var parameters []*commonpb.KeyValuePair
//	for key, value := range parameterMap {
//		parameters = append(parameters, &commonpb.KeyValuePair{
//			Key:   key,
//			Value: value,
//		})
//	}
//	payload := &commonpb.QueryPayload{
//		ContractName: contractName,
//		Method:       "method",
//		Parameters:   parameters,
//	}
//	payloadBytes, _ := proto.Marshal(payload)
//	return &commonpb.Transaction{
//		Header: &commonpb.TxHeader{
//			ChainId:        "",
//			Sender:         nil,
//			TxType:         commonpb.TxType_QUERY_USER_CONTRACT,
//			TxId:           txId,
//			Timestamp:      0,
//			ExpirationTime: 0,
//		},
//		RequestPayload:   payloadBytes,
//		RequestSignature: nil,
//		Result:           &commonpb.Result{
//			Code:           commonpb.TxStatusCode_SUCCESS,
//			ContractResult: nil,
//			RwSetHash:      nil,
//		},
//	}
//}
//
//func getTxs() []*commonpb.Transaction {
//	contractId := &commonpb.ContractId{
//		ContractName:    "ContractName",
//		ContractVersion: "1",
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
//
//	txList := []*commonpb.Transaction{tx0, tx1, tx2, tx3, tx4, tx5, tx6, tx7}
//
//	return txList
//}
//
//func createBatch(blockHash []byte, height int64, txList []*commonpb.Transaction) *commonpb.Block {
//	b0 := CreateNewTestBlock(height)
//	hash0 := blockHash
//	b0.Header.BlockHash = hash0
//	b0.Txs = txList
//
//	return b0
//}
//
//func addTxBatch(branchID1, branchID2, branchID3, branchID4, branchID5, branchID6 []byte) *cache.AbftCache {
//	txList := getTxs()
//
//	hc := cache.NewAbftCache()
//	rwSetMap0 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap0[txList[2].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[2].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	rwSetMap0[txList[3].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[3].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	hash0 := branchID1
//	b0 := createBatch(hash0, 3, []*commonpb.Transaction{txList[2], txList[3]})
//	hc.AddVerifiedTxBatch(b0, true, rwSetMap0)
//
//	rwSetMap1 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap1[txList[3].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[3].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	rwSetMap1[txList[4].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[4].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	hash1 := branchID2
//	b1 := createBatch(hash1, 3, []*commonpb.Transaction{txList[3], txList[4]})
//	hc.AddVerifiedTxBatch(b1, true, rwSetMap1)
//
//	rwSetMap2 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap2[txList[4].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[4].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	rwSetMap2[txList[6].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[6].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//
//	hash2 := branchID3
//	b2 := createBatch(hash2, 3, []*commonpb.Transaction{txList[4], txList[6]})
//	hc.AddVerifiedTxBatch(b2, true, rwSetMap2)
//
//	rwSetMap3 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap3[txList[4].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[4].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	hash3 := branchID4
//	b3 := createBatch(hash3, 3, []*commonpb.Transaction{txList[4]})
//	hc.AddVerifiedTxBatch(b3, true, rwSetMap3)
//
//	rwSetMap4 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap4[txList[0].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[0].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	hash4 := branchID5
//	b4 := createBatch(hash4, 3, []*commonpb.Transaction{txList[0]})
//	hc.AddVerifiedTxBatch(b4, true, rwSetMap4)
//
//	rwSetMap5 := make(map[string]*commonpb.TxRWSet)
//	rwSetMap5[txList[0].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[0].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	rwSetMap5[txList[5].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[5].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	rwSetMap5[txList[6].Header.TxId] = &commonpb.TxRWSet{
//		TxId:     txList[6].Header.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//	hash5 := branchID6
//	b5 := createBatch(hash5, 3, []*commonpb.Transaction{txList[0], txList[5], txList[6]})
//	hc.AddVerifiedTxBatch(b5, true, rwSetMap5)
//
//	/**
//	b0		t2,t3
//	b1		t3,t4
//	b2		t4      fail
//	b3		t4
//	b4		t0
//	b5		t0,t5,t6   fail
//	*/
//
//	//retry: t5,t6
//	return hc
//}
//
//
//func CreateNewTestBlock(height int64) *commonpb.Block {
//	var hash = []byte("0123456789")
//	var version = []byte("0")
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
//			Proposer:       hash,
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
//	var hash = []byte("0123456789")
//	return &commonpb.Transaction{
//		Header: &commonpb.TxHeader{
//			ChainId:        "",
//			Sender:         nil,
//			TxType:         0,
//			TxId:           "",
//			Timestamp:      0,
//			ExpirationTime: 0,
//		},
//		RequestPayload:   hash,
//		RequestSignature: hash,
//		Result: &commonpb.Result{
//			Code:           commonpb.TxStatusCode_SUCCESS,
//			ContractResult: nil,
//			RwSetHash:      nil,
//		},
//	}
//}
