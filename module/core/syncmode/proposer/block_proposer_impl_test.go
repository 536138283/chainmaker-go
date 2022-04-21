/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proposer

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/module/core/cache"
	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker/common/v2/json"
	mbusmock "chainmaker.org/chainmaker/common/v2/msgbus/mock"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	configpb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	txpoolpb "chainmaker.org/chainmaker/pb-go/v2/txpool"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"chainmaker.org/chainmaker/utils/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	chainId      = "Chain1"
	contractName = "contractName"
)

func TestProposeStatusChange(t *testing.T) {
	ctl := gomock.NewController(t)
	txPool := mock.NewMockTxPool(ctl)
	snapshotMgr := mock.NewMockSnapshotManager(ctl)
	msgBus := mbusmock.NewMockMessageBus(ctl)
	msgBus.EXPECT().Register(gomock.Any(), gomock.Any()).AnyTimes()
	identity := mock.NewMockSigningMember(ctl)
	ledgerCache := cache.NewLedgerCache(chainId)
	proposedCache := cache.NewProposalCache(nil, ledgerCache)
	txScheduler := mock.NewMockTxScheduler(ctl)
	blockChainStore := mock.NewMockBlockchainStore(ctl)
	chainConf := mock.NewMockChainConf(ctl)
	storeHelper := common.NewKVStoreHelper("chain1")

	ledgerCache.SetLastCommittedBlock(createNewTestBlock(0))

	txs := make([]*commonpb.Transaction, 0)

	for i := 0; i < 5; i++ {
		txs = append(txs, createNewTestTx("txId"+fmt.Sprint(i)))
		blockChainStore.EXPECT().TxExists("txId" + fmt.Sprint(i)).AnyTimes()
	}

	identity.EXPECT().GetMember().AnyTimes()
	txPool.EXPECT().FetchTxBatch(gomock.Any()).Return(txs).AnyTimes()
	txPool.EXPECT().GetPoolStatus().Return(&txpoolpb.TxPoolStatus{}).AnyTimes()
	//msgBus.EXPECT().Publish(gomock.Any(), gomock.Any())
	txPool.EXPECT().RetryAndRemoveTxs(gomock.Any(), gomock.Any())

	consensus := configpb.ConsensusConfig{
		Type: consensus.ConsensusType_TBFT,
	}
	blockConf := configpb.BlockConfig{
		TxTimestampVerify: false,
		TxTimeout:         1000000000,
		BlockTxCapacity:   100,
		BlockSize:         100000,
		BlockInterval:     1000,
	}
	crypro := configpb.CryptoConfig{Hash: "SHA256"}
	contract := configpb.ContractConfig{EnableSqlSupport: false}
	chainConfig := configpb.ChainConfig{
		Consensus: &consensus,
		Block:     &blockConf,
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
	snapshot := mock.NewMockSnapshot(ctl)
	snapshot.EXPECT().GetBlockchainStore().AnyTimes()
	snapshotMgr.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes().Return(snapshot)
	txScheduler.EXPECT().Schedule(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	logger := logger.GetLoggerByChain(logger.MODULE_CORE, chainId)
	blockBuilderConf := &common.BlockBuilderConf{
		ChainId:         "chain1",
		TxPool:          txPool,
		TxScheduler:     txScheduler,
		SnapshotManager: snapshotMgr,
		Identity:        identity,
		LedgerCache:     ledgerCache,
		ProposalCache:   proposedCache,
		ChainConf:       chainConf,
		Log:             logger,
		StoreHelper:     storeHelper,
	}
	blockBuilder := common.NewBlockBuilder(blockBuilderConf)

	blockProposer := &BlockProposerImpl{
		chainId:         chainId,
		isProposer:      false, // not proposer when initialized
		idle:            true,
		msgBus:          msgBus,
		canProposeC:     make(chan bool),
		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
		proposeTimer:    nil,
		exitC:           make(chan bool),
		txPool:          txPool,
		snapshotManager: snapshotMgr,
		txScheduler:     txScheduler,
		identity:        identity,
		ledgerCache:     ledgerCache,
		proposalCache:   proposedCache,
		log:             logger,
		finishProposeC:  make(chan bool),
		blockchainStore: blockChainStore,
		chainConf:       chainConf,

		blockBuilder: blockBuilder,
		storeHelper:  storeHelper,
	}
	require.False(t, blockProposer.isProposer)
	require.Nil(t, blockProposer.proposeTimer)

	blockProposer.proposeBlock()
	blockProposer.OnReceiveYieldProposeSignal(true)
}

func TestShouldPropose(t *testing.T) {
	ctl := gomock.NewController(t)
	txPool := mock.NewMockTxPool(ctl)
	snapshotMgr := mock.NewMockSnapshotManager(ctl)
	msgBus := mbusmock.NewMockMessageBus(ctl)
	identity := mock.NewMockSigningMember(ctl)
	ledgerCache := cache.NewLedgerCache(chainId)
	proposedCache := cache.NewProposalCache(nil, ledgerCache)
	txScheduler := mock.NewMockTxScheduler(ctl)

	ledgerCache.SetLastCommittedBlock(createNewTestBlock(0))
	blockProposer := &BlockProposerImpl{
		chainId:         chainId,
		isProposer:      false, // not proposer when initialized
		idle:            true,
		msgBus:          msgBus,
		canProposeC:     make(chan bool),
		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
		proposeTimer:    nil,
		exitC:           make(chan bool),
		txPool:          txPool,
		snapshotManager: snapshotMgr,
		txScheduler:     txScheduler,
		identity:        identity,
		ledgerCache:     ledgerCache,
		proposalCache:   proposedCache,
		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
	}

	b0 := createNewTestBlock(0)
	ledgerCache.SetLastCommittedBlock(b0)
	require.True(t, blockProposer.shouldProposeByBFT(b0.Header.BlockHeight+1))

	b := createNewTestBlock(1)
	proposedCache.SetProposedBlock(b, nil, nil, false)
	require.Nil(t, proposedCache.GetSelfProposedBlockAt(1))
	b1, _, _ := proposedCache.GetProposedBlock(b)
	require.NotNil(t, b1)

	b2 := createNewTestBlock(1)
	b2.Header.BlockHash = nil
	proposedCache.SetProposedBlock(b2, nil, nil, true)
	require.False(t, blockProposer.shouldProposeByBFT(b2.Header.BlockHeight+1))
	require.NotNil(t, proposedCache.GetSelfProposedBlockAt(1))
	ledgerCache.SetLastCommittedBlock(b2)
	require.True(t, blockProposer.shouldProposeByBFT(b2.Header.BlockHeight+1))

	b3, _, _ := proposedCache.GetProposedBlock(b2)
	require.NotNil(t, b3)

	proposedCache.SetProposedAt(b3.Header.BlockHeight)
	require.False(t, blockProposer.shouldProposeByBFT(b3.Header.BlockHeight))
}

func TestShouldProposeByMaxBFT(t *testing.T) {
	ctl := gomock.NewController(t)
	txPool := mock.NewMockTxPool(ctl)
	snapshotMgr := mock.NewMockSnapshotManager(ctl)
	msgBus := mbusmock.NewMockMessageBus(ctl)
	identity := mock.NewMockSigningMember(ctl)
	ledgerCache := cache.NewLedgerCache(chainId)
	proposedCache := cache.NewProposalCache(nil, ledgerCache)
	txScheduler := mock.NewMockTxScheduler(ctl)

	ledgerCache.SetLastCommittedBlock(createNewTestBlock(0))
	blockProposer := &BlockProposerImpl{
		chainId:         chainId,
		isProposer:      false, // not proposer when initialized
		idle:            true,
		msgBus:          msgBus,
		canProposeC:     make(chan bool),
		txPoolSignalC:   make(chan *txpoolpb.TxPoolSignal),
		proposeTimer:    nil,
		exitC:           make(chan bool),
		txPool:          txPool,
		snapshotManager: snapshotMgr,
		txScheduler:     txScheduler,
		identity:        identity,
		ledgerCache:     ledgerCache,
		proposalCache:   proposedCache,
		log:             logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
	}

	b0 := createNewTestBlock(0)
	ledgerCache.SetLastCommittedBlock(b0)
	require.True(t, blockProposer.shouldProposeByMaxBFT(b0.Header.BlockHeight+1, b0.Header.BlockHash))
	require.False(t, blockProposer.shouldProposeByMaxBFT(b0.Header.BlockHeight+1, []byte("xyz")))
	require.False(t, blockProposer.shouldProposeByMaxBFT(b0.Header.BlockHeight, b0.Header.PreBlockHash))

	b := createNewTestBlock(1)
	proposedCache.SetProposedBlock(b, nil, nil, false)
	require.Nil(t, proposedCache.GetSelfProposedBlockAt(1))
	b1, _, _ := proposedCache.GetProposedBlock(b)
	require.NotNil(t, b1)

	b2 := createNewTestBlock(1)
	b2.Header.BlockHash = nil
	proposedCache.SetProposedBlock(b2, nil, nil, true)
	require.NotNil(t, proposedCache.GetSelfProposedBlockAt(1))
	require.True(t, blockProposer.shouldProposeByMaxBFT(b2.Header.BlockHeight, b0.Header.BlockHash))

	b3, _, _ := proposedCache.GetProposedBlock(b2)
	require.NotNil(t, b3)

}

func TestYieldGoRountine(t *testing.T) {
	exitC := make(chan bool)
	go func() {
		time.Sleep(3 * time.Second)
		exitC <- true
	}()

	sig := <-exitC
	require.True(t, sig)
	fmt.Println("exit1")
}

func TestHash(t *testing.T) {
	txCount := 50000
	txs := make([][]byte, 0)
	for i := 0; i < txCount; i++ {
		txId := uuid.GetUUID() + uuid.GetUUID()
		txs = append(txs, []byte(txId))
	}
	require.Equal(t, txCount, len(txs))
	hf := sha256.New()

	start := utils.CurrentTimeMillisSeconds()
	for _, txId := range txs {
		hf.Write(txId)
		hf.Sum(nil)
		hf.Reset()
	}
	fmt.Println(utils.CurrentTimeMillisSeconds() - start)
}

func TestFinalize(t *testing.T) {
	txCount := 50000
	dag := &commonpb.DAG{Vertexes: make([]*commonpb.DAG_Neighbor, txCount)}
	txRead := &commonpb.TxRead{
		Key:          []byte("key"),
		Value:        []byte("value"),
		ContractName: contractName,
		Version:      nil,
	}
	txReads := make([]*commonpb.TxRead, 5)
	for i := 0; i < 5; i++ {
		txReads[i] = txRead
	}
	block := &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "chain1",
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			PreConfHeight:  0,
			BlockVersion:   1,
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: 0,
			Proposer: &accesscontrol.Member{
				OrgId:      "org1",
				MemberType: 0,
				MemberInfo: nil,
			},
			ConsensusArgs: nil,
			TxCount:       uint32(txCount),
			Signature:     nil,
		},
		Dag:            nil,
		Txs:            nil,
		AdditionalData: nil,
	}
	txs := make([]*commonpb.Transaction, 0)
	rwSetMap := make(map[string]*commonpb.TxRWSet)
	for i := 0; i < txCount; i++ {
		dag.Vertexes[i] = &commonpb.DAG_Neighbor{
			Neighbors: nil,
		}
		txId := uuid.GetUUID() + uuid.GetUUID()
		payload := parsePayload(txId)
		payloadBytes, _ := json.Marshal(payload)
		tx := parseTx(txId, payloadBytes)
		txs = append(txs, tx)
		txWrite := &commonpb.TxWrite{
			Key:          []byte(txId),
			Value:        payloadBytes,
			ContractName: contractName,
		}
		txWrites := make([]*commonpb.TxWrite, 0)
		txWrites = append(txWrites, txWrite)
		rwSetMap[txId] = &commonpb.TxRWSet{
			TxId:     txId,
			TxReads:  txReads,
			TxWrites: txWrites,
		}
	}
	require.Equal(t, txCount, len(txs))
	block.Txs = txs
	block.Dag = dag

	kvs := []*configpb.ConfigKeyValue{
		{Key: "IsExtreme", Value: "true"},
	}

	err := localconf.UpdateDebugConfig(kvs)
	require.Nil(t, err)
}

func TestTxDuplicateCheck(t *testing.T) {
	// init
	// 1. init transactions
	const (
		originalTx   = "QmXDdHkYEbAshxDnHxpDAvog7a2y3zknuKJgFnx4YLfYD"
		duplicateTx3 = originalTx + "3"
		duplicateTx5 = originalTx + "5"
		duplicateTx7 = originalTx + "7"
	)
	var duplicateTxs []*commonpb.Transaction
	duplicateTxs = append(duplicateTxs, &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId: "chain1",
			TxType:  commonpb.TxType_INVOKE_CONTRACT,
			TxId:    originalTx + strconv.Itoa(3),
		},
	})
	duplicateTxs = append(duplicateTxs, &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId: "chain1",
			TxType:  commonpb.TxType_INVOKE_CONTRACT,
			TxId:    originalTx + strconv.Itoa(5),
		},
	})
	duplicateTxs = append(duplicateTxs, &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId: "chain1",
			TxType:  commonpb.TxType_INVOKE_CONTRACT,
			TxId:    originalTx + strconv.Itoa(7),
		},
	})
	var txs []*commonpb.Transaction
	for i := 0; i < 10; i++ {
		tx := &commonpb.Transaction{
			Payload: &commonpb.Payload{
				ChainId: "chain1",
				TxType:  commonpb.TxType_INVOKE_CONTRACT,
				TxId:    originalTx + strconv.Itoa(i),
			},
		}
		txs = append(txs, tx)
	}
	// 2. init store
	ctl := gomock.NewController(t)
	// Three repeat transactions
	blockchainStore1 := mock.NewMockBlockchainStore(ctl)
	blockchainStore1.EXPECT().TxExists(gomock.Eq(duplicateTx3)).Return(true, nil).AnyTimes()
	blockchainStore1.EXPECT().TxExists(gomock.Eq(duplicateTx5)).Return(true, nil).AnyTimes()
	blockchainStore1.EXPECT().TxExists(gomock.Eq(duplicateTx7)).Return(true, nil).AnyTimes()
	// No repeat transactions
	blockchainStore2 := mock.NewMockBlockchainStore(ctl)
	// All repeat transactions
	blockchainStore3 := mock.NewMockBlockchainStore(ctl)
	for i := 0; i < 10; i++ {
		blockchainStore3.EXPECT().TxExists(gomock.Eq(originalTx+strconv.Itoa(i))).Return(true, nil).AnyTimes()
	}
	// execute case
	cases := []struct {
		comment      string
		duplicateTxs []*commonpb.Transaction
		store        *mock.MockBlockchainStore
	}{
		{comment: "Three repeat transactions", duplicateTxs: duplicateTxs, store: blockchainStore1},
		{comment: "No repeat transactions", duplicateTxs: []*commonpb.Transaction{}, store: blockchainStore2},
		{comment: "All repeat transactions", duplicateTxs: txs, store: blockchainStore3},
	}
	for _, case_ := range cases {
		t.Logf("comment: %s", case_.comment)
		blockProposerImpl := &BlockProposerImpl{
			blockchainStore: case_.store,
		}
		// 3. test Duplicate
		_, duplicates := blockProposerImpl.txDuplicateCheck(case_.duplicateTxs)
		if len(duplicates) != len(case_.duplicateTxs) {
			t.Errorf("duplicates size error result: %v, original: %v", duplicates, duplicateTxs)
		}
		// note: For convenience, uses an empty slice
		if duplicates == nil {
			duplicates = []*commonpb.Transaction{}
		}
		equal := reflect.DeepEqual(duplicates, case_.duplicateTxs)
		if !equal {
			t.Errorf("duplicates error result: %v, original: %v", duplicates, duplicateTxs)
		}
		t.Logf("comment: %s success", case_.comment)
	}
}

func finalizeBlockRoots() (interface{}, interface{}) {
	return nil, nil
}

func parseTxs(num int) []*commonpb.Transaction {
	txs := make([]*commonpb.Transaction, 0)
	for i := 0; i < num; i++ {
		txId := uuid.GetUUID() + uuid.GetUUID()
		payload := parsePayload(txId)
		payloadBytes, _ := json.Marshal(payload)
		txs = append(txs, parseTx(txId, payloadBytes))
	}
	return txs
}

func parsePayload(txId string) *commonpb.Payload {
	pairs := []*commonpb.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(txId)[len(txId)/2:],
		},
	}

	return &commonpb.Payload{
		ChainId:        "chain1",
		TxType:         0,
		TxId:           "txId1",
		Timestamp:      0,
		ExpirationTime: 0,
		ContractName:   contractName,
		Method:         "save",
		Parameters:     pairs,
		Sequence:       1,
		Limit:          nil,
	}
}

func parseTx(txId string, payloadBytes []byte) *commonpb.Transaction {
	return &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId:        "chain1",
			TxType:         0,
			TxId:           "txId1",
			Timestamp:      0,
			ExpirationTime: 0,
			ContractName:   contractName,
			Method:         "save",
			Parameters:     nil,
			Sequence:       1,
			Limit:          nil,
		},
		Sender:    nil,
		Endorsers: nil,
		Result: &commonpb.Result{
			Code: 0,
			ContractResult: &commonpb.ContractResult{
				Code:    0,
				Result:  payloadBytes,
				Message: "SUCCESS",
				GasUsed: 0,
			},
			RwSetHash: nil,
		},
	}

}

func createNewTestBlock(height uint64) *commonpb.Block {
	var hash = []byte("0123456789")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			BlockVersion:   1,
			BlockType:      0,
			ChainId:        "chain1",
			BlockHeight:    height,
			BlockHash:      hash,
			PreBlockHash:   hash,
			PreConfHeight:  0,
			TxCount:        0,
			TxRoot:         hash,
			DagHash:        hash,
			RwSetRoot:      hash,
			BlockTimestamp: 0,
			ConsensusArgs:  hash,
			Proposer: &accesscontrol.Member{
				OrgId:      "org1",
				MemberType: 0,
				MemberInfo: hash,
			},
			Signature: hash,
		},
		Dag:            &commonpb.DAG{Vertexes: nil},
		Txs:            nil,
		AdditionalData: nil,
	}

	tx := createNewTestTx("txId1")
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	block.Txs = txs
	return block
}

func createNewTestTx(txId string) *commonpb.Transaction {
	var hash = []byte("0123456789")
	return &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId:        "chain1",
			TxType:         0,
			TxId:           txId,
			Timestamp:      0,
			ExpirationTime: 0,
			ContractName:   "fact",
			Method:         "set",
			Parameters:     nil,
			Sequence:       1,
			Limit:          nil,
		},
		Sender: &commonpb.EndorsementEntry{
			Signer:    nil,
			Signature: nil,
		},
		Endorsers: nil,
		Result: &commonpb.Result{
			Code:           0,
			ContractResult: nil,
			RwSetHash:      hash,
			Message:        "",
		},
	}
}
