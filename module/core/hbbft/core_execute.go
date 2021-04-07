/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/core/hbbft/committer"
	"chainmaker.org/chainmaker-go/core/hbbft/packager"
	"chainmaker.org/chainmaker-go/core/hbbft/scheduler"
	"chainmaker.org/chainmaker-go/core/hbbft/verifier"
	"chainmaker.org/chainmaker-go/logger"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
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

	committer committer.Committer
	packager  packager.Packager
	scheduler scheduler.Scheduler
	verifier  verifier.Verifier
}

func (ce *CoreExecute) Package(txBatch []*commonpb.Transaction) error {
	//return packager.Packaged(txBatch []*commonpb.Transaction);
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

func (ce *CoreExecute) Commit() (*commonpb.BlockInfo, error) {
	return nil, nil
}
