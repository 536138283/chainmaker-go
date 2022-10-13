/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	batch "chainmaker.org/chainmaker/txpool-batch/v2"

	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	"chainmaker.org/chainmaker-go/module/txfilter/filtercommon"

	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"
	txpoolpb "chainmaker.org/chainmaker/pb-go/v2/txpool"
	"github.com/prometheus/client_golang/prometheus"
)

// BlockProposerImpl implements BlockProposer interface.
// In charge of propose a new block.
type BlockProposerImpl struct {
	// chain id, to identity this chain
	chainId string
	// tx pool provides tx batch
	txPool protocol.TxPool
	// scheduler orders tx batch into DAG form and returns a block
	txScheduler protocol.TxScheduler
	// snapshot manager
	snapshotManager protocol.SnapshotManager
	// identity manager
	identity protocol.SigningMember
	// ledger cache
	ledgerCache protocol.LedgerCache
	// channel to give out proposed block
	msgBus msgbus.MessageBus
	// access control provider
	ac              protocol.AccessControlProvider
	blockchainStore protocol.BlockchainStore
	// Verify the transaction rules with TxFilter
	txFilter protocol.TxFilter
	// whether current node can propose block now
	isProposer bool
	// whether current node is proposing or not
	idle bool
	// timer controls the proposing periods
	proposeTimer *time.Timer
	// channel to handle propose status change from consensus module
	canProposeC chan bool
	// channel to handle propose signal from tx pool
	txPoolSignalC chan *txpoolpb.TxPoolSignal
	// channel to stop proposing loop
	exitC chan bool
	// prposal cache
	proposalCache protocol.ProposalCache
	// chain config
	chainConf protocol.ChainConf
	// for proposeBlock reentrant lock
	idleMu sync.Mutex
	// for propose status change lock
	statusMu sync.Mutex
	// for isProposer lock, avoid race
	proposerMu sync.RWMutex
	// logger
	log protocol.Logger
	// channel to receive signal to yield propose block
	finishProposeC chan bool
	// metric block package time
	metricBlockPackageTime *prometheus.HistogramVec
	// proposer by pbac member
	proposer *pbac.Member
	// block builder
	blockBuilder *common.BlockBuilder
	// store helper
	storeHelper conf.StoreHelper
}

// BlockProposerConfig block proposer config
type BlockProposerConfig struct {
	// chain id
	ChainId string
	// tx pool
	TxPool protocol.TxPool
	// snapshot manager
	SnapshotManager protocol.SnapshotManager
	// message bus
	MsgBus msgbus.MessageBus
	// signing member
	Identity protocol.SigningMember
	// ledger cache
	LedgerCache protocol.LedgerCache
	// tx scheduler
	TxScheduler protocol.TxScheduler
	// proposal cache
	ProposalCache protocol.ProposalCache
	// chain config
	ChainConf protocol.ChainConf
	// access control provider
	AC protocol.AccessControlProvider
	// block chain store
	BlockchainStore protocol.BlockchainStore
	// store helper
	StoreHelper conf.StoreHelper
	// tx filter
	TxFilter protocol.TxFilter
}

const (
	// DEFAULTDURATION default proposal duration, millis seconds
	DEFAULTDURATION = 1000
	// DEFAULTVERSION default version of chain
	DEFAULTVERSION = "v1.0.0"
)

// NewBlockProposer new block proposer, return block proposer error
func NewBlockProposer(config BlockProposerConfig, log protocol.Logger) (protocol.BlockProposer, error) {
	// construct block proposer implement
	blockProposerImpl := &BlockProposerImpl{
		chainId:         config.ChainId,
		isProposer:      false, // not proposer when initialized
		idle:            true,
		msgBus:          config.MsgBus,
		blockchainStore: config.BlockchainStore,
		canProposeC:     make(chan bool),
		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
		exitC:           make(chan bool),
		txPool:          config.TxPool,
		snapshotManager: config.SnapshotManager,
		txScheduler:     config.TxScheduler,
		identity:        config.Identity,
		ledgerCache:     config.LedgerCache,
		proposalCache:   config.ProposalCache,
		chainConf:       config.ChainConf,
		ac:              config.AC,
		log:             log,
		finishProposeC:  make(chan bool),
		storeHelper:     config.StoreHelper,
		txFilter:        config.TxFilter,
	}

	var err error
	// set the proposer by identity
	blockProposerImpl.proposer, err = blockProposerImpl.identity.GetMember()
	if err != nil {
		blockProposerImpl.log.Warnf("identity serialize failed, %s", err)
		return nil, err
	}

	// start propose timer
	blockProposerImpl.proposeTimer = time.NewTimer(blockProposerImpl.getDuration())
	if !blockProposerImpl.isSelfProposer() {
		blockProposerImpl.proposeTimer.Stop()
	}

	// monitor config open case
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		blockProposerImpl.metricBlockPackageTime = monitor.NewHistogramVec(
			monitor.SUBSYSTEM_CORE_PROPOSER,
			"metric_block_package_time",
			"block package time metric",
			[]float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 2, 5, 10},
			"chainId",
		)
	}

	// set the store helper by config store helper
	blockProposerImpl.storeHelper = config.StoreHelper

	// construct block builder config
	bbConf := &common.BlockBuilderConf{
		ChainId:         blockProposerImpl.chainId,
		TxPool:          blockProposerImpl.txPool,
		TxScheduler:     blockProposerImpl.txScheduler,
		SnapshotManager: blockProposerImpl.snapshotManager,
		Identity:        blockProposerImpl.identity,
		LedgerCache:     blockProposerImpl.ledgerCache,
		ProposalCache:   blockProposerImpl.proposalCache,
		ChainConf:       blockProposerImpl.chainConf,
		Log:             blockProposerImpl.log,
		StoreHelper:     config.StoreHelper,
	}

	// set the block builder by NewBlockBuilder func
	blockProposerImpl.blockBuilder = common.NewBlockBuilder(bbConf)

	return blockProposerImpl, nil
}

// Start proposer
func (bp *BlockProposerImpl) Start() error {
	defer bp.log.Info("block proposer starts")
	// use on gorourine  startProposingLoop
	go bp.startProposingLoop()

	return nil
}

// Stop proposing loop
func (bp *BlockProposerImpl) Stop() error {
	defer bp.log.Infof("block proposer stopped")
	bp.exitC <- true
	return nil
}

// Start, start proposing loop
func (bp *BlockProposerImpl) startProposingLoop() {
	for {
		select {
		case <-bp.proposeTimer.C:
			if !bp.isSelfProposer() {
				break
			}
			go bp.proposeBlock()

		case signal := <-bp.txPoolSignalC:
			if !bp.isSelfProposer() {
				break
			}
			if signal.SignalType != txpoolpb.SignalType_BLOCK_PROPOSE {
				break
			}
			go bp.proposeBlock()

		case <-bp.exitC:
			bp.proposeTimer.Stop()
			bp.log.Info("block proposer loop stopped")
			return
		}
	}
}

/*
 * shouldProposeByBFT, check if node should propose new block
 * Only for *BFT consensus
 * if node is proposer, and node is not propose right now, and last proposed block is committed, then return true
 */
func (bp *BlockProposerImpl) shouldProposeByBFT(height uint64) bool {
	if !bp.isIdle() {
		// concurrent control, proposer is proposing now
		bp.log.Debugf("proposer is busy, not propose [%d] ", height)
		return false
	}
	committedBlock := bp.ledgerCache.GetLastCommittedBlock()
	if committedBlock == nil {
		bp.log.Errorf("no committed block found")
		return false
	}
	currentHeight := committedBlock.Header.BlockHeight
	// proposing height must higher than current height
	return currentHeight+1 == height
}

// proposeBlock, to check if proposer can propose block right now
// if so, start proposing
func (bp *BlockProposerImpl) proposeBlock() {
	defer func() {
		if bp.isSelfProposer() {
			bp.proposeTimer.Reset(bp.getDuration())
		}
	}()
	lastBlock := bp.ledgerCache.GetLastCommittedBlock()
	if lastBlock == nil {
		bp.log.Errorf("no committed block found")
		return
	}

	proposingHeight := lastBlock.Header.BlockHeight + 1
	//if !bp.shouldProposeByBFT(proposingHeight) {
	//	return
	//}
	if !bp.isIdle() {
		// concurrent control, proposer is proposing now
		bp.log.Debugf("proposer is busy, not propose [%d] ", proposingHeight)
		return
	}
	if !bp.setNotIdle() {
		bp.log.Infof("concurrent propose block [%d], yield!", proposingHeight)
		return
	}
	defer bp.setIdle()

	go bp.proposing(proposingHeight, lastBlock.Header.BlockHash)
	// #DEBUG MODE#
	if localconf.ChainMakerConfig.DebugConfig.IsHaltPropose {
		go func() {
			bp.OnReceiveYieldProposeSignal(true)
		}()
	}

	<-bp.finishProposeC
}

// proposing, propose a block in new height
func (bp *BlockProposerImpl) proposing(height uint64, preHash []byte) *commonpb.Block {
	startTick := utils.CurrentTimeMillisSeconds()
	defer bp.yieldProposing()

	bp.log.Debugf("syncmode::BlockProposerImpl::proposing() => tx_pool status = %#v, height:%d", bp.txPool, height)

	selfProposedBlock := bp.proposalCache.GetSelfProposedBlockAt(height)
	if selfProposedBlock != nil {
		if needPropose := bp.dealProposalRequestWithProposalCache(height, selfProposedBlock, preHash); !needPropose {
			return nil
		}
	}

	var (
		fetchLasts          int64
		filterValidateLasts int64
		fetchTotalLasts     int64 // The total time consuming
		totalTimes          int   // loop count
		fetchBatch          []*commonpb.Transaction
		batchIds            []string
		fetchBatches        [][]*commonpb.Transaction // record the order about transaction in tx pool
	)
	// 根据TxFilter时间规则过滤交易，如果剩余的交易为0，则再次从交易池拉取交易，重复执行
	// The transaction is filtered according to txFilter time rule. If the remaining transaction is 0, the transaction
	// is pulled from the trading pool again and executed repeatedly
	fetchTotalFirst := utils.CurrentTimeMillisSeconds()
	for {
		totalTimes++
		// retrieve tx batch from tx pool
		fetchFirst := utils.CurrentTimeMillisSeconds()
		batchIds, fetchBatch, fetchBatches = bp.getFetchBatchFromPool(height)
		fetchLasts += utils.CurrentTimeMillisSeconds() - fetchFirst
		bp.log.DebugDynamic(filtercommon.LoggingFixLengthFunc("begin proposing block[%d], fetch tx num[%d]",
			height, len(fetchBatch)))
		if len(fetchBatch) == 0 {
			bp.log.DebugDynamic(filtercommon.LoggingFixLengthFunc("no txs in tx pool, proposing block stoped"))
			return nil
		}
		// validate txFilter rules
		filterValidateFirst := utils.CurrentTimeMillisSeconds()
		removeTxs, remainTxs := common.ValidateTxRules(bp.txFilter, fetchBatch)
		filterValidateLasts += utils.CurrentTimeMillisSeconds() - filterValidateFirst
		if len(removeTxs) > 0 {
			batchIds, fetchBatches, fetchBatch =
				bp.removeTx(height, batchIds, removeTxs, fetchBatch, fetchBatches)

			bp.log.Warnf("remove the overtime transactions, total:%d, remain:%d, remove:%d",
				len(fetchBatch), len(remainTxs), len(removeTxs))
		}
		if len(remainTxs) > 0 {
			// 剩余交易大于0则跳出循环
			fetchBatch = remainTxs
			break
		}
	}
	fetchTotalLasts = utils.CurrentTimeMillisSeconds() - fetchTotalFirst

	if !utils.CanProposeEmptyBlock(bp.chainConf.ChainConfig().Consensus.Type) && len(fetchBatch) == 0 {
		// can not propose empty block and tx batch is empty, then yield proposing.
		bp.log.Debugf("no txs in tx pool, proposing block stopped")
		return nil
	}

	txCapacity := int(bp.chainConf.ChainConfig().Block.BlockTxCapacity)
	if len(fetchBatch) > txCapacity {
		// check if checkedBatch > txCapacity, if so, strict block tx count according to  config,
		// and put other txs back to txpool.
		txRetry := fetchBatch[txCapacity:]
		fetchBatch = fetchBatch[:txCapacity]

		if common.TxPoolType != batch.TxPoolType {
			bp.txPool.RetryAndRemoveTxs(txRetry, nil)
		} else {
			batchIds, fetchBatches = bp.txPool.ReGenTxBatchesWithRetryTxs(height, batchIds, txRetry)
			fetchBatch = getFetchBatch(fetchBatches)
		}

		bp.log.Warnf("txbatch oversize expect <= %d, got %d", txCapacity, len(fetchBatch))
	}

	block, timeLasts, err := bp.generateNewBlock(
		height,
		preHash,
		fetchBatch,
		batchIds,
		fetchBatches)

	if err != nil {
		// rollback sql
		if sqlErr := bp.storeHelper.RollBack(block, bp.blockchainStore); sqlErr != nil {
			bp.log.Errorf("block [%d] rollback sql failed: %s", height, sqlErr)
		}

		if common.TxPoolType != batch.TxPoolType {
			bp.txPool.RetryAndRemoveTxs(fetchBatch, nil) // put txs back to txpool
		} else {
			bp.txPool.RetryAndRemoveTxBatches(batchIds, nil)
		}

		bp.log.Warnf("generate new block failed, %s", err.Error())
		return nil
	}
	_, rwSetMap, _ := bp.proposalCache.GetProposedBlock(block)
	bp.log.Debugf("proposing block \n %s", utils.FormatBlock(block))

	cutBlock := bp.getCutBlock(block)
	bp.msgBus.Publish(msgbus.ProposedBlock,
		&consensuspb.ProposalBlock{Block: block, TxsRwSet: rwSetMap, CutBlock: cutBlock})

	//bp.log.Debugf("finalized block \n%s", utils.FormatBlock(block))
	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	bp.log.Infof("proposer success [%d](txs:%d),fetch(times:%v,fetch:%v,filter:%v,total:%d), time used("+
		"begin DB transaction:%v, new snapshot:%v, vm:%v, finalize block:%v,total:%d)",
		block.Header.BlockHeight, block.Header.TxCount,
		totalTimes, fetchLasts, filterValidateLasts, fetchTotalLasts,
		timeLasts[0], timeLasts[1], timeLasts[2], timeLasts[3], elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		bp.metricBlockPackageTime.WithLabelValues(bp.chainId).Observe(float64(elapsed) / 1000)
	}
	return block
}

// OnReceiveTxPoolSignal receive txpool signal and deliver to chan txpool signal
func (bp *BlockProposerImpl) OnReceiveTxPoolSignal(txPoolSignal *txpoolpb.TxPoolSignal) {
	bp.txPoolSignalC <- txPoolSignal
}

// OnReceiveProposeStatusChange to update isProposer status when received proposeStatus from consensus
// if node is proposer, then reset the timer, otherwise stop the timer
func (bp *BlockProposerImpl) OnReceiveProposeStatusChange(proposeStatus bool) {
	bp.log.Debugf("OnReceiveProposeStatusChange(%t)", proposeStatus)
	bp.statusMu.Lock()
	defer bp.statusMu.Unlock()
	if proposeStatus == bp.isSelfProposer() {
		// 状态一致，忽略
		return
	}
	height, _ := bp.ledgerCache.CurrentHeight()
	bp.proposalCache.ResetProposedAt(height + 1) // proposer status changed, reset this round proposed status
	bp.setIsSelfProposer(proposeStatus)
	if !bp.isSelfProposer() {
		bp.yieldProposing() // try to yield if proposer self is proposing right now.
		bp.log.Debug("current node is not proposer ")
		return
	}
	bp.proposeTimer.Reset(bp.getDuration())
	bp.log.Debugf("current node is proposer, timeout period is %v", bp.getDuration())
}

// OnReceiveMaxBFTProposal to check if this proposer should propose a new block
// Only for maxbft consensus
func (bp *BlockProposerImpl) OnReceiveMaxBFTProposal(proposal *maxbft.BuildProposal) {

}

// OnReceiveYieldProposeSignal receive yield propose signal
func (bp *BlockProposerImpl) OnReceiveYieldProposeSignal(isYield bool) {
	if !isYield {
		return
	}
	if bp.yieldProposing() {
		// halt scheduler execution
		bp.txScheduler.Halt()
		height, _ := bp.ledgerCache.CurrentHeight()
		bp.proposalCache.ResetProposedAt(height + 1)
	}
}

// yieldProposing, to yield proposing handle
func (bp *BlockProposerImpl) yieldProposing() bool {
	// signal finish propose only if proposer is not idle
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	if !bp.idle {
		bp.finishProposeC <- true
		bp.idle = true
		return true
	}
	return false
}

// getDuration, get propose duration from config.
// If not access from config, use default value.
func (bp *BlockProposerImpl) getDuration() time.Duration {
	if bp.chainConf == nil || bp.chainConf.ChainConfig() == nil {
		return DEFAULTDURATION * time.Millisecond
	}
	chainConfig := bp.chainConf.ChainConfig()
	duration := chainConfig.Block.BlockInterval
	if duration <= 0 {
		return DEFAULTDURATION * time.Millisecond
	}
	return time.Duration(duration) * time.Millisecond
}

// getChainVersion, get chain version from config.
// If not access from config, use default value.
// @Deprecated
//nolint: unused
func (bp *BlockProposerImpl) getChainVersion() []byte {
	if bp.chainConf == nil || bp.chainConf.ChainConfig() == nil {
		return []byte(DEFAULTVERSION)
	}
	return []byte(bp.chainConf.ChainConfig().Version)
}

// setNotIdle, set not idle status
func (bp *BlockProposerImpl) setNotIdle() bool {
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	if bp.idle {
		bp.idle = false
		return true
	}
	return false
}

// isIdle, to check if proposer is idle
func (bp *BlockProposerImpl) isIdle() bool {
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	return bp.idle
}

// setIdle, set idle status
func (bp *BlockProposerImpl) setIdle() {
	bp.idleMu.Lock()
	defer bp.idleMu.Unlock()
	bp.idle = true
}

// setIsSelfProposer, set isProposer status of this node
func (bp *BlockProposerImpl) setIsSelfProposer(isSelfProposer bool) {
	bp.proposerMu.Lock()
	defer bp.proposerMu.Unlock()
	bp.isProposer = isSelfProposer
	if !bp.isProposer {
		bp.proposeTimer.Stop()
	} else {
		bp.proposeTimer.Reset(bp.getDuration())
	}
}

// isSelfProposer, return if this node is consensus proposer
func (bp *BlockProposerImpl) isSelfProposer() bool {
	bp.proposerMu.RLock()
	defer bp.proposerMu.RUnlock()
	return bp.isProposer
}

// ProposeBlock propose block
func (bp *BlockProposerImpl) ProposeBlock(proposal *maxbft.BuildProposal) (*consensuspb.ProposalBlock, error) {

	return nil, nil
}

// OnReceiveRwSetVerifyFailTxs remove verify fail txs, deal with rw set verify fail txs
func (bp *BlockProposerImpl) OnReceiveRwSetVerifyFailTxs(rwSetVerifyFailTxs *consensuspb.RwSetVerifyFailTxs) {

	if common.TxPoolType == batch.TxPoolType {
		bp.log.Warnf("batch tx pool not support recover the problem about rwSet in conformity")
		return
	}

	// get block by height from proposal cache
	height := rwSetVerifyFailTxs.BlockHeight
	block := bp.proposalCache.GetSelfProposedBlockAt(height)

	bp.log.DebugDynamic(func() string {
		return fmt.Sprintf("remove rw set verify failed txs, block height:%d", height)
	})

	// if block is nil, remove tx from tx pool
	if block == nil {
		txsRet, _ := bp.txPool.GetTxsByTxIds(rwSetVerifyFailTxs.TxIds)
		txs := make([]*commonpb.Transaction, 0)
		for _, v := range txsRet {
			txs = append(txs, v)
		}
		bp.txPool.RetryAndRemoveTxs(nil, txs)
		return
	}

	// collect retry txs and remove txs
	retryTxs := make([]*commonpb.Transaction, 0, len(block.Txs))
	removeTxs := make([]*commonpb.Transaction, 0, len(block.Txs))
	txsMap := make(map[string]*commonpb.Transaction, len(block.Txs))
	for _, tx := range block.Txs {
		for _, txId := range rwSetVerifyFailTxs.TxIds {
			if tx.Payload.TxId == txId {
				txsMap[txId] = tx
				removeTxs = append(removeTxs, tx)
				break
			}
		}
	}

	for _, tx := range block.Txs {
		if _, ok := txsMap[tx.Payload.TxId]; !ok {
			retryTxs = append(retryTxs, tx)
		}
	}

	// retry txs and remove txs in tx pool
	bp.txPool.RetryAndRemoveTxs(retryTxs, removeTxs)
	// clear proposal cache at the height
	bp.proposalCache.ClearProposedBlockAt(height)

}

/*
 * getLastProposeTimeByBlockFinger, get proposer block time by block finger, it delayed by some second
 */
func (bp *BlockProposerImpl) getLastProposeTimeByBlockFinger(blockFinger string) (int64, error) {
	// load the proposer repeat timer map, if not exist, new and store the time
	timeValue, ok := common.ProposeRepeatTimerMap.Load(blockFinger)
	if !ok {
		timeNow := utils.CurrentTimeMillisSeconds()
		common.ProposeRepeatTimerMap.Store(blockFinger, timeNow)
		return timeNow, nil
	}

	switch timeNow := timeValue.(type) {
	case int64:
		return timeNow, nil
	default:
		// default case, new and store the time
		timeNow = utils.CurrentTimeMillisSeconds()
		common.ProposeRepeatTimerMap.Store(blockFinger, timeNow)
		errMsg := "propose repeat time map type is wrong"
		return 0, errors.New(errMsg)
	}
}

// getCutBlock get cut block, return cut block
func (bp *BlockProposerImpl) getCutBlock(block *commonpb.Block) *commonpb.Block {
	cutBlock := new(commonpb.Block)
	if common.IfOpenConsensusMessageTurbo(bp.chainConf) ||
		common.TxPoolType == batch.TxPoolType {
		cutBlock = common.GetTurboBlock(block, cutBlock, bp.chainConf, bp.log)
	} else {
		cutBlock = block
	}

	return cutBlock
}

// getFetchBatchFromPool fetch txs from tx pool at the height, return batch ids, fetch batch txs, fetch batches
func (bp *BlockProposerImpl) getFetchBatchFromPool(
	height uint64) ([]string, []*commonpb.Transaction, [][]*commonpb.Transaction) {
	if common.TxPoolType == batch.TxPoolType {
		batchIds, fetchBatches := bp.txPool.FetchTxBatches(height)

		fetchBatch := getFetchBatch(fetchBatches)

		return batchIds, fetchBatch, fetchBatches
	}

	return nil, bp.txPool.FetchTxs(height), nil
}

// getFetchBatch return fetchBatch
func getFetchBatch(fetchBatches [][]*commonpb.Transaction) []*commonpb.Transaction {

	fetchBatch := make([]*commonpb.Transaction, 0)
	for _, v := range fetchBatches {
		fetchBatch = append(fetchBatch, v...)
	}

	return fetchBatch
}

// removeTx remove tx
func (bp *BlockProposerImpl) removeTx(
	height uint64, batchIds []string, removeTxs, fetchBatch []*commonpb.Transaction,
	fetchBatches [][]*commonpb.Transaction) ([]string, [][]*commonpb.Transaction, []*commonpb.Transaction) {
	// don't remove tx when is batchTx pool
	if common.TxPoolType == batch.TxPoolType {
		// remove and get new batchIds
		batchIds, fetchBatches = bp.txPool.ReGenTxBatchesWithRemoveTxs(height, batchIds, removeTxs)
		fetchBatch = getFetchBatch(fetchBatches)

		return batchIds, fetchBatches, fetchBatch
	}

	// remove txs in tx pool
	bp.txPool.RetryAndRemoveTxs(nil, removeTxs)
	return batchIds, fetchBatches, fetchBatch
}

func (bp *BlockProposerImpl) dealProposalRequestWithProposalCache(
	height uint64, selfProposedBlock *commonpb.Block, preHash []byte) (needPropose bool) {

	if bytes.Equal(selfProposedBlock.Header.PreBlockHash, preHash) {

		// when this block has some wrong tx and could not to reach an agreement.
		// we need to clear the old proposal cache when the old block's tx timeout.
		// we need to remove these txs from tx pool.
		if utils.CurrentTimeSeconds()-selfProposedBlock.Header.BlockTimestamp >=
			int64(bp.chainConf.ChainConfig().Block.TxTimeout) {

			bp.proposalCache.ClearTheBlock(selfProposedBlock)
			if common.TxPoolType == batch.TxPoolType {
				batchIds, _, err := common.GetBatchIds(selfProposedBlock)
				if err != nil {
					// no need to handle this err,propose a new block.
					return true
				}
				bp.txPool.RetryAndRemoveTxBatches(nil, batchIds)
				return true
			}

			bp.txPool.RetryAndRemoveTxs(nil, selfProposedBlock.Txs)
			return true
		}

		blockFinger := utils.CalcBlockFingerPrint(selfProposedBlock)
		timeNow, err := bp.getLastProposeTimeByBlockFinger(string(blockFinger))

		if err != nil {
			bp.log.Errorf("proposer fail, get last propose time by hash err %s", err.Error())
			return false
		}

		if timeNow == 0 {
			return false
		}

		if utils.CurrentTimeMillisSeconds()-timeNow >= 1000 {
			// Repeat propose block if node has proposed before at the same height
			bp.proposalCache.SetProposedAt(height)
			_, txsRwSet, _ := bp.proposalCache.GetProposedBlock(selfProposedBlock)

			cutBlock := new(commonpb.Block)
			if common.IfOpenConsensusMessageTurbo(bp.chainConf) ||
				common.TxPoolType == batch.TxPoolType {
				cutBlock = common.GetTurboBlock(selfProposedBlock, cutBlock, bp.chainConf, bp.log)
			} else {
				cutBlock = selfProposedBlock
			}

			bp.msgBus.Publish(msgbus.ProposedBlock, &consensuspb.ProposalBlock{Block: selfProposedBlock,
				TxsRwSet: txsRwSet, CutBlock: cutBlock})
			bp.log.Infof("proposer success repeat [%d](txs:%d,hash:%x)",
				selfProposedBlock.Header.BlockHeight, selfProposedBlock.Header.TxCount,
				selfProposedBlock.Header.BlockHash)
		}
		return false

	}
	bp.proposalCache.ClearTheBlock(selfProposedBlock)
	// Note: It is not possible to re-add the transactions in the deleted block to txpool; because some
	// transactions may be included in other blocks to be confirmed, and it is impossible to quickly exclude
	// these pending transactions that have been entered into the block. Comprehensive considerations,
	// directly discard this block is the optimal choice. This processing method may only cause partial
	// transaction loss at the current node, but it can be solved by rebroadcasting on the client side.

	if common.TxPoolType == batch.TxPoolType {
		batchIds, _, err := common.GetBatchIds(selfProposedBlock)
		if err != nil {
			return true
		}
		bp.txPool.RetryAndRemoveTxBatches(nil, batchIds)
		return true
	}

	bp.txPool.RetryAndRemoveTxs(nil, selfProposedBlock.Txs)

	return true
}
