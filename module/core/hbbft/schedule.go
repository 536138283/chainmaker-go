/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"encoding/hex"
	"sync"
)

type Scheduler struct {
	block *commonpb.Block // the commited block
	branchInfo    map[string]*BranchInfo // key -> branchId
	branchIDList []string
}

type BranchInfo struct {
	confirmedBranch *commonpb.Block
	rwSetMap map[string]*commonpb.TxRWSet // key->txId
}

func NewScheduler(block *commonpb.Block, branchMap sync.Map, branchIDList []string) *Scheduler {
	return &Scheduler{
		block:        block,
		branchIDList: branchIDList,
	}
}

func (s *Scheduler) Schedule() (map[string]*commonpb.TxRWSet, map[string]bool, error) {
	// todo change english
	// 1. 根据第一批次，后续批次进行去重操作
	// 2. 冲突检测，不同批次的DAG按顺序将冲突的DAG剔除（将排在后面的剔除）
	// 3. 按照顺序依次合并各个批次的DAG
	// 4. 重新生成读写集
	// 5. 返回新生成的读写集

	// 去除重复交易(todo DAG 进行去重的冲突检测)

	txMap := make(map[string]bool)        // 用于检验是否有重复交易(branchID->bool)
	repeatTrans := make(map[string][]int) // 用于记录被删除的重复交易(branchID->这笔交易所在批次的位置)

	for _, branchID := range s.branchIDList {
		if branchInfo, ok := s.branchInfo[branchID]; ok{
			newTxs := make([]*commonpb.Transaction, 0)
			txs := branchInfo.confirmedBranch.Txs
			for i, _ := range txs {
				if _, ok := txMap[txs[i].Header.TxId]; !ok {
					txMap[txs[i].Header.TxId] = true
					newTxs = append(newTxs, txs[i])
				} else {
					branchID := hex.EncodeToString(branchInfo.confirmedBranch.Header.BlockHash)
					repeatTrans[branchID] = append(repeatTrans[branchID], i)
				}
			}
			branchInfo.confirmedBranch.Txs = newTxs
			s.branchInfo[branchID] = branchInfo
		}

	}

	// 除第一批次外，把其余批次的DAG中因重复引起的冲突的DAG删除（只针对重复冲突）
	for branchID, v := range repeatTrans {
		branch := s.branchInfo[branchID].confirmedBranch
		rwSetMap := s.branchInfo[branchID].rwSetMap
		s.deleteRepeatDAG(branch.Dag, v, rwSetMap)
	}

	//todo DAG进行键对的冲突检测 + 合并DAG
	dag, txs := s.mergeTheDag()

	//todo 重新生成读写集
	txRWSetMap := s.rebuiltRwSetMap()

	s.block.Dag = dag
	s.block.Txs = txs

	return txRWSetMap, txMap, nil
}

// todo 第一版 ，出现重复交易后，该批次这笔交易后的所有交易全部丢弃；后续需要完善
func (s *Scheduler) deleteRepeatDAG(dag *commonpb.DAG, repeatTrans []int, rwSetMap map[string]*commonpb.TxRWSet) {

	//for i, _ := range dag.Vertexes {
	//
	//}

}

func (s *Scheduler) mergeTheDag() (*commonpb.DAG, []*commonpb.Transaction) {

	//todo
	var dag *commonpb.DAG

	txs := make([]*commonpb.Transaction, 0)

	return dag, txs
}

func (s *Scheduler) rebuiltRwSetMap() map[string]*commonpb.TxRWSet {
	txRWSetMap := make(map[string]*commonpb.TxRWSet)

	return txRWSetMap
}