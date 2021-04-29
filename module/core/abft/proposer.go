/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/utils"
	"context"
	"encoding/hex"
	"sync"
	"time"
)

const DEFAULT_WAIT_TXS_TIMEOUT = time.Second * 2

type Proposer struct {
	lock            sync.Mutex
	chainId         string
	txPool          protocol.TxPool
	ledgerCache     protocol.LedgerCache
	log             *logger.CMLogger
	identity        protocol.SigningMember
	snapshotManager protocol.SnapshotManager
	chainConf       protocol.ChainConf
	vmMgr           protocol.VmManager
	abftCache       *cache.AbftCache
	msgBus          msgbus.MessageBus
	txBatch         []*commonpb.Transaction
	getTxBatchC     chan struct{}
	retryInterval   time.Duration //(Millisecond)
	txScheduler     *common.TxScheduler
}

func NewProposer(ceConfig *CoreExecuteConfig) *Proposer {
	return &Proposer{
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
		txScheduler:     common.NewTxScheduler(ceConfig.VmMgr, ceConfig.ChainId),
	}
}

func (p *Proposer) getProposeStatus() *commonpb.Block {
	txBatch := p.abftCache.GetProposedTxBatch()
	if txBatch == nil {
		return nil
	}
	return txBatch.GetTxBatch()
}

func (p *Proposer) Propose(proposedSignal *abft.PackagedSignal) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	//check height
	err := common.VerifyHeight(proposedSignal.BlockHeight, p.ledgerCache)
	if err != nil {
		return err
	}

	//check propose status
	txBatch := p.getProposeStatus()
	if txBatch != nil {
		p.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		p.log.Infof("The proposal has been completed, height: (%d)", txBatch.Header.BlockHeight)
		return nil
	}

	//start propose
	lastBlock := p.ledgerCache.GetLastCommittedBlock()
	blockBatch, err := common.InitNewBlock(lastBlock, p.identity, p.chainId, p.chainConf)
	if err != nil {
		return err
	}
	emptyBlockBatch := *blockBatch
	//get a random number of transactions
	ticker := time.NewTicker(DEFAULT_WAIT_TXS_TIMEOUT)
	ctx, cancel := context.WithCancel(context.Background())
	go p.getTxBatchFromTxPool(proposedSignal.BlockHeight, ctx)
	select {
	case <-ticker.C:
		cancel()
		p.log.Debugf("there are no transactions in the tx pool, proposing an empty tx batch, height: (%d)", emptyBlockBatch.Header.BlockHeight)
		p.msgBus.Publish(msgbus.ProposedBlock, &emptyBlockBatch)
		p.abftCache.SetProposedTxBatch(blockBatch, nil)
		return nil
	case <-p.getTxBatchC:
		if err := p.doPropose(lastBlock, blockBatch, &emptyBlockBatch); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proposer) doPropose(lastBlock, blockBatch, emptyBlockBatch *commonpb.Block) error {
	timeLasts := make([]int64, 0)
	ssStartTick := utils.CurrentTimeMillisSeconds()

	snapshot := p.snapshotManager.NewSnapshot(lastBlock, blockBatch)

	vmStartTick := utils.CurrentTimeMillisSeconds()
	ssLasts := vmStartTick - ssStartTick

	txRWSetMap, err := p.txScheduler.Schedule(blockBatch, p.txBatch, snapshot)

	vmLasts := utils.CurrentTimeMillisSeconds() - vmStartTick
	timeLasts = append(timeLasts, ssLasts, vmLasts)
	if err != nil {
		p.log.Errorf("schedule txBatch(%d,%x) error %s",
			blockBatch.Header.BlockHeight, blockBatch.Header.BlockHash, err)
		p.msgBus.Publish(msgbus.ProposedBlock, emptyBlockBatch)
		p.abftCache.SetProposedTxBatch(emptyBlockBatch, nil)
		return err
	}

	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	err = common.FinalizeBlock(blockBatch, txRWSetMap, aclFailTxs, p.chainConf.ChainConfig().Crypto.Hash)
	if err != nil {
		p.log.Errorf("finalizeBlock txBatch(%d,%s) error %s",
			blockBatch.Header.BlockHeight, hex.EncodeToString(blockBatch.Header.BlockHash), err)
		p.msgBus.Publish(msgbus.ProposedBlock, emptyBlockBatch)
		p.abftCache.SetProposedTxBatch(emptyBlockBatch, nil)
		return err
	}

	var txsTimeout = make([]*commonpb.Transaction, 0)
	if len(txRWSetMap) < len(p.txBatch) {
		for _, tx := range p.txBatch {
			if _, ok := txRWSetMap[tx.Header.TxId]; !ok {
				txsTimeout = append(txsTimeout, tx)
			}
		}
		p.txPool.RetryAndRemoveTxs(txsTimeout, nil)
	}

	p.abftCache.SetProposedTxBatch(blockBatch, txRWSetMap)
	p.msgBus.Publish(msgbus.ProposedBlock, blockBatch)
	p.log.Infof("proposer success [%d](txs:%d)", blockBatch.Header.BlockHeight, blockBatch.Header.TxCount)

	return nil
}

func (p *Proposer) getTxBatchFromTxPool(height int64, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			txBatch := p.txPool.FetchTxBatch(height)
			if txBatch != nil || len(txBatch) != 0 {
				p.txBatch = txBatch
				p.getTxBatchC <- struct{}{}
				return
			}
			time.Sleep(time.Millisecond * p.retryInterval)
		}
	}
}
