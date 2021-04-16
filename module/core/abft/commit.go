/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"encoding/hex"
	"errors"
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
