/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
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
		return nil, fmt.Errorf("new verifier failed: %s", err.Error())
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		verifier.metricBlockVerifyTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_VERIFIER, "metric_block_verify_time",
			"block verify time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return verifier, nil
}

func (v *Verifier) verifyHeight(block *commonPb.Block) error {
	currentHeight, err := v.ledgerCache.CurrentHeight()
	if err != nil {
		return err
	}
	if currentHeight+1 != block.Header.BlockHeight {
		return errors.New("the packaging signal height is inconsistent with the cache")
	}
	return nil
}

// todo add repeat verify!
func (v *Verifier) verify(block *commonPb.Block) (bool, map[string]*commonPb.TxRWSet, error) {
	startTick := utils.CurrentTimeMillisSeconds()
	if err := utils.IsEmptyBlock(block); err != nil {
		return false, nil, err
	}
	err := v.verifyHeight(block)
	if err != nil {
		return false, nil, err
	}
	v.log.Debugf("verify receive [%d](%x,%d,%d)",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs))

	//nodes that pack the txBatch do not need to verify
	proposedTxBatchCache := v.abftCache.GetProposedTxBatchCache()
	if proposedTxBatchCache != nil {
		if bytes.Equal(proposedTxBatchCache.GetTxBatch().Header.BlockHash, block.Header.BlockHash) {
			return true, proposedTxBatchCache.GetRwSetMap(), nil
		}
	}
	txRwSetMap, timeLasts, err := v.verifyBlock.ValidateBlock(block)
	if err != nil {
		return false, nil, err
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
	verifyResult, rwSetMap, err := v.verify(block)
	if err != nil {
		v.log.Errorf("verify block failed: %s, blockHeignt: (%d)", err.Error(), block.Header.BlockHeight)
	}
	if mode == protocol.CONSENSUS_VERIFY {
		err = v.verifyResult(block, rwSetMap, verifyResult)
		if err != nil {
			return err
		}
		return nil
	}
	//after verifing block,sync nodes cache the block
	err = v.abftCache.AddVerifiedTxBatch(block, verifyResult, rwSetMap)
	if err != nil {
		err = fmt.Errorf("sync cache the verified block faield: %s, blockHeight(%d), blockHash(%s)", err.Error(),
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
		return err
	}
	return nil
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

func (v *Verifier) verifyResult(block *commonPb.Block, rwSet map[string]*commonPb.TxRWSet, verifyResult bool) error {
	err := v.abftCache.AddVerifiedTxBatch(block, verifyResult, rwSet)
	if err != nil {
		return fmt.Errorf("abft add tx batch faield: %s, blockHeight(%d), txBatchHash(%s)", err.Error(),
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
	}
	v.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, verifyResult))
	return nil
}
