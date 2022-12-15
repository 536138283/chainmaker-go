/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package committer

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/consensus-maxbft/v3/epoch"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	"chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/pb-go/v3/consensus/maxbft"
	systemPb "chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
	"github.com/gogo/protobuf/proto"
)

// BlockCommitter Decorator for the maxBft block commit feature
type BlockCommitter struct {
	conf     protocol.ChainConf
	msgBus   msgbus.MessageBus
	store    protocol.BlockchainStore
	delegate protocol.BlockCommitter

	epochStrategy string
}

// NewBLockCommitter new feature
func NewBLockCommitter(delegate protocol.BlockCommitter,
	conf protocol.ChainConf, msgBus msgbus.MessageBus, store protocol.BlockchainStore) (*BlockCommitter, error) {

	bc := &BlockCommitter{delegate: delegate, conf: conf, msgBus: msgBus, store: store}
	strategy, _, err := epoch.GetEpochStrategyFromConfig(conf.ChainConfig())
	if err != nil {
		return nil, err
	}
	bc.epochStrategy = strategy
	return bc, nil
}

// AddBlock add block to db and do some things with maxbft
func (bc *BlockCommitter) AddBlock(blk *commonPb.Block) error {
	err := bc.delegate.AddBlock(blk)
	if err != nil {
		return err
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

func (bc *BlockCommitter) getGovernanceFromBlock(block *commonPb.Block) (*maxbft.GovernanceContract, error) {
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
