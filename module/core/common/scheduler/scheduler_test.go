/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package scheduler

import (
	"fmt"
	"runtime"

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

func prepare(t *testing.T, enableSenderGroup, enableConflictsBitWindow bool, txCount int) (*mock.MockVmManager, []*commonpb.TxRWSet, []*commonpb.Transaction,
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
	chainConfig := &configpb.ChainConfig{Crypto: &crypto, Contract: &contractConf, Core: &configpb.CoreConfig{
		EnableSenderGroup:        enableSenderGroup,
		EnableConflictsBitWindow: enableConflictsBitWindow,
	}}
	chainConf.EXPECT().ChainConfig().AnyTimes().Return(chainConfig)
	//chainConf :=

	storeHelper := mock.NewMockStoreHelper(ctl)
	storeHelper.EXPECT().GetPoolCapacity().Return(runtime.NumCPU() * 4)
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
	blockChainStore := mock.NewMockBlockchainStore(ctl)
	blockChainStore.EXPECT().GetContractByName(contractId.Name).Return(contractId, nil).AnyTimes()
	blockChainStore.EXPECT().GetContractBytecode(contractId.Name).AnyTimes()

	snapshot.EXPECT().GetBlockchainStore().AnyTimes().Return(blockChainStore)
	//snapshot.EXPECT().Seal()

	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(contractResult, protocol.ExecOrderTxTypeNormal, commonpb.TxStatusCode_SUCCESS)
	return vmMgr, txRWSetTable, txTable, snapshot, scheduler, contractId, block
}

func TestSchedule(t *testing.T) {

	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare(t, false, false, 2)

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

	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare(t, true, false, 1)

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

	_, txRWSetTable, txTable, snapshot, scheduler, contractId, block := prepare(t, true, true, 1)

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

func TestSimulateWithDag(t *testing.T) {

	_, _, _, snapshot, scheduler, contractId, block := prepare(t, false, false, 2)

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
