/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/hbbft"
	"chainmaker.org/chainmaker-go/protocol"
)

type CoreExecute struct {
	chainId         string // chain id, to identity this chain
	hbbftCache      cache.HbbftCache
	ledgerCache     protocol.LedgerCache     // ledger cache
	txPool          protocol.TxPool          // tx pool provides tx batch
	snapshotManager protocol.SnapshotManager // snapshot manager
	identity        protocol.SigningMember   // identity manager
	msgBus          msgbus.MessageBus        // channel to give out proposed block
	ac              protocol.AccessControlProvider
	blockchainStore protocol.BlockchainStore
	chainConf       protocol.ChainConf // chain config
	log             *logger.CMLogger   // logger

	Committer Committer
	Packager  Packager
	Scheduler Scheduler
	Verifier  Verifier
}

type CoreExecuteConfig struct {
	ChainId         string
	TxPool          protocol.TxPool
	SnapshotManager protocol.SnapshotManager
	MsgBus          msgbus.MessageBus
	Identity        protocol.SigningMember
	LedgerCache     protocol.LedgerCache
	HbbftCache      cache.HbbftCache
	ChainConf       protocol.ChainConf
	AC              protocol.AccessControlProvider
	BlockchainStore protocol.BlockchainStore
	Log             *logger.CMLogger
}

func NewCoreExecute(ceConfig *CoreExecuteConfig) *CoreExecute {
	ce := &CoreExecute{
		chainId:         ceConfig.ChainId,
		hbbftCache:      ceConfig.HbbftCache,
		ledgerCache:     ceConfig.LedgerCache,
		txPool:          ceConfig.TxPool,
		snapshotManager: ceConfig.SnapshotManager,
		identity:        ceConfig.Identity,
		msgBus:          ceConfig.MsgBus,
		ac:              ceConfig.AC,
		blockchainStore: ceConfig.BlockchainStore,
		chainConf:       ceConfig.ChainConf,
		log:             ceConfig.Log,
	}

	return ce
}

func (ce *CoreExecute) Package() error {
	return nil
}

func (ce *CoreExecute) Schedule() (map[string]*commonpb.TxRWSet, error) {
	//todo
	return ce.scheduler.Schedule()
}

func (ce *CoreExecute) Verify(block *commonpb.Block) error {
	//
	return nil
}

func (ce *CoreExecute) Commit() error {
	return nil, nil
}

// OnQuit called when quit subsribe message from message bus
func (c *CoreExecute) OnQuit() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreExecute) OnMessage(message *msgbus.Message) {

	switch message.Topic {
	case msgbus.PackageSignal:
		if packagedSignal, ok := message.Payload.(hbbft.PackagedSignal); ok {
			c.Packager.packagedSignal = &packagedSignal
		}
	case msgbus.VerifyBlock:

	case msgbus.CommitedTxBatchs:

	}
}

func (c *CoreExecute) Stop() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreExecute) Start() {

}

func (ce *CoreExecute) SetPackageStatus(status bool) {
	ce.Packager.packageStatus = status
}
