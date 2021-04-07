/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	"chainmaker.org/chainmaker-go/core/common"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/hbbft"

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
	chainId        string
	packagedSignal *hbbft.PackagedSignal
	packageStatus  PackageStatus
	txPool         protocol.TxPool
	ledgerCache    protocol.LedgerCache
	log            *logger.CMLogger
	identity       protocol.SigningMember
	chainConf      protocol.ChainConf
	vmMgr          protocol.VmManager
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

}
