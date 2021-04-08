/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
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
		return true
	}
	if p.packageStatus == Packaging {
		return false
	}
	if p.packageStatus == Packaged {
		//TODO 重新发送缓存打包好的批次
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
	if !ok {
		return nil
	}
	lastBlock := p.ledgerCache.GetLastCommittedBlock()
	block, err := common.InitNewBlock(lastBlock, p.identity, p.chainId, p.chainConf)
	if err != nil {
		return err
	}
	checkedBatch := p.txPool.FetchTxBatch(p.packagedSignal.BlockHeight)
	if checkedBatch == nil || len(checkedBatch) == 0 {
		// can not propose empty block and tx batch is empty, then yield proposing.
		p.log.Debugf("no txs in tx pool, packaging txBatch stoped")
		return nil
	}
	timeLasts := make([]int64, 0)
	ssStartTick := utils.CurrentTimeMillisSeconds()
	snapshot := p.snapshotManager.NewSnapshot(lastBlock, block)
	vmStartTick := utils.CurrentTimeMillisSeconds()
	ssLasts := vmStartTick - ssStartTick
	txScheduler := common.NewTxScheduler(p.vmMgr, p.chainId)
	txRWSetMap, err := common.Schedule(txScheduler, block, checkedBatch, snapshot)
	vmLasts := utils.CurrentTimeMillisSeconds() - vmStartTick
	timeLasts = append(timeLasts, ssLasts, vmLasts)
	if err != nil {
		p.log.Errorf("schedule txBatch(%d,%x) error %s",
			block.Header.BlockHeight, block.Header.BlockHash, err)
	}
	var aclFailTxs = make([]*commonpb.Transaction, 0) // No need to ACL check, this slice is empty
	err = common.FinalizeBlock(block, txRWSetMap, aclFailTxs, p.chainConf.ChainConfig().Crypto.Hash)
	if err != nil {
		p.log.Errorf("finalizeBlock txBatch(%d,%s) error %s",
			block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash), err)
	}
	// get txs schedule timeout and put back to txpool
	var txsTimeout = make([]*commonpb.Transaction, 0)
	if len(txRWSetMap) < len(checkedBatch) {
		// if tx not in txRWSetMap, tx should be put back to txpool
		for _, tx := range checkedBatch {
			if _, ok := txRWSetMap[tx.Header.TxId]; !ok {
				txsTimeout = append(txsTimeout, tx)
			}
		}
		p.txPool.RetryAndRemoveTxs(txsTimeout, nil)
	}
	//TODO 缓存该批次 ===》msgBus RBC
}
