/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"
	txpoolpb "chainmaker.org/chainmaker/pb-go/v2/txpool"

	"chainmaker.org/chainmaker-go/module/core/cache"
	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/abft"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

// DEFAULT_WAIT_TXS_TIMEOUT default wait txs timeout
const DEFAULT_WAIT_TXS_TIMEOUT = time.Second * 2

// BlockProposerImpl struct
type BlockProposerImpl struct {
	lock            sync.Mutex
	chainId         string
	txPool          protocol.TxPool
	ledgerCache     protocol.LedgerCache
	log             protocol.Logger
	identity        protocol.SigningMember
	snapshotManager protocol.SnapshotManager
	chainConf       protocol.ChainConf
	vmMgr           protocol.VmManager
	abftCache       *cache.AbftCache
	msgBus          msgbus.MessageBus
	txBatch         []*commonpb.Transaction
	getTxBatchC     chan struct{}
	retryInterval   time.Duration //(Millisecond)
	txScheduler     protocol.TxScheduler
}

// NewBlockProposer params CoreEngineConfig, TxScheduler, return BlockProposerImpl, error
func NewBlockProposer(ceConfig *conf.CoreEngineConfig, txScheduler protocol.TxScheduler) (*BlockProposerImpl, error) {
	blockProposerImpl := &BlockProposerImpl{
		lock:            sync.Mutex{},
		chainId:         ceConfig.ChainId,
		txPool:          ceConfig.TxPool,
		ledgerCache:     ceConfig.LedgerCache,
		log:             ceConfig.Log,
		identity:        ceConfig.Identity,
		chainConf:       ceConfig.ChainConf,
		vmMgr:           ceConfig.VmMgr,
		snapshotManager: ceConfig.SnapshotManager,
		msgBus:          ceConfig.MsgBus,
		abftCache:       ceConfig.ABFTCache,
		getTxBatchC:     make(chan struct{}),
		retryInterval:   100,
		txScheduler:     txScheduler,
	}

	//blockProposerImpl

	return blockProposerImpl, nil
}

// Start proposer
func (bp *BlockProposerImpl) Start() error {
	defer bp.log.Info("block proposer starts")

	//go bp.startProposingLoop()

	return nil
}

// Stop proposing loop
func (bp *BlockProposerImpl) Stop() error {
	defer bp.log.Infof("block proposer stopped")
	//bp.exitC <- true
	return nil
}

// OnReceiveTxPoolSignal receive txpool signal and deliver to chan txpool signal
func (bp *BlockProposerImpl) OnReceiveTxPoolSignal(txPoolSignal *txpoolpb.TxPoolSignal) {
	//bp.txPoolSignalC <- txPoolSignal
}

// OnReceiveProposeStatusChange to update isProposer status when received proposeStatus from consensus
// if node is proposer, then reset the timer, otherwise stop the timer
func (bp *BlockProposerImpl) OnReceiveProposeStatusChange(proposeStatus bool) {

}

// OnReceiveMaxBFTProposal to check if this proposer should propose a new block
// Only for maxbft consensus
func (bp *BlockProposerImpl) OnReceiveMaxBFTProposal(proposal *maxbft.BuildProposal) {

}

// OnReceiveYieldProposeSignal receive yield propose signal
func (bp *BlockProposerImpl) OnReceiveYieldProposeSignal(isYield bool) {

}

// OnReceiveRwSetVerifyFailTxs remove verify fail txs
func (bp *BlockProposerImpl) OnReceiveRwSetVerifyFailTxs(rwSetVerifyFailTxs *consensuspb.RwSetVerifyFailTxs) {

}

// ProposeBlock params BuildProposal, return ProposalBlock, error
func (bp *BlockProposerImpl) ProposeBlock(proposal *maxbft.BuildProposal) (*consensuspb.ProposalBlock, error) {

	return nil, nil
}

// getTxBatchFromABFTCache return Block
func (bp *BlockProposerImpl) getTxBatchFromABFTCache() *commonpb.Block {
	txBatch := bp.abftCache.GetProposedTxBatch()
	if txBatch == nil {
		return nil
	}
	return txBatch.GetTxBatch()
}

// Propose params PackagedSignal, return error
func (bp *BlockProposerImpl) Propose(proposedSignal *abft.PackagedSignal) error {
	bp.lock.Lock()
	defer bp.lock.Unlock()

	//check height
	err := common.VerifyHeight(proposedSignal.BlockHeight, bp.ledgerCache)
	if err != nil {
		return err
	}

	//check propose status
	txBatch := bp.getTxBatchFromABFTCache()
	if txBatch != nil && txBatch.Header.BlockHeight == proposedSignal.BlockHeight {
		bp.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		bp.log.Debugf("The proposal has been completed, height: (%d)", txBatch.Header.BlockHeight)
		return nil
	}

	//start propose
	lastBlock := bp.ledgerCache.GetLastCommittedBlock()
	blockBatch, err := common.InitNewBlock(lastBlock, bp.identity, bp.chainId, bp.chainConf, false)
	if err != nil {
		return err
	}
	emptyBlockBatch := *blockBatch
	//get a random number of transactions
	ticker := time.NewTicker(DEFAULT_WAIT_TXS_TIMEOUT)
	ctx, cancel := context.WithCancel(context.Background())
	go bp.getTxBatchFromTxPool(ctx, proposedSignal.BlockHeight)
	select {
	case <-ticker.C:
		cancel()
		bp.log.Debugf("there are no transactions in the tx pool, proposing an empty tx batch, height: (%d)",
			emptyBlockBatch.Header.BlockHeight)
		err = common.FinalizeBlock(blockBatch, nil, nil, bp.chainConf.ChainConfig().Crypto.Hash, bp.log)
		if err != nil {
			return err
		}
		bp.msgBus.Publish(msgbus.ProposedBlock, &emptyBlockBatch)
		rwSetMap := make(map[string]*commonpb.TxRWSet)
		bp.abftCache.SetProposedTxBatch(&emptyBlockBatch, rwSetMap)
		bp.log.Infof("proposer success [%d](txs:%d)", emptyBlockBatch.Header.BlockHeight, emptyBlockBatch.Header.TxCount)
		return nil
	case <-bp.getTxBatchC:
		cancel()
		if err := bp.doPropose(lastBlock, blockBatch); err != nil {
			return err
		}
	}
	return nil
}

// doPropose params lastBlock, blockBatch, return error
func (bp *BlockProposerImpl) doPropose(lastBlock, blockBatch *commonpb.Block) error {
	emptyBlockBatch := *blockBatch
	snapshot := bp.snapshotManager.NewSnapshot(lastBlock, blockBatch)
	vmStartTick := utils.CurrentTimeMillisSeconds()
	txRWSetMap, _, err := bp.txScheduler.Schedule(blockBatch, bp.txBatch, snapshot)
	vmLasts := utils.CurrentTimeMillisSeconds() - vmStartTick
	rwSetMap := make(map[string]*commonpb.TxRWSet)
	if err != nil {
		bp.log.Errorf("schedule txBatch(%d,%x) error %s",
			blockBatch.Header.BlockHeight, blockBatch.Header.BlockHash, err)
		bp.msgBus.Publish(msgbus.ProposedBlock, emptyBlockBatch)
		bp.abftCache.SetProposedTxBatch(&emptyBlockBatch, rwSetMap)
		return err
	}

	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	finalizeStartTick := utils.CurrentTimeMillisSeconds()
	err = common.FinalizeBlock(blockBatch, txRWSetMap, aclFailTxs, bp.chainConf.ChainConfig().Crypto.Hash, bp.log)
	finalizeLasts := utils.CurrentTimeMillisSeconds() - finalizeStartTick
	if err != nil {
		bp.log.Errorf("finalizeBlock txBatch(%d,%s) error %s",
			blockBatch.Header.BlockHeight, hex.EncodeToString(blockBatch.Header.BlockHash), err)
		bp.msgBus.Publish(msgbus.ProposedBlock, emptyBlockBatch)
		bp.abftCache.SetProposedTxBatch(&emptyBlockBatch, rwSetMap)
		return err
	}

	var txsTimeout = make([]*commonpb.Transaction, 0)
	if len(txRWSetMap) < len(bp.txBatch) {
		for _, tx := range bp.txBatch {
			if _, ok := txRWSetMap[tx.Payload.TxId]; !ok {
				txsTimeout = append(txsTimeout, tx)
			}
		}
		bp.txPool.RetryTxs(txsTimeout)
	}

	bp.log.Debugf("schedule success [%d](txs:%d), time used(vm:%d,finalizeBlock:%d)",
		blockBatch.Header.BlockHeight, blockBatch.Header.TxCount,
		vmLasts, finalizeLasts)
	bp.abftCache.SetProposedTxBatch(blockBatch, txRWSetMap)
	bp.msgBus.Publish(msgbus.ProposedBlock, blockBatch)
	bp.log.Infof("proposer success [%d](txs:%d)", blockBatch.Header.BlockHeight, blockBatch.Header.TxCount)

	return nil
}

func (bp *BlockProposerImpl) getTxBatchFromTxPool(ctx context.Context, height uint64) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			txBatch := bp.txPool.FetchTxs(height)
			if txBatch != nil || len(txBatch) != 0 {
				bp.txBatch = txBatch
				bp.getTxBatchC <- struct{}{}
				return
			}
			time.Sleep(time.Millisecond * bp.retryInterval)
		}
	}
}
