package abft

import (
	"chainmaker.org/chainmaker-go/core/cache"
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	"encoding/hex"
	"fmt"
	"github.com/golang/mock/gomock"
	"testing"
)

func newBlock() *commonpb.Block {
	return &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "",
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			BlockVersion:   nil,
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: 0,
			Proposer:       nil,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
		AdditionalData: &commonpb.AdditionalData{
			ExtraData: nil,
		},
	}
}
func proposePrepare(t *testing.T) *Proposer {
	ctl := gomock.NewController(t)
	log := logger.GetLoggerByChain(logger.MODULE_CORE, "chain1")
	chainConf := mock.NewMockChainConf(ctl)
	ledgerCache := mock.NewMockLedgerCache(ctl)
	ac := mock.NewMockAccessControlProvider(ctl)
	snapshotManager := mock.NewMockSnapshotManager(ctl)
	vmMgr := mock.NewMockVmManager(ctl)
	txPool := mock.NewMockTxPool(ctl)
	store := mock.NewMockBlockchainStore(ctl)
	identity := mock.NewMockSigningMember(ctl)
	msgBus := mock.NewMockMessageBus(ctl)

	//txPool mock
	contractId := &commonpb.ContractId{
		ContractName:    "ContractName",
		ContractVersion: "1",
		RuntimeType:     commonpb.RuntimeType_WASMER,
	}

	parameters := make(map[string]string, 1)
	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
	txBatch := make([]*commonpb.Transaction, 0)
	txBatch = append(txBatch, tx0)
	txPool.EXPECT().FetchTxBatch(gomock.Any()).AnyTimes().Return(txBatch)

	//ledgerCache mock
	lastBlock := newBlock()
	blockHash, _ := hex.DecodeString("f4b43ff2d2fbdd2563b406f833ecfd03c5b5d67726326d65c60cdf1f270f10fd")
	lastBlock.Header.BlockHash = blockHash
	ledgerCache.EXPECT().GetLastCommittedBlock().AnyTimes().Return(lastBlock)
	ledgerCache.EXPECT().CurrentHeight().AnyTimes().Return(int64(0), nil)

	//identity mock
	identity.EXPECT().Serialize(gomock.Any()).AnyTimes().Return([]byte("testNode1"), nil)

	//chainConf mock
	config := &config.ChainConfig{
		Version: "1.0",
		ChainId: "chain1",
		Crypto: &config.CryptoConfig{
			Hash: "SHA256",
		},
		Block: &config.BlockConfig{
			BlockTxCapacity: 1000,
			BlockSize:       1,
			BlockInterval:   1000,
		},
	}
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(config)

	//snapshotManager mock
	snapshot := mock.NewMockSnapshot(ctl)
	snapshotManager.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes().Return(snapshot)

	var txTable = make([]*commonpb.Transaction, 1)
	txTable[0] = tx0

	var txRWSetTable = make([]*commonpb.TxRWSet, 1)
	txRWSetTable[0] = &commonpb.TxRWSet{
		TxId: tx0.Header.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.ContractName,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.ContractName,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	rwHash, _ := hex.DecodeString("d02f421ed76e0e26e9def824a8b84c7c223d484762d6d060a8b71e1649d1abbf")
	result := &commonpb.Result{
		Code: commonpb.TxStatusCode_SUCCESS,
		ContractResult: &commonpb.ContractResult{
			Code:    0,
			Result:  nil,
			Message: "",
			GasUsed: 0,
		},
		RwSetHash: rwHash,
	}
	txResultMap := make(map[string]*commonpb.Result, 1)
	txResultMap[tx0.Header.TxId] = result
	snapshot.EXPECT().GetTxTable().AnyTimes().Return(txTable)
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRWSetTable)
	snapshot.EXPECT().GetSnapshotSize().AnyTimes().Return(1)
	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().AnyTimes().Return()

	snapshot.EXPECT().ApplyTxSimContext(gomock.Any(), true).Return(true, 1)
	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}
	snapshot.EXPECT().BuildDAG().Return(dag)
	snapshot.EXPECT().GetTxResultMap().Return(txResultMap)

	//msgBus mock
	msgBus.EXPECT().Publish(gomock.Any(), gomock.Any()).AnyTimes().Return()

	//vm mock
	contractResult := &commonpb.ContractResult{
		Code:    0,
		Result:  nil,
		Message: "",
		GasUsed: 0,
	}
	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, commonpb.TxStatusCode_SUCCESS)

	//abftcache mock
	abftCache := cache.NewAbftCache()
	ce := &CoreExecute{
		chainId:         "chain1",
		ledgerCache:     ledgerCache,
		txPool:          txPool,
		snapshotManager: snapshotManager,
		identity:        identity,
		msgBus:          msgBus,
		ac:              ac,
		blockchainStore: store,
		chainConf:       chainConf,
		log:             log,
		vmMgr:           vmMgr,
		abftCache:       abftCache,
	}
	return NewProposer(ce)
}

func TestPropose(t *testing.T) {
	proposer := proposePrepare(t)
	proposer.proposedSignal = &abft.PackagedSignal{
		BlockHeight: 1,
	}
	err := proposer.Propose()
	if err != nil {
		fmt.Println("error: " + err.Error())
	}
}
