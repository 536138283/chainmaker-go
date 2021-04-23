/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	"chainmaker.org/chainmaker-go/utils"

	"errors"

	"chainmaker.org/chainmaker-go/protocol"
)

const DEFAULT_WAIT_TXS_TIMEOUT = time.Second * 2

type Proposer struct {
	lock                sync.Mutex
	chainId             string
	proposedSignal      *abft.PackagedSignal
	txPool              protocol.TxPool
	ledgerCache         protocol.LedgerCache
	log                 *logger.CMLogger
	identity            protocol.SigningMember
	snapshotManager     protocol.SnapshotManager
	chainConf           protocol.ChainConf
	vmMgr               protocol.VmManager
	abftCache           *cache.AbftCache
	msgBus              msgbus.MessageBus
	txBatch             []*commonpb.Transaction
	getTxBatchC         chan struct{}
	proposeEmptyTxBatch commonpb.Block
}

func NewProposer(ce *CoreExecute) *Proposer {
	return &Proposer{
		lock:            sync.Mutex{},
		chainId:         ce.chainId,
		txPool:          ce.txPool,
		ledgerCache:     ce.ledgerCache,
		log:             ce.log,
		identity:        ce.identity,
		chainConf:       ce.chainConf,
		vmMgr:           ce.vmMgr,
		snapshotManager: ce.snapshotManager,
		msgBus:          ce.msgBus,
		abftCache:       ce.abftCache,
		getTxBatchC:     make(chan struct{}),
	}
}

func (p *Proposer) verifyHeight() error {
	currentHeight, err := p.ledgerCache.CurrentHeight()
	if err != nil {
		return err
	}
	if currentHeight+1 != p.proposedSignal.BlockHeight {
		return errors.New("the propose signal height is inconsistent with the cache")
	}
	return nil
}

func (p *Proposer) proposeStatus() *commonpb.Block {
	txBatch := p.abftCache.GetProposedTxBatchCache()
	if txBatch == nil {
		return nil
	}
	return txBatch.GetTxBatch()
}

func (p *Proposer) Propose() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	//check height
	err := p.verifyHeight()

	//check propose status
	txBatch := p.proposeStatus()
	if txBatch != nil {
		p.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		p.log.Infof("The proposal has been completed, height: (%d)", txBatch.Header.BlockHeight)
		return nil
	}

	//start propose
	lastBlock := p.ledgerCache.GetLastCommittedBlock()
	txBatch, err = common.InitNewBlock(lastBlock, p.identity, p.chainId, p.chainConf)
	if err != nil {
		return err
	}
	p.proposeEmptyTxBatch = *txBatch
	//get a random number of transactions
	ticker := time.NewTicker(DEFAULT_WAIT_TXS_TIMEOUT)
	ctx, cancel := context.WithCancel(context.Background())
	go p.getTxBatchFromTxPool(p.proposedSignal.BlockHeight, ctx)
	select {
	case <-ticker.C:
		cancel()
		p.log.Debugf("there are no transactions in the tx pool, proposing an empty tx batch, height: (%d)", txBatch.Header.BlockHeight)
		p.msgBus.Publish(msgbus.ProposedBlock, &p.proposeEmptyTxBatch)
		p.abftCache.SetProposedTxBatchCache(txBatch, nil)
		return nil
	case <-p.getTxBatchC:
		timeLasts := make([]int64, 0)
		ssStartTick := utils.CurrentTimeMillisSeconds()

		snapshot := p.snapshotManager.NewSnapshot(lastBlock, txBatch)

		vmStartTick := utils.CurrentTimeMillisSeconds()
		ssLasts := vmStartTick - ssStartTick

		txScheduler := common.NewTxScheduler(p.vmMgr, p.chainId)
		txRWSetMap, err := txScheduler.Schedule(txBatch, p.txBatch, snapshot)

		vmLasts := utils.CurrentTimeMillisSeconds() - vmStartTick
		timeLasts = append(timeLasts, ssLasts, vmLasts)
		if err != nil {
			p.log.Errorf("schedule txBatch(%d,%x) error %s",
				txBatch.Header.BlockHeight, txBatch.Header.BlockHash, err)
			p.msgBus.Publish(msgbus.ProposedBlock, &p.proposeEmptyTxBatch)
			p.abftCache.SetProposedTxBatchCache(&p.proposeEmptyTxBatch, nil)
			return err
		}

		var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
		err = common.FinalizeBlock(txBatch, txRWSetMap, aclFailTxs, p.chainConf.ChainConfig().Crypto.Hash)
		if err != nil {
			p.log.Errorf("finalizeBlock txBatch(%d,%s) error %s",
				txBatch.Header.BlockHeight, hex.EncodeToString(txBatch.Header.BlockHash), err)
			p.msgBus.Publish(msgbus.ProposedBlock, &p.proposeEmptyTxBatch)
			p.abftCache.SetProposedTxBatchCache(&p.proposeEmptyTxBatch, nil)
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
		p.abftCache.SetProposedTxBatchCache(txBatch, txRWSetMap)
		p.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		p.log.Infof("proposer success [%d](txs:%d)", txBatch.Header.BlockHeight, txBatch.Header.TxCount)
	}
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
			time.Sleep(time.Millisecond * 100)
		}
	}
}
