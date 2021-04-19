/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/localconf"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
)

type Committer struct {
	chainID      string
	blockHeight  int64
	branchIDList []string // branchID After ABA
	ledgerCache  protocol.LedgerCache
	abftCache    cache.AbftCache
	scheduler    *Scheduler
	log          *logger.CMLogger // logger
	txPool       protocol.TxPool
	identity     protocol.SigningMember
	chainConf    protocol.ChainConf
	retryList    []*commonpb.Transaction
	commonCommit *common.CommitBlock
	proposer     *Proposer
}

func NewCommitter(coreExecute *CoreExecute, proposer *Proposer) *Committer {
	committer := &Committer{
		chainID:      coreExecute.chainId,
		blockHeight:  0,
		branchIDList: make([]string, 0),
		ledgerCache:  coreExecute.ledgerCache,
		abftCache:    *coreExecute.abftCache,
		log:          coreExecute.log,
		txPool:       coreExecute.txPool,
		identity:     coreExecute.identity,
		chainConf:    coreExecute.chainConf,
		proposer:     proposer,
	}
	cbConf := &common.CommitBlockConf{
		Store:           coreExecute.blockchainStore,
		Log:             coreExecute.log,
		SnapshotManager: coreExecute.snapshotManager,
		TxPool:          coreExecute.txPool,
		LedgerCache:     coreExecute.ledgerCache,
		ChainConf:       coreExecute.chainConf,
		MsgBus:          coreExecute.msgBus,
	}
	committer.commonCommit = common.NewCommitBlock(cbConf)
	committer.scheduler = NewScheduler()
	return committer
}

func (c *Committer) Commit() error {
	// sort branchID
	c.sortBranchID()

	// new block
	lastBlock := c.ledgerCache.GetLastCommittedBlock()
	block, err := common.InitNewBlock(lastBlock, c.identity, c.chainID, c.chainConf)
	if err != nil {
		return err
	}

	c.scheduler.block = block
	c.scheduler.branchIDList = c.branchIDList
	// get the new RWSetMap after conflict detection
	newRWSetMap, err := c.scheduler.Schedule()
	if err != nil {
		return err
	}

	// get the	retryList after schedule
	c.retryList = c.scheduler.retryList

	// get the verified branch from cache TODO ABFT
	branchCacheList := c.abftCache.GetVerifiedAbftTxBatchsByCode(cache.SUCCESS)

	// get the branchID list before ABA
	branchIDListBeforeABA := make([]string, 0)
	txBranchMapBeforeABA := make(map[string]*commonpb.Block)
	for _, branchCache := range branchCacheList {
		branchID := hex.EncodeToString(branchCache.GetTxBatch().Header.BlockHash)
		branchIDListBeforeABA = append(branchIDListBeforeABA, branchID)
		txBranchMapBeforeABA[branchID] = branchCache.GetTxBatch()
	}

	// get the branch which ABA fail
	branchIDListFailABA := c.getTheABAFailBranchID(branchIDListBeforeABA)

	// handle the tx which ABA fail
	c.handelABAFailTranstraction(branchIDListFailABA, txBranchMapBeforeABA)

	block.Header.BlockTimestamp = baseBranchInfo.Header.BlockTimestamp
	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	err = common.FinalizeBlock(block, newRWSetMap, aclFailTxs, c.chainConf.ChainConfig().Crypto.Hash)
	if err != nil {
		return err
	}

	hash, sig, err := utils.SignBlock(c.chainConf.ChainConfig().Crypto.Hash, c.identity, block)
	if err != nil {
		c.log.Errorf("[%s]sign block failed, %s", c.identity.GetMemberId(), err)
	}

	// get the base branch info
	baseBranchId := c.branchIDList[0]

	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig

	//ear abft catche
	c.abftCache.ClearAbftCache()

	//CommitBlock the action that all consensus types do when a block is committed
	err = c.commonCommit.CommitBlock(block, newRWSetMap)
	if err != nil {
		c.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
	}

	//sync txpool
	c.txPool.RetryAndRemoveTxs(c.retryList, block.Txs)

	//set propose status
	c.proposer.SetProposeStatus(NoPackaging)
	return nil
}

func (c *Committer) sortBranchID() {
	sort.Strings(c.branchIDList)
}

func (c *Committer) getConfirmedBranchInfo(branchID []byte) error {
	branch, err := c.abftCache.GetVerifiedTxBatchByHash(branchID)
	if err != nil {
		return err
	}

	if branch.GetCode() == cache.SUCCESS {
		var branchInfo *BranchInfo
		branchInfo.branch = branch.GetTxBatch()
		branchInfo.rwSetMap = branch.GetTxBatchRwSet()
		c.scheduler.branchInfo[hex.EncodeToString(branchID)] = branchInfo
	}
	return nil
}

func (c *Committer) getTheABAFailBranchID(branchIDListBeforeABA []string) []string {
	failedBranchIDs := make([]string, 0)
	for _, branchID := range branchIDListBeforeABA {
		if _, ok := c.scheduler.branchInfo[branchID]; !ok {
			failedBranchIDs = append(failedBranchIDs, branchID)
		}
	}
	return failedBranchIDs
}

func (c *Committer) handelABAFailTranstraction(failBranchIDList []string, txBranchMapBeforeABA map[string]*commonpb.Block) {
	// find the repeat tx and delete it and put the other tx back to the txpool
	for _, branchID := range failBranchIDList {
		branch := txBranchMapBeforeABA[branchID]
		for _, tx := range branch.Txs {
			if _, ok := c.scheduler.allTransMap[tx.Header.TxId]; !ok {
				c.retryList = append(c.retryList, tx)
			}
		}
	}
}

func (c *Committer) verifyHeight(height int64) (bool, error) {
	currentHeight, err := c.ledgerCache.CurrentHeight()
	if err != nil {
		return false, err
	}
	if currentHeight+1 != height {
		return false, errors.New("the ABA signal height is inconsistent with the cache")
	}
	return true, nil
}

func (c *Committer) AddBlock(block *commonpb.Block) error {
	startTick := utils.CurrentTimeMillisSeconds()
	c.log.Debugf("add block(%d,%x)=(%x,%d,%d)",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash, block.Header.TxCount, len(block.Txs))
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error

	height := block.Header.BlockHeight
	if err = chain.isBlockLegal(block); err != nil {
		chain.log.Errorf("block illegal [%d](hash:%x), %s", height, block.Header.BlockHash, err)
		return err
	}

	lastProposed, rwSetMap := chain.proposalCache.GetProposedBlock(block)
	if err = chain.checkLastProposedBlock(block, lastProposed, err, height, rwSetMap); err != nil {
		return err
	}

	// record block
	rwSet := chain.rearrangeRWSet(block, rwSetMap)

	checkLasts := utils.CurrentTimeMillisSeconds() - startTick
	startDBTick := utils.CurrentTimeMillisSeconds()
	if err = chain.blockchainStore.PutBlock(block, rwSet); err != nil {
		// if put db error, then panic
		chain.log.Error(err)
		panic(err)
	}
	dbLasts := utils.CurrentTimeMillisSeconds() - startDBTick

	// clear snapshot
	startSnapshotTick := utils.CurrentTimeMillisSeconds()
	if err = chain.snapshotManager.NotifyBlockCommitted(block); err != nil {
		err = fmt.Errorf("notify snapshot error [%d](hash:%x)",
			lastProposed.Header.BlockHeight, lastProposed.Header.BlockHash)
		chain.log.Error(err)
		return err
	}
	snapshotLasts := utils.CurrentTimeMillisSeconds() - startSnapshotTick

	// notify chainConf to update config when config block committed
	startConfTick := utils.CurrentTimeMillisSeconds()
	if err = chain.notifyChainConf(block, err); err != nil {
		return err
	}
	confLasts := utils.CurrentTimeMillisSeconds() - startConfTick

	// Remove txs from txpool. Remove will invoke proposeSignal from txpool if pool size > txcount
	startPoolTick := utils.CurrentTimeMillisSeconds()
	txRetry := chain.syncWithTxPool(block, height)
	chain.log.Infof("remove txs[%d] and retry txs[%d] in add block", len(block.Txs), len(txRetry))
	chain.txPool.RetryAndRemoveTxs(txRetry, block.Txs)
	poolLasts := utils.CurrentTimeMillisSeconds() - startPoolTick

	startOtherTick := utils.CurrentTimeMillisSeconds()
	chain.ledgerCache.SetLastCommittedBlock(block)
	chain.proposalCache.ClearProposedBlockAt(height)
	bi := &commonpb.BlockInfo{
		Block:     block,
		RwsetList: rwSet,
	}
	// synchronize new block height to consensus and sync module
	chain.msgBus.Publish(msgbus.BlockInfo, bi)

	if err = chain.monitorCommit(bi); err != nil {
		return err
	}

	otherLasts := utils.CurrentTimeMillisSeconds() - startOtherTick
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	chain.log.Infof("commit block [%d](count:%d,hash:%x), time used(check:%d,db:%d,ss:%d,conf:%d,pool:%d,other:%d,total:%d)",
		height, block.Header.TxCount, block.Header.BlockHash, checkLasts, dbLasts, snapshotLasts, confLasts, poolLasts, otherLasts, elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		chain.metricBlockCommitTime.WithLabelValues(chain.chainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}