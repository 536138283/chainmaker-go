/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package committer

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/consensus-maxbft/v2/epoch"
	consensusUtils "chainmaker.org/chainmaker/consensus-maxbft/v2/utils"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"
	systemPb "chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"

	"github.com/gogo/protobuf/proto"
)

type BLockCommitter struct {
	conf     protocol.ChainConf
	msgBus   msgbus.MessageBus
	store    protocol.BlockchainStore
	delegate protocol.BlockCommitter

	epochStrategy string
}

func NewBLockCommitter(delegate protocol.BlockCommitter,
	conf protocol.ChainConf, msgBus msgbus.MessageBus, store protocol.BlockchainStore) (*BLockCommitter, error) {

	bc := &BLockCommitter{delegate: delegate, conf: conf, msgBus: msgBus, store: store}
	strategy, _, err := epoch.GetEpochStrategyFromConfig(conf.ChainConfig())
	if err != nil {
		return nil, err
	}
	bc.epochStrategy = strategy
	return bc, nil
}

func (bc *BLockCommitter) AddBlock(blk *commonPb.Block) error {
	err := bc.delegate.AddBlock(blk)
	if err != nil {
		return err
	}

	// 如果是nilStrategy策略，表示不会再更新世代合约；
	// 为了在其它模块屏蔽epoch的不同策略，所以在此处理epoch的不同策略
	if bc.epochStrategy == epoch.NilStrategy {
		if utils.IsConfBlock(blk) {
			// publish governance contract
			cfg, err := consensusUtils.GetChainConfigFromChainStore(bc.store)
			if err != nil {
				return err
			}
			contract := &maxbft.GovernanceContract{
				ChainConfig: cfg,
			}
			bc.msgBus.PublishSafe(msgbus.MaxbftEpochConf, contract)
		}
		return nil
	}

	// 其它策略模式下，当且仅当区块头的ConsensusArgs包含数据时，表示存在世代合约内容.
	// 通知其它模块进行链配置更新
	if len(blk.Header.ConsensusArgs) == 0 {
		return nil
	}
	governance, err := bc.getGovernanceFromBlock(blk)
	if err != nil {
		err = fmt.Errorf("get governance from block failed. error: %+v", err)
		return err
	}
	if governance != nil {
		bc.msgBus.PublishSafe(msgbus.MaxbftEpochConf, governance)
	}
	return nil
}

func (bc *BLockCommitter) getGovernanceFromBlock(block *commonPb.Block) (*maxbft.GovernanceContract, error) {
	var (
		err  error
		args = new(consensus.BlockHeaderConsensusArgs)
	)
	if err = proto.Unmarshal(block.Header.ConsensusArgs, args); err != nil {
		err = fmt.Errorf("unmarshal consensus args failed. error:%+v", err)
		return nil, err
	}

	// get the governanceContract from the txWrite
	contractName := systemPb.SystemContract_GOVERNANCE.String()
	if args.ConsensusData == nil || len(args.ConsensusData.TxWrites) == 0 ||
		args.ConsensusData.TxWrites[0].ContractName != contractName {
		// there is no governance contract information in the block, need not to switch epoch
		return nil, nil
	}

	// get governance contract from the block, to get the configurations of the next epoch
	governanceContract := new(maxbft.GovernanceContract)
	if err = proto.Unmarshal(args.ConsensusData.TxWrites[0].GetValue(), governanceContract); err != nil {
		err = fmt.Errorf("unmarshal txWrites value failed. error:%+v", err)
		return nil, err
	}
	return governanceContract, nil
}
