/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	commonErrors "chainmaker.org/chainmaker/common/v2/errors"
	batch "chainmaker.org/chainmaker/txpool-batch/v2"

	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/utils/v2"

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

// DEFAULT_VERIFY_TIMEOUT default verify timeout
const DEFAULT_VERIFY_TIMEOUT = time.Second * 10

// BlockVerifier struct
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
	proposalCache         protocol.ProposalCache
	blockchainStore       protocol.BlockchainStore
	ac                    protocol.AccessControlProvider
	storeHelper           protocol.StoreHelper
	reentrantLocks        *common.ReentrantLocks   // reentrant lock for avoid concurrent verify block
	metricBlockVerifyTime *prometheus.HistogramVec // metrics monitor
}

// NewVerifier params CoreEngineConfig, TxScheduler, return BlockVerifier, error
func NewVerifier(ceConfig *conf.CoreEngineConfig, txScheduler protocol.TxScheduler) (protocol.BlockVerifier, error) {
	verifier := &BlockVerifier{
		chainId:         ceConfig.ChainId,
		wg:              sync.WaitGroup{},
		log:             ceConfig.Log,
		abftCache:       ceConfig.ABFTCache,
		ledgerCache:     ceConfig.LedgerCache,
		msgBus:          ceConfig.MsgBus,
		verifyTimeout:   DEFAULT_VERIFY_TIMEOUT,
		txPool:          ceConfig.TxPool,
		chainConf:       ceConfig.ChainConf,
		storeHelper:     ceConfig.StoreHelper,
		proposalCache:   ceConfig.ProposalCache,
		blockchainStore: ceConfig.BlockchainStore,
		ac:              ceConfig.AC,
		reentrantLocks: &common.ReentrantLocks{
			ReentrantLocks: make(map[string]interface{}),
		},
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
		ProposalCache:   ceConfig.ProposalCache,
		TxFilter:        ceConfig.TxFilter,
	}
	verifier.verifierBlock = common.NewVerifierBlock(conf)
	var err error
	verifier.goRoutinePool, err = ants.NewPool(
		len(ceConfig.ChainConf.ChainConfig().Consensus.Nodes), ants.WithPreAlloc(true))
	if err != nil {
		return nil, fmt.Errorf("new verifier failed: %s", err.Error())
	}
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		verifier.metricBlockVerifyTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_VERIFIER, "metric_block_verify_time",
			"block verify time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
	}
	return verifier, nil
}

func (bv *BlockVerifier) verifyBlock(block *commonPb.Block,
	mode protocol.VerifyMode) (bool, map[string]*commonPb.TxRWSet, error) {
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

// VerifyBlock params block, VerifyMode, return error
func (bv *BlockVerifier) VerifyBlock(block *commonPb.Block, mode protocol.VerifyMode) error {
	return bv.goRoutinePool.Submit(bv.verifyTask(block, mode))
}

// VerifyBlockSync params Block, VerifyMode, return VerifyResult, error
func (bv *BlockVerifier) VerifyBlockSync(block *commonPb.Block,
	mode protocol.VerifyMode) (*consensuspb.VerifyResult, error) {
	panic("implement me")
}

// VerifyBlockWithRwSets to check if block is valid
func (bv *BlockVerifier) VerifyBlockWithRwSets(block *commonPb.Block,
	rwsets []*commonPb.TxRWSet, mode protocol.VerifyMode) (err error) {

	if mode == protocol.CONSENSUS_VERIFY {
		return fmt.Errorf("consensus verify could not call this method")
	}

	startTick := utils.CurrentTimeMillisSeconds()
	if err = utils.IsEmptyBlock(block); err != nil {
		bv.log.Error(err)
		bv.log.Debugf("empty block. height:%+v, hash:%+v, chainId:%+v, preHash:%+v, signature:%+v",
			block.Header.BlockHeight, block.Header.BlockHash,
			block.Header.ChainId, block.Header.PreBlockHash, block.Header.Signature)
		return err
	}

	bv.log.Debugf("verify receive [%d](%x,%d,%d), from sync %d",
		block.Header.BlockHeight, block.Header.BlockHash, block.Header.TxCount, len(block.Txs), mode)
	// avoid concurrent verify, only one block hash can be verified at the same time
	if !bv.reentrantLocks.Lock(string(block.Header.BlockHash)) {
		bv.log.Warnf("block(%d,%x) concurrent verify, yield", block.Header.BlockHeight, block.Header.BlockHash)
		return commonErrors.ErrConcurrentVerify
	}
	defer bv.reentrantLocks.Unlock(string(block.Header.BlockHash))

	// No duplicate verify
	isRepeat := bv.verifyRepeat(block, startTick, mode)
	if isRepeat {
		return nil
	}

	var contractEventMap map[string][]*commonPb.ContractEvent
	txRWSetMap := make(map[string]*commonPb.TxRWSet)
	for _, txRWSet := range rwsets {
		if txRWSet != nil {
			txRWSetMap[txRWSet.TxId] = txRWSet
		}
	}

	// avoid to recover the committed block.
	lastBlock, err := bv.verifierBlock.FetchLastBlock(block)
	if err != nil {
		return err
	}

	startPoolTick := utils.CurrentTimeMillisSeconds()
	newBlock := &commonPb.Block{
		Header:         block.Header,
		Dag:            block.Dag,
		Txs:            block.Txs,
		AdditionalData: block.AdditionalData,
	}

	lastPool := utils.CurrentTimeMillisSeconds() - startPoolTick
	contractEventMap, timeLasts, _, err := bv.validateBlockWithRWSets(newBlock, lastBlock, mode, txRWSetMap)
	if err != nil {
		bv.log.Warnf("verify failed [%d](%x),preBlockHash:%x, %s",
			newBlock.Header.BlockHeight, newBlock.Header.BlockHash, newBlock.Header.PreBlockHash, err.Error())

		// rollback sql
		if sqlErr := bv.storeHelper.RollBack(newBlock, bv.blockchainStore); sqlErr != nil {
			bv.log.Errorf("block [%d] rollback sql failed: %s", newBlock.Header.BlockHeight, sqlErr)
		}
		return err
	}

	// sync mode, need to verify consensus vote signature
	beginConsensCheck := utils.CurrentTimeMillisSeconds()
	// ABFT not need to verify vote sig

	//if protocol.SYNC_VERIFY == mode {
	//	if err = bv.verifyVoteSig(newBlock); err != nil {
	//		bv.log.Warnf("verify failed [%d](%x), votesig %s",
	//			newBlock.Header.BlockHeight, newBlock.Header.BlockHash, err.Error())
	//		return err
	//	}
	//}
	consensusCheckUsed := utils.CurrentTimeMillisSeconds() - beginConsensCheck

	// verify success, cache block and read write set
	// solo need this，too！！！
	bv.log.Debugf("set proposed block(%d,%x)", newBlock.Header.BlockHeight, newBlock.Header.BlockHash)
	if err = bv.proposalCache.SetProposedBlock(newBlock, txRWSetMap, contractEventMap, false); err != nil {
		return err
	}

	err = bv.abftCache.AddVerifiedTxBatch(block, true, txRWSetMap)
	if err != nil {
		err = fmt.Errorf("sync cache the verified block faield: %s, blockHeight(%d), blockHash(%s)", err.Error(),
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
		return err
	}

	// mark transactions in block as pending status in txpool
	if common.TxPoolType == batch.TxPoolType {
		batchIds, _, err := common.GetBatchIds(block)
		if err != nil {
			return err
		}
		bv.txPool.AddTxBatchesToPendingCache(batchIds, newBlock.Header.BlockHeight)
	} else {
		bv.txPool.AddTxsToPendingCache(newBlock.Txs, newBlock.Header.BlockHeight)
	}

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	bv.log.Infof("verify success [%d,%x]"+
		"(blockSig:%d,vm:%d,txVerify:%d,txRoot:%d,pool:%d,consensusCheckUsed:%d,total:%d)",
		newBlock.Header.BlockHeight, newBlock.Header.BlockHash, timeLasts[common.BlockSig], timeLasts[common.VM],
		timeLasts[common.TxVerify], timeLasts[common.TxRoot], lastPool, consensusCheckUsed, elapsed)

	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		bv.metricBlockVerifyTime.WithLabelValues(bv.chainId).Observe(float64(elapsed) / 1000)
	}
	return nil
}

func (bv *BlockVerifier) validateBlockWithRWSets(block, lastBlock *commonPb.Block, mode protocol.VerifyMode,
	txRWSetMap map[string]*commonPb.TxRWSet) (
	map[string][]*commonPb.ContractEvent, map[string]int64, *common.RwSetVerifyFailTx, error) {
	hashType := bv.chainConf.ChainConfig().Crypto.Hash
	timeLasts := make(map[string]int64)
	var err error
	txCapacity := uint32(bv.chainConf.ChainConfig().Block.BlockTxCapacity)
	if block.Header.TxCount > txCapacity {
		return nil, timeLasts, nil, fmt.Errorf("txcapacity expect <= %d, got %d)", txCapacity, block.Header.TxCount)
	}

	if err = common.IsTxCountValid(block); err != nil {
		return nil, timeLasts, nil, err
	}

	err = common.CheckPreBlock(block, lastBlock)
	if err != nil {
		return nil, timeLasts, nil, err
	}

	return bv.verifierBlock.ValidateBlockWithRWSets(block, lastBlock, hashType, timeLasts, txRWSetMap, mode)
}

// Verify params Block, VerifyMode, return error
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
		bv.log.Errorf("verify failed:%s,[%d],(%s)", err.Error(),
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
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

// verifyRepeat to check if the block has verified before
func (bv *BlockVerifier) verifyRepeat(block *commonPb.Block, startTick int64,
	mode protocol.VerifyMode) (isRepeat bool) {
	b, _, _ := bv.proposalCache.GetProposedBlock(block)
	// Return not repeat if SQL is not enabled or if it is not solo
	if b == nil {
		return false
	}
	isSqlDb := bv.chainConf.ChainConfig().Contract.EnableSqlSupport
	if consensuspb.ConsensusType_SOLO != bv.chainConf.ChainConfig().Consensus.Type || isSqlDb {
		elapsed := utils.CurrentTimeMillisSeconds() - startTick
		// the block has verified before
		bv.log.Infof("verify success repeat [%d](%x), total: %d", block.Header.BlockHeight, block.Header.BlockHash, elapsed)
		//if protocol.CONSENSUS_VERIFY == mode {
		//	// consensus mode, publish verify result to message bus
		//	bv.msgBus.Publish(msgbus.VerifyResult, parseVerifyResult(block, true, txRwSet, nil))
		//}
		lastBlock, _ := bv.proposalCache.GetProposedBlockByHashAndHeight(
			block.Header.PreBlockHash, block.Header.BlockHeight-1)
		if lastBlock == nil {
			bv.log.Debugf(
				"no pre-block be found, preHeight:%d, preBlockHash:%x",
				block.Header.BlockHeight-1,
				block.Header.PreBlockHash,
			)
			return true
		}
		cutBlocks := bv.proposalCache.KeepProposedBlock(lastBlock.Header.BlockHash, lastBlock.Header.BlockHeight)
		if len(cutBlocks) > 0 {
			bv.log.Infof(
				"received block hash: %s, height: %v",
				hex.EncodeToString(lastBlock.Header.BlockHash),
				lastBlock.Header.BlockHeight,
			)
			bv.cutBlocks(cutBlocks, lastBlock)
		}

		return true
	}
	return false
}

//func (bv *BlockVerifier) verifyVoteSig(block *commonPb.Block) error {
//	return consensus.VerifyBlockSignatures(bv.chainConf, bv.ac, bv.blockchainStore, block, bv.ledgerCache)
//}

func (bv *BlockVerifier) cutBlocks(blocksToCut []*commonPb.Block, blockToKeep *commonPb.Block) {
	if common.TxPoolType == batch.TxPoolType {
		bv.cutBlocksForBatchPool(blocksToCut, blockToKeep)
		return
	}

	cutTxs := make([]*commonPb.Transaction, 0)
	txMap := make(map[string]interface{})
	for _, tx := range blockToKeep.Txs {
		txMap[tx.Payload.TxId] = struct{}{}
	}
	for _, blockToCut := range blocksToCut {
		bv.log.Infof("cut block hash: %x, height: %v", blockToCut.Header.BlockHash, blockToCut.Header.BlockHeight)
		for _, txToCut := range blockToCut.Txs {
			if _, ok := txMap[txToCut.Payload.TxId]; ok {
				// this transaction is kept, do NOT cut it.
				continue
			}
			bv.log.Debugf("cut tx hash: %s", txToCut.Payload.TxId)
			cutTxs = append(cutTxs, txToCut)
		}
	}
	if len(cutTxs) > 0 {
		bv.txPool.RetryTxs(cutTxs)
	}
}

func (bv *BlockVerifier) cutBlocksForBatchPool(blocksToCut []*commonPb.Block, blockToKeep *commonPb.Block) {

	keepBatchIdsMap := make(map[string]interface{})
	batchIds, _, _ := common.GetBatchIds(blockToKeep)
	for _, batchId := range batchIds {
		keepBatchIdsMap[batchId] = struct{}{}
	}

	finalCutBatchIds := make([]string, 0)
	for _, blockToCut := range blocksToCut {
		bv.log.Infof("cut block hash: %x, height: %v", blockToCut.Header.BlockHash, blockToCut.Header.BlockHeight)
		cutBatchIds, _, _ := common.GetBatchIds(blockToCut)
		for _, cutBatchId := range cutBatchIds {
			if _, ok := keepBatchIdsMap[cutBatchId]; ok {
				// this transaction is kept, do NOT cut it.
				continue
			}
			bv.log.Debugf("cut tx batchId: %s", cutBatchId)
			finalCutBatchIds = append(finalCutBatchIds, cutBatchId)
		}
	}

	if len(finalCutBatchIds) > 0 {
		bv.txPool.RetryTxBatches(finalCutBatchIds)
	}

}
