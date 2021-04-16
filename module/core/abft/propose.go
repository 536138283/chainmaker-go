/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"encoding/hex"

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

type ProposeStatus int32

const (
	NoPackaging ProposeStatus = iota
	Packaging
	Proposed
)

type Proposer struct {
	chainId         string
	proposedSignal  *abft.PackagedSignal
	proposeStatus   ProposeStatus
	txPool          protocol.TxPool
	ledgerCache     protocol.LedgerCache
	log             *logger.CMLogger
	identity        protocol.SigningMember
	snapshotManager protocol.SnapshotManager
	chainConf       protocol.ChainConf
	vmMgr           protocol.VmManager
	abftCache       *cache.AbftCache
	msgBus          msgbus.MessageBus
}

func NewProposer(ce *CoreExecute) *Proposer {
	return &Proposer{
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
	}
}
func (p *Proposer) SetProposeStatus(status ProposeStatus) {
	p.proposeStatus = status
}
func (p *Proposer) GetProposeStatus() ProposeStatus {
	return p.proposeStatus
}
func (p *Proposer) verifyHeight() (bool, error) {
	currentHeight, err := p.ledgerCache.CurrentHeight()
	if err != nil {
		return false, err
	}
	if currentHeight+1 != p.proposedSignal.BlockHeight {
		return false, errors.New("the packaging signal height is inconsistent with the cache")
	}
	return true, nil
}

//优化
func (p *Proposer) checkProposeStatus() bool {
	//TODO
	switch p.proposeStatus {
	case NoPackaging:
		p.SetProposeStatus(Packaging)
		return true
	case Packaging:
		return false
	case Proposed:
		txBatch := p.abftCache.GetTxBatchCache()
		p.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		return false
	default:
		p.log.Errorf(
			"Invalid Propose Status: %v",
			p.proposeStatus,
		)
		return false
	}
}

//TODO IF 优化
func (p *Proposer) Propose() error {
	ok, err := p.verifyHeight()
	if !ok {
		return err
	}
	ok = p.checkProposeStatus()
	if ok {

		lastBlock := p.ledgerCache.GetLastCommittedBlock()
		txBatch, err := common.InitNewBlock(lastBlock, p.identity, p.chainId, p.chainConf)
		if err != nil {
			return err
		}
		checkedBatch := p.txPool.FetchTxBatch(p.proposedSignal.BlockHeight)
		if checkedBatch == nil || len(checkedBatch) == 0 {
			p.log.Debugf("no txs in tx pool, packaging txBatch stoped")
			return nil
		}
		//TODO check batch

		timeLasts := make([]int64, 0)
		ssStartTick := utils.CurrentTimeMillisSeconds()
		snapshot := p.snapshotManager.NewSnapshot(lastBlock, txBatch)
		vmStartTick := utils.CurrentTimeMillisSeconds()
		ssLasts := vmStartTick - ssStartTick

		txScheduler := common.NewTxScheduler(p.vmMgr, p.chainId)
		txRWSetMap, err := txScheduler.Schedule(txBatch, checkedBatch, snapshot)

		vmLasts := utils.CurrentTimeMillisSeconds() - vmStartTick
		timeLasts = append(timeLasts, ssLasts, vmLasts)
		if err != nil {
			p.log.Errorf("schedule txBatch(%d,%x) error %s",
				txBatch.Header.BlockHeight, txBatch.Header.BlockHash, err)
		}

		var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
		err = common.FinalizeBlock(txBatch, txRWSetMap, aclFailTxs, p.chainConf.ChainConfig().Crypto.Hash)
		if err != nil {
			p.log.Errorf("finalizeBlock txBatch(%d,%s) error %s",
				txBatch.Header.BlockHeight, hex.EncodeToString(txBatch.Header.BlockHash), err)
		}
		var txsTimeout = make([]*commonpb.Transaction, 0)
		if len(txRWSetMap) < len(checkedBatch) {
			for _, tx := range checkedBatch {
				if _, ok := txRWSetMap[tx.Header.TxId]; !ok {
					txsTimeout = append(txsTimeout, tx)
				}
			}
			p.txPool.RetryAndRemoveTxs(txsTimeout, nil)
		}
		p.abftCache.SetTxBatchCache(txBatch)
		p.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		p.log.Infof("proposer success [%d](txs:%d)", txBatch.Header.BlockHeight, txBatch.Header.TxCount)
		p.SetProposeStatus(Proposed)
	}
	return nil
}
