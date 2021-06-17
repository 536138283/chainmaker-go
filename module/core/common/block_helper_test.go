package common

import (
	"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker-go/mock"
	commonpb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	"encoding/hex"
	"fmt"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestFinalizeBlock(t *testing.T) {
	ctl := gomock.NewController(t)
	identity := mock.NewMockSigningMember(ctl)
	chainConf := mock.NewMockChainConf(ctl)
	identity.EXPECT().Serialize(true).AnyTimes().Return([]byte("DEFAULTPROPOSER"), nil)
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(nil)

	block := CreateNewTestBlock(0)
	chainId := "123"
	nblock, er := InitNewBlock(block, identity, chainId, chainConf)
	fmt.Println(er)

	txRWSetMap := make(map[string]*commonpb.TxRWSet)
	aclFailTxs := make([]*commonpb.Transaction, 0, 0)
	hashtype := "456"
	er = FinalizeBlock(nblock, txRWSetMap, aclFailTxs, hashtype)
	fmt.Println(er)
}

func verifyBlockPrepare(t *testing.T) (*VerifierBlock, *commonpb.Block) {
	ctl := gomock.NewController(t)
	log := logger.GetLoggerByChain(logger.MODULE_CORE, "chain1")
	chainConf := mock.NewMockChainConf(ctl)
	ledgerCache := mock.NewMockLedgerCache(ctl)
	ac := mock.NewMockAccessControlProvider(ctl)
	snapshotManager := mock.NewMockSnapshotManager(ctl)
	vmMgr := mock.NewMockVmManager(ctl)
	txPool := mock.NewMockTxPool(ctl)
	store := mock.NewMockBlockchainStore(ctl)

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
	}
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(config)

	//ledgerCache mock
	lastBlock := newBlock()
	block := newBlock()
	lastBlock.Header.BlockHash = []byte("111222333444555")
	block.Header.PreBlockHash = lastBlock.Header.BlockHash
	blockHash, _ := hex.DecodeString("f4b43ff2d2fbdd2563b406f833ecfd03c5b5d67726326d65c60cdf1f270f10fd")
	block.Header.BlockHash = blockHash
	ledgerCache.EXPECT().GetLastCommittedBlock().AnyTimes().Return(lastBlock)

	//ac mock
	principal := mock.NewMockPrincipal(ctl)
	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(principal, nil)
	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)

	//vmMgr mock
	contractId := &commonpb.ContractId{
		ContractName:    "ContractName",
		ContractVersion: "1",
		RuntimeType:     commonpb.RuntimeType_WASMER,
	}
	contractResult := &commonpb.ContractResult{
		Code:    0,
		Result:  nil,
		Message: "",
	}
	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
	txs := make([]*commonpb.Transaction, 0)
	txs = append(txs, tx0)
	block.Txs = txs
	block.Header.TxCount = 1
	block.Header.BlockHeight = 1

	var txRWSetMap = make(map[string]*commonpb.TxRWSet, 3)
	txRWSetMap[tx0.Header.TxId] = &commonpb.TxRWSet{
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
	tx0.Result = result
	txResultMap := make(map[string]*commonpb.Result, 1)
	txResultMap[tx0.Header.TxId] = result
	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, commonpb.TxStatusCode_SUCCESS)

	//snapshotManager mock
	snapshot := mock.NewMockSnapshot(ctl)
	snapshotManager.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes().Return(snapshot)
	var txTable = make([]*commonpb.Transaction, 1)
	var txRWSetTable = make([]*commonpb.TxRWSet, 1)
	txTable[0] = tx0
	txRWSetTable[0] = txRWSetMap[tx0.Header.TxId]
	snapshot.EXPECT().GetTxTable().AnyTimes().Return(txTable)
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRWSetTable)
	snapshot.EXPECT().GetSnapshotSize().AnyTimes().Return(2)
	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()
	txSimCache0 := NewTxSimContext(vmMgr, snapshot, tx0)
	txSimCache0.SetTxResult(result)
	snapshot.EXPECT().ApplyTxSimContext(txSimCache0, true).Return(true, 1)
	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}
	snapshot.EXPECT().BuildDAG().Return(dag)
	snapshot.EXPECT().GetTxResultMap().Return(txResultMap)

	//txPool mock
	txsMap := make(map[string]*commonpb.Transaction)
	txsMap[tx0.Header.TxId] = tx0
	txPool.EXPECT().GetTxsByTxIds([]string{tx0.Header.TxId}).Return(txsMap, nil)

	//store mock
	store.EXPECT().TxExists(tx0).AnyTimes().Return(false, nil)

	conf := &ValidateBlockConf{
		ChainConf:       chainConf,
		Log:             log,
		LedgerCache:     ledgerCache,
		Ac:              ac,
		SnapshotManager: snapshotManager,
		VmMgr:           vmMgr,
		TxPool:          txPool,
		BlockchainStore: store,
	}
	return NewVerifierBlock(conf), block
}

func TestBlockVerify(t *testing.T) {
	verifyBlock, block := verifyBlockPrepare(t)
	RWSetMap, _, _ := verifyBlock.ValidateBlock(block)
	fmt.Printf("rwset : %v\n", RWSetMap)
}

func CreateNewTestBlock(height int64) *commonpb.Block {
	var hash = []byte("0123456789")
	var version = []byte("0")
	var block = &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "Chain1",
			BlockHeight:    height,
			PreBlockHash:   hash,
			BlockHash:      hash,
			PreConfHeight:  0,
			BlockVersion:   version,
			DagHash:        hash,
			RwSetRoot:      hash,
			TxRoot:         hash,
			BlockTimestamp: 0,
			Proposer:       hash,
			ConsensusArgs:  nil,
			TxCount:        1,
			Signature:      []byte(""),
		},
		Dag: &commonpb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
	}
	tx := CreateNewTestTx()
	txs := make([]*commonpb.Transaction, 1)
	txs[0] = tx
	block.Txs = txs
	return block
}

func CreateNewTestTx() *commonpb.Transaction {
	var hash = []byte("0123456789")
	return &commonpb.Transaction{
		Header: &commonpb.TxHeader{
			ChainId:        "",
			Sender:         nil,
			TxType:         0,
			TxId:           "",
			Timestamp:      0,
			ExpirationTime: 0,
		},
		RequestPayload:   hash,
		RequestSignature: hash,
		Result: &commonpb.Result{
			Code:           commonpb.TxStatusCode_SUCCESS,
			ContractResult: nil,
			RwSetHash:      nil,
		},
	}
}