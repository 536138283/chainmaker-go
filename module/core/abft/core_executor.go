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
	ABFTCache       *cache.AbftCache
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
		abftCache:       ceConfig.ABFTCache,
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
	ce.Proposer = NewProposer(ceConfig)
	ce.Verifier, err = NewVerifier(ceConfig)
	if err != nil {
		return nil, err
	}
	ce.Committer = NewCommitter(ceConfig)
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
		proposedSignal, ok := message.Payload.(*abft.PackagedSignal)
		if !ok {
			c.log.Warnf("propose failed, Invalid Signal Type")
			return
		}
		c.log.Debugf("handle package signal, block height [%d]", proposedSignal.BlockHeight)
		if err := c.Proposer.Propose(proposedSignal); err != nil {
			c.log.Warnf("propose failed, error %s", err.Error())
		}
	case msgbus.VerifyBlock:
		block, ok := message.Payload.(*commonPb.Block)
		if !ok {
			c.log.Warnf("verify block failed, Invalid Signal Type")
			return
		}
		c.log.Debugf("handle verify block signal, block height [%d]", block.Header.BlockHeight)
		if err := c.Verifier.verify(block); err != nil {
			c.log.Warnf("verify failed, error %s", err.Error())
		}
	case msgbus.CommitedTxBatchs:
		txBatchAfterABA, ok := message.Payload.(*abft.TxBatchAfterABA)
		if !ok {
			c.log.Warnf("commited txBatch failed, Invalid Signal Type")
			return
		}
		c.log.Debugf("handle commit tx batch signal, block height [%d]", txBatchAfterABA.BlockHeight)
		if err := c.Committer.Commit(txBatchAfterABA); err != nil {
			c.log.Warnf("commit fail, error %s", err.Error())
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
