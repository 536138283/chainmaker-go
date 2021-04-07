/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/common/log"
	"runtime"
	"time"
)

// SimulateWithDag based on the dag in the block, perform scheduling and execution transactions
func SimulateWithDag(txScheduler *TxScheduler, block *commonpb.Block,
	snapshot protocol.Snapshot) (map[string]*commonpb.TxRWSet, map[string]*commonpb.Result, error) {

	txScheduler.lock.Lock()
	defer txScheduler.lock.Unlock()

	startTime := time.Now()
	log.Debugf("simulate with dag start, size %d", len(block.Txs))
	txMapping := make(map[int]*commonpb.Transaction)
	for index, tx := range block.Txs {
		txMapping[index] = tx
	}

	// Construct the adjacency list of dag, which describes the subsequent adjacency transactions of all transactions
	dag := block.Dag
	dagRemain := make(map[int]dagNeighbors)
	for txIndex, neighbors := range dag.Vertexes {
		dn := make(dagNeighbors)
		for _, neighbor := range neighbors.Neighbors {
			dn[int(neighbor)] = true
		}
		dagRemain[txIndex] = dn
	}

	txBatchSize := len(block.Dag.Vertexes)
	runningTxC := make(chan int, txBatchSize)
	doneTxC := make(chan int, txBatchSize)

	timeoutC := time.After(txScheduler.scheduleWithDagTimeout * time.Second)
	finishC := make(chan bool)

	var goRoutinePool *ants.Pool
	var err error
	if goRoutinePool, err = ants.NewPool(runtime.NumCPU()*4, ants.WithPreAlloc(true)); err != nil {
		return nil, nil, err
	}
	defer goRoutinePool.Release()

	go func() {
		for {
			select {
			case txIndex := <-runningTxC:
				tx := txMapping[txIndex]
				err := goRoutinePool.Submit(func() {
					log.Debugf("run vm with dag for tx id %s", tx.Header.GetTxId())
					txSimContext := newTxSimContext(txScheduler.VmManager, snapshot, tx)

					runVmSuccess := true
					var txResult *commonpb.Result
					var err error

					if txResult, err = runVM(tx, txSimContext, txScheduler.VmManager, txScheduler.log); err != nil {
						runVmSuccess = false
						txSimContext.SetTxResult(txResult)
						log.Errorf("failed to run vm for tx id:%s during simulate with dag, tx result:%+v, error:%+v", tx.Header.GetTxId(), txResult, err)
					} else {
						//ts.log.Debugf("success to run vm for tx id:%s during simulate with dag, tx result:%+v", tx.Header.GetTxId(), txResult)
						txSimContext.SetTxResult(txResult)
					}

					applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, runVmSuccess)
					if !applyResult {
						log.Debugf("failed to apply according to dag with tx %s ", tx.Header.TxId)
						runningTxC <- txIndex
					} else {
						log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d", tx.Header.GetTxId(), txResult, applySize)
						doneTxC <- txIndex
					}
					// If all transactions in current batch have been successfully added to dag
					if applySize >= txBatchSize {
						finishC <- true
					}
				})
				if err != nil {
					log.Warnf("failed to submit tx id %s during simulate with dag, %+v", tx.Header.GetTxId(), err)
				}
			case doneTxIndex := <-doneTxC:
				shrinkDag(doneTxIndex, dagRemain)

				txIndexBatch := popNextTxBatchFromDag(dagRemain)
				log.Debugf("pop next tx index batch %v", txIndexBatch)
				for _, tx := range txIndexBatch {
					runningTxC <- tx
				}
			case <-finishC:
				log.Debugf("schedule with dag finish")
				txScheduler.scheduleFinishC <- true
				return
			case <-timeoutC:
				log.Errorf("schedule with dag timeout")
				txScheduler.scheduleFinishC <- true
				return
			}
		}
	}()

	txIndexBatch := popNextTxBatchFromDag(dagRemain)

	go func() {
		for _, tx := range txIndexBatch {
			runningTxC <- tx
		}
	}()

	<-txScheduler.scheduleFinishC
	snapshot.Seal()

	log.Infof("simulate with dag end, size %d, time cost %+v", len(block.Txs), time.Since(startTime))

	// Return the read and write set after the scheduled execution
	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	for _, txRWSet := range snapshot.GetTxRWSetTable() {
		if txRWSet != nil {
			txRWSetMap[txRWSet.TxId] = txRWSet
		}
	}
	return txRWSetMap, snapshot.GetTxResultMap(), nil
}

func shrinkDag(txIndex int, dagRemain map[int]dagNeighbors) {
	for _, neighbors := range dagRemain {
		delete(neighbors, txIndex)
	}
}

func popNextTxBatchFromDag(dagRemain map[int]dagNeighbors) []int {
	var txIndexBatch []int
	for checkIndex, neighbors := range dagRemain {
		if len(neighbors) == 0 {
			txIndexBatch = append(txIndexBatch, checkIndex)
			delete(dagRemain, checkIndex)
		}
	}
	return txIndexBatch
}