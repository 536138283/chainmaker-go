package common

import (
	"encoding/hex"
	"testing"

	mbusmock "chainmaker.org/chainmaker/common/v2/msgbus/mock"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	consensusPb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
)

func TestCommitBlock_CommitBlock(t *testing.T) {

	ctl := gomock.NewController(t)
	log := logger.GetLoggerByChain(logger.MODULE_CORE, "chain1")
	block := createNewTestBlock(12)

	// snapshotManager
	snapshotManager := mock.NewMockSnapshotManager(ctl)
	snapshotManager.EXPECT().NotifyBlockCommitted(block).Return(nil)

	// 	ledgerCache
	ledgerCache := mock.NewMockLedgerCache(ctl)
	ledgerCache.EXPECT().SetLastCommittedBlock(block)

	// msgbus
	msgbus := mbusmock.NewMockMessageBus(ctl)
	//msgbus.EXPECT().PublishSafe(gomock.Any(), gomock.Any()).Return()

	//chainConf mock
	config := &config.ChainConfig{
		ChainId: "chain1",
		Crypto: &config.CryptoConfig{
			Hash: "SHA256",
		},
		Block: &config.BlockConfig{
			BlockTxCapacity: 1000,
			BlockSize:       1,
			BlockInterval:   DEFAULTDURATION,
		},
		Consensus: &config.ConsensusConfig{
			Type: consensusPb.ConsensusType_RAFT,
		},
	}
	chainConf := mock.NewMockChainConf(ctl)
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(config)

	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	tx0 := block.Txs[0]
	contractName := "testContract"
	txRWSetMap[tx0.Payload.TxId] = &commonpb.TxRWSet{
		TxId: tx0.Payload.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}

	// Mock blockChain Store
	store := mock.NewMockBlockchainStore(ctl)
	txRWSets := []*commonpb.TxRWSet{
		txRWSetMap[tx0.Payload.TxId],
	}
	log.Infof("init block(%d,%s)", block.Header.BlockHeight, hex.EncodeToString(block.Header.BlockHash))
	store.EXPECT().PutBlock(block, txRWSets).Return(nil)

	cbConf := &CommitBlockConf{
		Store:           store,
		Log:             log,
		SnapshotManager: snapshotManager,
		LedgerCache:     ledgerCache,
		ChainConf:       chainConf,
		MsgBus:          msgbus,
	}

	conEventMap := make(map[string][]*commonpb.ContractEvent)

	commiter := NewCommitBlock(cbConf)
	_, _, _, _, _, _, err := commiter.CommitBlock(block, txRWSetMap, conEventMap)
	if err != nil {
		panic(err)
	}
}

func createNewTestBlock(height uint64) *commonpb.Block {
	var hash = []byte("0123456789")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "Chain1",
			BlockHeight:    height,
			PreBlockHash:   hash,
			BlockHash:      hash,
			PreConfHeight:  0,
			BlockVersion:   1,
			DagHash:        hash,
			RwSetRoot:      hash,
			TxRoot:         hash,
			BlockTimestamp: 0,
			Proposer: &accesscontrol.Member{
				OrgId:      "org1",
				MemberType: 0,
				MemberInfo: nil,
			},
			TxCount:   0,
			Signature: nil,
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}
	tx := createNewTestTx("0123456789")
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	block.Txs = txs
	return block
}
