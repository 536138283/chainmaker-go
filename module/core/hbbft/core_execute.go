/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/hbbft"
	"chainmaker.org/chainmaker-go/protocol"
	"encoding/hex"
)

type CoreExecute struct {
	chainId         string // chain id, to identity this chain
	hbbftCache      *cache.HbbftCache
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
	Packager  *Packager
	Verifier  *Verifier
}

func NewCoreExecute(ceConfig *core.CoreExecuteConfig) *CoreExecute {
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
	ce.hbbftCache = cache.NewHbbftCache()
	ce.Packager = NewPackager(ce)
	ce.Verifier = NewVerifier(ce)
	ce.Committer = NewCommitter(ce, ce.Packager)
	return ce
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
		c.Packager.Package()
	case msgbus.VerifyBlock:
		if block, ok := message.Payload.(commonPb.Block); ok {
			c.Verifier.verifier(&block)
		}
	case msgbus.CommitedTxBatchs:
		if txBatchAfterABA, ok := message.Payload.(hbbft.TxBatchAfterABA); ok {
			ok, err := c.Committer.verifyHeight(txBatchAfterABA.BlockHeight)
			if !ok {
				c.log.Errorf("after ABA the tx batch height is wrong: %s, height: (%d)", err.Error(), txBatchAfterABA.BlockHeight)
				return
			}
			c.Committer.blockHeight = txBatchAfterABA.BlockHeight
			for i, _ := range txBatchAfterABA.TxBatchHash {
				c.Committer.getConfirmedBranchInfo(txBatchAfterABA.TxBatchHash[i])                                              // branchInfo After ABA
				c.Committer.branchIDList = append(c.Committer.branchIDList, hex.EncodeToString(txBatchAfterABA.TxBatchHash[i])) // branchIDList After ABA
			}

			if err := c.Committer.Commit(); err != nil {
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
