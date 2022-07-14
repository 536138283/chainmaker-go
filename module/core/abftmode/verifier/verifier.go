/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/module/core/cache"
	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/panjf2000/ants/v2"
	"github.com/prometheus/client_golang/prometheus"
)

const DEFAULT_VERIFY_TIMEOUT = time.Second * 10

type BlockVerifier struct {
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
	chainConf             protocol.ChainConf
	metricBlockVerifyTime *prometheus.HistogramVec // metrics monitor
}

func NewVerifier(ceConfig *conf.CoreEngineConfig, txScheduler protocol.TxScheduler) (protocol.BlockVerifier, error) {
	verifier := &BlockVerifier{
		chainId:       ceConfig.ChainId,
		wg:            sync.WaitGroup{},
		log:           ceConfig.Log,
		abftCache:     ceConfig.ABFTCache,
		ledgerCache:   ceConfig.LedgerCache,
		msgBus:        ceConfig.MsgBus,
		verifyTimeout: DEFAULT_VERIFY_TIMEOUT,
		txPool:        ceConfig.TxPool,
		chainConf:     ceConfig.ChainConf,
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
		TxScheduler:     txScheduler,
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

func (bv *BlockVerifier) verifyBlock(block *commonPb.Block, mode protocol.VerifyMode) (bool, map[string]*commonPb.TxRWSet, error) {
	startTick := utils.CurrentTimeMillisSeconds()
	emptyTxRwSetMap := make(map[string]*commonPb.TxRWSet)
	if err := utils.IsEmptyBlock(block); err != nil {
		return false, emptyTxRwSetMap, err
	}
	err := common.VerifyHeight(block.Header.BlockHeight, bv.ledgerCache)
	if err != nil {
		return false, emptyTxRwSetMap, err
	}
	bv.log.Debugf("verify receive [%d](%x,%d,%d)",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs))

	if err = common.IsTxCountValid(block); err != nil {
		return false, emptyTxRwSetMap, err
	}

	lastBlock := bv.ledgerCache.GetLastCommittedBlock()
	err = common.CheckPreBlock(block, lastBlock)
	if err != nil {
		return false, emptyTxRwSetMap, err
	}

	hashType := bv.chainConf.ChainConfig().Crypto.Hash
	timeLasts := make(map[string]int64)
	txRwSetMap, _, timeLasts, _, err := bv.verifierBlock.ValidateBlock(block, lastBlock, hashType, timeLasts, mode)
	if err != nil {
		return false, emptyTxRwSetMap, err
	}
	// mark transactions in block as pending status in txpool
	bv.txPool.AddTxsToPendingCache(block.Txs, block.Header.BlockHeight)

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	bv.log.Infof("verify success [%d,%x](%v,%d)", block.Header.BlockHeight, block.Header.BlockHash, timeLasts, elapsed)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		bv.metricBlockVerifyTime.WithLabelValues(bv.chainId).Observe(float64(elapsed) / 1000)
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

func (bv *BlockVerifier) VerifyBlock(block *commonPb.Block, mode protocol.VerifyMode) error {
	return bv.goRoutinePool.Submit(bv.verifyTask(block,mode))
}

func (bv *BlockVerifier) VerifyBlockSync(block *commonPb.Block, mode protocol.VerifyMode) (*consensuspb.VerifyResult, error) {
	panic("implement me")
}

func (bv *BlockVerifier) VerifyBlockWithRwSets(block *commonPb.Block, rwsets []*commonPb.TxRWSet, mode protocol.VerifyMode) error {
	panic("implement me")
}

func (bv *BlockVerifier) Verify(block *commonPb.Block, mode protocol.VerifyMode) error {
	if block == nil {
		return fmt.Errorf("verify failed, block is nil")
	}

	// repeat verify
	if bv.abftCache.HasVerifiedTxBatch(block.Header.BlockHash) {
		if mode == protocol.CONSENSUS_VERIFY {
			verifyResult, _ := bv.abftCache.IsVerifiedTxBatchSuccess(block.Header.BlockHash)
			bv.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, verifyResult))
		}
		return nil
	}

	//nodes that pack the txBatch do not need to verify
	proposedTxBatchCache := bv.abftCache.GetProposedTxBatch()
	fingerPrint := utils.CalcBlockFingerPrint(block)
	if proposedTxBatchCache != nil &&
		string(proposedTxBatchCache.GetFingerPrint()) == string(fingerPrint) &&
		hex.EncodeToString(block.Header.BlockHash) == hex.EncodeToString(proposedTxBatchCache.GetTxBatch().Header.BlockHash) {
		verifyResult := true
		err := bv.abftCache.AddVerifiedTxBatch(block, verifyResult, proposedTxBatchCache.GetRwSetMap())
		if err != nil {
			err = fmt.Errorf("sync cache the verified block faield: %s, blockHeight(%d), blockHash(%s)", err.Error(),
				block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
			return err
		}
		bv.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, verifyResult))
		return nil
	}
	verifyResult, rwSetMap, err := bv.verifyBlock(block, mode)
	if err != nil {
		bv.log.Errorf("verify failed:%s,[%d],(%s)", err.Error(), block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
	}

	err = bv.abftCache.AddVerifiedTxBatch(block, verifyResult, rwSetMap)
	if err != nil {
		err = fmt.Errorf("sync cache the verified block faield: %s, blockHeight(%d), blockHash(%s)", err.Error(),
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
		return err
	}

	if mode == protocol.CONSENSUS_VERIFY {
		bv.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, verifyResult))
	}

	bv.log.Debugf("verify block[%d] finish", block.Header.BlockHeight)
	return nil
}

func (bv *BlockVerifier) verifyTask(block *commonPb.Block, mode protocol.VerifyMode) func() {
	return func() {
		err := bv.Verify(block, mode)
		if err != nil {
			bv.log.Errorf("verify txBatch failed: %s, height: %d, txBatchHash: %s", err, block.Header.BlockHeight,
				hex.EncodeToString(block.Header.BlockHash))
		}
	}
}
