/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package helper

import (
	"chainmaker.org/chainmaker-go/module/core/common"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	batch "chainmaker.org/chainmaker/txpool-batch/v2"
)

// maxBftHelper max bft heleper
type maxBftHelper struct {
	// tx pool used by maxBftHelper
	txPool protocol.TxPool
	// chain config used by maxBftHelper
	chainConf protocol.ChainConf
	// proposal cache used by maxBftHelper
	proposalCache protocol.ProposalCache
	logger        protocol.Logger
}

// NewMaxbftHelper new max bft helper, return NewMaxbftHelper
func NewMaxbftHelper(txPool protocol.TxPool, chainConf protocol.ChainConf,
	proposalCache protocol.ProposalCache, log protocol.Logger) protocol.MaxbftHelper {
	return &maxBftHelper{
		txPool:        txPool,
		chainConf:     chainConf,
		proposalCache: proposalCache,
		logger:        log}
}

// DiscardBlocks discard blocks
func (hp *maxBftHelper) DiscardBlocks(baseHeight uint64) {
	// only deal with consensus type equal max bft

	if hp.chainConf.ChainConfig().Consensus.Type != consensusPb.ConsensusType_MAXBFT {
		return
	}

	// discard the block when height > baseHeight, delete the block in lastProposedBlock at the height
	delBlocks := hp.proposalCache.DiscardBlocks(baseHeight)
	if len(delBlocks) == 0 {
		return
	}

	if common.TxPoolType == batch.TxPoolType {
		for _, delBlock := range delBlocks {
			batchIds, _, err := common.GetBatchIds(delBlock)
			if err != nil {
				// if get batch ids fail,discard other blocks.
				hp.logger.Warnf("get batch ids from block[%d,%x] failed, err:%v",
					delBlock.Header.BlockHeight, delBlock.Header.BlockHash, err)
				continue
			}
			hp.txPool.RetryTxBatches(batchIds)
		}
		return
	}

	txs := make([]*commonpb.Transaction, 0, 100)
	for _, blk := range delBlocks {
		txs = append(txs, blk.Txs...)
	}

	hp.txPool.RetryTxs(txs)
}
