/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/hex"
	"fmt"
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
	log           *logger.CMLogger // logger
	txPool        protocol.TxPool
	identity      protocol.SigningMember
	chainConf     protocol.ChainConf
	retryList     []*commonpb.Transaction
	commonCommit  *common.CommitBlock
	lock          sync.Mutex
}

func NewCommitter(ceConfig *CoreExecuteConfig) *Committer {
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
	cbConf := &common.CommitBlockConf{
		Store:           ceConfig.BlockchainStore,
		Log:             ceConfig.Log,
		SnapshotManager: ceConfig.SnapshotManager,
		TxPool:          ceConfig.TxPool,
		LedgerCache:     ceConfig.LedgerCache,
		ChainConf:       ceConfig.ChainConf,
		MsgBus:          ceConfig.MsgBus,
	}
	committer.commonCommit = common.NewCommitBlock(cbConf)
	committer.merger = NewMerger()
	committer.merger.log = ceConfig.Log
	return committer
}

func (c *Committer) Commit(txBatchAfterABA *abft.TxBatchAfterABA) error {
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
		return err
	}

	// sort BatchID
	c.sortTxBatchID()
	c.log.Debugf("sort branchID index: %s", c.txBatchIDList)

	//var block *commonpb.Block
	rwSetMap := make(map[string]*commonpb.TxRWSet, 0)
	// new block
	lastBlock := c.ledgerCache.GetLastCommittedBlock()
	block, err := common.InitNewBlock(lastBlock, c.identity, c.chainID, c.chainConf)
	if err != nil {
		return err
	}

	// set base TxBatch Id
	c.merger.baseTxBatchID = c.txBatchIDList[0]
	// rewrite block's Timestamp
	baseTxBatchInfo := c.merger.txBatchInfo[c.merger.baseTxBatchID].txBatch
	block.Header.BlockTimestamp = baseTxBatchInfo.Header.BlockTimestamp

	if !c.isEmptyBlock() {
		c.log.Debugf("baseRWSetMap::: %s", c.merger.txBatchInfo[c.merger.baseTxBatchID])

		// get the new RWSetMap after conflict detection
		if err = c.merger.Merge(block, c.txBatchIDList); err != nil {
			return err
		}

		rwSetMap = c.merger.rwSetMap
		var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
		err = common.FinalizeBlock(block, rwSetMap, aclFailTxs, c.chainConf.ChainConfig().Crypto.Hash)
		if err != nil {
			return err
		}
	}

	// set proposer nil
	block.Header.Proposer = []byte{}
	c.log.Debugf("block before sig:::", c.merger.txBatchInfo)
	hash, sig, err := utils.SignBlock(c.chainConf.ChainConfig().Crypto.Hash, c.identity, block)
	if err != nil {
		c.log.Errorf("[%s]sign block failed, %s", c.identity.GetMemberId(), err)
	}

	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig
	//todo set sig empty
	block.Header.Signature = []byte{}
	c.log.Debugf("commit block: %s", block)
	err = c.commonCommit.CommitBlock(block, rwSetMap)
	if err != nil {
		c.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
	}

	// deal with tx(ABA fail)
	c.handleABAFailTxs()

	c.log.Debug("remove txs")
	//sync txpool(put retryList back txpool & delete blocked tx)
	c.txPool.RetryAndRemoveTxs(c.retryList, block.Txs)

	c.log.Debug("clear abft cache")
	//clear abft catche
	c.abftCache.ClearAbftCache()

	c.log.Debugf("commit finish, block: %s", block.Header)

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

func (c *Committer) setTxBatchInfo(txBatchHash []byte) (error, bool) {
	txBatch, err := c.abftCache.GetVerifiedTxBatchByHash(txBatchHash)
	if err != nil {
		return err, false
	}

	if !txBatch.GetVerifyResult() {
		return nil, false
	}

	c.merger.txBatchInfo[hex.EncodeToString(txBatchHash)] = &TxBatchInfo{
		txBatch:  txBatch.GetTxBatch(),
		rwSetMap: txBatch.GetTxBatchRwSet(),
	}
	return nil, true
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

	c.initCommiter()
	for _, hash := range txBatchHashs {
		// set txBatchInfo
		err, ok := c.setTxBatchInfo(hash)
		if err != nil {
			return err
		}

		// set txBatchIDList
		if ok {
			c.txBatchIDList = append(c.txBatchIDList, hex.EncodeToString(hash))
		}
	}
	return nil
}

func (c *Committer) initCommiter() {
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
		c.log.Debugf("txBatchInfo::: %s", c.merger.txBatchInfo)
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
	c.lock.Lock()
	defer c.lock.Unlock()

	c.log.Debugf("AddBlock::: start")

	//verify height
	err := common.VerifyHeight(block.Header.BlockHeight, c.ledgerCache)
	if err != nil {
		return err
	}
	c.log.Debugf("VerifyHeight::: finish!")
	abftBlock, err := c.abftCache.GetVerifiedTxBatchByHash(block.Header.BlockHash)
	if err != nil {
		return err
	}
	if abftBlock == nil {
		return fmt.Errorf("[AddBlock] the block is not in the cache, blockHeight(%d), blockHash(%s)", block.Header.BlockHeight,
			hex.EncodeToString(block.Header.BlockHash))
	}
	c.log.Debugf("CommitBlock::: start!")
	err = c.commonCommit.CommitBlock(abftBlock.GetTxBatch(), abftBlock.GetTxBatchRwSet())
	if err != nil {
		c.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
		return err
	}
	if err != nil {
		return err
	}

	c.log.Debug("remove txs")
	//sync txpool(put retryList back txpool & delete blocked tx)
	c.txPool.RetryAndRemoveTxs(nil, block.Txs)

	c.log.Debug("clear abft cache")
	//clear abft catche
	c.abftCache.ClearAbftCache()

	c.log.Debugf("AddBlock::: finish!")
	return nil
}
