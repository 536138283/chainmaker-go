/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"fmt"
	"runtime"

	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"

	commonpb "chainmaker.org/chainmaker/pb-go/v2/common"
	configpb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/gogo/protobuf/proto"

	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var (
	TestPrivKeyFile = "../../../../config/wx-org1/certs/node/consensus1/consensus1.sign.key"
	TestCertFile    = "../../../../config/wx-org1/certs/node/consensus1/consensus1.sign.crt"
)

//func TestDag(t *testing.T) {
//	for i := 0; i < 10; i++ {
//
//		neb1 := &commonpb.DAG_Neighbor{
//			Neighbors: []int32{1, 2, 3, 4},
//		}
//		neb2 := &commonpb.DAG_Neighbor{
//			Neighbors: []int32{1, 2, 3, 4},
//		}
//		neb3 := &commonpb.DAG_Neighbor{
//			Neighbors: []int32{1, 2, 3, 4},
//		}
//		vs := make([]*commonpb.DAG_Neighbor, 3)
//		vs[0] = neb1
//		vs[1] = neb2
//		vs[2] = neb3
//		dag := &commonpb.DAG{
//			Vertexes: vs,
//		}
//		marshal, _ := proto.Marshal(dag)
//		println("Dag", hex.EncodeToString(marshal))
//	}
//}
//
func newTx(txId string, contractId *commonpb.Contract, parameterMap map[string]string) *commonpb.Transaction {

	var parameters []*commonpb.KeyValuePair
	for key, value := range parameterMap {
		parameters = append(parameters, &commonpb.KeyValuePair{
			Key:   key,
			Value: []byte(value),
		})
	}

	return &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId:        "Chain1",
			TxType:         0,
			TxId:           txId,
			ContractName:   contractId.Name,
			Method:         "method",
			Parameters:     parameters,
			Timestamp:      0,
			ExpirationTime: 0,
		},
		Result: &commonpb.Result{
			Code: commonpb.TxStatusCode_SUCCESS,
			ContractResult: &commonpb.ContractResult{
				Code:          0,
				Result:        nil,
				Message:       "",
				GasUsed:       0,
				ContractEvent: nil,
			},
			RwSetHash: nil,
		},
		Sender: &commonpb.EndorsementEntry{Signer: &acPb.Member{OrgId: "org1", MemberInfo: []byte("cert1...")},
			Signature: []byte("sign1"),
		},
	}
}

func newTxWithPubKeyAndGasLimit(txId string, contractId *commonpb.Contract, parameterMap map[string]string, gasLimit uint64) *commonpb.Transaction {

	var parameters []*commonpb.KeyValuePair
	for key, value := range parameterMap {
		parameters = append(parameters, &commonpb.KeyValuePair{
			Key:   key,
			Value: []byte(value),
		})
	}

	return &commonpb.Transaction{
		Payload: &commonpb.Payload{
			ChainId:        "Chain1",
			TxType:         0,
			TxId:           txId,
			ContractName:   contractId.Name,
			Method:         "method",
			Parameters:     parameters,
			Timestamp:      0,
			ExpirationTime: 0,
			Limit:          &commonpb.Limit{GasLimit: gasLimit},
		},
		Result: &commonpb.Result{
			Code: commonpb.TxStatusCode_SUCCESS,
			ContractResult: &commonpb.ContractResult{
				Code:          0,
				Result:        nil,
				Message:       "",
				GasUsed:       0,
				ContractEvent: nil,
			},
			RwSetHash: nil,
		},
		Sender: &commonpb.EndorsementEntry{
			Signer: &acPb.Member{
				OrgId:      "org1",
				MemberType: acPb.MemberType_PUBLIC_KEY,
				MemberInfo: []byte("-----BEGIN PUBLIC KEY-----\nMIIBCgKCAQEAvIU7PHVzanE3V6GHHS5OQLYRAh8gjKIzSVI+UKPRcy6hB8u/z7Is\n2oNPeOLW/N9umreCgi1nBhcjczOlbpIzq8YIMP/7HN3gnyPpsSp4y6GelKzl0YNy\nAN5huqyNU8dn2Du0xFeyzK6UGqmKb9Le1nfLZq6YtVB0NEfPfxzkTG15RrJg/eRn\nc0Lywl8tMwAptRE3ZJA791/aEJWdJLB52vqhM+fGn5+ol6OO/0mQAHdopIutYrZI\nzvM9GBZHdDEdz3f+44IRmc9qmzhoEEp5epD2LJDCtfNnwbKP/cwBaTMNCMqSibA4\nlMMMSwU88dmY6ZH4RCxDXaI9suMGzFh/fwIDAQAB\n-----END PUBLIC KEY-----"),
			},
			Signature: []byte("sign1"),
		},
	}

}

func newBlock() *commonpb.Block {
	return &commonpb.Block{
		Header: &commonpb.BlockHeader{
			ChainId:        "",
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			BlockVersion:   1,
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

func prepare(t *testing.T, enableOptimizeChargeGas, enableSenderGroup, enableConflictsBitWindow bool, txCount int) (
	*mock.MockVmManager, []*commonpb.TxRWSet, []*commonpb.Transaction,
	*mock.MockSnapshot, protocol.TxScheduler, *commonpb.Contract, *commonpb.Block) {
	var txRWSetTable = make([]*commonpb.TxRWSet, txCount)
	var txTable = make([]*commonpb.Transaction, txCount)

	ctl := gomock.NewController(t)
	snapshot := mock.NewMockSnapshot(ctl)
	vmMgr := mock.NewMockVmManager(ctl)
	chainConf := mock.NewMockChainConf(ctl)
	crypto := configpb.CryptoConfig{
		Hash: "SHA256",
	}
	contractConf := configpb.ContractConfig{EnableSqlSupport: false}
	chainConfig := &configpb.ChainConfig{
		Crypto:   &crypto,
		Contract: &contractConf,
		AuthType: protocol.Identity,
		Core: &configpb.CoreConfig{
			EnableOptimizeChargeGas:  enableOptimizeChargeGas,
			EnableSenderGroup:        enableSenderGroup,
			EnableConflictsBitWindow: enableConflictsBitWindow,
		},
	}
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(chainConfig)

	storeHelper := mock.NewMockStoreHelper(ctl)
	storeHelper.EXPECT().GetPoolCapacity().Return(runtime.NumCPU() * 4).AnyTimes()
	var schedulerFactory TxSchedulerFactory
	scheduler := schedulerFactory.NewTxScheduler(vmMgr, chainConf, storeHelper)
	contractId := &commonpb.Contract{
		Name:        "ContractName",
		Version:     "1",
		RuntimeType: commonpb.RuntimeType_WASMER,
	}

	contractResult := &commonpb.ContractResult{
		Code:    0,
		Result:  nil,
		Message: "",
	}
	block := newBlock()

	snapshot.EXPECT().GetTxTable().AnyTimes().Return(txTable)
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRWSetTable)
	snapshot.EXPECT().GetSnapshotSize().AnyTimes().Return(len(txTable))
	snapshot.EXPECT().GetSpecialTxTable().AnyTimes().Return([]*commonpb.Transaction{})
	snapshot.EXPECT().GetKey(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("1000000000"), nil)
	blockChainStore := mock.NewMockBlockchainStore(ctl)
	blockChainStore.EXPECT().GetContractByName(contractId.Name).Return(contractId, nil).AnyTimes()
	blockChainStore.EXPECT().GetContractBytecode(contractId.Name).AnyTimes()

	snapshot.EXPECT().GetBlockchainStore().AnyTimes().Return(blockChainStore)
	//snapshot.EXPECT().Seal()

	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, protocol.ExecOrderTxTypeNormal, commonpb.TxStatusCode_SUCCESS)
	return vmMgr, txRWSetTable, txTable, snapshot, scheduler, contractId, block
}

// prepare4 is used only by TestSchedule4
func prepare4(t *testing.T, enableOptimizeChargeGas, enableSenderGroup, enableConflictsBitWindow bool, txCount int) (
	*mock.MockVmManager, []*commonpb.TxRWSet, []*commonpb.Transaction,
	*mock.MockSnapshot, protocol.TxScheduler, *commonpb.Contract, *commonpb.Block) {
	var txRWSetTable = make([]*commonpb.TxRWSet, txCount)
	var txTable = make([]*commonpb.Transaction, txCount)

	ctl := gomock.NewController(t)
	snapshot := mock.NewMockSnapshot(ctl)
	vmMgr := mock.NewMockVmManager(ctl)
	chainConf := mock.NewMockChainConf(ctl)
	crypto := configpb.CryptoConfig{
		Hash: "SHA256",
	}
	contractConf := configpb.ContractConfig{EnableSqlSupport: false}
	chainConfig := &configpb.ChainConfig{
		Crypto:   &crypto,
		Contract: &contractConf,
		AuthType: protocol.Identity,
		Core: &configpb.CoreConfig{
			EnableOptimizeChargeGas:  enableOptimizeChargeGas,
			EnableSenderGroup:        enableSenderGroup,
			EnableConflictsBitWindow: enableConflictsBitWindow,
		},
		AccountConfig: &configpb.GasAccountConfig{
			EnableGas: true,
		},
	}
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(chainConfig)

	storeHelper := mock.NewMockStoreHelper(ctl)
	storeHelper.EXPECT().GetPoolCapacity().Return(runtime.NumCPU() * 4).AnyTimes()
	var schedulerFactory TxSchedulerFactory
	scheduler := schedulerFactory.NewTxScheduler(vmMgr, chainConf, storeHelper)
	contractId := &commonpb.Contract{
		Name:        "ContractName",
		Version:     "1",
		RuntimeType: commonpb.RuntimeType_WASMER,
	}

	sysContractId := &commonpb.Contract{
		Name:        syscontract.SystemContract_ACCOUNT_MANAGER.String(),
		Version:     "1",
		RuntimeType: commonpb.RuntimeType_NATIVE,
	}

	contractResult := &commonpb.ContractResult{
		Code:    0,
		Result:  nil,
		Message: "",
	}
	block := newBlock()

	snapshot.EXPECT().GetTxTable().AnyTimes().Return(txTable)
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRWSetTable)
	snapshot.EXPECT().GetSnapshotSize().AnyTimes().Return(len(txTable))
	snapshot.EXPECT().GetSpecialTxTable().AnyTimes().Return([]*commonpb.Transaction{})
	snapshot.EXPECT().GetKey(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("1000000000"), nil)
	blockChainStore := mock.NewMockBlockchainStore(ctl)
	reqCall1 := blockChainStore.EXPECT().GetContractByName(contractId.Name).Return(contractId, nil).Times(2)
	blockChainStore.EXPECT().GetContractByName(sysContractId.Name).After(reqCall1).Return(sysContractId, nil).Times(1)
	blockChainStore.EXPECT().GetContractBytecode(contractId.Name).AnyTimes()
	blockChainStore.EXPECT().GetContractBytecode(sysContractId.Name).AnyTimes()

	snapshot.EXPECT().GetBlockchainStore().AnyTimes().Return(blockChainStore)
	//snapshot.EXPECT().Seal()

	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, protocol.ExecOrderTxTypeNormal, commonpb.TxStatusCode_SUCCESS)
	return vmMgr, txRWSetTable, txTable, snapshot, scheduler, contractId, block
}

// prepare5 is used only by TestSchedule5
func prepare5(t *testing.T, enableOptimizeChargeGas, enableSenderGroup, enableConflictsBitWindow bool, txCount int) (
	*mock.MockVmManager, []*commonpb.TxRWSet, []*commonpb.Transaction,
	*mock.MockSnapshot, protocol.TxScheduler, *commonpb.Contract, *commonpb.Block) {
	var txRWSetTable = make([]*commonpb.TxRWSet, txCount)
	var txTable = make([]*commonpb.Transaction, txCount)

	ctl := gomock.NewController(t)
	snapshot := mock.NewMockSnapshot(ctl)
	vmMgr := mock.NewMockVmManager(ctl)
	chainConf := mock.NewMockChainConf(ctl)
	crypto := configpb.CryptoConfig{
		Hash: "SHA256",
	}
	contractConf := configpb.ContractConfig{EnableSqlSupport: false}
	chainConfig := &configpb.ChainConfig{
		Crypto:   &crypto,
		Contract: &contractConf,
		AuthType: protocol.Identity,
		Core: &configpb.CoreConfig{
			EnableOptimizeChargeGas:  enableOptimizeChargeGas,
			EnableSenderGroup:        enableSenderGroup,
			EnableConflictsBitWindow: enableConflictsBitWindow,
		},
		AccountConfig: &configpb.GasAccountConfig{
			EnableGas: true,
		},
	}
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(chainConfig)

	storeHelper := mock.NewMockStoreHelper(ctl)
	storeHelper.EXPECT().GetPoolCapacity().Return(runtime.NumCPU() * 4).AnyTimes()
	var schedulerFactory TxSchedulerFactory
	scheduler := schedulerFactory.NewTxScheduler(vmMgr, chainConf, storeHelper)
	contractId := &commonpb.Contract{
		Name:        "ContractName",
		Version:     "1",
		RuntimeType: commonpb.RuntimeType_WASMER,
	}

	sysContractId := &commonpb.Contract{
		Name:        syscontract.SystemContract_ACCOUNT_MANAGER.String(),
		Version:     "1",
		RuntimeType: commonpb.RuntimeType_NATIVE,
	}

	contractResult := &commonpb.ContractResult{
		Code:    0,
		Result:  nil,
		Message: "",
	}
	block := newBlock()

	snapshot.EXPECT().GetTxTable().AnyTimes().Return(txTable)
	snapshot.EXPECT().GetTxRWSetTable().AnyTimes().Return(txRWSetTable)
	snapshot.EXPECT().GetSnapshotSize().AnyTimes().Return(len(txTable))
	snapshot.EXPECT().GetSpecialTxTable().AnyTimes().Return([]*commonpb.Transaction{})
	snapshot.EXPECT().GetKey(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("1000000000"), nil)
	blockChainStore := mock.NewMockBlockchainStore(ctl)
	reqCall1 := blockChainStore.EXPECT().GetContractByName(contractId.Name).Return(contractId, nil).Times(2)
	blockChainStore.EXPECT().GetContractByName(sysContractId.Name).After(reqCall1).Return(sysContractId, nil).Times(1)
	blockChainStore.EXPECT().GetContractBytecode(contractId.Name).AnyTimes()
	blockChainStore.EXPECT().GetContractBytecode(sysContractId.Name).AnyTimes()

	snapshot.EXPECT().GetBlockchainStore().AnyTimes().Return(blockChainStore)
	//snapshot.EXPECT().Seal()

	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, protocol.ExecOrderTxTypeNormal, commonpb.TxStatusCode_SUCCESS)
	return vmMgr, txRWSetTable, txTable, snapshot, scheduler, contractId, block
}

func TestSchedule(t *testing.T) {

	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare(t, false, false, false, 2)

	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000001", contractId, parameters)
	tx1 := newTx("a0000000000000000000000000000002", contractId, parameters)

	txTable[0] = tx0
	txTable[1] = tx1
	txRWSetTable[0] = &commonpb.TxRWSet{
		TxId: tx0.Payload.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	txRWSetTable[1] = &commonpb.TxRWSet{
		TxId: tx1.Payload.TxId,
		TxReads: []*commonpb.TxRead{
			{
				ContractName: contractId.Name,
				Key:          []byte("K2"),
				Value:        []byte("V"),
			},
			{
				ContractName: contractId.Name,
				Key:          []byte("K2"),
				Value:        []byte("V"),
			},
		},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K3"),
			Value:        []byte("V"),
		}},
	}

	snapshot.EXPECT().ApplyTxSimContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, 2).AnyTimes()
	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()

	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}
	snapshot.EXPECT().BuildDAG(gomock.Any()).Return(dag)

	txBatch := []*commonpb.Transaction{tx0, tx1}
	txSet, contractEven, err := scheduler.Schedule(block, txBatch, snapshot)
	require.Nil(t, err)
	require.NotNil(t, txSet)
	require.NotNil(t, contractEven)

	fmt.Println(txSet)
	fmt.Println(contractEven)
}

func TestSchedule2(t *testing.T) {

	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare(t, false, true, false, 1)

	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000001", contractId, parameters)
	//tx1 := newTx("a0000000000000000000000000000002", contractId, parameters)

	txTable[0] = tx0
	//txTable[1] = tx1
	txRWSetTable[0] = &commonpb.TxRWSet{
		TxId: tx0.Payload.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	//txRWSetTable[1] = &commonpb.TxRWSet{
	//	TxId: tx1.Payload.TxId,
	//	TxReads: []*commonpb.TxRead{
	//		{
	//			ContractName: contractId.Name,
	//			Key:          []byte("K2"),
	//			Value:        []byte("V"),
	//		},
	//		{
	//			ContractName: contractId.Name,
	//			Key:          []byte("K2"),
	//			Value:        []byte("V"),
	//		},
	//	},
	//	TxWrites: []*commonpb.TxWrite{{
	//		ContractName: contractId.Name,
	//		Key:          []byte("K3"),
	//		Value:        []byte("V"),
	//	}},
	//}

	snapshot.EXPECT().ApplyTxSimContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, 1).AnyTimes()
	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()

	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}
	snapshot.EXPECT().BuildDAG(gomock.Any()).Return(dag)

	txBatch := []*commonpb.Transaction{tx0}
	txSet, contractEven, err := scheduler.Schedule(block, txBatch, snapshot)
	require.Nil(t, err)
	require.NotNil(t, txSet)
	require.NotNil(t, contractEven)

	fmt.Println(txSet)
	fmt.Println(contractEven)
}

func TestSchedule3(t *testing.T) {

	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare(t, false, true, true, 1)

	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000001", contractId, parameters)
	//tx1 := newTx("a0000000000000000000000000000002", contractId, parameters)

	txTable[0] = tx0
	//txTable[1] = tx1
	txRWSetTable[0] = &commonpb.TxRWSet{
		TxId: tx0.Payload.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K2"),
			Value:        []byte("V"),
		}},
	}
	//txRWSetTable[1] = &commonpb.TxRWSet{
	//	TxId: tx1.Payload.TxId,
	//	TxReads: []*commonpb.TxRead{
	//		{
	//			ContractName: contractId.Name,
	//			Key:          []byte("K2"),
	//			Value:        []byte("V"),
	//		},
	//		{
	//			ContractName: contractId.Name,
	//			Key:          []byte("K2"),
	//			Value:        []byte("V"),
	//		},
	//	},
	//	TxWrites: []*commonpb.TxWrite{{
	//		ContractName: contractId.Name,
	//		Key:          []byte("K3"),
	//		Value:        []byte("V"),
	//	}},
	//}

	snapshot.EXPECT().ApplyTxSimContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, 1).AnyTimes()
	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()

	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}
	snapshot.EXPECT().BuildDAG(gomock.Any()).Return(dag)

	txBatch := []*commonpb.Transaction{tx0}
	txSet, contractEven, err := scheduler.Schedule(block, txBatch, snapshot)
	require.Nil(t, err)
	require.NotNil(t, txSet)
	require.NotNil(t, contractEven)

	fmt.Println(txSet)
	fmt.Println(contractEven)
}

// TestSchedule4 test the flag `enableOptimizeChargeGas` is opened.
func TestSchedule4(t *testing.T) {

	fmt.Println("===== TestSchedule4() begin ==== ")
	localconf.ChainMakerConfig.NodeConfig.PrivKeyFile = TestPrivKeyFile
	localconf.ChainMakerConfig.NodeConfig.CertFile = TestCertFile
	localconf.ChainMakerConfig.NodeConfig.PrivKeyPassword = "11111111"
	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare4(t, true, false, false, 2)

	parameters := make(map[string]string, 8)
	tx0 := newTxWithPubKeyAndGasLimit("a0000000000000000000000000000001", contractId, parameters, 101)
	tx1 := newTxWithPubKeyAndGasLimit("a0000000000000000000000000000002", contractId, parameters, 102)

	txTable[0] = tx0
	txTable[1] = tx1
	txRWSetTable[0] = &commonpb.TxRWSet{
		TxId: tx0.Payload.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V1"),
		}},
	}
	txRWSetTable[1] = &commonpb.TxRWSet{
		TxId: tx1.Payload.TxId,
		TxReads: []*commonpb.TxRead{
			{
				ContractName: contractId.Name,
				Key:          []byte("K1"),
				Value:        []byte("V"),
			},
			{
				ContractName: contractId.Name,
				Key:          []byte("K2"),
				Value:        []byte("V"),
			},
		},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K2"),
			Value:        []byte("V2"),
		}},
	}

	snapshot.EXPECT().ApplyTxSimContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, 2).AnyTimes()
	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()

	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}
	snapshot.EXPECT().BuildDAG(gomock.Any()).Return(dag)

	txBatch := []*commonpb.Transaction{tx0, tx1}
	txSet, contractEven, err := scheduler.Schedule(block, txBatch, snapshot)
	require.Nil(t, err)
	require.NotNil(t, txSet)
	require.NotNil(t, contractEven)
}

// TestSchedule5 test the conflictsBitWindows features under flag `enableOptimizeChargeGas` is opened.
func TestSchedule5(t *testing.T) {

	fmt.Println("===== TestSchedule5() begin ==== ")
	localconf.ChainMakerConfig.NodeConfig.PrivKeyFile = TestPrivKeyFile
	localconf.ChainMakerConfig.NodeConfig.CertFile = TestCertFile
	localconf.ChainMakerConfig.NodeConfig.PrivKeyPassword = "11111111"
	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare5(t, true, false, true, 2)

	parameters := make(map[string]string, 8)
	tx0 := newTxWithPubKeyAndGasLimit("a0000000000000000000000000000001", contractId, parameters, 101)
	tx1 := newTxWithPubKeyAndGasLimit("a0000000000000000000000000000002", contractId, parameters, 102)

	txTable[0] = tx0
	txTable[1] = tx1
	txRWSetTable[0] = &commonpb.TxRWSet{
		TxId: tx0.Payload.TxId,
		TxReads: []*commonpb.TxRead{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V"),
		}},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K1"),
			Value:        []byte("V1"),
		}},
	}
	txRWSetTable[1] = &commonpb.TxRWSet{
		TxId: tx1.Payload.TxId,
		TxReads: []*commonpb.TxRead{
			{
				ContractName: contractId.Name,
				Key:          []byte("K3"),
				Value:        []byte("V"),
			},
			{
				ContractName: contractId.Name,
				Key:          []byte("K4"),
				Value:        []byte("V"),
			},
		},
		TxWrites: []*commonpb.TxWrite{{
			ContractName: contractId.Name,
			Key:          []byte("K3"),
			Value:        []byte("V3"),
		}},
	}

	snapshot.EXPECT().ApplyTxSimContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, 2).AnyTimes()
	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()

	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{{}},
	}
	snapshot.EXPECT().BuildDAG(gomock.Any()).Return(dag)

	txBatch := []*commonpb.Transaction{tx0, tx1}
	txSet, contractEven, err := scheduler.Schedule(block, txBatch, snapshot)
	require.Nil(t, err)
	require.NotNil(t, txSet)
	require.NotNil(t, contractEven)
}

func TestSimulateWithDag(t *testing.T) {

	_, _, _, snapshot, scheduler, contractId, block := prepare(t, false, false, false, 2)

	parameters := make(map[string]string, 8)
	tx0 := newTx("a0000000000000000000000000000000", contractId, parameters)
	tx1 := newTx("a0000000000000000000000000000001", contractId, parameters)
	tx2 := newTx("a0000000000000000000000000000002", contractId, parameters)

	block.Txs = []*commonpb.Transaction{tx0, tx1, tx2}
	block.Dag = &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{
			{
				Neighbors: nil,
			},
			{
				Neighbors: []uint32{0},
			},
			{
				Neighbors: []uint32{0},
			},
		},
	}

	snapshot.EXPECT().IsSealed().AnyTimes().Return(false)
	snapshot.EXPECT().Seal().Return()
	snapshot.EXPECT().ApplyTxSimContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, 3).AnyTimes()

	txRWSets := make(map[string]*commonpb.Result, len(block.Txs))
	//
	snapshot.EXPECT().GetTxResultMap().AnyTimes().Return(txRWSets)

	txRwSet, result, err := scheduler.SimulateWithDag(block, snapshot)
	require.Nil(t, err)
	require.NotNil(t, txRwSet)
	require.NotNil(t, result)
	fmt.Println("txRWSet: ", txRwSet)
	fmt.Println("result: ", result)
}

func TestMarshalDag(t *testing.T) {
	dag := &commonpb.DAG{
		Vertexes: []*commonpb.DAG_Neighbor{
			{
				Neighbors: []uint32{0},
			},
			{
				Neighbors: []uint32{0, 1, 2},
			},
		},
	}

	mar, _ := proto.Marshal(dag)

	dag2 := &commonpb.DAG{}
	proto.Unmarshal(mar, dag2)

	require.Equal(t, len(dag2.Vertexes), 2)
}
