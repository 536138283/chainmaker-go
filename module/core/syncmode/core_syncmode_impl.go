/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package syncmode means commit new block in sync way
package syncmode

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker-go/module/core/common/scheduler"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker-go/module/core/syncmode/proposer"
	"chainmaker.org/chainmaker-go/module/core/syncmode/verifier"
	"chainmaker.org/chainmaker-go/module/subscriber"
	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/localconf/v3"
	commonpb "chainmaker.org/chainmaker/pb-go/v3/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v3/consensus"
	txpoolpb "chainmaker.org/chainmaker/pb-go/v3/txpool"
	"chainmaker.org/chainmaker/protocol/v3"
)

// CoreEngine is a block handle engine.
// One core engine for one chain.
//nolint: structcheck,unused
type CoreEngine struct {
	// chainId, identity of a chain
	chainId string
	// chain config
	chainConf protocol.ChainConf
	// message bus, transfer messages with other modules
	msgBus msgbus.MessageBus
	// block proposer, to generate new block when node is proposer
	blockProposer protocol.BlockProposer
	// block verifier, to verify block that proposer generated
	BlockVerifier protocol.BlockVerifier
	// block committer, to commit block to store after consensus
	BlockCommitter protocol.BlockCommitter
	// transaction scheduler, schedule transactions run in vm
	txScheduler protocol.TxScheduler
	// max bft helper
	MaxbftHelper protocol.MaxbftHelper
	// transaction pool, cache transactions to be pack in block
	txPool protocol.TxPool
	// vm manager
	vmMgr protocol.VmManager
	// blockchain store, to store block, transactions in DB
	blockchainStore protocol.BlockchainStore
	// snapshot manager, manage state data that not store yet
	snapshotManager protocol.SnapshotManager
	// quit chan, reserved for stop core engine running
	quitC <-chan interface{}
	// cache proposed block and proposal status
	proposedCache protocol.ProposalCache
	// logger
	log protocol.Logger
	// block subscriber
	subscriber *subscriber.EventSubscriber
	// net service
	netService protocol.NetService
}

// NewCoreEngine new a core engine.
func NewCoreEngine(cf *conf.CoreEngineConfig) (*CoreEngine, error) {
	core := &CoreEngine{
		msgBus:          cf.MsgBus,
		txPool:          cf.TxPool,
		vmMgr:           cf.VmMgr,
		blockchainStore: cf.BlockchainStore,
		snapshotManager: cf.SnapshotManager,
		proposedCache:   cf.ProposalCache,
		chainConf:       cf.ChainConf,
		log:             cf.Log,
		netService:      cf.NetService,
	}

	var schedulerFactory scheduler.TxSchedulerFactory
	// new tx scheduler to set the core engine
	core.txScheduler = schedulerFactory.NewTxScheduler(
		cf.VmMgr,
		cf.ChainConf,
		cf.StoreHelper,
		cf.LedgerCache)
	core.quitC = make(<-chan interface{})

	var err error
	// new a bock proposer
	proposerConfig := proposer.BlockProposerConfig{
		ChainId:         cf.ChainId,
		TxPool:          cf.TxPool,
		SnapshotManager: cf.SnapshotManager,
		MsgBus:          cf.MsgBus,
		Identity:        cf.Identity,
		LedgerCache:     cf.LedgerCache,
		TxScheduler:     core.txScheduler,
		ProposalCache:   cf.ProposalCache,
		ChainConf:       cf.ChainConf,
		AC:              cf.AC,
		BlockchainStore: cf.BlockchainStore,
		StoreHelper:     cf.StoreHelper,
		TxFilter:        cf.TxFilter,
	}
	// new block proposer to set the core engine
	core.blockProposer, err = proposer.NewBlockProposer(proposerConfig, cf.Log)
	if err != nil {
		return nil, err
	}

	// new a block verifier
	verifierConfig := verifier.BlockVerifierConfig{
		ChainId:         cf.ChainId,
		MsgBus:          cf.MsgBus,
		SnapshotManager: cf.SnapshotManager,
		BlockchainStore: cf.BlockchainStore,
		LedgerCache:     cf.LedgerCache,
		TxScheduler:     core.txScheduler,
		ProposedCache:   cf.ProposalCache,
		ChainConf:       cf.ChainConf,
		AC:              cf.AC,
		TxPool:          cf.TxPool,
		VmMgr:           cf.VmMgr,
		StoreHelper:     cf.StoreHelper,
		NetService:      cf.NetService,
		TxFilter:        cf.TxFilter,
	}
	// new block verifier to set the core engine
	core.BlockVerifier, err = verifier.NewBlockVerifier(verifierConfig, cf.Log)
	if err != nil {
		return nil, err
	}

	// new a block committer
	committerConfig := common.BlockCommitterConfig{
		ChainId:         cf.ChainId,
		BlockchainStore: cf.BlockchainStore,
		SnapshotManager: cf.SnapshotManager,
		TxPool:          cf.TxPool,
		LedgerCache:     cf.LedgerCache,
		ProposedCache:   cf.ProposalCache,
		ChainConf:       cf.ChainConf,
		MsgBus:          cf.MsgBus,
		Subscriber:      cf.Subscriber,
		Verifier:        core.BlockVerifier,
		StoreHelper:     cf.StoreHelper,
		TxFilter:        cf.TxFilter,
	}
	// new block committer to set the core engine
	core.BlockCommitter, err = common.NewBlockCommitter(committerConfig, cf.Log)
	if err != nil {
		return nil, err
	}

	// get the type of tx pool
	if value, ok := localconf.ChainMakerConfig.TxPoolConfig["pool_type"]; ok {
		common.TxPoolType, _ = value.(string)
		common.TxPoolType = strings.ToUpper(common.TxPoolType)
	}

	return core, nil
}

// OnQuit called when quit subsribe message from message bus
func (c *CoreEngine) OnQuit() {
	c.log.Info("on quit")
}

// OnMessage consume a message from message bus
func (c *CoreEngine) OnMessage(message *msgbus.Message) {
	// 1. receive proposal status from consensus
	// 2. receive verify block from consensus
	// 3. receive commit block message from consensus
	// 4. receive propose signal from txpool
	// 5. receive rw set verify fail txs from maxbft consensus

	switch message.Topic {
	case msgbus.ProposeState:
		if proposeStatus, ok := message.Payload.(bool); ok {
			c.blockProposer.OnReceiveProposeStatusChange(proposeStatus)
		}
	case msgbus.VerifyBlock:
		go func() {
			if block, ok := message.Payload.(*commonpb.Block); ok {
				c.BlockVerifier.VerifyBlock(block, protocol.CONSENSUS_VERIFY) //nolint: errcheck
			}
		}()
	case msgbus.CommitBlock:
		go func() {
			if block, ok := message.Payload.(*commonpb.Block); ok {
				if err := c.BlockCommitter.AddBlock(block); err != nil {
					c.log.Warnf("put block(%d,%x) error %s",
						block.Header.BlockHeight,
						block.Header.BlockHash,
						err.Error())
				}
			}
		}()
	case msgbus.TxPoolSignal:
		if signal, ok := message.Payload.(*txpoolpb.TxPoolSignal); ok {
			c.blockProposer.OnReceiveTxPoolSignal(signal)
		}
	case msgbus.RwSetVerifyFailTxs:
		if signal, ok := message.Payload.(*consensuspb.RwSetVerifyFailTxs); ok {
			c.log.DebugDynamic(func() string {
				return fmt.Sprintf("received consensus rw set verify fail txs block height:%d", signal.BlockHeight)
			})
			c.blockProposer.OnReceiveRwSetVerifyFailTxs(signal)
		}
	}
}

// Start initialize core engine
func (c *CoreEngine) Start() {
	// 1. register msgbus ProposeState
	// 2. register msgbus VerifyBlock
	// 3. register msgbus CommitBlock
	// 4. register msgbus TxPoolSignal
	// 5. register msgbus RwSetVerifyFailTxs
	c.msgBus.Register(msgbus.ProposeState, c)
	c.msgBus.Register(msgbus.VerifyBlock, c)
	c.msgBus.Register(msgbus.CommitBlock, c)
	c.msgBus.Register(msgbus.TxPoolSignal, c)
	c.msgBus.Register(msgbus.RwSetVerifyFailTxs, c)
	//c.msgBus.Register(msgbus.BuildProposal, c)
	c.blockProposer.Start() //nolint: errcheck
}

// Stop core engine
func (c *CoreEngine) Stop() {
	defer c.log.Infof("core stopped.")
	c.blockProposer.Stop() //nolint: errcheck
}

// GetBlockProposer get block proposer
func (c *CoreEngine) GetBlockProposer() protocol.BlockProposer {
	return c.blockProposer
}

// GetBlockCommitter get block committer
func (c *CoreEngine) GetBlockCommitter() protocol.BlockCommitter {
	return c.BlockCommitter
}

// GetBlockVerifier get block verifier
func (c *CoreEngine) GetBlockVerifier() protocol.BlockVerifier {
	return c.BlockVerifier
}

// GetMaxbftHelper get max bft helper
func (c *CoreEngine) GetMaxbftHelper() protocol.MaxbftHelper {
	return c.MaxbftHelper
}
