package hbbft

import (
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"encoding/hex"
	"sort"
)

type Committer struct {
	chainID       string
	blockHeight   int64
	branchIDList  []string // branchID After ABA
	ledgerCache   protocol.LedgerCache
	hbbftCache    cache.HbbftCache
	scheduler     *Scheduler
	log           *logger.CMLogger // logger
	txPool        protocol.TxPool
	identity      protocol.SigningMember
	chainConf     protocol.ChainConf
	blockCommiter protocol.BlockCommitter
}

func NewCommitter(coreExecute *CoreExecute) (*Committer, error) {
	return &Committer{
		chainID:      coreExecute.chainId,
		blockHeight:  0,
		branchIDList: nil,
		ledgerCache:  coreExecute.ledgerCache,
		hbbftCache:   *coreExecute.hbbftCache,
		log:          coreExecute.log,
		txPool:       coreExecute.txPool,
		identity:     coreExecute.identity,
		chainConf:    coreExecute.chainConf,
	}, nil
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

	// 冲突检测后，返回新的读写集
	newTxRWSetMap, txMap, err := c.scheduler.Schedule()
	if err != nil {
		return err
	}

	// get the verified branch from cache
	branchCacheList := c.hbbftCache.GetVerifiedHbbftTxBatchsByCode(cache.SUCCESS)

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
	c.handelABAFailTranstraction(branchIDListFailABA, txBranchMapBeforeABA, txMap)

	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	err = common.FinalizeBlock(block, newTxRWSetMap, aclFailTxs, c.chainConf.ChainConfig().Crypto.Hash)
	if err != nil {
		return err
	}

	// todo AddBlock

	return nil
}

func (c *Committer) sortBranchID() {
	sort.Strings(c.branchIDList)
}

func (c *Committer) getConfirmedBranchInfo(branchID []byte) error {
	branch, err := c.hbbftCache.GetVerifiedTxBatchByHash(branchID)
	if err != nil {
		return err
	}

	if branch.GetCode() == cache.SUCCESS {
		var branchInfo *BranchInfo
		branchInfo.confirmedBranch = branch.GetTxBatch()
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

func (c *Committer) handelABAFailTranstraction(failBranchIDList []string, txBranchMapBeforeABA map[string]*commonpb.Block, txMap map[string]bool) {
	// find the repeat tx and delete it and put the other tx back to the txpool

	retryTxList := make([]*commonpb.Transaction, 0)
	for _, branchID := range failBranchIDList {
		branch := txBranchMapBeforeABA[branchID]
		for _, tx := range branch.Txs {
			if _, ok := txMap[tx.Header.TxId]; !ok {
				retryTxList = append(retryTxList, tx)
			}
		}
	}

	c.txPool.RetryAndRemoveTxs(retryTxList, nil)
}
