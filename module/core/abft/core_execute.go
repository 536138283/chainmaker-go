/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	"chainmaker.org/chainmaker-go/protocol"
)

type CoreExecute struct {
	chainId         string // chain id, to identity this chain
	abftCache       *cache.AbftCache
	ledgerCache     protocol.LedgerCache     // ledger cache
	txPool          protocol.TxPool          // tx pool provides tx batch
	snapshotManager protocol.SnapshotManager // snapshot manager
	identity        protocol.SigningMember   // identity manager
	msgBus          msgbus.MessageBus        // channel to give out proposed block
	vmMgr           protocol.VmManager
	ac              protocol.AccessControlProvider
	blockchainStore protocol.BlockchainStore
	chainConf       protocol.ChainConf // chain config
	log             *logger.CMLogger   // logger

	Committer *Committer
	Proposer  *Proposer
	Verifier  *Verifier
}

type CoreExecuteConfig struct {
	ChainId         string
	TxPool          protocol.TxPool
	SnapshotManager protocol.SnapshotManager
	MsgBus          msgbus.MessageBus
	Identity        protocol.SigningMember
	LedgerCache     protocol.LedgerCache
	ChainConf       protocol.ChainConf
	AC              protocol.AccessControlProvider
	BlockchainStore protocol.BlockchainStore
	Log             *logger.CMLogger
	VmMgr           protocol.VmManager
}

func NewCoreExecute(ceConfig *CoreExecuteConfig) (*CoreExecute, error) {
	ce := &CoreExecute{
		chainId:         ceConfig.ChainId,
		ledgerCache:     ceConfig.LedgerCache,
		txPool:          ceConfig.TxPool,
		snapshotManager: ceConfig.SnapshotManager,
		identity:        ceConfig.Identity,
		msgBus:          ceConfig.MsgBus,
		ac:              ceConfig.AC,
		blockchainStore: ceConfig.BlockchainStore,
		chainConf:       ceConfig.ChainConf,
		log:             ceConfig.Log,
		vmMgr:           ceConfig.VmMgr,
	}
	var err error
	ce.abftCache = cache.NewAbftCache()
	ce.Proposer = NewProposer(ce)
	ce.Verifier, err = NewVerifier(ce)
	if err != nil {
		return nil, err
	}
	ce.Committer = NewCommitter(ce)
	return ce, nil
}

// OnQuit called when quit subsribe message from message bus
func (c *CoreExecute) OnQuit() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreExecute) OnMessage(message *msgbus.Message) {

	switch message.Topic {
	case msgbus.PackageSignal:
		if proposedSignal, ok := message.Payload.(abft.PackagedSignal); ok {
			c.Proposer.proposedSignal = &proposedSignal
		}
		if err := c.Proposer.Propose(); err != nil {
			c.log.Warnf("propose failed, error %s",
				err.Error())
		}
	case msgbus.VerifyBlock:
		if block, ok := message.Payload.(commonPb.Block); ok {
			c.Verifier.goRoutinePool.Submit(c.Verifier.verifyTask(&block, protocol.CONSENSUS_VERIFY))
		}
	case msgbus.CommitedTxBatchs:
		if txBatchAfterABA, ok := message.Payload.(abft.TxBatchAfterABA); ok {
			if err := c.Committer.Commit(
				txBatchAfterABA.BlockHeight,
				txBatchAfterABA.TxBatchHash); err != nil {
				c.log.Warnf("commit fail, error %s",
					err.Error())
			}
		}
	}
}

func (c *CoreExecute) Stop() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreExecute) Start() {
	c.msgBus.Register(msgbus.ProposeState, c)
	c.msgBus.Register(msgbus.VerifyBlock, c)
	c.msgBus.Register(msgbus.CommitBlock, c)
	c.msgBus.Register(msgbus.PackageSignal, c)
	c.msgBus.Register(msgbus.CommitedTxBatchs, c)
}
