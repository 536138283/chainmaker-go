/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/hbbft"
	"chainmaker.org/chainmaker-go/utils"
	"encoding/hex"

	"chainmaker.org/chainmaker-go/protocol"
	"errors"
)

type PackageStatus int32

const (
	NoPackaging PackageStatus = iota
	Packaging
	Packaged
)

type Packager struct {
	chainId         string
	packagedSignal  *hbbft.PackagedSignal
	packageStatus   PackageStatus
	txPool          protocol.TxPool
	ledgerCache     protocol.LedgerCache
	log             *logger.CMLogger
	identity        protocol.SigningMember
	snapshotManager protocol.SnapshotManager
	chainConf       protocol.ChainConf
	vmMgr           protocol.VmManager
	hbbftCache      *cache.HbbftCache
	msgBus          msgbus.MessageBus
}

func NewPackager(ce *CoreExecute) *Packager {
	return &Packager{
		chainId:         ce.chainId,
		txPool:          ce.txPool,
		ledgerCache:     ce.ledgerCache,
		log:             ce.log,
		identity:        ce.identity,
		chainConf:       ce.chainConf,
		vmMgr:           ce.vmMgr,
		snapshotManager: ce.snapshotManager,
		msgBus:          ce.msgBus,
		hbbftCache:      ce.hbbftCache,
	}
}
func (p *Packager) SetPackageStatus(status PackageStatus) {
	p.packageStatus = status
}
func (p *Packager) GetPackageStatus() PackageStatus {
	return p.packageStatus
}
func (p *Packager) verifyHeight() (bool, error) {
	currentHeight, err := p.ledgerCache.CurrentHeight()
	if err != nil {
		return false, err
	}
	if currentHeight+1 != p.packagedSignal.BlockHeight {
		return false, errors.New("the packaging signal height is inconsistent with the cache")
	}
	return true, nil
}

func (p *Packager) checkPackageStatus() bool {
	if p.packageStatus == NoPackaging {
		p.SetPackageStatus(Packaging)
		return true
	}
	if p.packageStatus == Packaging {
		return false
	}
	if p.packageStatus == Packaged {
		txBatch := p.hbbftCache.GetTxBatchCache()
		p.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		return false
	}
	return false
}

func (p *Packager) Package() error {
	ok, err := p.verifyHeight()
	if !ok {
		return err
	}
	ok = p.checkPackageStatus()
	if ok {

		lastBlock := p.ledgerCache.GetLastCommittedBlock()
		txBatch, err := common.InitNewBlock(lastBlock, p.identity, p.chainId, p.chainConf)
		if err != nil {
			return err
		}
		checkedBatch := p.txPool.FetchTxBatch(p.packagedSignal.BlockHeight)
		if checkedBatch == nil || len(checkedBatch) == 0 {
			p.log.Debugf("no txs in tx pool, packaging txBatch stoped")
			return nil
		}
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
		p.hbbftCache.SetTxBatchCache(txBatch)
		p.msgBus.Publish(msgbus.ProposedBlock, txBatch)
		p.log.Infof("proposer success [%d](txs:%d)", txBatch.Header.BlockHeight, txBatch.Header.TxCount)
		p.SetPackageStatus(Packaged)
	}
	return nil
}
