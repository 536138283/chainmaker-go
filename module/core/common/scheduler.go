/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	acpb "chainmaker.org/chainmaker-go/pb/protogo/accesscontrol"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/hex"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"runtime"
	"sync"
	"time"
)

type TxScheduler struct {
	lock            sync.Mutex
	VmManager       protocol.VmManager
	scheduleFinishC chan bool
	log             *logger.CMLogger
	scheduleTimeOut time.Duration
	scheduleWithDagTimeout time.Duration
	metricVMRunTime *prometheus.HistogramVec
}

// Transaction dependency in adjacency table representation
type dagNeighbors map[int]bool

func newTxSimContext(vmManager protocol.VmManager, snapshot protocol.Snapshot, tx *commonpb.Transaction) protocol.TxSimContext {
	return &txSimContextImpl{
		txExecSeq:     snapshot.GetSnapshotSize(),
		tx:            tx,
		txReadKeyMap:  make(map[string]*commonpb.TxRead, 8),
		txWriteKeyMap: make(map[string]*commonpb.TxWrite, 8),
		snapshot:      snapshot,
		vmManager:     vmManager,
		gasUsed:       0,
		currentDeep:   0,
		hisResult:     make([]*callContractResult, 0),
	}
}

// Schedule according to a batch of transactions, and generating DAG according to the conflict relationship
func Schedule(txScheduler *TxScheduler, block *commonpb.Block, txBatch []*commonpb.Transaction,
	snapshot protocol.Snapshot) (map[string]*commonpb.TxRWSet, error) {

	txScheduler.lock.Lock()
	defer txScheduler.lock.Unlock()
	txBatchSize := len(txBatch)
	runningTxC := make(chan *commonpb.Transaction, txBatchSize)
	timeoutC := time.After(txScheduler.scheduleTimeOut * time.Second)
	finishC := make(chan bool)
	log.Infof("schedule tx batch start, size %d", txBatchSize)
	var goRoutinePool *ants.Pool
	var err error
	if goRoutinePool, err = ants.NewPool(runtime.NumCPU()*4, ants.WithPreAlloc(true)); err != nil {
		return nil, err
	}
	defer goRoutinePool.Release()
	startTime := time.Now()
	go func() {
		for {
			select {
			case tx := <-runningTxC:
				err := goRoutinePool.Submit(func() {
					// If snapshot is sealed, no more transaction will be added into snapshot
					if snapshot.IsSealed() {
						return
					}
					log.Debugf("run vm for tx id:%s", tx.Header.GetTxId())
					txSimContext := newTxSimContext(txScheduler.VmManager, snapshot, tx)
					runVmSuccess := true
					var txResult *commonpb.Result
					var err error
					var start time.Time
					if localconf.ChainMakerConfig.MonitorConfig.Enabled {
						start = time.Now()
					}
					if txResult, err = runVM(tx, txSimContext, txScheduler.VmManager, txScheduler.log); err != nil {
						runVmSuccess = false
						tx.Result = txResult
						txSimContext.SetTxResult(txResult)
						log.Errorf("failed to run vm for tx id:%s during schedule, tx result:%+v, error:%+v", tx.Header.GetTxId(), txResult, err)
					} else {
						tx.Result = txResult
						txSimContext.SetTxResult(txResult)
					}
					applyResult, applySize := snapshot.ApplyTxSimContext(txSimContext, runVmSuccess)
					if !applyResult {
						runningTxC <- tx
					} else {
						if localconf.ChainMakerConfig.MonitorConfig.Enabled {
							elapsed := time.Since(start)
							txScheduler.metricVMRunTime.WithLabelValues(tx.Header.ChainId).Observe(elapsed.Seconds())
						}
						log.Debugf("apply to snapshot tx id:%s, result:%+v, apply count:%d", tx.Header.GetTxId(), txResult, applySize)
					}
					// If all transactions have been successfully added to dag
					if applySize >= txBatchSize {
						finishC <- true
					}
				})
				if err != nil {
					log.Warnf("failed to submit tx id %s during schedule, %+v", tx.Header.GetTxId(), err)
				}
			case <-timeoutC:
				txScheduler.scheduleFinishC <- true
				log.Debugf("schedule reached time limit")
				return
			case <-finishC:
				log.Debugf("schedule finish")
				txScheduler.scheduleFinishC <- true
				return
			}
		}
	}()
	// Put the pending transaction into the running queue
	go func() {
		for _, tx := range txBatch {
			runningTxC <- tx
		}
	}()
	// Wait for schedule finish signal
	<-txScheduler.scheduleFinishC
	// Build DAG from read-write table
	snapshot.Seal()
	timeCostA := time.Since(startTime)
	block.Dag = snapshot.BuildDAG()
	block.Txs = snapshot.GetTxTable()
	timeCostB := time.Since(startTime)
	log.Infof("schedule tx batch end, success %d, time cost %v, time cost(dag include) %v ",
		len(block.Dag.Vertexes), timeCostA, timeCostB)
	txRWSetTable := snapshot.GetTxRWSetTable()
	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	for _, txRWSet := range txRWSetTable {
		if txRWSet != nil {
			txRWSetMap[txRWSet.TxId] = txRWSet
		}
	}
	//ts.dumpDAG(block.Dag, block.Txs)
	return txRWSetMap, nil
}

func parseParameter(parameterPairs []*commonpb.KeyValuePair) map[string]string {
	parameters := make(map[string]string, 16)
	for i := 0; i < len(parameterPairs); i++ {
		key := parameterPairs[i].Key
		// ignore the following input from the user's invoke parameters
		if key == protocol.ContractCreatorOrgIdParam ||
			key == protocol.ContractCreatorRoleParam ||
			key == protocol.ContractCreatorPkParam ||
			key == protocol.ContractSenderOrgIdParam ||
			key == protocol.ContractSenderRoleParam ||
			key == protocol.ContractSenderPkParam ||
			key == protocol.ContractBlockHeightParam ||
			key == protocol.ContractTxIdParam {
			continue
		}
		value := parameterPairs[i].Value
		parameters[key] = value
	}
	return parameters
}

func acVerify(txSimContext protocol.TxSimContext, methodName string, endorsements []*commonpb.EndorsementEntry, msg []byte, parameters map[string]string) error {
	var ac protocol.AccessControlProvider
	var targetOrgId string
	var err error

	tx := txSimContext.GetTx()

	if ac, err = txSimContext.GetAccessControl(); err != nil {
		return fmt.Errorf("failed to get access control from tx sim context for tx: %s, error: %s", tx.Header.TxId, err.Error())
	}
	if orgId, ok := parameters[protocol.ConfigNameOrgId]; ok {
		targetOrgId = orgId
	} else {
		targetOrgId = ""
	}

	var fullCertEndorsements []*commonpb.EndorsementEntry
	for _, endorsement := range endorsements {
		if endorsement == nil || endorsement.Signer == nil {
			return fmt.Errorf("failed to get endorsement signer for tx: %s, endorsement: %+v", tx.Header.TxId, endorsement)
		}
		if endorsement.Signer.IsFullCert {
			fullCertEndorsements = append(fullCertEndorsements, endorsement)
		} else {
			fullCertEndorsement := &commonpb.EndorsementEntry{
				Signer: &acpb.SerializedMember{
					OrgId:      endorsement.Signer.OrgId,
					MemberInfo: nil,
					IsFullCert: true,
				},
				Signature: endorsement.Signature,
			}
			memberInfoHex := hex.EncodeToString(endorsement.Signer.MemberInfo)
			if fullMemberInfo, err := txSimContext.Get(commonpb.ContractName_SYSTEM_CONTRACT_CERT_MANAGE.String(), []byte(memberInfoHex)); err != nil {
				return fmt.Errorf("failed to get full cert from tx sim context for tx: %s, error: %s", tx.Header.TxId, err.Error())
			} else {
				fullCertEndorsement.Signer.MemberInfo = fullMemberInfo
			}
			fullCertEndorsements = append(fullCertEndorsements, fullCertEndorsement)
		}
	}
	if verifyResult, err := utils.VerifyConfigUpdateTx(methodName, fullCertEndorsements, msg, targetOrgId, ac); err != nil {
		return fmt.Errorf("failed to verify endorsements for tx: %s, error: %s", tx.Header.TxId, err.Error())
	} else if !verifyResult {
		return fmt.Errorf("failed to verify endorsements for tx: %s", tx.Header.TxId)
	} else {
		return nil
	}
}
