/*
Copyright (C) BABEbc. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package commiter

import (
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/utils/v3"

	"chainmaker.org/chainmaker-go/module/core/cache"
	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/pb-go/v3/accesscontrol"
	commonpb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/consensus/abft"
	"chainmaker.org/chainmaker/protocol/v3"
)

// BlockCommitter block committer
type BlockCommitter struct {
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
	msgbus        msgbus.MessageBus
	retryList     []*commonpb.Transaction
	commonCommit  *common.CommitBlock
	lock          sync.Mutex
}

// NewCommitter param ceconfig, return BlockCommitter
func NewCommitter(ceConfig *conf.CoreEngineConfig) *BlockCommitter {
	committer := &BlockCommitter{
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
		msgbus:        ceConfig.MsgBus,
	}

	committer.commonCommit = common.NewCommitBlock(ceConfig)
	committer.merger = NewMerger()
	committer.merger.log = ceConfig.Log
	return committer
}

// Commit BlockCommitter Commit func, return error
func (bc *BlockCommitter) Commit(txBatchAfterABA *abft.TxBatchAfterABA) error {
	startTick := utils.CurrentTimeMillisSeconds()

	bc.lock.Lock()
	defer bc.lock.Unlock()
	blockHeight := txBatchAfterABA.BlockHeight
	txBatchHashs := txBatchAfterABA.TxBatchHash

	// check block height
	if err := common.VerifyHeight(blockHeight, bc.ledgerCache); err != nil {
		bc.log.Errorf("height verify fail,err: %s, height: (%d)", err.Error(), blockHeight)
		return err
	}

	// set txBatchID list & txBatchInfo
	err := bc.prepare(txBatchHashs)
	if err != nil {
		bc.log.Error("prepare commit fail,err: %s, height: (%d)", err.Error(), blockHeight)
		return err
	}

	// sort BatchID
	bc.sortTxBatchID()
	bc.log.Debugf("receive tx batch id [%s], height[%d], length[%d]", bc.txBatchIDList, blockHeight, len(bc.txBatchIDList))

	//var block *commonpb.Block
	rwSetMap := make(map[string]*commonpb.TxRWSet)
	// new block
	lastBlock := bc.ledgerCache.GetLastCommittedBlock()
	block, err := common.InitNewBlock(lastBlock, bc.identity, bc.chainID, bc.chainConf, false)
	if err != nil {
		bc.log.Error("init new block fail,err: %s,height: (%d)", err.Error(), blockHeight)
		return err
	}

	// set base TxBatch Id
	bc.merger.baseTxBatchID = bc.txBatchIDList[0]
	// rewrite block's Timestamp
	baseTxBatchInfo := bc.merger.txBatchInfo[bc.merger.baseTxBatchID].txBatch
	block.Header.BlockTimestamp = baseTxBatchInfo.Header.BlockTimestamp

	retryTxs := make([]*commonpb.Transaction, 0)
	if !bc.isEmptyBlock() {
		// get the new RWSetMap after conflict detection
		retryTxs, err = bc.merger.Merge(block, bc.txBatchIDList)
		if err != nil {
			bc.log.Error("merge txBatch fail,err: %s, height: (%d)", err.Error(), blockHeight)
			return err
		}

		rwSetMap = bc.merger.rwSetMap
		var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
		if err = common.FinalizeBlock(block,
			rwSetMap, aclFailTxs, bc.chainConf.ChainConfig().Crypto.Hash, bc.log); err != nil {
			bc.log.Error("finalize block fail,err: %s, height: (%d)", err.Error(), blockHeight)
			return err
		}
	}

	// set proposer nil
	block.Header.Proposer = &accesscontrol.Member{}
	hash, sig, err := utils.SignBlock(
		bc.chainConf.ChainConfig().Crypto.Hash, bc.identity, block)
	if err != nil {
		bc.log.Errorf("[%s]sign block failed, %s", bc.identity.GetMemberId(), err)
	}

	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig
	dbLasts, snapshotLasts, confLasts, otherLasts, pubEvent, filterLasts, blockInfo, err :=
		bc.commonCommit.CommitBlock(block, rwSetMap, nil)
	if err != nil {
		bc.log.Errorf("block common commit failed: %s, blockHeight: (%d)",
			err.Error(), block.Header.BlockHeight)
	}

	// synchronize new block height to consensus and sync module
	bc.msgbus.PublishSafe(msgbus.BlockInfo, blockInfo)

	// deal with tx(ABA fail)
	bc.handleABAFailTxs()

	//sync txpool(put retryList back txpool & delete blocked tx)
	if len(retryTxs) != 0 {
		bc.retryList = append(bc.retryList, retryTxs...)
	}
	bc.txPool.RetryTxs(bc.retryList)
	bc.txPool.RemoveTxs(block.Txs, protocol.NORMAL)

	//clear abft catche
	bc.abftCache.ClearAbftCache()

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	bc.log.Infof("commit block [%d](count:%d,hash:%x), "+
		"time used(db:%d,ss:%d,conf:%d,pubConEvent:%d, filter:%d,other:%d,total:%d)",
		blockHeight, block.Header.TxCount, block.Header.BlockHash, dbLasts,
		snapshotLasts, confLasts, pubEvent, filterLasts, otherLasts, elapsed)

	return nil
}

func (bc *BlockCommitter) handleABAFailTxs() {

	// get the verified txBatch from cache
	txBatchCacheList := bc.abftCache.GetVerifiedTxBatchsByResult(true)

	// get the txBatchID list before ABA
	txBatchIDListBeforeABA := make([]string, 0)
	txBatchMapBeforeABA := make(map[string]*commonpb.Block)
	for _, txBatchCache := range txBatchCacheList {
		txBatchID := hex.EncodeToString(txBatchCache.GetTxBatch().Header.BlockHash)
		txBatchIDListBeforeABA = append(txBatchIDListBeforeABA, txBatchID)
		txBatchMapBeforeABA[txBatchID] = txBatchCache.GetTxBatch()
	}

	// get the txBatch which ABA fail
	txBatchIDListFailABA := getABAFailTxBatchIDs(txBatchIDListBeforeABA, bc.merger.txBatchInfo)

	// record the tx which ABA fail
	bc.setRetryList(txBatchIDListFailABA, txBatchMapBeforeABA)
}

func (bc *BlockCommitter) sortTxBatchID() {
	if len(bc.txBatchIDList) > 1 {
		sort.Strings(bc.txBatchIDList)
	}
}

func (bc *BlockCommitter) setTxBatchInfo(txBatchHash []byte) bool {
	txBatch, err := bc.abftCache.GetVerifiedTxBatchByHash(txBatchHash)
	if err != nil {
		return false
	}

	if !txBatch.GetVerifyResult() {
		return false
	}

	bc.merger.txBatchInfo[hex.EncodeToString(txBatchHash)] = &TxBatchInfo{
		txBatch:  txBatch.GetTxBatch(),
		rwSetMap: txBatch.GetTxBatchRwSet(),
	}
	return true
}

func (bc *BlockCommitter) setRetryList(failTxBatchIDList []string, txBatchMapBeforeABA map[string]*commonpb.Block) {
	// find the repeat tx and delete it and put the other tx back to the txpool
	for _, batchID := range failTxBatchIDList {
		batch := txBatchMapBeforeABA[batchID]
		for _, tx := range batch.Txs {
			if _, ok := bc.merger.allTxsMap[tx.Payload.TxId]; !ok {
				bc.retryList = append(bc.retryList, tx)
			}
		}
	}
}

func (bc *BlockCommitter) prepare(txBatchHashs [][]byte) error {
	bc.clearCommiter()
	for _, hash := range txBatchHashs {
		// set txBatchInfo
		if ok := bc.setTxBatchInfo(hash); ok {
			// set txBatchIDList
			bc.txBatchIDList = append(bc.txBatchIDList, hex.EncodeToString(hash))
		}
	}
	return nil
}

func (bc *BlockCommitter) clearCommiter() {
	bc.txBatchIDList = make([]string, 0)
	bc.retryList = make([]*commonpb.Transaction, 0)

	// init merger
	bc.merger.txBatchInfo = make(map[string]*TxBatchInfo)
	bc.merger.baseTxBatchID = ""
	bc.merger.rwSetMap = make(map[string]*commonpb.TxRWSet)
	bc.merger.allTxsMap = make(map[string]*commonpb.Transaction)
}

func (bc *BlockCommitter) isEmptyBlock() bool {
	for _, txBatchID := range bc.txBatchIDList {
		if len(bc.merger.txBatchInfo[txBatchID].txBatch.Txs) != 0 {
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

// AddBlock params block, return error
func (bc *BlockCommitter) AddBlock(block *commonpb.Block) error {
	startTick := utils.CurrentTimeMillisSeconds()
	bc.lock.Lock()
	defer bc.lock.Unlock()

	//verify height
	err := common.VerifyHeight(block.Header.BlockHeight, bc.ledgerCache)
	if err != nil {
		return err
	}

	abftBlock, err := bc.abftCache.GetVerifiedTxBatchByHash(block.Header.BlockHash)
	if err != nil {
		return err
	}
	if abftBlock == nil {
		return fmt.Errorf("[AddBlock] the block is not in the cache, "+
			"blockHeight(%d), blockHash(%s)", block.Header.BlockHeight,
			hex.EncodeToString(block.Header.BlockHash))
	}

	dbLasts, snapshotLasts, confLasts, otherLasts, pubEvent, filterLasts, blockInfo, err :=
		bc.commonCommit.CommitBlock(abftBlock.GetTxBatch(), abftBlock.GetTxBatchRwSet(), nil)
	if err != nil {
		bc.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
		return err
	}

	// synchronize new block height to consensus and sync module
	bc.msgbus.PublishSafe(msgbus.BlockInfo, blockInfo)

	//sync txpool(put retryList back txpool & delete blocked tx)
	bc.txPool.RemoveTxs(block.Txs, protocol.NORMAL)

	//clear abft catche
	bc.abftCache.ClearAbftCache()

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	bc.log.Infof("add block [%d](count:%d,hash:%x), "+
		"time used(db:%d,ss:%d,conf:%d,pubConEvent:%d,filter:%d,other:%d,total:%d)",
		block.Header.BlockHeight, block.Header.TxCount, block.Header.BlockHash,
		dbLasts, snapshotLasts, confLasts, pubEvent, filterLasts, otherLasts, elapsed)

	return nil
}
