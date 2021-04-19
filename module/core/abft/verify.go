/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Workiva/go-datastructures/threadsafe/err"
	"runtime"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/monitor"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const DEFAULT_VERIFY_TIMEOUT = time.Second * 10

type Verifier struct {
	chainId               string
	wg                    sync.WaitGroup
	log                   *logger.CMLogger
	abftCache             *cache.AbftCache
	verifyBlock           *common.VerifyBlock
	ledgerCache           protocol.LedgerCache
	msgBus                msgbus.MessageBus
	verifyTimeout         time.Duration
	txPool                protocol.TxPool
	goRoutinePool         *ants.Pool
	metricBlockVerifyTime *prometheus.HistogramVec // metrics monitor
}

func NewVerifier(ce *CoreExecute) (*Verifier, error) {
	verifier := &Verifier{
		wg:            sync.WaitGroup{},
		log:           ce.log,
		abftCache:     ce.abftCache,
		ledgerCache:   ce.ledgerCache,
		msgBus:        ce.msgBus,
		verifyTimeout: DEFAULT_VERIFY_TIMEOUT,
		txPool:        ce.txPool,
		chainId:       ce.chainId,
	}
	//verifier.finishVerifyC = make(chan struct{})
	conf := &common.ValidateBlockConf{
		ChainConf:       ce.chainConf,
		Log:             ce.log,
		LedgerCache:     ce.ledgerCache,
		Ac:              ce.ac,
		SnapshotManager: ce.snapshotManager,
		VmMgr:           ce.vmMgr,
		TxPool:          ce.txPool,
		BlockchainStore: ce.blockchainStore,
	}
	verifier.verifyBlock = common.NewVerifyBlock(conf)
	var err error
	verifier.goRoutinePool, err = ants.NewPool(len(ce.chainConf.ChainConfig().Consensus.Nodes), ants.WithPreAlloc(true))
	if err != nil {
		return fmt.Errorf('')
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		verifier.metricBlockVerifyTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_VERIFIER, "metric_block_verify_time",
			"block verify time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return verifier
}

func (v *Verifier) checkHeight(block *commonPb.Block) (bool, error) {
	currentHeight, err := v.ledgerCache.CurrentHeight()
	if err != nil {
		return false, err
	}
	if currentHeight+1 != block.Header.BlockHeight {
		return false, errors.New("the packaging signal height is inconsistent with the cache")
	}
	return true, nil
}

func (v *Verifier) verify(block *commonPb.Block) error {

	startTick := utils.CurrentTimeMillisSeconds()
	if err := utils.IsEmptyBlock(block); err != nil {
		return err
	}
	ok, err := v.checkHeight(block)
	if !ok {
		return err
	}
	v.log.Debugf("verify receive [%d](%x,%d,%d)",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs))

	txRWSetMap, timeLasts, err := v.verifyBlock.ValidateBlock(block)
	var isValid bool
	if err != nil {
		isValid = false
		v.log.Warnf("verify failed [%d](%x),preBlockHash:%x, %s",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash, err.Error())
		v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
		err := v.abftCache.AddAbftTxBatch(block, cache.FAIL, txRWSetMap)
		if err != nil {
			v.log.Warnf("add abft cache tx batch [%d](%x),preBlockHash:%x, %s",
				block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash, err.Error())
		}
		return err
	}
	// mark transactions in block as pending status in txpool
	v.txPool.AddTxsToPendingCache(block.Txs, block.Header.BlockHeight)
	isValid = true
	v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, isValid))
	err = v.abftCache.AddAbftTxBatch(block, cache.SUCCESS, txRWSetMap)
	if err != nil {
		v.log.Warnf("add abft cache tx batch [%d](%x),preBlockHash:%x, %s",
			block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash, err.Error())
	}
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	v.log.Infof("verify success [%d,%x](%v,%d)", block.Header.BlockHeight, block.Header.BlockHash,
		timeLasts, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		v.metricBlockVerifyTime.WithLabelValues(v.chainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}

func parseVerifyResult(block *commonPb.Block, isValid bool) *consensuspb.VerifyResult {
	verifyResult := &consensuspb.VerifyResult{
		VerifiedBlock: block,
	}
	if isValid {
		verifyResult.Code = consensuspb.VerifyResult_SUCCESS
		verifyResult.Msg = "OK"
	} else {
		verifyResult.Msg = "FAIL"
		verifyResult.Code = consensuspb.VerifyResult_FAIL
	}
	return verifyResult
}

func (v *Verifier) VerifyBlock(block *commonPb.Block, mode protocol.VerifyMode) error {
	var goRoutinePool *ants.Pool
	var err error
	if goRoutinePool, err = ants.NewPool(runtime.NumCPU()*4, ants.WithPreAlloc(true)); err != nil {
		v.log.Errorf("ants new go routine pool failed: %s", err.Error())
		return err
	}
	defer goRoutinePool.Release()
	goRoutinePool.Submit(v.verifyTask(block))
}

func (v *Verifier) verifyTask(block *commonPb.Block) func() {
	return func() {
		err := v.verify(block)
		if err != nil {
			v.log.Errorf("verify txBatch failed: %s, height: %d, txBatchHash: %s", err, block.Header.BlockHeight,
				hex.EncodeToString(block.Header.BlockHash))
		}
	}
}
