/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker-go/module/txfilter/filtercommon"
	"chainmaker.org/chainmaker/common/v2/monitor"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/net-common/utils"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"
	txpoolpb "chainmaker.org/chainmaker/pb-go/v2/txpool"
	"chainmaker.org/chainmaker/protocol/v2"
	batch "chainmaker.org/chainmaker/txpool-batch/v2"
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
	// access controller provider
	ac protocol.AccessControlProvider
	// block chain store
	blockchainStore protocol.BlockchainStore
	// Verify the transaction rules with TxFilter
	txFilter protocol.TxFilter
	// whether current node can propose block now
	isProposer bool
	//idle whether current node is proposing or not
	idle bool
	//proposeTimer timer controls the proposing periods
	proposeTimer *time.Timer
	//canProposeC channel to handle propose status change from consensus module
	//canProposeC chan bool
	//txPoolSignalC channel to handle propose signal from tx pool
	txPoolSignalC chan *txpoolpb.TxPoolSignal
	//exitC channel to stop proposing loop
	exitC chan bool
	//proposalCache proposal cache
	proposalCache protocol.ProposalCache
	//chainConf chain config
	chainConf protocol.ChainConf
	//idleMu for proposeBlock reentrant lock
	idleMu sync.Mutex
	//statusMu for propose status change lock
	statusMu sync.Mutex
	//proposerMu for isProposer lock, avoid race
	proposerMu sync.RWMutex
	log        protocol.Logger
	//finishProposeC channel to receive signal to yield propose block
	finishProposeC chan bool
	// metric block package time
	metricBlockPackageTime *prometheus.HistogramVec
	// proposer from pbac member
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
	// sign member
	Identity protocol.SigningMember
	// leger cache
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
	// store heloer
	StoreHelper conf.StoreHelper
	// tx filter
	TxFilter protocol.TxFilter
}

const (
	//DEFAULTDURATION default proposal duration, millis seconds
	DEFAULTDURATION = 1000
	//RETRY 0
	RETRY = 0
	//REMOVE 1
	REMOVE = 1
)

// NewBlockProposer return block proposer error
func NewBlockProposer(config BlockProposerConfig, log protocol.Logger) (protocol.BlockProposer, error) {
	blockProposerImpl := &BlockProposerImpl{
		chainId:         config.ChainId,
		isProposer:      false, // not proposer when initialized
		idle:            true,
		msgBus:          config.MsgBus,
		blockchainStore: config.BlockchainStore,
		//canProposeC:     make(chan bool),
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

	// get proposer from identity
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

	// monitor open case
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		blockProposerImpl.metricBlockPackageTime = monitor.NewHistogramVec(
			monitor.SUBSYSTEM_CORE_PROPOSER,
			"metric_block_package_time",
			"block package time metric",
			[]float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 2, 5, 10},
			"chainId",
		)
	}

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
		StoreHelper:     blockProposerImpl.storeHelper,
	}

	// new block builder at blockProposerImpl.blockBuilder
	blockProposerImpl.blockBuilder = common.NewBlockBuilder(bbConf)

	return blockProposerImpl, nil
}

// Start start proposer
func (bp *BlockProposerImpl) Start() error {
	defer bp.log.Info("block proposer starts")
	// use one goroutine to startProposingLoop
	go bp.startProposingLoop()

	return nil
}

// startProposingLoop, start proposing loop
func (bp *BlockProposerImpl) startProposingLoop() {
	for {
		select {
		// deal proposer timer case
		case <-bp.proposeTimer.C:
			if !bp.isSelfProposer() {
				break
			}

			poolStatus := bp.txPool.GetPoolStatus()
			if poolStatus != nil {
				if poolStatus.ConfigTxNumInQueue != 0 || poolStatus.CommonTxNumInQueue != 0 {
					bp.log.DebugDynamic(func() string {
						return "publish msgbus proposeTimer propose blocks propose true"
					})
					go bp.msgBus.Publish(msgbus.ProposeBlock, &maxbft.ProposeBlock{IsPropose: true})
				}
				bp.proposeTimer.Reset(bp.getDuration())
			}
		case signal := <-bp.txPoolSignalC:
			if !bp.isSelfProposer() {
				break
			}

			if signal.SignalType != txpoolpb.SignalType_BLOCK_PROPOSE {
				break
			}
			go bp.msgBus.Publish(msgbus.ProposeBlock, &maxbft.ProposeBlock{IsPropose: true})
		// deal with exit channal
		case <-bp.exitC:
			// propose timer stop
			bp.proposeTimer.Stop()
			bp.log.Info("block proposer loop stopped")
			return
		}
	}
}

// Stop stop proposing loop
func (bp *BlockProposerImpl) Stop() error {
	defer bp.log.Infof("block proposer stopped")
	bp.exitC <- true
	return nil
}

// proposing, propose a block in new height
func (bp *BlockProposerImpl) proposing(height uint64, preHash []byte) (*consensuspb.ProposalBlock, error) {
	startTick := utils.CurrentTimeMillisSeconds()

	defer bp.yieldProposing()

	bp.log.DebugDynamic(func() string {
		return fmt.Sprintf(
			"maxbftmode::BlockProposerImpl::proposing() => tx_pool status = %#v, height: %d", bp.txPool, height)
	})

	selfProposedBlock := bp.proposalCache.GetSelfProposedBlockAt(height)
	if selfProposedBlock != nil {
		/**
		1. preHash not equal.
		// Note: It is not possible to re-add the transactions in the deleted block to txpool; because some transactions may
		// be included in other blocks to be confirmed, and it is impossible to quickly exclude these pending transactions
		// that have been entered into the block. Comprehensive considerations, directly discard this block is the optimal
		// choice. This processing method may only cause partial transaction loss at the current node, but it can be solved
		// by rebroadcasting on the client side.
		*/
		if !bytes.Equal(selfProposedBlock.Header.PreBlockHash, preHash) {
			bp.log.Warnf(fmt.Sprintf(
				"remove self proposed block, height: %d, selfPreHash:%x, hash: %x, preHash:%x",
				selfProposedBlock.Header.BlockHeight,
				selfProposedBlock.Header.PreBlockHash,
				selfProposedBlock.Header.BlockHash, preHash))

			bp.proposalCache.ClearTheBlock(selfProposedBlock)

			if common.TxPoolType == batch.TxPoolType {
				batchIds, _, err := common.GetBatchIds(selfProposedBlock)
				if err != nil {
					return nil, err
				}

				bp.txPool.RetryAndRemoveTxBatches(nil, batchIds)
			} else {
				bp.txPool.RetryAndRemoveTxs(nil, selfProposedBlock.Txs)
			}
		}
	}
	var (
		fetchLasts               int64
		fetchFromOtherBlockLasts int64
		filterValidateLasts      int64
		// The total time consuming
		fetchTotalLasts int64
		// loop count
		totalTimes   int
		fetchBatch   []*commonpb.Transaction
		batchIds     []string
		fetchBatches [][]*commonpb.Transaction // record the order about transaction in tx poolerr                      error
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
			bp.log.DebugDynamic(filtercommon.LoggingFixLengthFunc("no txs in tx pool, proposing block stopped"))
			// fetch Tx from other block
			fetchFromOtherBlockStart := utils.CurrentTimeMillisSeconds()
			fetchBatch, batchIds = bp.FetchTxFromOtherBlock(height, preHash)
			// re_gen new txBatches by retry txs
			if len(fetchBatch) != 0 && len(batchIds) != 0 {
				batchIds, fetchBatches = bp.txPool.ReGenTxBatchesWithRetryTxs(height, batchIds, fetchBatch)
				fetchBatch = getFetchBatch(fetchBatches)
			}
			fetchFromOtherBlockLasts += utils.CurrentTimeMillisSeconds() - fetchFromOtherBlockStart
			break
		}
		// validate txFilter rules
		filterValidateFirst := utils.CurrentTimeMillisSeconds()
		removeTxs, remainTxs := common.ValidateTxRules(bp.txFilter, fetchBatch)
		filterValidateLasts += utils.CurrentTimeMillisSeconds() - filterValidateFirst
		if len(removeTxs) > 0 {
			batchIds, fetchBatch, fetchBatches = bp.removeAndRetryTx(height, batchIds, removeTxs, remainTxs, REMOVE)
			bp.log.Warnf("remove the overtime transactions, remain:%d, remove:%d",
				len(remainTxs), len(removeTxs))
		}

		// 剩余交易大于0则跳出循环
		if len(fetchBatch) > 0 {
			break
		}
	}
	fetchTotalLasts = utils.CurrentTimeMillisSeconds() - fetchTotalFirst

	batchIds, fetchBatch, fetchBatches = bp.fetchBatchWithoutDupTxInSameBranch(height, preHash, batchIds, fetchBatch,
		fetchBatches)

	txCapacity := int(bp.chainConf.ChainConfig().Block.BlockTxCapacity)
	if len(fetchBatch) > txCapacity {
		// check if checkedBatch > txCapacity, if so, strict block tx count according to  config,
		// and put other txs back to txpool.
		txRetry := fetchBatch[txCapacity:]
		fetchBatch = fetchBatch[:txCapacity]

		if common.TxPoolType != batch.TxPoolType {
			bp.txPool.RetryAndRemoveTxs(txRetry, nil)
		} else {
			batchIds, fetchBatches = bp.txPool.ReGenTxBatchesWithRetryTxs(height, batchIds, fetchBatch)
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
		bp.log.Warnf("generate new block failed, %s", err.Error())
		// rollback sql
		if sqlErr := bp.storeHelper.RollBack(block, bp.blockchainStore); sqlErr != nil {
			bp.log.Errorf("block [%d] rollback sql failed: %s", height, sqlErr)
		}

		if common.TxPoolType != batch.TxPoolType {
			bp.txPool.RetryAndRemoveTxs(fetchBatch, nil) // put txs back to txpool
		} else {
			bp.txPool.RetryAndRemoveTxBatches(batchIds, nil)
		}

		return nil, err
	}
	_, txsRwSet, _ := bp.proposalCache.GetProposedBlock(block)

	cutBlock := new(commonpb.Block)
	if common.IfOpenConsensusMessageTurbo(bp.chainConf) ||
		common.TxPoolType == batch.TxPoolType {
		cutBlock = common.GetTurboBlock(block, cutBlock, bp.log)
	} else {
		cutBlock = block
	}

	bp.msgBus.Publish(msgbus.ProposedBlock,
		&consensuspb.ProposalBlock{Block: block, TxsRwSet: txsRwSet, CutBlock: cutBlock})

	elapsed := utils.CurrentTimeMillisSeconds() - startTick
	bp.log.Infof("proposer success [%d](txs:%d), fetch(times:%v,fetch:%v,filter:%v,fetch from other block:%v,total:%d) "+
		"time used(begin DB transaction:%v, "+
		"new snapshot:%v, vm:%v, finalize block:%v,total:%d)", block.Header.BlockHeight, block.Header.TxCount,
		totalTimes, fetchLasts, filterValidateLasts, fetchFromOtherBlockLasts, fetchTotalLasts,
		timeLasts[0], timeLasts[1], timeLasts[2], timeLasts[3], elapsed)
	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
		bp.metricBlockPackageTime.WithLabelValues(bp.chainId).Observe(float64(elapsed) / 1000)
	}
	return &consensuspb.ProposalBlock{Block: block, TxsRwSet: txsRwSet, CutBlock: cutBlock}, nil
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
	bp.setIsSelfProposer(proposeStatus)
	if !bp.isSelfProposer() {
		//bp.yieldProposing() // try to yield if proposer self is proposing right now.
		bp.log.Debug("current node is not proposer ")
		return
	}
	bp.proposeTimer.Reset(bp.getDuration())
	bp.log.Debugf("current node is proposer, timeout period is %v", bp.getDuration())
}

// OnReceiveMaxBFTProposal to check if this proposer should propose a new block
// Only for max bft consensus
func (bp *BlockProposerImpl) OnReceiveMaxBFTProposal(proposal *maxbft.BuildProposal) {
	proposingHeight := proposal.Height
	preHash := proposal.PreHash
	if !bp.shouldProposeByMaxBFT(proposingHeight, preHash) {
		bp.log.Infof("not a legal proposal request [%d](%x)", proposingHeight, preHash)
		return
	}

	if !bp.setNotIdle() {
		bp.log.Warnf("concurrent propose block [%d](%x), yield!", proposingHeight, preHash)
		return
	}
	defer bp.setIdle()

	bp.log.Infof("trigger proposal from maxBFT, height[%d]", proposal.Height)
	go func() {
		if _, err := bp.proposing(proposingHeight, preHash); err != nil {
			bp.log.Warnf("proposing err:%s", err.Error())
		}
	}()
	<-bp.finishProposeC
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

// FetchTxFromOtherBlock fetch tx
// @param height
// @param preHash
// @return []*commonpb.Transaction
// @return []string
func (bp *BlockProposerImpl) FetchTxFromOtherBlock(height uint64, preHash []byte) (
	[]*commonpb.Transaction, []string) {

	newFetchBatch := make([]*commonpb.Transaction, 0)
	newBatchIds := make([]string, 0)
	var err error

	txInSameBranch := bp.getTxInSameBranch(preHash, height)
	// 交易有可能锁在前面4个块中
	for i := int64(0); i < 4; i++ {
		if int64(height)-i < 0 {
			break
		}

		proposedBlocks := bp.proposalCache.GetProposedBlocksAt(height - uint64(i))
		if len(proposedBlocks) == 0 {
			continue
		}

		newFetchBatch, newBatchIds, err = bp.fetchFromProposalCache(proposedBlocks, txInSameBranch)
		if err != nil {
			continue
		}

		if len(newFetchBatch) != 0 {
			break
		}
	}
	return newFetchBatch, newBatchIds
}

func (bp *BlockProposerImpl) getTxInSameBranch(preHash []byte, height uint64) map[string]interface{} {
	hash := preHash
	txInSameBranch := make(map[string]interface{})
	// 同一分支下只存在3个区块
	for i := int64(1); i <= 3; i++ {
		if int64(height)-i < 0 {
			break
		}

		b, _ := bp.proposalCache.GetProposedBlockByHashAndHeight(hash, height-uint64(i))
		if b == nil || b.Header == nil {
			continue
		}

		for _, tx := range b.Txs {
			txInSameBranch[tx.Payload.TxId] = struct {
			}{}
		}
		hash = b.Header.PreBlockHash
	}
	return txInSameBranch
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
func (bp *BlockProposerImpl) getChainVersion() uint32 {
	if bp.chainConf == nil || bp.chainConf.ChainConfig() == nil {
		bp.log.Warnf("No chain config found, use default block version:%d", protocol.DefaultBlockVersion)
		return protocol.DefaultBlockVersion
	}
	return bp.chainConf.ChainConfig().GetBlockVersion()
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
//func (bp *BlockProposerImpl) isIdle() bool {
//	bp.idleMu.Lock()
//	defer bp.idleMu.Unlock()
//	return bp.idle
//}

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

//isSelfProposer, return if this node is consensus proposer
func (bp *BlockProposerImpl) isSelfProposer() bool {
	bp.proposerMu.RLock()
	defer bp.proposerMu.RUnlock()
	return bp.isProposer
}

/*
 * shouldProposeByMaxBFT, check if node should propose new block
 * Only for max bft consensus
 */
func (bp *BlockProposerImpl) shouldProposeByMaxBFT(height uint64, preHash []byte) bool {
	committedBlock := bp.ledgerCache.GetLastCommittedBlock()
	if committedBlock == nil {
		bp.log.Errorf("no committed block found")
		return false
	}
	currentHeight := committedBlock.Header.BlockHeight
	// proposing height must higher than current height
	if currentHeight >= height {
		bp.log.Errorf("current commit block height: %d, propose height: %d", currentHeight, height)
		return false
	}
	if height == currentHeight+1 {
		// height follows the last committed block
		if bytes.Equal(committedBlock.Header.BlockHash, preHash) {
			return true
		}
		bp.log.Errorf("block pre hash error, expect %x, got %x, can not propose",
			committedBlock.Header.BlockHash, preHash)
		return false
	}
	// if height not follows the last committed block, then check last proposed block
	b, _ := bp.proposalCache.GetProposedBlockByHashAndHeight(preHash, height-1)
	if b == nil {
		bp.log.Errorf("not find preBlock: [%d:%x]", height-1, preHash)
	}
	return b != nil
}

// ProposeBlock proposer block
func (bp *BlockProposerImpl) ProposeBlock(proposal *maxbft.BuildProposal) (*consensuspb.ProposalBlock, error) {
	defer func() {
		// change proposed status when call proposing by consensus.
		bp.OnReceiveProposeStatusChange(true)

		if bp.isSelfProposer() {
			bp.proposeTimer.Reset(bp.getDuration())
		}
	}()

	height := proposal.Height
	preHash := proposal.PreHash
	if ok, err := bp.shouldProposeByMaxBFTSync(height, preHash); !ok {
		bp.log.Errorf("not a legal proposal request [%d](%x), err: %v", height, preHash, err)
		return nil, err
	}

	//if !bp.setNotIdle() {
	//	bp.log.Infof("concurrent propose block [%d], yield!", height)
	//	return nil, nil
	//}
	//defer bp.setIdle()

	bp.log.Infof("trigger proposal from maxBFT, height[%d]", height)
	proposalBlock, err := bp.proposing(height, preHash)
	if err != nil {
		return nil, err
	}

	//<-bp.finishProposeC

	return proposalBlock, nil
}

/*
 * shouldProposeByMaxBFT, check if node should propose new block
 * Only for maxbft consensus
 */
func (bp *BlockProposerImpl) shouldProposeByMaxBFTSync(height uint64, preHash []byte) (bool, error) {
	var err error
	committedBlock := bp.ledgerCache.GetLastCommittedBlock()
	if committedBlock == nil {
		err = fmt.Errorf("no committed block found")
		return false, err
	}
	currentHeight := committedBlock.Header.BlockHeight
	// proposing height must higher than current height
	if currentHeight >= height {
		err = fmt.Errorf("current commit block height: %d, propose height: %d", currentHeight, height)
		return false, err
	}
	if height == currentHeight+1 {
		// height follows the last committed block
		if bytes.Equal(committedBlock.Header.BlockHash, preHash) {
			return true, nil
		}
		err = fmt.Errorf("block pre hash error, expect %x, got %x, can not propose",
			committedBlock.Header.BlockHash, preHash)
		return false, err

	}
	// if height not follows the last committed block, then check last proposed block
	//b, _ := bp.proposalCache.GetProposedBlockByHashAndHeight(preHash, height-1)
	//if b == nil {
	//	err = fmt.Errorf("not find preBlock: [%d:%x]", height-1, preHash)
	//	return false, err
	//}
	return true, nil
}

// OnReceiveRwSetVerifyFailTxs remove verify fail txs, deal with rw set verify fail txs
func (bp *BlockProposerImpl) OnReceiveRwSetVerifyFailTxs(rwSetVerifyFailTxs *consensuspb.RwSetVerifyFailTxs) {
	// deal case tx pool type not equal tx pool type batch

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

func (bp *BlockProposerImpl) fetchFromProposalCache(
	proposedBlocks []*commonpb.Block, txInSameBranch map[string]interface{}) (
	[]*commonpb.Transaction, []string, error) {
	txTimeout := int64(bp.chainConf.ChainConfig().Block.TxTimeout)
	newBatchIds := make([]string, 0)
	for _, proposedBlock := range proposedBlocks {
		if proposedBlock.Header.TxCount != 0 {

			// if block timeout, use the other block.
			blockTimeStamp := proposedBlock.Header.BlockTimestamp
			if utils.CurrentTimeSeconds()-blockTimeStamp >= txTimeout {
				bp.log.DebugDynamic(
					func() string {
						return fmt.Sprintf("block is time out,use the other block.(height:%d,hash:%x)",
							proposedBlock.Header.BlockHeight, proposedBlock.Header.BlockHash)
					})
				continue
			}

			removeTxs := make([]*commonpb.Transaction, 0)
			retryTxs := make([]*commonpb.Transaction, 0)
			keepTx := make([]*commonpb.Transaction, 0)
			for _, tx := range proposedBlock.Txs {
				txId := tx.Payload.TxId
				if _, exit := txInSameBranch[txId]; exit {
					retryTxs = append(retryTxs, tx)
					continue
				}

				if utils.CurrentTimeSeconds()-tx.Payload.Timestamp >= txTimeout {
					removeTxs = append(removeTxs, tx)
					continue
				}

				keepTx = append(keepTx, tx)
			}

			// all transaction time out,use the other block's tx.
			if len(keepTx) == 0 {
				bp.log.DebugDynamic(
					func() string {
						return fmt.Sprintf("no txs need to keep,use the other block.(height:%d,hash:%x)",
							proposedBlock.Header.BlockHeight, proposedBlock.Header.BlockHash)
					})
				continue
			}

			fetchBatch := keepTx
			batchIds, _, err := common.GetBatchIds(proposedBlock)
			if err != nil {
				return nil, nil, err
			}

			if len(removeTxs) != 0 || len(retryTxs) != 0 {
				newBatchIds, fetchBatch, _ = bp.removeAndRetryTx(proposedBlock.Header.BlockHeight, batchIds, removeTxs,
					keepTx, RETRY)
				bp.log.Infof("remove the overtime transactions, total:%d, fetch:%d, remove:%d",
					len(proposedBlock.Txs), len(fetchBatch), len(removeTxs))
			} else {
				// no tx need to remove or retry,use the old batchIds.
				newBatchIds = batchIds
			}

			// schedule tx again
			bp.log.Infof(fmt.Sprintf("fetch tx from cache,fetch:%d,remove:%d,height:%d,hash:%x",
				len(fetchBatch), len(removeTxs), proposedBlock.Header.BlockHeight, proposedBlock.Header.BlockHash))

			return fetchBatch, newBatchIds, nil
		}
	}
	return nil, newBatchIds, nil
}

func (bp *BlockProposerImpl) removeAndRetryTx(
	height uint64, batchIds []string, removeTxs, fetchBatch []*commonpb.Transaction, mode int) (
	[]string, []*commonpb.Transaction, [][]*commonpb.Transaction) {
	// don't remove tx when is batchTx pool
	if common.TxPoolType == batch.TxPoolType {
		// remove and get new batchIds
		if mode == RETRY {
			newBatchIds, fetchBatches := bp.txPool.ReGenTxBatchesWithRetryTxs(height, batchIds, fetchBatch)
			newFetchBatch := getFetchBatch(fetchBatches)

			return newBatchIds, newFetchBatch, fetchBatches
		}

		newBatchIds, fetchBatches := bp.txPool.ReGenTxBatchesWithRemoveTxs(height, batchIds, removeTxs)
		newFetchBatch := getFetchBatch(fetchBatches)

		return newBatchIds, newFetchBatch, fetchBatches

	}
	bp.txPool.RetryAndRemoveTxs(nil, removeTxs)
	return batchIds, fetchBatch, nil

}

func (bp *BlockProposerImpl) fetchBatchWithoutDupTxInSameBranch(height uint64, preHash []byte, batchIds []string,
	fetchBatch []*commonpb.Transaction, fetchBatches [][]*commonpb.Transaction) ([]string, []*commonpb.Transaction,
	[][]*commonpb.Transaction) {
	if common.TxPoolType == batch.TxPoolType {
		dupTxs := make([]*commonpb.Transaction, 0)
		for _, tx := range fetchBatch {
			if isExit, _ := common.IfExitInSameBranch(height, tx.Payload.TxId, bp.proposalCache, preHash); isExit {
				dupTxs = append(dupTxs, tx)
			}
		}
		if len(dupTxs) != 0 {
			batchIds, fetchBatches = bp.txPool.ReGenTxBatchesWithRemoveTxs(height, batchIds, dupTxs)
			fetchBatch = getFetchBatch(fetchBatches)
		}
		return batchIds, fetchBatch, fetchBatches
	}
	finalBatch := make([]*commonpb.Transaction, 0)
	for _, tx := range fetchBatch {
		if isExit, _ := common.IfExitInSameBranch(
			height, tx.Payload.TxId, bp.proposalCache, preHash); !isExit {
			finalBatch = append(finalBatch, tx)
		}
	}
	fetchBatch = finalBatch
	return batchIds, fetchBatch, fetchBatches
}

//// isIdle, to check if proposer is idle
//func (bp *BlockProposerImpl) isIdle() bool {
//	bp.idleMu.Lock()
//	defer bp.idleMu.Unlock()
//	return bp.idle
//}
//
///*
// * shouldProposeByBFT, check if node should propose new block
// * Only for *BFT consensus
// * if node is proposer, and node is not propose right now, and last proposed block is committed, then return true
// */
//func (bp *BlockProposerImpl) shouldProposeByBFT(height uint64) bool {
//	if !bp.isIdle() {
//		// concurrent control, proposer is proposing now
//		bp.log.Debugf("proposer is busy, not propose [%d] ", height)
//		return false
//	}
//	committedBlock := bp.ledgerCache.GetLastCommittedBlock()
//	if committedBlock == nil {
//		bp.log.Errorf("no committed block found")
//		return false
//	}
//	currentHeight := committedBlock.Header.BlockHeight
//	// proposing height must higher than current height
//	return currentHeight+1 == height
//}

// getFetchBatch return fetchBatch
func getFetchBatch(fetchBatches [][]*commonpb.Transaction) []*commonpb.Transaction {

	fetchBatch := make([]*commonpb.Transaction, 0)
	for _, txs := range fetchBatches {
		fetchBatch = append(fetchBatch, txs...)
	}

	return fetchBatch
}
