/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package conf

import (
	"chainmaker.org/chainmaker-go/module/core/cache"
	"chainmaker.org/chainmaker-go/module/subscriber"
	"chainmaker.org/chainmaker/common/v3/msgbus"
	commonpb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/protocol/v3"
)

// CoreEngineConfig core engine config struct
type CoreEngineConfig struct {
	// chain id
	ChainId string
	// tx pool
	TxPool protocol.TxPool
	// snapshot manager
	SnapshotManager protocol.SnapshotManager
	// message bus
	MsgBus msgbus.MessageBus
	// sininging member
	Identity protocol.SigningMember
	// ledger cache
	LedgerCache protocol.LedgerCache
	// proposal cache
	ProposalCache protocol.ProposalCache
	// chain config
	ChainConf protocol.ChainConf
	// access control provider
	AC protocol.AccessControlProvider
	// block chain store
	BlockchainStore protocol.BlockchainStore
	// logger
	Log protocol.Logger
	// vm manager
	VmMgr protocol.VmManager
	// block subscriber
	Subscriber *subscriber.EventSubscriber
	// store helper
	StoreHelper StoreHelper
	// net service
	NetService protocol.NetService
	// tx filter
	TxFilter protocol.TxFilter
	// abft cache
	ABFTCache *cache.AbftCache
}

// StoreHelper store helper interface
type StoreHelper interface {
	// RollBack roll back func return error
	RollBack(*commonpb.Block, protocol.BlockchainStore) error
	// BeginDbTransaction begin db transaction
	BeginDbTransaction(protocol.BlockchainStore, string)
	// GetPoolCapacity get pool capacity
	GetPoolCapacity() int
}
