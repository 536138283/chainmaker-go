/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abftmode

import (
	"chainmaker.org/chainmaker-go/module/core/abftmode/commiter"
	"chainmaker.org/chainmaker-go/module/core/abftmode/proposer"
	"chainmaker.org/chainmaker-go/module/core/abftmode/verifier"
	"chainmaker.org/chainmaker-go/module/core/cache"
	"chainmaker.org/chainmaker-go/module/core/common/scheduler"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/abft"
	"chainmaker.org/chainmaker/protocol/v2"
)

type CoreEngine struct {
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
	log             protocol.Logger    // logger
	blockProposer  *proposer.BlockProposerImpl  // block proposer, to generate new block when node is proposer
	BlockVerifier  protocol.BlockVerifier  // block verifier, to verify block that proposer generated
	BlockCommitter *commiter.BlockCommitter // block committer, to commit block to store after consensus
	MaxbftHelper   protocol.MaxbftHelper
}

func NewCoreEngine(ceConfig *conf.CoreEngineConfig) (*CoreEngine, error) {
	ce := &CoreEngine{
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

	var schedulerFactory scheduler.TxSchedulerFactory
	txScheduler := schedulerFactory.NewTxScheduler(ceConfig.VmMgr, ceConfig.ChainConf, ceConfig.StoreHelper)

	var err error
	// init block proposer
	blockProposer, err := proposer.NewBlockProposer(ceConfig, txScheduler)
	if err != nil {
		return nil, err
	}
	ce.blockProposer = blockProposer

	// init block verifier
	ce.BlockVerifier, err = verifier.NewVerifier(ceConfig, txScheduler)
	if err != nil {
		return nil, err
	}

	// init block committer
	ce.BlockCommitter = commiter.NewCommitter(ceConfig)

	return ce, nil
}

// OnQuit called when quit subsribe message from message bus
func (c *CoreEngine) OnQuit() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreEngine) OnMessage(message *msgbus.Message) {

	switch message.Topic {
	case msgbus.PackageSignal:
		proposedSignal, ok := message.Payload.(*abft.PackagedSignal)
		if !ok {
			c.log.Warnf("propose failed, Invalid Signal Type")
			return
		}
		c.log.Debugf("handle package signal, block height [%d]", proposedSignal.BlockHeight)
		if err := c.blockProposer.Propose(proposedSignal); err != nil {
			c.log.Warnf("propose failed, error %s", err.Error())
		}
	case msgbus.VerifyBlock:
		block, ok := message.Payload.(*commonPb.Block)
		if !ok {
			c.log.Warnf("verify block failed, Invalid Signal Type")
			return
		}
		c.log.Debugf("handle verify block signal, block height [%d]", block.Header.BlockHeight)
		if err := c.BlockVerifier.VerifyBlock(block,  protocol.CONSENSUS_VERIFY); err != nil {
			c.log.Warnf("verify failed, error %s", err.Error())
		}
	case msgbus.CommitedTxBatchs:
		txBatchAfterABA, ok := message.Payload.(*abft.TxBatchAfterABA)
		if !ok {
			c.log.Warnf("commited txBatch failed, Invalid Signal Type")
			return
		}
		c.log.Debugf("handle commit tx batch signal, block height [%d]", txBatchAfterABA.BlockHeight)
		if err := c.BlockCommitter.Commit(txBatchAfterABA); err != nil {
			c.log.Warnf("commit fail, error %s", err.Error())
		}
	}
}

func (c *CoreEngine) Stop() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreEngine) Start() {
	c.msgBus.Register(msgbus.ProposeState, c)
	c.msgBus.Register(msgbus.VerifyBlock, c)
	c.msgBus.Register(msgbus.CommitBlock, c)
	c.msgBus.Register(msgbus.PackageSignal, c)
	c.msgBus.Register(msgbus.CommitedTxBatchs, c)
}

func (c *CoreEngine) GetBlockProposer() protocol.BlockProposer {
	return c.blockProposer
}

func (c *CoreEngine) GetBlockCommitter() protocol.BlockCommitter {
	return c.BlockCommitter
}

func (c *CoreEngine) GetBlockVerifier() protocol.BlockVerifier {
	return c.BlockVerifier
}

func (c *CoreEngine) GetMaxbftHelper() protocol.MaxbftHelper {
	return c.MaxbftHelper
}
