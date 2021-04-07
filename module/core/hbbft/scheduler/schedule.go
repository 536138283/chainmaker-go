/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
)

type Scheduler struct {
	block      *commonpb.Block    // the commited block
	blanchList []*commonpb.Block  // the total branch of transactions which sorted by branchID
	snapshot   protocol.Snapshot
}

func (sd *Scheduler) NewScheduler(block *commonpb.Block, blanchList []*commonpb.Block,
	snapshot protocol.Snapshot) *Scheduler {
	return &Scheduler{
		block:      block,
		blanchList: blanchList,
		snapshot:   snapshot,
	}
}

func (sd *Scheduler) Schedule() (map[string]*commonpb.TxRWSet, error) {
	// todo change english
	// 1. 根据第一批次，后续批次进行去重操作
	// 2. 冲突检测，不同批次的DAG按顺序将冲突的DAG剔除（将排在后面的剔除）
	// 3. 按照顺序依次合并各个批次的DAG
	// 4. 重新生成读写集
	// 5. 返回新生成的读写集

	// 去除重复交易(todo DAG 进行去重的冲突检测)

	txMap := make(map[string]bool, 0)
	for _, branch := range sd.blanchList {
		newTxList := make([]*commonpb.Transaction, 0)
		for _, tx := range branch.Txs {
			if _, ok := txMap[tx.Header.TxId]; !ok {
				txMap[tx.Header.TxId] = true
				newTxList = append(newTxList, tx)
			}
		}
		branch.Txs = newTxList

	}

	//todo DAG进行键对的冲突检测


	//todo 合并DAG
	//var newDag *commonpb.DAG
	
	//todo 重新生成读写集



	txRWSetMap := make(map[string]*commonpb.TxRWSet)

	return txRWSetMap, nil
}

func deleteRepeatDAG(fatherDag, dag *commonpb.DAG) {


}

func mergeTheDag(fatherDag, dag *commonpb.DAG)  {

}