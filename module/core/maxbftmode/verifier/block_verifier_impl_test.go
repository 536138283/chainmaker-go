/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"errors"
	"testing"

	"chainmaker.org/chainmaker-go/module/core/common"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	mock2 "chainmaker.org/chainmaker/common/v2/msgbus/mock"
	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus"
)

//
//import (
//	"chainmaker.org/chainmaker/common/v2/crypto/hash"
//	"chainmaker.org/chainmaker/common/v2/msgbus"
//	"chainmaker.org/chainmaker-go/module/core/cache"
//	"chainmaker.org/chainmaker/logger/v2"
//	"chainmaker.org/chainmaker/protocol/v2/mock"
//	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
//	configpb "chainmaker.org/chainmaker/pb-go/v2/config"
//	"chainmaker.org/chainmaker/pb-go/v2/consensus"
//	"chainmaker.org/chainmaker/protocol/v2"
//	"chainmaker.org/chainmaker/utils/v2"
//	"fmt"
//	"github.com/golang/mock/gomock"
//	"github.com/stretchr/testify/require"
//	"strings"
//	"sync/atomic"
//	"testing"
//	"time"
//)
//
//var hashType = "SHA256"
//
//func TestBlockVerifierImpl_VerifyBlock(t *testing.T) {
//	ctl := gomock.NewController(t)
//	var chainId = "Chain1"
//
//	msgBus := msgbus.NewMessageBus()
//	txScheduler := mock.NewMockTxScheduler(ctl)
//	snapshotMgr := mock.NewMockSnapshotManager(ctl)
//	ledgerCache := cache.NewLedgerCache(chainId)
//	blockchainStoreImpl := mock.NewMockBlockchainStore(ctl)
//	proposedCache := cache.NewProposalCache(mock.NewMockChainConf(ctl), ledgerCache)
//	signerMember := mock.NewMockSigningMember(ctl)
//	chainConf := mock.NewMockChainConf(ctl)
//	ac := mock.NewMockAccessControlProvider(ctl)
//	txpool := mock.NewMockTxPool(ctl)
//
//	consensus := configpb.ConsensusConfig{
//		Type: consensus.ConsensusType_TBFT,
//	}
//	block := configpb.BlockConfig{
//		TxTimestampVerify: false,
//		TxTimeout:         1000000000,
//		BlockTxCapacity:   100,
//		BlockSize:         100000,
//		BlockInterval:     1000,
//	}
//	crypro := configpb.CryptoConfig{Hash: hashType}
//	contract := configpb.ContractConfig{EnableSqlSupport: false}
//	chainConfig := configpb.ChainConfig{Consensus: &consensus, Block: &block, Contract: &contract, Crypto: &crypro}
//	chainConf.EXPECT().ChainConfig().Return(&chainConfig).AnyTimes()
//
//	verifier := &BlockVerifierImpl{
//		chainId:         chainId,
//		msgBus:          msgBus,
//		txScheduler:     txScheduler,
//		snapshotManager: snapshotMgr,
//		ledgerCache:     ledgerCache,
//		blockchainStore: blockchainStoreImpl,
//		reentrantLocks: &reentrantLocks{
//			reentrantLocks: make(map[string]interface{}),
//		},
//		proposalCache:  proposedCache,
//		log:            logger.GetLoggerByChain(logger.MODULE_CORE, chainId),
//		chainConf:      chainConf,
//		blockValidator: NewBlockValidator(chainId, hashType),
//		ac:             ac,
//		txPool:         txpool,
//	}
//	verifier.txValidator = NewTxValidator(verifier.log, chainId, hashType, verifier.chainConf.ChainConfig().Consensus.Type, verifier.blockchainStore, txpool, ac)
//
//	sig_default := []byte("DEFAULT_SIGNATURE")
//	signerMember.EXPECT().Sign(gomock.Any(), gomock.Any()).Return(sig_default, nil).Times(100)
//	signerMember.EXPECT().Serialize(gomock.Any()).AnyTimes()
//	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
//	ac.EXPECT().VerifyPrincipal(gomock.Any()).Return(true, nil).AnyTimes()
//	snapshotMgr.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes()
//	tx := createNewTestTx()
//	txs := make([]*commonpb.Transaction, 1)
//	txs[0] = tx
//	rwSetmap := make(map[string]*commonpb.TxRWSet, 0)
//	rwSetmap[tx.Payload.TxId] = &commonpb.TxRWSet{
//		TxId:     tx.Payload.TxId,
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//
//	txList := make(map[string]*commonpb.Transaction, 0)
//	txList[tx.Payload.TxId] = tx
//	heights := make(map[string]int64, 0)
//	heights[tx.Payload.TxId] = 1
//
//	txpool.EXPECT().GetTxsByTxIds(gomock.Any()).Return(txList, heights).AnyTimes()
//	txpool.EXPECT().AddTxsToPendingCache(gomock.Any(), gomock.Any()).AnyTimes()
//	txResultMap := make(map[string]*commonpb.Result)
//	txResultMap[tx.Payload.TxId] = tx.Result
//	txScheduler.EXPECT().SimulateWithDag(gomock.Any(), gomock.Any()).Return(rwSetmap, txResultMap, nil).AnyTimes()
//
//	proposer, err := signerMember.Serialize(true)
//	require.Nil(t, err)
//
//	tx.Result.RwSetHash, err  = utils.CalcRWSetHash(hashType, rwSetmap[tx.Payload.TxId])
//
//	txHash, err := utils.CalcTxHash(hashType, tx)
//	require.Nil(t, err)
//
//	b0 := createNewTestBlockWithoutProposer(0)
//	ledgerCache.SetLastCommittedBlock(b0)
//	b1 := createNewTestBlock(1, proposer, txs)
//
//	txHashs := make([][]byte, 0)
//	txHashs= append(txHashs, txHash)
//	txRoot, err := hash.GetMerkleRoot(hashType, txHashs)
//	require.Nil(t, err)
//	b1.Header.TxRoot = txRoot
//
//	dagHash, err := utils.CalcDagHash(hashType, b1.Dag)
//	require.Nil(t, err)
//	b1.Header.DagHash = dagHash
//
//	rwSetRoot, err := utils.CalcRWSetRoot(hashType, txs)
//	require.Nil(t, err)
//	b1.Header.RwSetRoot = rwSetRoot
//
//	blockHash, err := utils.CalcBlockHash("SHA256", b1)
//	require.Nil(t, err)
//	b1.Header.BlockHash = blockHash
//
//	err = verifier.VerifyBlock(b1, protocol.CONSENSUS_VERIFY)
//	require.Nil(t, err)
//}
//
//func Test_ReentrantLock(t *testing.T) {
//	lock := &reentrantLocks{
//		reentrantLocks: make(map[string]interface{}),
//	}
//
//	for i := 0; i < 3; i++ {
//		go func() {
//			j := i
//			if lock.lock("") {
//				require.False(t, lock.lock(""))
//				defer lock.unlock("")
//				fmt.Println(fmt.Sprintf("%d get lock", j))
//				time.Sleep(2 * time.Second)
//			}
//		}()
//	}
//
//	for i := 0; i < 3; i++ {
//		j := i
//		go func() {
//			for {
//				if lock.lock("") {
//					defer lock.unlock("")
//					fmt.Println(fmt.Sprintf("finally %d get lock", j))
//					break
//				}
//			}
//		}()
//	}
//
//	time.Sleep(5 * time.Second)
//}
//
//func Test_ReentrantLocks(t *testing.T) {
//	locks := &reentrantLocks{
//		reentrantLocks: make(map[string]interface{}),
//	}
//	for i := 0; i < 3; i++ {
//		go func() {
//			j := i
//			if locks.lock("1") {
//				require.False(t, locks.lock("1"))
//				defer locks.unlock("1")
//				fmt.Println(fmt.Sprintf("%d get lock", j))
//				time.Sleep(2 * time.Second)
//			}
//		}()
//	}
//
//	for i := 0; i < 3; i++ {
//		j := i
//		go func() {
//			for {
//				if locks.lock("2") {
//					defer locks.unlock("2")
//					fmt.Println(fmt.Sprintf("finally %d get lock", j))
//					time.Sleep(1 * time.Second)
//					break
//				}
//			}
//		}()
//	}
//	time.Sleep(5 * time.Second)
//
//}
//
//type reentrantLock struct {
//	reentrantLock *int32
//}
//
//func (l *reentrantLock) lock(key string) bool {
//	return atomic.CompareAndSwapInt32(l.reentrantLock, 0, 1)
//}
//
//func (l *reentrantLock) unlock(key string) bool {
//	return atomic.CompareAndSwapInt32(l.reentrantLock, 1, 0)
//}
//
//func Test_Hashprefix(t *testing.T) {
//	b := []byte(":B:1.0.0")
//	require.True(t, strings.HasPrefix(string(b), protocol.ContractByteCode))
//}
//
//func Test_DispatchTask(t *testing.T) {
//	tasks := utils.DispatchTxVerifyTask(nil)
//	fmt.Println(tasks)
//	txs := make([]*commonpb.Transaction, 0)
//	for i := 0; i < 5; i++ {
//		txs = append(txs, createNewTestTx())
//	}
//	require.Equal(t, 5, len(txs))
//	verifyTasks := utils.DispatchTxVerifyTask(txs)
//	fmt.Println(len(verifyTasks))
//	for i := 0; i < len(verifyTasks); i++ {
//		fmt.Println(fmt.Sprintf("%v", verifyTasks[i]))
//	}
//
//	for i := 0; i < 123; i++ {
//		txs = append(txs, createNewTestTx())
//	}
//	verifyTasks = utils.DispatchTxVerifyTask(txs)
//	fmt.Println(len(verifyTasks))
//	for i := 0; i < len(verifyTasks); i++ {
//		fmt.Println(fmt.Sprintf("%v", verifyTasks[i]))
//	}
//
//	for i := 0; i < 896; i++ {
//		txs = append(txs, createNewTestTx())
//	}
//	verifyTasks = utils.DispatchTxVerifyTask(txs)
//	fmt.Println(len(verifyTasks))
//	for i := 0; i < len(verifyTasks); i++ {
//		fmt.Println(fmt.Sprintf("%v", verifyTasks[i]))
//	}
//}
//
//func createNewTestBlock(height uint64, proposer []byte, txs []*commonpb.Transaction) *commonpb.Block {
//	var hash = []byte("0123456789")
//	var version = []byte("0")
//
//	var block = &commonpb.Block{
//		Header: &commonpb.BlockHeader{
//			ChainId:        "Chain1",
//			BlockHeight:    height,
//			PreBlockHash:   hash,
//			BlockHash:      hash,
//			PreConfHeight:  0,
//			BlockVersion:   version,
//			DagHash:        hash,
//			RwSetRoot:      hash,
//			TxRoot:         hash,
//			BlockTimestamp: 0,
//			Proposer:       proposer,
//			ConsensusArgs:  nil,
//			TxCount:        1,
//			Signature:      hash,
//		},
//		Dag: &commonpb.DAG{
//			Vertexes: nil,
//		},
//		Txs: txs,
//	}
//
//	return block
//}
//
//func createNewTestTx() *commonpb.Transaction {
//	var hash = []byte("0123456789")
//	return &commonpb.Transaction{
//		Header: &commonpb.TxHeader{
//			ChainId:        "",
//			Sender:         nil,
//			TxType:         0,
//			TxId:           "",
//			Timestamp:      0,
//			ExpirationTime: 0,
//		},
//		RequestPayload:   hash,
//		RequestSignature: hash,
//		Result: &commonpb.Result{
//			Code:           commonpb.TxStatusCode_CONTRACT_REVOKE_FAILED,
//			ContractResult: &commonpb.ContractResult{
//				Code:          0,
//				Result:        nil,
//				Message:       "",
//				GasUsed:       0,
//				ContractEvent: nil,
//			},
//			RwSetHash:      nil,
//		},
//	}
//}
//
//func createNewTestBlockWithoutProposer(height uint64) *commonpb.Block {
//	var hash = []byte("0123456789")
//	var version = []byte("0")
//	var block = &commonpb.Block{
//		Header: &commonpb.BlockHeader{
//			ChainId:        "Chain1",
//			BlockHeight:    height,
//			PreBlockHash:   hash,
//			BlockHash:      hash,
//			PreConfHeight:  0,
//			BlockVersion:   version,
//			DagHash:        hash,
//			RwSetRoot:      hash,
//			TxRoot:         hash,
//			BlockTimestamp: 0,
//			Proposer:       hash,
//			ConsensusArgs:  nil,
//			TxCount:        1,
//			Signature:      []byte(""),
//		},
//		Dag: &commonpb.DAG{
//			Vertexes: nil,
//		},
//		Txs: nil,
//	}
//	tx := createNewTestTx()
//	txs := make([]*commonpb.Transaction, 1)
//	txs[0] = tx
//	block.Txs = txs
//	return block
//}

func TestBlockVerifierImpl_verifyRepeat(t *testing.T) {
	c := gomock.NewController(t)

	type fields struct {
		chainId               string
		msgBus                msgbus.MessageBus
		txScheduler           protocol.TxScheduler
		snapshotManager       protocol.SnapshotManager
		ledgerCache           protocol.LedgerCache
		blockchainStore       protocol.BlockchainStore
		reentrantLocks        *common.ReentrantLocks
		proposalCache         protocol.ProposalCache
		chainConf             protocol.ChainConf
		ac                    protocol.AccessControlProvider
		log                   protocol.Logger
		txPool                protocol.TxPool
		verifierBlock         *common.VerifierBlock
		storeHelper           conf.StoreHelper
		metricBlockVerifyTime *prometheus.HistogramVec
	}
	type args struct {
		block     *commonpb.Block
		startTick int64
		mode      protocol.VerifyMode
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantIsRepeat bool
		wantErr      bool
	}{
		{
			name: "正常流 cc(sole,sqltrue)",
			fields: fields{
				proposalCache: func() protocol.ProposalCache {
					proposalCache := mock.NewMockProposalCache(c)
					proposalCache.EXPECT().GetProposedBlock(gomock.Any()).Return(nil, nil, nil)
					return proposalCache
				}(),
				chainConf: getCc(consensus.ConsensusType_SOLO, true, c),
				msgBus:    getMb(c),
				log: func() protocol.Logger {
					logger := mock.NewMockLogger(c)
					logger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
					logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
					return logger
				}(),
			},
			args: args{
				block:     getBlock(),
				startTick: 1,
				mode:      protocol.CONSENSUS_VERIFY,
			},
			wantIsRepeat: false,
			wantErr:      false,
		},
		{
			name: "正常流 cc(notsole,sqlfalse)",
			fields: fields{
				proposalCache: func() protocol.ProposalCache {
					proposalCache := mock.NewMockProposalCache(c)
					proposalCache.EXPECT().GetProposedBlock(gomock.Any()).Return(nil, nil, nil)
					return proposalCache
				}(),
				chainConf: getCc(consensus.ConsensusType_SOLO, false, c),
				msgBus:    getMb(c),
				log: func() protocol.Logger {
					logger := mock.NewMockLogger(c)
					logger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
					logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
					return logger
				}(),
			},
			args: args{
				block:     getNilBlock(),
				startTick: 1,
				mode:      protocol.CONSENSUS_VERIFY,
			},
			wantIsRepeat: false,
			wantErr:      false,
		},
		{
			name: "正常流 cc(notsole,sqlfalse)",
			fields: fields{
				proposalCache: func() protocol.ProposalCache {
					proposalCache := mock.NewMockProposalCache(c)
					proposalCache.EXPECT().GetProposedBlock(gomock.Any()).Return(nil, nil, nil)
					return proposalCache
				}(),
				chainConf: getCc(consensus.ConsensusType_TBFT, false, c),
				msgBus:    getMb(c),
				log: func() protocol.Logger {
					logger := mock.NewMockLogger(c)
					logger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
					logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
					return logger
				}(),
			},
			args: args{
				block:     getBlock(),
				startTick: 1,
				mode:      protocol.CONSENSUS_VERIFY,
			},
			wantIsRepeat: false,
			wantErr:      false,
		},
		{
			name: "正常流 cc(notsole,sqltrue) CONSENSUS_VERIFY GetProposedBlockByHashAndHeight()nil",
			fields: fields{
				proposalCache: func() protocol.ProposalCache {
					proposalCache := mock.NewMockProposalCache(c)
					proposalCache.EXPECT().GetProposedBlock(gomock.Any()).Return(getBlock(), nil, nil)
					proposalCache.EXPECT().GetProposedBlockByHashAndHeight(gomock.Any(), gomock.Any()).
						Return(nil, nil)
					return proposalCache
				}(),
				chainConf: getCc(consensus.ConsensusType_TBFT, true, c),
				msgBus:    getMb(c),
				log: func() protocol.Logger {
					logger := mock.NewMockLogger(c)
					logger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
					logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
					return logger
				}(),
			},
			args: args{
				block:     getBlock(),
				startTick: 1,
				mode:      protocol.CONSENSUS_VERIFY,
			},
			wantIsRepeat: true,
			wantErr:      false,
		},
		{
			name: "正常流 cc(notsole,sqlfalse) SYNC_VERIFY GetProposedBlockByHashAndHeight()notnil " +
				"KeepProposedBlock()len1",
			fields: fields{
				proposalCache: func() protocol.ProposalCache {
					proposalCache := mock.NewMockProposalCache(c)
					proposalCache.EXPECT().GetProposedBlock(gomock.Any()).Return(getBlock(), nil, nil)
					proposalCache.EXPECT().GetProposedBlockByHashAndHeight(gomock.Any(), gomock.Any()).
						Return(getBlock(), nil)
					proposalCache.EXPECT().KeepProposedBlock(gomock.Any(), gomock.Any()).
						Return([]*commonpb.Block{getBlock()})
					proposalCache.EXPECT().SetProposedBlock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil)
					proposalCache.EXPECT().IsProposedAt(gomock.Any()).Return(true)
					return proposalCache
				}(),
				chainConf: getCc(consensus.ConsensusType_TBFT, true, c),
				msgBus:    getMb(c),
				log: func() protocol.Logger {
					logger := mock.NewMockLogger(c)
					logger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
					logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
					return logger
				}(),
			},
			args: args{
				block:     getBlock(),
				startTick: 1,
				mode:      protocol.SYNC_VERIFY,
			},
			wantIsRepeat: true,
			wantErr:      false,
		},
		{
			name: "正常流 cc(notsole,sqlfalse) CONSENSUS_VERIFY GetProposedBlockByHashAndHeight()notnil " +
				"KeepProposedBlock()len0",
			fields: fields{
				proposalCache: func() protocol.ProposalCache {
					proposalCache := mock.NewMockProposalCache(c)
					proposalCache.EXPECT().GetProposedBlock(gomock.Any()).Return(getBlock(), nil, nil)
					proposalCache.EXPECT().GetProposedBlockByHashAndHeight(gomock.Any(), gomock.Any()).
						Return(getBlock(), nil)
					proposalCache.EXPECT().KeepProposedBlock(gomock.Any(), gomock.Any()).Return([]*commonpb.Block{})
					proposalCache.EXPECT().SetProposedBlock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil)
					proposalCache.EXPECT().IsProposedAt(gomock.Any()).Return(true)
					return proposalCache
				}(),
				chainConf: getCc(consensus.ConsensusType_TBFT, true, c),
				msgBus:    getMb(c),
				log: func() protocol.Logger {
					logger := mock.NewMockLogger(c)
					logger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
					logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
					return logger
				}(),
			},
			args: args{
				block:     getBlock(),
				startTick: 1,
				mode:      protocol.SYNC_VERIFY,
			},
			wantIsRepeat: true,
			wantErr:      false,
		},
		{
			name: "正常流 cc(notsole,sqlfalse) CONSENSUS_VERIFY GetProposedBlockByHashAndHeight()notnil " +
				"KeepProposedBlock()len0",
			fields: fields{
				proposalCache: func() protocol.ProposalCache {
					proposalCache := mock.NewMockProposalCache(c)
					proposalCache.EXPECT().GetProposedBlock(gomock.Any()).Return(getBlock(), nil, nil)
					proposalCache.EXPECT().GetProposedBlockByHashAndHeight(gomock.Any(), gomock.Any()).
						Return(getBlock(), nil)
					proposalCache.EXPECT().KeepProposedBlock(gomock.Any(), gomock.Any()).Return([]*commonpb.Block{})
					proposalCache.EXPECT().SetProposedBlock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(errors.New("block with invalid height"))
					proposalCache.EXPECT().IsProposedAt(gomock.Any()).Return(true)
					return proposalCache
				}(),
				chainConf: getCc(consensus.ConsensusType_TBFT, true, c),
				msgBus:    getMb(c),
				log: func() protocol.Logger {
					logger := mock.NewMockLogger(c)
					logger.EXPECT().Infof(gomock.Any(), gomock.Any()).AnyTimes()
					logger.EXPECT().Debugf(gomock.Any(), gomock.Any()).AnyTimes()
					return logger
				}(),
			},
			args: args{
				block:     getBlock(),
				startTick: 1,
				mode:      protocol.SYNC_VERIFY,
			},
			wantIsRepeat: true,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &BlockVerifierImpl{
				chainId:               tt.fields.chainId,
				msgBus:                tt.fields.msgBus,
				txScheduler:           tt.fields.txScheduler,
				snapshotManager:       tt.fields.snapshotManager,
				ledgerCache:           tt.fields.ledgerCache,
				blockchainStore:       tt.fields.blockchainStore,
				reentrantLocks:        tt.fields.reentrantLocks,
				proposalCache:         tt.fields.proposalCache,
				chainConf:             tt.fields.chainConf,
				ac:                    tt.fields.ac,
				log:                   tt.fields.log,
				txPool:                tt.fields.txPool,
				verifierBlock:         tt.fields.verifierBlock,
				storeHelper:           tt.fields.storeHelper,
				metricBlockVerifyTime: tt.fields.metricBlockVerifyTime,
			}
			gotIsRepeat, err := v.verifyRepeat(tt.args.block, tt.args.startTick, tt.args.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyRepeat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotIsRepeat != tt.wantIsRepeat {
				t.Errorf("verifyRepeat() = %v, want %v", gotIsRepeat, tt.wantIsRepeat)
			}
		})
	}
}

func getBlock() *commonpb.Block {
	return &commonpb.Block{Header: &commonpb.BlockHeader{
		BlockHeight: 56744,
		BlockHash:   []byte("fdasfdasfdsa"),
	}}
}

func getNilBlock() *commonpb.Block {
	return &commonpb.Block{}
}

func getMb(c *gomock.Controller) msgbus.MessageBus {
	messageBus := mock2.NewMockMessageBus(c)
	messageBus.EXPECT().Publish(gomock.Any(), gomock.Any()).AnyTimes()
	return messageBus
}

func getCc(csus consensus.ConsensusType, sql bool, c *gomock.Controller) protocol.ChainConf {
	cc := mock.NewMockChainConf(c)
	cc.EXPECT().ChainConfig().AnyTimes().Return(&config.ChainConfig{
		Consensus: &config.ConsensusConfig{
			Type: csus,
		},
		Contract: &config.ContractConfig{
			EnableSqlSupport: sql,
		},
	})
	return cc
}
