/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"fmt"
	"sync"
	"testing"

	"chainmaker.org/chainmaker-go/module/core/cache"
	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	configpb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var hashType = "SHA256"

func TestBlockVerifierImpl_VerifyBlock(t *testing.T) {
	ctl := gomock.NewController(t)
	var chainId = "Chain1"

	msgBus := msgbus.NewMessageBus()
	txScheduler := mock.NewMockTxScheduler(ctl)
	snapshotMgr := mock.NewMockSnapshotManager(ctl)
	ledgerCache := cache.NewLedgerCache(chainId)
	blockchainStoreImpl := mock.NewMockBlockchainStore(ctl)
	proposedCache := cache.NewProposalCache(mock.NewMockChainConf(ctl), ledgerCache)
	chainConf := mock.NewMockChainConf(ctl)
	ac := mock.NewMockAccessControlProvider(ctl)
	txpool := mock.NewMockTxPool(ctl)
	netService := mock.NewMockNetService(ctl)

	tx := createNewTestTx()
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	rwSetmap := make(map[string]*commonpb.TxRWSet)
	rwSetmap[tx.Payload.TxId] = &commonpb.TxRWSet{
		TxId:     tx.Payload.TxId,
		TxReads:  nil,
		TxWrites: nil,
	}

	txList := make(map[string]*commonpb.Transaction)
	txList[tx.Payload.TxId] = tx
	heights := make(map[string]uint64)
	heights[tx.Payload.TxId] = 1
	txRwSetTable := make([]*commonpb.TxRWSet, 0)
	txRwSetTable = append(txRwSetTable, &commonpb.TxRWSet{
		TxId:     tx.Payload.TxId,
		TxReads:  nil,
		TxWrites: nil,
	})

	var err error
	tx.Result.RwSetHash, err = utils.CalcRWSetHash(hashType, rwSetmap[tx.Payload.TxId])
	require.Nil(t, err)

	txHash, err := utils.CalcTxHash(hashType, tx)
	require.Nil(t, err)

	b0 := createNewTestBlockWithoutProposer(0)
	ledgerCache.SetLastCommittedBlock(b0)
	b1 := createNewTestBlock(1, &accesscontrol.Member{
		OrgId:      "org1",
		MemberType: 0,
		MemberInfo: []byte("1234567890"),
	}, txs)

	txHashs := make([][]byte, 0)
	txHashs = append(txHashs, txHash)
	txRoot, err := hash.GetMerkleRoot(hashType, txHashs)
	require.Nil(t, err)
	b1.Header.TxRoot = txRoot

	dagHash, err := utils.CalcDagHash(hashType, b1.Dag)
	require.Nil(t, err)
	b1.Header.DagHash = dagHash

	rwSetRoot, err := utils.CalcRWSetRoot(hashType, txs)
	require.Nil(t, err)
	b1.Header.RwSetRoot = rwSetRoot

	blockHash, err := utils.CalcBlockHash("SHA256", b1)
	require.Nil(t, err)
	b1.Header.BlockHash = blockHash

	member := mock.NewMockMember(ctl)
	member.EXPECT().GetMemberId().Return("123").AnyTimes()
	ac.EXPECT().NewMember(b1.Header.Proposer).Return(member, nil)

	txpool.EXPECT().GetTxsByTxIds(gomock.Any()).Return(txList, nil).AnyTimes()
	txpool.EXPECT().GetAllTxsByTxIds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(txList, nil).AnyTimes()
	txpool.EXPECT().AddTxsToPendingCache(gomock.Any(), gomock.Any()).AnyTimes()
	txResultMap := make(map[string]*commonpb.Result)
	txResultMap[tx.Payload.TxId] = tx.Result

	snapshot := mock.NewMockSnapshot(ctl)
	snapshot.EXPECT().GetBlockchainStore().AnyTimes()
	snapshot.EXPECT().Seal().AnyTimes()
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRwSetTable)
	snapshot.EXPECT().GetTxResultMap().AnyTimes().Return(txResultMap)

	netService.EXPECT().GetNodeUidByCertId(gomock.Any()).Return("123", nil)

	snapshotMgr.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes().Return(snapshot)
	blockchainStoreImpl.EXPECT().BeginDbTransaction(gomock.Any()).AnyTimes()
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	ac.EXPECT().VerifyPrincipal(gomock.Any()).Return(true, nil).AnyTimes()
	txScheduler.EXPECT().SimulateWithDag(gomock.Any(), gomock.Any()).Return(rwSetmap, txResultMap, nil).AnyTimes()

	consensus := configpb.ConsensusConfig{
		Type: consensus.ConsensusType_TBFT,
	}
	block := configpb.BlockConfig{
		TxTimestampVerify: false,
		TxTimeout:         1000000000,
		BlockTxCapacity:   100,
		BlockSize:         100000,
		BlockInterval:     1000,
	}
	crypro := configpb.CryptoConfig{Hash: hashType}
	contract := configpb.ContractConfig{EnableSqlSupport: false}
	chainConfig := configpb.ChainConfig{
		Consensus: &consensus,
		Block:     &block,
		Contract:  &contract,
		Crypto:    &crypro,
		Core: &configpb.CoreConfig{
			TxSchedulerTimeout:         0,
			TxSchedulerValidateTimeout: 0,
			ConsensusTurboConfig: &configpb.ConsensusTurboConfig{
				ConsensusMessageTurbo: false,
				RetryTime:             0,
				RetryInterval:         0,
			},
		}}
	chainConf.EXPECT().ChainConfig().Return(&chainConfig).AnyTimes()

	verifier := &BlockVerifierImpl{
		chainId:         chainId,
		msgBus:          msgBus,
		txScheduler:     txScheduler,
		snapshotManager: snapshotMgr,
		ledgerCache:     ledgerCache,
		blockchainStore: blockchainStoreImpl,
		reentrantLocks: &common.ReentrantLocks{
			ReentrantLocks: make(map[string]interface{}),
		},
		proposalCache:         proposedCache,
		chainConf:             chainConf,
		ac:                    ac,
		log:                   logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
		txPool:                txpool,
		mu:                    sync.Mutex{},
		verifierBlock:         nil,
		storeHelper:           common.NewKVStoreHelper(chainId),
		metricBlockVerifyTime: nil,
		netService:            netService,
	}

	conf := &common.VerifierBlockConf{
		ChainConf:       verifier.chainConf,
		Log:             verifier.log,
		LedgerCache:     verifier.ledgerCache,
		Ac:              verifier.ac,
		SnapshotManager: verifier.snapshotManager,
		TxPool:          verifier.txPool,
		BlockchainStore: verifier.blockchainStore,
		ProposalCache:   verifier.proposalCache,
		VmMgr:           nil,
		StoreHelper:     verifier.storeHelper,
		TxScheduler:     verifier.txScheduler,
	}

	conf.ChainConf.ChainConfig().AuthType = protocol.Identity
	verifier.verifierBlock = common.NewVerifierBlock(conf)

	err = verifier.VerifyBlock(b1, protocol.CONSENSUS_VERIFY)
	require.Nil(t, err)
}

func TestBlockVerifierImpl_VerifyBlockWithRwSets(t *testing.T) {
	ctl := gomock.NewController(t)
	var chainId = "Chain1"

	msgBus := msgbus.NewMessageBus()
	txScheduler := mock.NewMockTxScheduler(ctl)
	snapshotMgr := mock.NewMockSnapshotManager(ctl)
	ledgerCache := cache.NewLedgerCache(chainId)
	blockchainStoreImpl := mock.NewMockBlockchainStore(ctl)
	proposedCache := cache.NewProposalCache(mock.NewMockChainConf(ctl), ledgerCache)
	chainConf := mock.NewMockChainConf(ctl)
	ac := mock.NewMockAccessControlProvider(ctl)
	txpool := mock.NewMockTxPool(ctl)
	netService := mock.NewMockNetService(ctl)

	tx := createNewTestTx()
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	rwSetmap := make(map[string]*commonpb.TxRWSet)
	rwSetmap[tx.Payload.TxId] = &commonpb.TxRWSet{
		TxId:     tx.Payload.TxId,
		TxReads:  nil,
		TxWrites: nil,
	}

	txList := make(map[string]*commonpb.Transaction)
	txList[tx.Payload.TxId] = tx
	heights := make(map[string]uint64)
	heights[tx.Payload.TxId] = 1
	txRwSetTable := make([]*commonpb.TxRWSet, 0)
	txRwSetTable = append(txRwSetTable, &commonpb.TxRWSet{
		TxId:     tx.Payload.TxId,
		TxReads:  nil,
		TxWrites: nil,
	})

	var err error
	tx.Result.RwSetHash, err = utils.CalcRWSetHash(hashType, rwSetmap[tx.Payload.TxId])
	require.Nil(t, err)

	txHash, err := utils.CalcTxHash(hashType, tx)
	require.Nil(t, err)

	b0 := createNewTestBlockWithoutProposer(0)
	ledgerCache.SetLastCommittedBlock(b0)
	b1 := createNewTestBlock(1, &accesscontrol.Member{
		OrgId:      "org1",
		MemberType: 0,
		MemberInfo: []byte("1234567890"),
	}, txs)

	txHashs := make([][]byte, 0)
	txHashs = append(txHashs, txHash)
	txRoot, err := hash.GetMerkleRoot(hashType, txHashs)
	require.Nil(t, err)
	b1.Header.TxRoot = txRoot

	dagHash, err := utils.CalcDagHash(hashType, b1.Dag)
	require.Nil(t, err)
	b1.Header.DagHash = dagHash

	rwSetRoot, err := utils.CalcRWSetRoot(hashType, txs)
	require.Nil(t, err)
	b1.Header.RwSetRoot = rwSetRoot

	blockHash, err := utils.CalcBlockHash("SHA256", b1)
	require.Nil(t, err)
	b1.Header.BlockHash = blockHash

	rwSet := make([]*commonpb.TxRWSet, 0)
	rwSet = append(rwSet, &commonpb.TxRWSet{
		TxId:     tx.Payload.TxId,
		TxReads:  nil,
		TxWrites: nil,
	})

	member := mock.NewMockMember(ctl)
	member.EXPECT().GetMemberId().Return("123").AnyTimes()
	ac.EXPECT().NewMember(b1.Header.Proposer).Return(member, nil)

	txpool.EXPECT().GetTxsByTxIds(gomock.Any()).Return(txList, nil).AnyTimes()
	txpool.EXPECT().GetAllTxsByTxIds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(txList, nil).AnyTimes()
	txpool.EXPECT().AddTxsToPendingCache(gomock.Any(), gomock.Any()).AnyTimes()
	txResultMap := make(map[string]*commonpb.Result)
	txResultMap[tx.Payload.TxId] = tx.Result

	snapshot := mock.NewMockSnapshot(ctl)
	snapshot.EXPECT().GetBlockchainStore().AnyTimes()
	snapshot.EXPECT().Seal().AnyTimes()
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRwSetTable)
	snapshot.EXPECT().GetTxResultMap().AnyTimes().Return(txResultMap)
	netService.EXPECT().GetNodeUidByCertId(gomock.Any()).Return("123", nil)

	snapshotMgr.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes().Return(snapshot)
	blockchainStoreImpl.EXPECT().BeginDbTransaction(gomock.Any()).AnyTimes()
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	ac.EXPECT().VerifyPrincipal(gomock.Any()).Return(true, nil).AnyTimes()
	txScheduler.EXPECT().SimulateWithDag(gomock.Any(), gomock.Any()).Return(rwSetmap, txResultMap, nil).AnyTimes()

	consensus := configpb.ConsensusConfig{
		Type: consensus.ConsensusType_TBFT,
	}
	block := configpb.BlockConfig{
		TxTimestampVerify: false,
		TxTimeout:         1000000000,
		BlockTxCapacity:   100,
		BlockSize:         100000,
		BlockInterval:     1000,
	}
	crypro := configpb.CryptoConfig{Hash: hashType}
	contract := configpb.ContractConfig{EnableSqlSupport: false}
	chainConfig := configpb.ChainConfig{
		Consensus: &consensus,
		Block:     &block,
		Contract:  &contract,
		Crypto:    &crypro,
		Core: &configpb.CoreConfig{
			TxSchedulerTimeout:         0,
			TxSchedulerValidateTimeout: 0,
			ConsensusTurboConfig: &configpb.ConsensusTurboConfig{
				ConsensusMessageTurbo: false,
				RetryTime:             0,
				RetryInterval:         0,
			},
		}}
	chainConf.EXPECT().ChainConfig().Return(&chainConfig).AnyTimes()

	verifier := &BlockVerifierImpl{
		chainId:         chainId,
		msgBus:          msgBus,
		txScheduler:     txScheduler,
		snapshotManager: snapshotMgr,
		ledgerCache:     ledgerCache,
		blockchainStore: blockchainStoreImpl,
		reentrantLocks: &common.ReentrantLocks{
			ReentrantLocks: make(map[string]interface{}),
		},
		proposalCache:         proposedCache,
		chainConf:             chainConf,
		ac:                    ac,
		log:                   logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
		txPool:                txpool,
		mu:                    sync.Mutex{},
		verifierBlock:         nil,
		storeHelper:           common.NewKVStoreHelper(chainId),
		metricBlockVerifyTime: nil,
		netService:            netService,
	}

	conf := &common.VerifierBlockConf{
		ChainConf:       verifier.chainConf,
		Log:             verifier.log,
		LedgerCache:     verifier.ledgerCache,
		Ac:              verifier.ac,
		SnapshotManager: verifier.snapshotManager,
		TxPool:          verifier.txPool,
		BlockchainStore: verifier.blockchainStore,
		ProposalCache:   verifier.proposalCache,
		VmMgr:           nil,
		StoreHelper:     verifier.storeHelper,
		TxScheduler:     verifier.txScheduler,
	}

	conf.ChainConf.ChainConfig().AuthType = protocol.Identity
	verifier.verifierBlock = common.NewVerifierBlock(conf)

	err = verifier.VerifyBlockWithRwSets(b1, rwSet, protocol.CONSENSUS_VERIFY)
	require.Nil(t, err)
}

func Test_DispatchTask(t *testing.T) {
	tasks := utils.DispatchTxVerifyTask(nil)
	fmt.Println(tasks)
	txs := make([]*commonpb.Transaction, 0)
	for i := 0; i < 5; i++ {
		txs = append(txs, createNewTestTx())
	}
	require.Equal(t, 5, len(txs))
	verifyTasks := utils.DispatchTxVerifyTask(txs)
	fmt.Println(len(verifyTasks))
	for i := 0; i < len(verifyTasks); i++ {
		fmt.Printf("%v \n", verifyTasks[i])
	}

	for i := 0; i < 123; i++ {
		txs = append(txs, createNewTestTx())
	}
	verifyTasks = utils.DispatchTxVerifyTask(txs)
	fmt.Println(len(verifyTasks))
	for i := 0; i < len(verifyTasks); i++ {
		fmt.Printf("%v \n", verifyTasks[i])
	}

	for i := 0; i < 896; i++ {
		txs = append(txs, createNewTestTx())
	}
	verifyTasks = utils.DispatchTxVerifyTask(txs)
	fmt.Println(len(verifyTasks))
	for i := 0; i < len(verifyTasks); i++ {
		fmt.Printf("%v \n", verifyTasks[i])
	}
}

func createNewTestBlock(height uint64, proposer *accesscontrol.Member, txs []*commonpb.Transaction) *commonpb.Block {
	var hash = []byte("0123456789")

	dag := &commonpb.DAG{Vertexes: []*commonpb.DAG_Neighbor{}}
	neighbor := make([]*commonpb.DAG_Neighbor, 0)
	neighbor = append(neighbor, &commonpb.DAG_Neighbor{Neighbors: []uint32{0}})
	dag.Vertexes = neighbor

	return &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockVersion:   0,
			BlockType:      0,
			ChainId:        "Chain1",
			BlockHeight:    height,
			BlockHash:      hash,
			PreBlockHash:   hash,
			PreConfHeight:  0,
			TxCount:        1,
			TxRoot:         hash,
			DagHash:        hash,
			RwSetRoot:      hash,
			BlockTimestamp: 0,
			ConsensusArgs:  nil,
			Proposer:       proposer,
			Signature:      hash,
		},
		Dag:            dag,
		Txs:            txs,
		AdditionalData: nil,
	}
}

func createNewTestTx() *commonpb.Transaction {
	return &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId:        "",
			TxType:         0,
			TxId:           "",
			Timestamp:      0,
			ExpirationTime: 0,
			ContractName:   "",
			Method:         "",
			Parameters:     nil,
			Sequence:       0,
			Limit:          nil,
		},
		Sender:    nil,
		Endorsers: nil,
		Result: &commonpb.Result{
			Code: commonpb.TxStatusCode_CONTRACT_REVOKE_FAILED,
			ContractResult: &commonpb.ContractResult{
				Code:          0,
				Result:        nil,
				Message:       "",
				GasUsed:       0,
				ContractEvent: nil,
			},
			RwSetHash: nil,
		},
	}

}

func createNewTestBlockWithoutProposer(height uint64) *commonpb.Block {
	var hash = []byte("0123456789")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "Chain1",
			BlockHeight:    height,
			PreBlockHash:   hash,
			BlockHash:      hash,
			PreConfHeight:  0,
			BlockVersion:   0,
			DagHash:        hash,
			RwSetRoot:      hash,
			TxRoot:         hash,
			BlockTimestamp: 0,
			Proposer: &accesscontrol.Member{
				OrgId:      "org1",
				MemberType: 0,
				MemberInfo: hash,
			},
			ConsensusArgs: nil,
			TxCount:       1,
			Signature:     []byte(""),
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}
	tx := createNewTestTx()
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	block.Txs = txs
	return block
}
