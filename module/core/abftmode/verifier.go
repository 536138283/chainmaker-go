/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abftmode

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/monitor"
	"chainmaker.org/chainmaker-go/utils"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const DEFAULT_VERIFY_TIMEOUT = time.Second * 10

type Verifier struct {
	chainId               string
	wg                    sync.WaitGroup
	log                   protocol.Logger
	abftCache             *cache.AbftCache
	verifierBlock         *common.VerifierBlock
	ledgerCache           protocol.LedgerCache
	msgBus                msgbus.MessageBus
	verifyTimeout         time.Duration
	txPool                protocol.TxPool
	goRoutinePool         *ants.Pool
	metricBlockVerifyTime *prometheus.HistogramVec // metrics monitor
}

func NewVerifier(ceConfig *conf.CoreEngineConfig) (*Verifier, error) {
	verifier := &Verifier{
		chainId:       ceConfig.ChainId,
		wg:            sync.WaitGroup{},
		log:           ceConfig.Log,
		abftCache:     ceConfig.ABFTCache,
		ledgerCache:   ceConfig.LedgerCache,
		msgBus:        ceConfig.MsgBus,
		verifyTimeout: DEFAULT_VERIFY_TIMEOUT,
		txPool:        ceConfig.TxPool,
	}
	conf := &common.VerifierBlockConf{
		ChainConf:       ceConfig.ChainConf,
		Log:             ceConfig.Log,
		LedgerCache:     ceConfig.LedgerCache,
		Ac:              ceConfig.AC,
		SnapshotManager: ceConfig.SnapshotManager,
		VmMgr:           ceConfig.VmMgr,
		TxPool:          ceConfig.TxPool,
		BlockchainStore: ceConfig.BlockchainStore,
		StoreHelper:     ceConfig.StoreHelper,
	}
	verifier.verifierBlock = common.NewVerifierBlock(conf)
	var err error
	verifier.goRoutinePool, err = ants.NewPool(len(ceConfig.ChainConf.ChainConfig().Consensus.Nodes), ants.WithPreAlloc(true))
	if err != nil {
		return nil, fmt.Errorf("new verifier failed: %s", err.Error())
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		verifier.metricBlockVerifyTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_VERIFIER, "metric_block_verify_time",
			"block verify time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return verifier, nil
}

func (v *Verifier) verifyBlock(block *commonPb.Block) (bool, map[string]*commonPb.TxRWSet, error) {
	startTick := utils.CurrentTimeMillisSeconds()
	emptyTxRwSetMap := make(map[string]*commonPb.TxRWSet)
	if err := utils.IsEmptyBlock(block); err != nil {
		return false, emptyTxRwSetMap, err
	}
	err := common.VerifyHeight(block.Header.BlockHeight, v.ledgerCache)
	if err != nil {
		return false, emptyTxRwSetMap, err
	}
	v.log.Debugf("verify receive [%d](%x,%d,%d)",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs))

	if err = common.IsTxCountValid(block); err != nil {
		return false, emptyTxRwSetMap, err
	}

	lastBlock := v.ledgerCache.GetLastCommittedBlock()
	err = common.CheckPreBlock(block, lastBlock)
	if err != nil {
		return false, emptyTxRwSetMap, err
	}

	timeLasts := make([]int64, 0)
	txRwSetMap, _, timeLasts, err := v.verifierBlock.ValidateBlock(block, lastBlock, timeLasts)
	if err != nil {
		return false, emptyTxRwSetMap, err
	}
	// mark transactions in block as pending status in txpool
	v.txPool.AddTxsToPendingCache(block.Txs, block.Header.BlockHeight)

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	v.log.Infof("verify success [%d,%x](%v,%d)", block.Header.BlockHeight, block.Header.BlockHash, timeLasts, elapsed)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		v.metricBlockVerifyTime.WithLabelValues(v.chainId).Observe(float64(elapsed) / 1000)
	}
	return true, txRwSetMap, nil
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
	if block == nil {
		return fmt.Errorf("verify failed, block is nil")
	}

	// repeat verify
	if v.abftCache.HasVerifiedTxBatch(block.Header.BlockHash) {
		if mode == protocol.CONSENSUS_VERIFY {
			verifyResult, _ := v.abftCache.IsVerifiedTxBatchSuccess(block.Header.BlockHash)
			v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, verifyResult))
		}
		return nil
	}

	//nodes that pack the txBatch do not need to verify
	proposedTxBatchCache := v.abftCache.GetProposedTxBatch()
	fingerPrint := utils.CalcBlockFingerPrint(block)
	if proposedTxBatchCache != nil &&
		string(proposedTxBatchCache.GetFingerPrint()) == string(fingerPrint) &&
		hex.EncodeToString(block.Header.BlockHash) == hex.EncodeToString(proposedTxBatchCache.GetTxBatch().Header.BlockHash) {
		verifyResult := true
		err := v.abftCache.AddVerifiedTxBatch(block, verifyResult, proposedTxBatchCache.GetRwSetMap())
		if err != nil {
			err = fmt.Errorf("sync cache the verified block faield: %s, blockHeight(%d), blockHash(%s)", err.Error(),
				block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
			return err
		}
		v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, verifyResult))
		return nil
	}
	verifyResult, rwSetMap, err := v.verifyBlock(block)
	if err != nil {
		v.log.Errorf("verify failed:%s,[%d],(%s)", err.Error(), block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
	}

	err = v.abftCache.AddVerifiedTxBatch(block, verifyResult, rwSetMap)
	if err != nil {
		err = fmt.Errorf("sync cache the verified block faield: %s, blockHeight(%d), blockHash(%s)", err.Error(),
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
		return err
	}

	if mode == protocol.CONSENSUS_VERIFY {
		v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, verifyResult))
	}

	v.log.Debugf("verify block[%d] finish", block.Header.BlockHeight)
	return nil
}

func (v *Verifier) verify(block *commonPb.Block) error {
	return v.goRoutinePool.Submit(v.verifyTask(block, protocol.CONSENSUS_VERIFY))
}

func (v *Verifier) verifyTask(block *commonPb.Block, mode protocol.VerifyMode) func() {
	return func() {
		err := v.VerifyBlock(block, mode)
		if err != nil {
			v.log.Errorf("verify txBatch failed: %s, height: %d, txBatchHash: %s", err, block.Header.BlockHeight,
				hex.EncodeToString(block.Header.BlockHash))
		}
	}
}
