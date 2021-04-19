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
	chainID       string
	blockHeight   int64
	txBatchIDList []string // BatchID After ABA
	ledgerCache   protocol.LedgerCache
	abftCache     cache.AbftCache
	merger        *Merger
	log           *logger.CMLogger // logger
	txPool        protocol.TxPool
	identity      protocol.SigningMember
	chainConf     protocol.ChainConf
	retryList     []*commonpb.Transaction
	commonCommit  *common.CommitBlock
	proposer      *Proposer
}

func NewCommitter(coreExecute *CoreExecute, proposer *Proposer) *Committer {
	committer := &Committer{
		chainID:       coreExecute.chainId,
		blockHeight:   0,
		txBatchIDList: make([]string, 0),
		ledgerCache:   coreExecute.ledgerCache,
		abftCache:     *coreExecute.abftCache,
		log:           coreExecute.log,
		txPool:        coreExecute.txPool,
		identity:      coreExecute.identity,
		chainConf:     coreExecute.chainConf,
		proposer:      proposer,
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
	committer.merger = NewMerger()
	return committer
}

func (c *Committer) Commit(blockHeight int64, txBatchHash [][]byte) error {
	// check block height
	ok, err := c.verifyHeight(blockHeight)
	if !ok {
		c.log.Errorf("after ABA the tx batch height is wrong: %s, height: (%d)", err.Error(), blockHeight)
		return err
	}

	// set txBatchID list
	err = c.setTxBatchIDList(txBatchHash)
	if err != nil {
		return err
	}

	// sort BatchID
	c.sortTxBatchID()

	var block *commonpb.Block
	var rwSetMap map[string]*commonpb.TxRWSet
	if !c.isEmptyBlock() {
		// new block
		lastBlock := c.ledgerCache.GetLastCommittedBlock()
		block, err = common.InitNewBlock(lastBlock, c.identity, c.chainID, c.chainConf)
		if err != nil {
			return err
		}

		c.merger.block = block
		c.merger.txBatchIDList = c.txBatchIDList
		// get the new RWSetMap after conflict detection
		if err = c.merger.Merge(); err != nil {
			return err
		}

		// get the base txBatch info
		baseTxBatchInfo := c.merger.txBatchInfo[c.merger.baseTxBatchID].txBatch
		block.Header.BlockTimestamp = baseTxBatchInfo.Header.BlockTimestamp

		rwSetMap = c.merger.rwSetMap
	} else {
		// empty block use the baseTxBatch's block
		block = c.merger.txBatchInfo[c.txBatchIDList[0]].txBatch

		rwSetMap = c.merger.txBatchInfo[c.txBatchIDList[0]].rwSetMap
	}

	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	err = common.FinalizeBlock(block, rwSetMap, aclFailTxs, c.chainConf.ChainConfig().Crypto.Hash)
	if err != nil {
		return err
	}

	hash, sig, err := utils.SignBlock(c.chainConf.ChainConfig().Crypto.Hash, c.identity, block)
	if err != nil {
		c.log.Errorf("[%s]sign block failed, %s", c.identity.GetMemberId(), err)
	}

	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig
	err = c.commonCommit.CommitBlock(block, rwSetMap)
	if err != nil {
		c.log.Errorf("block common commit failed: %s, blockHeight: (%d)", err.Error(), block.Header.BlockHeight)
	}

	// deal with tx(ABA fail)
	c.handleABAFailTxs()

	//sync txpool(put retryList back txpool & delete blocked tx)
	c.txPool.RetryAndRemoveTxs(c.retryList, block.Txs)

	//clear abft catche
	c.abftCache.ClearAbftCache()

	return nil
}

func (c *Committer) handleABAFailTxs() {

	// get the verified txBatch from cache
	txBatchCacheList := c.abftCache.GetVerifiedAbftTxBatchsByResult(true)

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

func (c *Committer) getConfirmedTxBatchInfo(txBatchID []byte) error {
	txBatch, err := c.abftCache.GetVerifiedTxBatchByHash(txBatchID)
	if err != nil {
		return err
	}

	if txBatch.GetVerifyResult() == true {
		var txBatchInfo *TxBatchInfo
		txBatchInfo.txBatch = txBatch.GetTxBatch()
		txBatchInfo.rwSetMap = txBatch.GetTxBatchRwSet()
		c.merger.txBatchInfo[hex.EncodeToString(txBatchID)] = txBatchInfo
	}
	return nil
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

func (c *Committer) verifyHeight(height int64) (bool, error) {
	currentHeight, err := c.ledgerCache.CurrentHeight()
	if err != nil {
		return false, err
	}
	if height != currentHeight+1 {
		return false, errors.New("the ABA signal height is inconsistent with the cache")
	}
	return true, nil
}

func (c *Committer) setTxBatchIDList(txBatchHash [][]byte) error {
	for i, _ := range txBatchHash {
		// BatchInfo After ABA
		if err := c.getConfirmedTxBatchInfo(txBatchHash[i]); err != nil {
			return err
		}

		// BatchIDList After ABA
		c.txBatchIDList = append(c.txBatchIDList, hex.EncodeToString(txBatchHash[i]))
	}
	return nil
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
