package cache

import (
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/stretchr/testify/require"

	//protocol "chainmaker.org/chainmaker/protocol/v2"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
)

func TestProposalCache_ClearTheBlock(t *testing.T) {

	ctl := gomock.NewController(t)
	chainConf := mock.NewMockChainConf(ctl)
	ledgerCache := mock.NewMockLedgerCache(ctl)
	ledgerCache.EXPECT().CurrentHeight().Return(uint64(0), nil)

	proposalCache := NewProposalCache(chainConf, ledgerCache)

	rwSetMap := make(map[string]*commonpb.TxRWSet)
	contractEvenMap := make(map[string][]*commonpb.ContractEvent)

	hash := []byte("123")
	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockVersion:   1,
			BlockType:      0,
			ChainId:        "chain1",
			BlockHeight:    0,
			BlockHash:      hash,
			PreBlockHash:   nil,
			PreConfHeight:  0,
			TxCount:        0,
			TxRoot:         nil,
			DagHash:        nil,
			RwSetRoot:      nil,
			BlockTimestamp: 0,
			ConsensusArgs:  nil,
			Proposer:       nil,
			Signature:      nil,
		},
		Dag:            nil,
		Txs:            nil,
		AdditionalData: nil,
	}

	err := proposalCache.SetProposedBlock(block, rwSetMap, contractEvenMap, false)
	require.Nil(t, err)

	b0 := proposalCache.GetProposedBlocksAt(0)
	require.NotNil(t, b0)

	b1 := proposalCache.GetProposedBlocksAt(1)
	require.Nil(t, b1)

	b2, txRWSet := proposalCache.GetProposedBlockByHashAndHeight(hash, 0)
	require.NotNil(t, b2)
	require.NotNil(t, txRWSet)

	b3 := proposalCache.GetSelfProposedBlockAt(0)
	require.Nil(t, b3)

	proposalCache.ClearProposedBlockAt(0)

	b2_1, txRWSet_2 := proposalCache.GetProposedBlockByHashAndHeight(hash, 0)
	require.Nil(t, b2_1)
	require.Nil(t, txRWSet_2)

	b0_1 := proposalCache.GetProposedBlocksAt(0)
	require.Nil(t, b0_1)

}

func TestProposalCache_ClearProposedBlockAt(t *testing.T) {

	ctl := gomock.NewController(t)
	chainConf := mock.NewMockChainConf(ctl)
	ledgerCache := mock.NewMockLedgerCache(ctl)
	ledgerCache.EXPECT().CurrentHeight().Return(uint64(0), nil)

	proposalCache := NewProposalCache(chainConf, ledgerCache)

	rwSetMap := make(map[string]*commonpb.TxRWSet)
	contractEvenMap := make(map[string][]*commonpb.ContractEvent)

	hash := []byte("123")
	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockVersion:   1,
			BlockType:      0,
			ChainId:        "chain1",
			BlockHeight:    0,
			BlockHash:      hash,
			PreBlockHash:   nil,
			PreConfHeight:  0,
			TxCount:        0,
			TxRoot:         nil,
			DagHash:        nil,
			RwSetRoot:      nil,
			BlockTimestamp: 0,
			ConsensusArgs:  nil,
			Proposer:       nil,
			Signature:      nil,
		},
		Dag:            nil,
		Txs:            nil,
		AdditionalData: nil,
	}

	err := proposalCache.SetProposedBlock(block, rwSetMap, contractEvenMap, true)
	require.Nil(t, err)

	require.Equal(t, proposalCache.IsProposedAt(block.Header.BlockHeight), true)

	b0 := proposalCache.GetSelfProposedBlockAt(block.Header.BlockHeight)
	require.NotNil(t, b0)

	b1 := proposalCache.GetSelfProposedBlockAt(block.Header.BlockHeight + 1)
	require.Nil(t, b1)

	require.Equal(t, proposalCache.HasProposedBlockAt(block.Header.BlockHeight), true)

	//proposalCache.ResetProposedAt(block.Header.BlockHeight)
	//
	//b2 := proposalCache.GetSelfProposedBlockAt(block.Header.BlockHeight)
	//require.Nil(t, b2)
	//
	//proposalCache.SetProposedAt(block.Header.BlockHeight)
	//b3 := proposalCache.GetSelfProposedBlockAt(block.Header.BlockHeight)
	//require.Nil(t, b3)

	//proposalCache.ClearProposedBlockAt(0)
	//
	//b2_1, txRWSet_2 := proposalCache.GetProposedBlockByHashAndHeight(hash, 0)
	//require.Nil(t, b2_1)
	//require.Nil(t, txRWSet_2)
	//
	//b0_1 := proposalCache.GetProposedBlocksAt(0)
	//require.Nil(t, b0_1)

}
