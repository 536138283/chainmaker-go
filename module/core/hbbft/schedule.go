/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"sort"
)

type Scheduler struct {
	block        *commonpb.Block        // the commited block
	branchInfo   map[string]*BranchInfo // key -> branchId
	branchIDList []string
	retryList    []*commonpb.Transaction
	allTransMap  map[string]*commonpb.Transaction // record all transaction(branchID->Transaction)
}

type BranchInfo struct {
	branch   *commonpb.Block
	rwSetMap map[string]*commonpb.TxRWSet // key->txId
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		branchInfo:  make(map[string]*BranchInfo),
		retryList:   make([]*commonpb.Transaction, 0),
		allTransMap: make(map[string]*commonpb.Transaction),
	}
}

func (s *Scheduler) Schedule() (map[string]*commonpb.TxRWSet, error) {
	// prepare
	repeatTrans := prepareForShedule(s.branchIDList, s.branchInfo, s.allTransMap)

	// delete repeat transaction
	s.delRepeatTransactions(repeatTrans)

	//merge DAG & RWSet
	txRWSetMap := s.mergeRwSetMapAndDAG()

	return txRWSetMap, nil
}

func (s *Scheduler) delRepeatTransactions(repeatTrans map[string][]int) {
	for branchID, site := range repeatTrans {
		delRepeatTransaction(branchID, site, s.branchInfo[branchID].branch, s.branchInfo[branchID].rwSetMap, s.retryList)
	}
}

func delRepeatTransaction(
	branchID string,
	deleteSites []int,
	branch *commonpb.Block,
	rwSetMap map[string]*commonpb.TxRWSet,
	retryList []*commonpb.Transaction) {

	// record the related Transaction' s position
	relatedTranSiteMap := recordTheReleatedTrans(deleteSites, branch)

	// record the relatedTransaction which need to be taken back to txpool
	for index, _ := range relatedTranSiteMap {
		// the conflict transaction 's position list
		deleteSites = append(deleteSites, index)
		retryList = append(retryList, branch.Txs[index])
	}

	// merge transaction's DAG & RWSet
	mergeRwSetMapAndDAG(
		deleteSites,
		branch,
		rwSetMap)
}

func recordTheReleatedTrans(deleteSites []int, branch *commonpb.Block) map[int]struct{} {
	relatedTranSiteMap := make(map[int]struct{})
	for _, site := range deleteSites {
		neighbors := branch.Dag.Vertexes[site].Neighbors
		for _, relatedTranSite := range neighbors {
			if _, ok := relatedTranSiteMap[int(relatedTranSite)]; !ok {
				relatedTranSiteMap[int(relatedTranSite)] = struct{}{}
			}
		}
	}
	return relatedTranSiteMap
}

func mergeRwSetMapAndDAG(deleteSites []int,
	branch *commonpb.Block, rwSetMap map[string]*commonpb.TxRWSet) {
	sort.Ints(deleteSites)
	for i := len(deleteSites) - 1; i >= 0; i-- {
		// delete the RWSetMap
		txId := branch.Txs[i].Header.TxId
		delete(rwSetMap, txId)

		// delete the transaction & delete the DAG
		if i != len(branch.Txs)-1 {
			branch.Txs = append(branch.Txs[:i], branch.Txs[i+1])
			branch.Dag.Vertexes = append(branch.Dag.Vertexes[:i], branch.Dag.Vertexes[i+1:]...)
		} else {
			branch.Txs = branch.Txs[:i]
			branch.Dag.Vertexes = branch.Dag.Vertexes[:i]
		}
	}
}

func (s *Scheduler) mergeRwSetMapAndDAG() map[string]*commonpb.TxRWSet {
	//  get the base writeTable
	baseBranchID := s.branchIDList[0]
	baseWriteTable := getBaseWriteTable(s.branchInfo[baseBranchID].rwSetMap)

	finalRWSetMap := s.branchInfo[baseBranchID].rwSetMap
	finalDAG := s.branchInfo[baseBranchID].branch.Dag
	finalTxs := s.branchInfo[baseBranchID].branch.Txs
	for _, branchID := range s.branchIDList {
		if branchID != baseBranchID {
			handleRWSetConflict(
				s.branchInfo[branchID].branch,
				s.branchInfo[branchID].rwSetMap,
				finalRWSetMap,
				baseWriteTable,
				s.retryList,
				s.allTransMap)
		}
	}

	// merge to the final DAG & Txs
	for _, branchID := range s.branchIDList[0:] {
		branch := s.branchInfo[branchID].branch
		finalDAG.Vertexes = append(finalDAG.Vertexes, branch.Dag.Vertexes...)
		finalTxs = append(finalTxs, branch.Txs...)
	}

	s.block.Txs = finalTxs
	s.block.Dag = finalDAG

	return finalRWSetMap
}

func handleRWSetConflict(branch *commonpb.Block, rwSetMap, finalRWSetMap map[string]*commonpb.TxRWSet, writeTable map[string]struct{},
	retryList []*commonpb.Transaction, allTransMap map[string]*commonpb.Transaction) {

	delSiteList := make([]int, 0)
	for site, tx := range branch.Txs {
		txId := tx.Header.TxId
		rwSet := rwSetMap[txId]
		for _, txRead := range rwSet.TxReads {
			finalKey := constructKey(txRead.ContractName, txRead.Key)
			// check if RWSet conflict
			if _, ok := writeTable[finalKey]; ok {
				// record the conflict transaction
				retryList = append(retryList, allTransMap[txId])
				delSiteList = append(delSiteList, site)
			} else {
				writeTable[finalKey] = struct{}{}
				finalRWSetMap[txId] = rwSet
			}
		}
	}

	// merge the DAG & RWSet
	mergeRwSetMapAndDAG(delSiteList, branch, rwSetMap)
}

func constructKey(contractName string, key []byte) string {
	return contractName + string(key)
}

func prepareForShedule(
	branchIDList []string,
	branchInfo map[string]*BranchInfo,
	allTransMap map[string]*commonpb.Transaction) map[string][]int {

	repeatTrans := make(map[string][]int) // record the deleted & repeated transaction(branchID->deleted tranction 's position)
	for _, branchID := range branchIDList {
		if info, ok := branchInfo[branchID]; ok {
			txs := info.branch.Txs
			for i, _ := range txs {
				txID := txs[i].Header.TxId
				if _, ok := allTransMap[txID]; !ok {
					allTransMap[txID] = txs[i]
				} else {
					repeatTrans[branchID] = append(repeatTrans[branchID], i)
				}
			}
		}
	}

	return repeatTrans
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
