/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package verifier

//
//import (
//	"chainmaker.org/chainmaker-go/core/cache"
//	"chainmaker.org/chainmaker-go/core/common"
//	"chainmaker.org/chainmaker-go/localconf"
//	"chainmaker.org/chainmaker-go/logger"
//	"chainmaker.org/chainmaker-go/monitor"
//	"chainmaker.org/chainmaker/protocol/mock"
//	commonpb "chainmaker.org/chainmaker/pb-go/common"
//	"chainmaker.org/chainmaker/pb-go/config"
//	"chainmaker.org/chainmaker/protocol"
//	"encoding/hex"
//	"encoding/json"
//	"fmt"
//	"github.com/golang/mock/gomock"
//	"github.com/panjf2000/ants/v3"
//	"sync"
//	"testing"
//)
//
//var success = "success"
//
//func verifyPrepare(t *testing.T) (*Verifier, *commonpb.Block, error) {
//	ctl := gomock.NewController(t)
//	log := logger.GetLoggerByChain(logger.MODULE_CORE, "chain1")
//	abftCache := cache.NewAbftCache()
//	ledgerCache := mock.NewMockLedgerCache(ctl)
//	msgBus := mock.NewMockMessageBus(ctl)
//	txPool := mock.NewMockTxPool(ctl)
//
//	//msgBus mock
//	msgBus.EXPECT().Publish(gomock.Any(), gomock.Any()).AnyTimes().Return()
//
//	verifier := &Verifier{
//		wg:            sync.WaitGroup{},
//		log:           log,
//		abftCache:     abftCache,
//		ledgerCache:   ledgerCache,
//		msgBus:        msgBus,
//		verifyTimeout: DEFAULT_VERIFY_TIMEOUT,
//		txPool:        txPool,
//		chainId:       "chain1",
//	}
//	var block *commonpb.Block
//	verifier.verifierBlock, block = verifyBlockPrepare(ctl, log, ledgerCache, txPool)
//	var err error
//	verifier.goRoutinePool, err = ants.NewPool(10, ants.WithPreAlloc(true))
//	if err != nil {
//		return nil, nil, fmt.Errorf("new verifier failed: %s", err.Error())
//	}
//	if localconf.ChainMakerConfig.MonitorConfig.Enabled {
//		verifier.metricBlockVerifyTime = monitor.NewHistogramVec(monitor.SUBSYSTEM_CORE_VERIFIER, "metric_block_verify_time",
//			"block verify time metric", []float64{0.005, 0.01, 0.015, 0.05, 0.1, 1, 10}, "chainId")
//	}
//	return verifier, block, nil
//
//}
//
//func verifyBlockPrepare(ctl *gomock.Controller, log *logger.CMLogger, ledgerCache *mock.MockLedgerCache, txPool *mock.MockTxPool) (*common.VerifierBlock, *commonpb.Block) {
//	chainConf := mock.NewMockChainConf(ctl)
//	ac := mock.NewMockAccessControlProvider(ctl)
//	snapshotManager := mock.NewMockSnapshotManager(ctl)
//	vmMgr := mock.NewMockVmManager(ctl)
//	store := mock.NewMockBlockchainStore(ctl)
//
//	//chainConf mock
//	config := &config.ChainConfig{
//		ChainId: "chain1",
//		Crypto: &config.CryptoConfig{
//			Hash: "SHA256",
//		},
//		Block: &config.BlockConfig{
//			BlockTxCapacity: 1000,
//			BlockSize:       1,
//			BlockInterval:   1000,
//		},
//	}
//	chainConf.EXPECT().ChainConfig().AnyTimes().Return(config)
//
//	//ledgerCache mock
//	lastBlock := newBlock()
//	block := newVerifyBlock()
//	lastBlock.Header.BlockHash = []byte("111222333444555")
//	block.Header.PreBlockHash = lastBlock.Header.BlockHash
//	blockHash, _ := hex.DecodeString("f96db8e4f48dbce4f44af1e794b3d1f347ce24167ce71535fbec8fdb7e8f83bc")
//	block.Header.BlockHash = blockHash
//	ledgerCache.EXPECT().GetLastCommittedBlock().AnyTimes().Return(lastBlock)
//	ledgerCache.EXPECT().CurrentHeight().AnyTimes().Return(int64(0), nil)
//
//	//ac mock
//	principal := mock.NewMockPrincipal(ctl)
//	ac.EXPECT().CreatePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(principal, nil)
//	ac.EXPECT().VerifyPrincipal(gomock.Any()).AnyTimes().Return(true, nil)
//
//	//vmMgr mock
//	contractId := &commonpb.ContractId{
//		ContractName:    "ContractName",
//		ContractVersion: "1",
//		RuntimeType:     commonpb.RuntimeType_WASMER,
//	}
//	contractResult := &commonpb.ContractResult{
//		Code:    0,
//		Result:  nil,
//		Message: "",
//	}
//	parameters := make(map[string]string, 8)
//	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
//	txs := make([]*commonpb.Transaction, 0)
//	txs = append(txs, tx0)
//	block.Txs = txs
//	block.Header.TxCount = 1
//	block.Header.BlockHeight = 1
//
//	var txRWSetMap = make(map[string]*commonpb.TxRWSet, 3)
//	txRWSetMap[tx0.Header.TxId] = &commonpb.TxRWSet{
//		TxId: tx0.Header.TxId,
//		TxReads: []*commonpb.TxRead{{
//			ContractName: contractId.ContractName,
//			Key:          []byte("K1"),
//			Value:        []byte("V"),
//		}},
//		TxWrites: []*commonpb.TxWrite{{
//			ContractName: contractId.ContractName,
//			Key:          []byte("K2"),
//			Value:        []byte("V"),
//		}},
//	}
//	rwHash, _ := hex.DecodeString("d02f421ed76e0e26e9def824a8b84c7c223d484762d6d060a8b71e1649d1abbf")
//	result := &commonpb.Result{
//		Code: commonpb.TxStatusCode_SUCCESS,
//		ContractResult: &commonpb.ContractResult{
//			Code:    0,
//			Result:  nil,
//			Message: "",
//			GasUsed: 0,
//		},
//		RwSetHash: rwHash,
//	}
//	tx0.Result = result
//	txResultMap := make(map[string]*commonpb.Result, 1)
//	txResultMap[tx0.Header.TxId] = result
//	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, commonpb.TxStatusCode_SUCCESS)
//
//	//snapshotManager mock
//	snapshot := mock.NewMockSnapshot(ctl)
//	snapshotManager.EXPECT().NewSnapshot(gomock.Any(), gomock.Any()).AnyTimes().Return(snapshot)
//	var txTable = make([]*commonpb.Transaction, 1)
//	var txRWSetTable = make([]*commonpb.TxRWSet, 1)
//	txTable[0] = tx0
//	txRWSetTable[0] = txRWSetMap[tx0.Header.TxId]
//	snapshot.EXPECT().GetTxTable().AnyTimes().Return(txTable)
//	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRWSetTable)
//	snapshot.EXPECT().GetSnapshotSize().AnyTimes().Return(2)
//	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
//	snapshot.EXPECT().Seal().Return()
//	txSimCache0 := common.NewTxSimContext(vmMgr, snapshot, tx0)
//	txSimCache0.SetTxResult(result)
//	snapshot.EXPECT().ApplyTxSimContext(txSimCache0, true).Return(true, 1)
//	dag := &commonpb.DAG{
//		Vertexes: []*commonpb.DAG_Neighbor{{}},
//	}
//	snapshot.EXPECT().BuildDAG().Return(dag)
//	snapshot.EXPECT().GetTxResultMap().Return(txResultMap)
//
//	//txPool mock
//	txsMap := make(map[string]*commonpb.Transaction)
//	txsMap[tx0.Header.TxId] = tx0
//	txPool.EXPECT().GetTxsByTxIds([]string{tx0.Header.TxId}).Return(txsMap, nil)
//	txPool.EXPECT().AddTxsToPendingCache(gomock.Any(), gomock.Any()).AnyTimes().Return()
//
//	//store mock
//	store.EXPECT().TxExists(tx0).AnyTimes().Return(false, nil)
//
//	conf := &common.ValidateBlockConf{
//		ChainConf:       chainConf,
//		Log:             log,
//		LedgerCache:     ledgerCache,
//		Ac:              ac,
//		SnapshotManager: snapshotManager,
//		VmMgr:           vmMgr,
//		TxPool:          txPool,
//		BlockchainStore: store,
//	}
//	return common.NewVerifierBlock(conf), block
//}
//
//func TestVerify(t *testing.T) {
//	verify, block, err := verifyPrepare(t)
//	if err != nil {
//		fmt.Println("verify prepare failed: " + err.Error())
//	}
//
//	blockByte, err := json.Marshal(block)
//
//	// empty block
//	block1 := new(commonpb.Block)
//	block1.GetDag()
//
//	// txCount error
//	block2 := new(commonpb.Block)
//	err = json.Unmarshal(blockByte, block2)
//	block2.Header.TxCount = 0
//
//	// txCount > txCap
//	block3 := new(commonpb.Block)
//	err = json.Unmarshal(blockByte, block3)
//	block3.Header.TxCount = 10
//
//	// sign is empty
//	block4 := new(commonpb.Block)
//	err = json.Unmarshal(blockByte, block4)
//	block.Header.Signature = []byte{}
//
//	//DagHash is empty
//	block5 := new(commonpb.Block)
//	err = json.Unmarshal(blockByte, block5)
//	block5.Header.DagHash = []byte{}
//
//	//PreBlockHash is empty
//	block6 := new(commonpb.Block)
//	err = json.Unmarshal(blockByte, block6)
//	block6.Header.PreBlockHash = []byte{}
//
//	//BlockHash is empty
//	block7 := new(commonpb.Block)
//	err = json.Unmarshal(blockByte, block7)
//	block7.Header.BlockHash = []byte{}
//
//	tests := []struct {
//		verifyCore   *Verifier
//		block        *commonpb.Block
//		expectResult string
//	}{
//		{verify, block, success}, // normal block
//		{verify, block, success}, // normal block (repeat block.)
//		{verify,block1, "invalid block, yield verify"}, // empty block
//		{verify,block2, ""}, // txcount error(no error,but record in cach)
//		{verify,block3, ""}, // txcount error(no error,but record in cach)
//		{verify,block4, ""}, // sign error(no error,but record in cach)
//	}
//
//	for _,v := range tests{
//		err = v.verifyCore.VerifyBlock(v.block, protocol.CONSENSUS_VERIFY)
//		fmt.Printf("block strcture: %v\n", v.block)
//		if v.expectResult != "" && v.expectResult != success {
//			checkResult(err, v.expectResult)
//		} else {
//			checkCache(v.verifyCore, v.block)
//		}
//	}
//
//	fmt.Println("verify finish.")
//}
//
//func checkResult(err error, expectResult string) {
//	if err.Error() != expectResult {
//		fmt.Printf("verify block failed; expected: %v, got: %v \n", expectResult, err)
//	}
//}
//
//func checkCache(verifyCore *Verifier, block *commonpb.Block) {
//	ifSuccess, err := verifyCore.abftCache.IsVerifiedTxBatchSuccess(block.Header.BlockHash)
//	if err != nil {
//		fmt.Printf("check cache fail, err : %v \n", err)
//	}
//
//	if ifSuccess {
//		fmt.Printf("check cache fail, this block should be fail.")
//	}
//
//	fmt.Printf("if success?, got : %v \n", ifSuccess)
//}
