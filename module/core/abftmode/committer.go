/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abftmode

import (
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/monitor"
	"chainmaker.org/chainmaker-go/utils"
	commonpb "chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/consensus/abft"
	"chainmaker.org/chainmaker/protocol"
	"encoding/hex"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sort"
	"sync"
)

type Committer struct {
	chainID       string
	blockHeight   int64
	txBatchIDList []string // BatchID After ABA
	ledgerCache   protocol.LedgerCache
	abftCache     *cache.AbftCache
	merger        *Merger
	log           protocol.Logger // logger
	txPool        protocol.TxPool
	identity      protocol.SigningMember
	chainConf     protocol.ChainConf
	retryList     []*commonpb.Transaction
	commonCommit  *common.CommitBlock
	lock          sync.Mutex
}

func NewCommitter(ceConfig *conf.CoreEngineConfig) *Committer {
	committer := &Committer{
		chainID:       ceConfig.ChainId,
		blockHeight:   0,
		txBatchIDList: make([]string, 0),
		ledgerCache:   ceConfig.LedgerCache,
		abftCache:     ceConfig.ABFTCache,
		log:           ceConfig.Log,
		txPool:        ceConfig.TxPool,
		identity:      ceConfig.Identity,
		chainConf:     ceConfig.ChainConf,
		lock:          sync.Mutex{},
	}

	var metricBlockSize *prometheus.HistogramVec
	var metricBlockCounter    *prometheus.CounterVec
	var metricTxCounter       *prometheus.CounterVec
	var metricBlockCommitTime *prometheus.HistogramVec
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		metricBlockSize = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockSize,
			monitor.HelpCurrentBlockSizeMetric, prometheus.ExponentialBuckets(1024, 2, 12), monitor.ChainId)

		metricBlockCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockCounter,
			monitor.HelpBlockCountsMetric, monitor.ChainId)

		metricTxCounter = monitor.NewCounterVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricTxCounter,
			monitor.HelpTxCountsMetric, monitor.ChainId)

		metricBlockCommitTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_COMMITTER, monitor.MetricBlockCommitTime,
			monitor.HelpBlockCommitTimeMetric, []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, monitor.ChainId)
	}

	cbConf := &common.CommitBlockConf{
		Store:                 ceConfig.BlockchainStore,
		Log:                   ceConfig.Log,
		SnapshotManager:       ceConfig.SnapshotManager,
		TxPool:                ceConfig.TxPool,
		LedgerCache:           ceConfig.LedgerCache,
		ChainConf:             ceConfig.ChainConf,
		MsgBus:                ceConfig.MsgBus,
		MetricBlockCommitTime: metricBlockCommitTime,
		MetricBlockCounter:    metricBlockCounter,
		MetricBlockSize:       metricBlockSize,
		MetricTxCounter:       metricTxCounter,
	}
	committer.commonCommit = common.NewCommitBlock(cbConf)
	committer.merger = NewMerger()
	committer.merger.log = ceConfig.Log
	return committer
}

func (c *Committer) Commit(txBatchAfterABA *abft.TxBatchAfterABA) error {
	startTick := utils.CurrentTimeMillisSeconds()

	c.lock.Lock()
	defer c.lock.Unlock()
	blockHeight := txBatchAfterABA.BlockHeight
	txBatchHashs := txBatchAfterABA.TxBatchHash

	// check block height
	if err := common.VerifyHeight(blockHeight, c.ledgerCache); err != nil {
		c.log.Errorf("height verify fail,err: %s, height: (%d)", err.Error(), blockHeight)
		return err
	}

	// set txBatchID list & txBatchInfo
	err := c.prepare(txBatchHashs)
	if err != nil {
		c.log.Error("prepare commit fail,err: %s, height: (%d)", err.Error(), blockHeight)
		return err
	}

	// sort BatchID
	c.sortTxBatchID()
	c.log.Debugf("receive tx batch id [%s], height[%d], length[%d]", c.txBatchIDList, blockHeight, len(c.txBatchIDList))

	//var block *commonpb.Block
	rwSetMap := make(map[string]*commonpb.TxRWSet, 0)
	// new block
	lastBlock := c.ledgerCache.GetLastCommittedBlock()
	block, err := common.InitNewBlock(lastBlock, c.identity, c.chainID, c.chainConf)
	if err != nil {
		c.log.Error("init new block fail,err: %s,height: (%d)", err.Error(), blockHeight)
		return err
	}

	// set base TxBatch Id
	c.merger.baseTxBatchID = c.txBatchIDList[0]
	// rewrite block's Timestamp
	baseTxBatchInfo := c.merger.txBatchInfo[c.merger.baseTxBatchID].txBatch
	block.Header.BlockTimestamp = baseTxBatchInfo.Header.BlockTimestamp

	if !c.isEmptyBlock() {
		// get the new RWSetMap after conflict detection
		if err = c.merger.Merge(block, c.txBatchIDList); err != nil {
			c.log.Error("merge txBatch fail,err: %s, height: (%d)", err.Error(), blockHeight)
			return err
		}

		rwSetMap = c.merger.rwSetMap
		var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
		err = common.FinalizeBlock(block, rwSetMap, aclFailTxs, c.chainConf.ChainConfig().Crypto.Hash, c.log)
		if err != nil {
			c.log.Error("finalize block fail,err: %s, height: (%d)", err.Error(), blockHeight)
			return err
		}
	}

	// set proposer nil
	block.Header.Proposer = []byte{}
	hash, sig, err := utils.SignBlock(c.chainConf.ChainConfig().Crypto.Hash, c.identity, block)
	if err != nil {
		c.log.Errorf("[%s]sign block failed, %s", c.identity.GetMemberId(), err)
	}

	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig
	dbLasts, snapshotLasts, confLasts, otherLasts, pubEvent, err := c.commonCommit.CommitBlock(block, rwSetMap, nil)
	if err != nil {
		c.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
	}

	// deal with tx(ABA fail)
	c.handleABAFailTxs()

	//sync txpool(put retryList back txpool & delete blocked tx)
	c.txPool.RetryAndRemoveTxs(c.retryList, block.Txs)

	//clear abft catche
	c.abftCache.ClearAbftCache()

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	c.log.Infof("commit block [%d](count:%d,hash:%x), time used(db:%d,ss:%d,conf:%d,pubConEvent:%d,other:%d,total:%d)",
		blockHeight, block.Header.TxCount, block.Header.BlockHash, dbLasts, snapshotLasts, confLasts, pubEvent, otherLasts, elapsed)

	return nil
}

func (c *Committer) handleABAFailTxs() {

	// get the verified txBatch from cache
	txBatchCacheList := c.abftCache.GetVerifiedTxBatchsByResult(true)

	// get the txBatchID list before ABA
	txBatchIDListBeforeABA := make([]string, 0)
	txBatchMapBeforeABA := make(map[string]*commonpb.Block)
	for _, txBatchCache := range txBatchCacheList {
		txBatchID := hex.EncodeToString(txBatchCache.GetTxBatch().Header.BlockHash)
		txBatchIDListBeforeABA = append(txBatchIDListBeforeABA, txBatchID)
		txBatchMapBeforeABA[txBatchID] = txBatchCache.GetTxBatch()
	}

	// get the txBatch which ABA fail
	txBatchIDListFailABA := getABAFailTxBatchIDs(txBatchIDListBeforeABA, c.merger.txBatchInfo)

	// record the tx which ABA fail
	c.setRetryList(txBatchIDListFailABA, txBatchMapBeforeABA)
}

func (c *Committer) sortTxBatchID() {
	if len(c.txBatchIDList) > 1 {
		sort.Strings(c.txBatchIDList)
	}
}

func (c *Committer) setTxBatchInfo(txBatchHash []byte) bool {
	txBatch, err := c.abftCache.GetVerifiedTxBatchByHash(txBatchHash)
	if err != nil {
		return false
	}

	if !txBatch.GetVerifyResult() {
		return false
	}

	c.merger.txBatchInfo[hex.EncodeToString(txBatchHash)] = &TxBatchInfo{
		txBatch:  txBatch.GetTxBatch(),
		rwSetMap: txBatch.GetTxBatchRwSet(),
	}
	return true
}

func (c *Committer) setRetryList(failTxBatchIDList []string, txBatchMapBeforeABA map[string]*commonpb.Block) {
	// find the repeat tx and delete it and put the other tx back to the txpool
	for _, BatchID := range failTxBatchIDList {
		Batch := txBatchMapBeforeABA[BatchID]
		for _, tx := range Batch.Txs {
			if _, ok := c.merger.allTxsMap[tx.Header.TxId]; !ok {
				c.retryList = append(c.retryList, tx)
			}
		}
	}
}

func (c *Committer) prepare(txBatchHashs [][]byte) error {
	c.clearCommiter()
	for _, hash := range txBatchHashs {
		// set txBatchInfo
		if ok := c.setTxBatchInfo(hash); ok {
			// set txBatchIDList
			c.txBatchIDList = append(c.txBatchIDList, hex.EncodeToString(hash))
		}
	}
	return nil
}

func (c *Committer) clearCommiter() {
	c.txBatchIDList = make([]string, 0)
	c.retryList = make([]*commonpb.Transaction, 0)

	// init merger
	c.merger.txBatchInfo = make(map[string]*TxBatchInfo)
	c.merger.baseTxBatchID = ""
	c.merger.rwSetMap = make(map[string]*commonpb.TxRWSet)
	c.merger.allTxsMap = make(map[string]*commonpb.Transaction)
}

func (c *Committer) isEmptyBlock() bool {
	for _, txBatchID := range c.txBatchIDList {
		if len(c.merger.txBatchInfo[txBatchID].txBatch.Txs) != 0 {
			return false
		}
	}

	return true
}

func getABAFailTxBatchIDs(txBatchIDListBeforeABA []string, txBatchInfo map[string]*TxBatchInfo) []string {
	failedBatchIDs := make([]string, 0)
	for _, BatchID := range txBatchIDListBeforeABA {
		if _, ok := txBatchInfo[BatchID]; !ok {
			failedBatchIDs = append(failedBatchIDs, BatchID)
		}
	}
	return failedBatchIDs
}

func (c *Committer) AddBlock(block *commonpb.Block) error {
	startTick := utils.CurrentTimeMillisSeconds()
	c.lock.Lock()
	defer c.lock.Unlock()

	//verify height
	err := common.VerifyHeight(block.Header.BlockHeight, c.ledgerCache)
	if err != nil {
		return err
	}

	abftBlock, err := c.abftCache.GetVerifiedTxBatchByHash(block.Header.BlockHash)
	if err != nil {
		return err
	}
	if abftBlock == nil {
		return fmt.Errorf("[AddBlock] the block is not in the cache, blockHeight(%d), blockHash(%s)", block.Header.BlockHeight,
			hex.EncodeToString(block.Header.BlockHash))
	}

	dbLasts, snapshotLasts, confLasts, otherLasts, pubEvent, err := c.commonCommit.CommitBlock(abftBlock.GetTxBatch(), abftBlock.GetTxBatchRwSet(), nil)
	if err != nil {
		c.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
		return err
	}

	//sync txpool(put retryList back txpool & delete blocked tx)
	c.txPool.RetryAndRemoveTxs(nil, block.Txs)

	//clear abft catche
	c.abftCache.ClearAbftCache()

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	c.log.Infof("add block [%d](count:%d,hash:%x), time used(db:%d,ss:%d,conf:%d,pubConEvent:%d,other:%d,total:%d)",
		block.Header.BlockHeight, block.Header.TxCount, block.Header.BlockHash, dbLasts, snapshotLasts, confLasts, pubEvent, otherLasts, elapsed)

	return nil
}
