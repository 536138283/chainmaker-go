/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package snapshot

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"chainmaker.org/chainmaker-go/module/core/common/scheduler"
	crypto2 "chainmaker.org/chainmaker/common/v3/crypto"
	"chainmaker.org/chainmaker/common/v3/log"
	"chainmaker.org/chainmaker/localconf/v3"
	"chainmaker.org/chainmaker/logger/v3"
	acPb "chainmaker.org/chainmaker/pb-go/v3/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v3/common"
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	configpb "chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"chainmaker.org/chainmaker/protocol/v3/test"
	"github.com/golang/mock/gomock"

	"chainmaker.org/chainmaker/protocol/v3/mock"
)

func TestNewSnapshotManager(t *testing.T) {
	t.Log("TestNewSnapshotManager")
	var (
		snapshotFactory Factory
		log             = &test.GoLogger{}
		ctl             = gomock.NewController(t)
		store           = mock.NewMockBlockchainStore(ctl)
	)

	manager := snapshotFactory.NewSnapshotManager(store, log)

	fmt.Println(manager)
	log.Debug("test NewSnapshotManager")
}

func TestNewSnapshotEvidenceMgr(t *testing.T) {
	t.Log("TestNewSnapshotEvidenceMgr")
	var (
		snapshotFactory Factory
		log             = &test.GoLogger{}
		ctl             = gomock.NewController(t)
		store           = mock.NewMockBlockchainStore(ctl)
	)

	manager := snapshotFactory.NewSnapshotManager(store, log)

	fmt.Println(manager)
	log.Debug("test NewSnapshotEvidenceMgr")
}

func prepareTx(txId string, contractId *commonPb.Contract, method string,
	parameterMap map[string]string) *commonPb.Transaction {

	var parameters []*commonPb.KeyValuePair
	for key, value := range parameterMap {
		parameters = append(parameters, &commonPb.KeyValuePair{
			Key:   key,
			Value: []byte(value),
		})
	}

	return &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ChainId:        "Chain1",
			TxType:         0,
			TxId:           txId,
			ContractName:   contractId.Name,
			Method:         method,
			Parameters:     parameters,
			Timestamp:      0,
			ExpirationTime: 0,
			Limit:          &commonPb.Limit{GasLimit: 100},
		},
		Sender: &commonPb.EndorsementEntry{
			//Signer: &acPb.Member{
			//	OrgId:      "public",
			//	MemberType: acPb.MemberType_PUBLIC_KEY,
			//	MemberInfo: []byte("-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA56Ts7nA8HqrApIkPFoHK\nNZSCo1SWxanjlkgBowLSjlatdYpTqKeE+mbNWFyl8R00JIuSsPf2pdIsdLhvNb6N\nL5uZ0bDlvaMv3Eg5q77Kt8TwJ12j6l3Gr8lrh7g8xYsIRbEUMjG0L/E4y4Fhlk7k\nDoGOrbiaA01vqlQDZVXCJCbK94oQOrokteMlyrl4/4bbilpWV8Sirc3mp12DMRPx\nGc3pGrGaxH8U263aHKFYj6+IKaPQ++RyL7L978fNCsnNuy8gnSynDMf1ddrGcIp0\nYIMXll3+58JO7EHvb2GQjhi6dPX057budvHfX3YJKFHnaDvXBBDCyV8V5lWrl5dV\n3QIDAQAB\n-----END PUBLIC KEY-----"),
			//},
			Signer: &acPb.Member{
				OrgId:      "org1",
				MemberType: acPb.MemberType_CERT,
				MemberInfo: []byte("-----BEGIN CERTIFICATE-----\nMIIChzCCAi2gAwIBAgIDAwGbMAoGCCqGSM49BAMCMIGKMQswCQYDVQQGEwJDTjEQ\nMA4GA1UECBMHQmVpamluZzEQMA4GA1UEBxMHQmVpamluZzEfMB0GA1UEChMWd3gt\nb3JnMS5jaGFpbm1ha2VyLm9yZzESMBAGA1UECxMJcm9vdC1jZXJ0MSIwIAYDVQQD\nExljYS53eC1vcmcxLmNoYWlubWFrZXIub3JnMB4XDTIwMTIwODA2NTM0M1oXDTI1\nMTIwNzA2NTM0M1owgY8xCzAJBgNVBAYTAkNOMRAwDgYDVQQIEwdCZWlqaW5nMRAw\nDgYDVQQHEwdCZWlqaW5nMR8wHQYDVQQKExZ3eC1vcmcxLmNoYWlubWFrZXIub3Jn\nMQ4wDAYDVQQLEwVhZG1pbjErMCkGA1UEAxMiYWRtaW4xLnNpZ24ud3gtb3JnMS5j\naGFpbm1ha2VyLm9yZzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABORqoYNAw8ax\n9QOD94VaXq1dCHguarSKqAruEI39dRkm8Vu2gSHkeWlxzvSsVVqoN6ATObi2ZohY\nKYab2s+/QA2jezB5MA4GA1UdDwEB/wQEAwIBpjAPBgNVHSUECDAGBgRVHSUAMCkG\nA1UdDgQiBCDZOtAtHzfoZd/OQ2Jx5mIMgkqkMkH4SDvAt03yOrRnBzArBgNVHSME\nJDAigCA1JD9xHLm3xDUukx9wxXMx+XQJwtng+9/sHFBf2xCJZzAKBggqhkjOPQQD\nAgNIADBFAiEAiGjIB8Wb8mhI+ma4F3kCW/5QM6tlxiKIB5zTcO5E890CIBxWDICm\nAod1WZHJajgnDQ2zEcFF94aejR9dmGBB/P//\n-----END CERTIFICATE-----"),
			},
			Signature: []byte("sign1"),
		},
	}

}

func prepareBlockV300() *commonPb.Block {
	return &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        "Chain01",
			BlockHeight:    10,
			PreBlockHash:   nil,
			BlockHash:      nil,
			BlockVersion:   uint32(3000000),
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: 0,
			Proposer:       nil,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag: &commonPb.DAG{
			Vertexes: nil,
		},
		Txs: nil,
		AdditionalData: &commonPb.AdditionalData{
			ExtraData: nil,
		},
	}
}

func prepareTxScheduler(ctl *gomock.Controller, chainConfig *configpb.ChainConfig, block *commonPb.Block) protocol.TxScheduler {

	vmMgr := mock.NewMockVmManager(ctl)
	vmMgr.EXPECT().BeforeSchedule(gomock.Any(), gomock.Any()).Return().AnyTimes()
	vmMgr.EXPECT().AfterSchedule(gomock.Any(), gomock.Any()).Return().AnyTimes()
	vmMgr.EXPECT().RunContract(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(contract *commonPb.Contract, method string, byteCode []byte,
			parameters map[string][]byte, txSimContext protocol.TxSimContext, gasUsed uint64, refTxType common.TxType,
		) (*commonPb.ContractResult, protocol.ExecOrderTxType, commonPb.TxStatusCode) {

			fmt.Printf("run contract => %v, %v \n", contract.Name, method)
			if len(txSimContext.GetTx().Payload.TxId) == 64 {
				return &commonPb.ContractResult{
					Code:    0,
					Message: "OK",
				}, protocol.ExecOrderTxTypeNormal, commonPb.TxStatusCode_SUCCESS
			}

			txType := protocol.ExecOrderTxTypeNormal
			if strings.Contains(method, "method-0") {
				txType = protocol.ExecOrderTxTypeIterator
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 0"), []byte("Value 0"))

			} else if strings.Contains(method, "method-1") {
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 1"), []byte("Value 1"))

			} else if strings.Contains(method, "method-2") {
				gasUsed += uint64(90)
				txType = protocol.ExecOrderTxTypeIterator
				txSimContext.Put(contract.Name, []byte("Key 2"), []byte("Value 2"))

			} else if strings.Contains(method, "method-3") {
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 3"), []byte("Value 3"))

			} else if strings.Contains(method, "method-4") {
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 4"), []byte("Value 4"))

			} else if strings.Contains(method, "method-5") {
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 5"), []byte("Value 5"))

			} else if strings.Contains(method, "method-6") {
				gasUsed += uint64(110)
				txSimContext.Put(contract.Name, []byte("Key 6"), []byte("Value 6"))
				return &commonPb.ContractResult{
					Code:    1,
					Message: "gasUsed(110) < gasLimit(100)",
					GasUsed: gasUsed,
				}, txType, commonPb.TxStatusCode_CONTRACT_FAIL

			} else if strings.Contains(method, "method-7") {
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 7"), []byte("Value 7"))

			} else if strings.Contains(method, "method-8") {
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 8"), []byte("Value 8"))

			} else if strings.Contains(method, "method-9") {
				gasUsed += uint64(90)
				txSimContext.Put(contract.Name, []byte("Key 9"), []byte("Value 9"))
			}

			return &commonPb.ContractResult{
				Code:    0,
				Message: "OK",
				GasUsed: gasUsed,
			}, txType, commonPb.TxStatusCode_SUCCESS
		}).AnyTimes()

	chainConf := mock.NewMockChainConf(ctl)
	chainConf.EXPECT().ChainConfig().Return(chainConfig).AnyTimes()

	storeHelper := mock.NewMockStoreHelper(ctl)
	storeHelper.EXPECT().GetPoolCapacity().Return(runtime.NumCPU() * 4).AnyTimes()

	ledgerCache := mock.NewMockLedgerCache(ctl)
	ledgerCache.EXPECT().CurrentHeight().Return(block.Header.BlockHeight-1, nil).AnyTimes()

	var schedulerFactory scheduler.TxSchedulerFactory
	scheduler := schedulerFactory.NewTxScheduler(vmMgr, chainConf, storeHelper, ledgerCache)

	return scheduler
}

func prepareTxBatch(size int) (*commonPb.Contract, []*commonPb.Transaction) {

	contract := commonPb.Contract{
		Name:        "TestContract-1",
		Version:     "1",
		RuntimeType: commonPb.RuntimeType_WASMER,
	}
	var txBatch []*commonPb.Transaction

	for i := 0; i < size; i++ {
		tx := prepareTx(
			fmt.Sprintf("a%d", i),
			&contract,
			fmt.Sprintf("method-%d", i),
			map[string]string{},
		)
		txBatch = append(txBatch, tx)
	}

	return &contract, txBatch
}

func prepareSnapshot(ctl *gomock.Controller, chainConfig *configpb.ChainConfig,
	contract *commonPb.Contract,
	block *commonPb.Block, size int) protocol.Snapshot {

	blockStore := mock.NewMockBlockchainStore(ctl)
	blockStore.EXPECT().GetLastChainConfig().Return(chainConfig, nil).AnyTimes()
	blockStore.EXPECT().GetContractByName(gomock.Any()).Return(contract, nil).AnyTimes()
	blockStore.EXPECT().GetContractBytecode(gomock.Any()).Return([]byte("Dummy Contract code !!!"), nil).AnyTimes()
	blockStore.EXPECT().ReadObject(
		syscontract.SystemContract_ACCOUNT_MANAGER.String(),
		gomock.Any()).Return([]byte("640"), nil).AnyTimes()

	var snapshotFactory Factory
	snapshotManager := snapshotFactory.NewSnapshotManager(blockStore, logger.GetLogger("TestUnit"))
	snapshot := snapshotManager.NewSnapshot(nil, block)
	return snapshot
}

func prepareLocalConfig(configPath string) (*localconf.CMConfig, error) {
	configFile, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	cmViper := viper.New()
	cmViper.SetConfigFile(configFile)
	if err = cmViper.ReadInConfig(); err != nil {
		return nil, err
	}
	chainmakerConfig := localconf.CMConfig{}
	if err = cmViper.Unmarshal(&chainmakerConfig); err != nil {
		return nil, err
	}

	return &chainmakerConfig, nil
}

func Test_TxSchedule_BuildDAG(t *testing.T) {

	defaultLogConfig := logger.DefaultLogConfig()
	defaultLogConfig.SystemLog.LogLevelDefault = log.INFO
	defaultLogConfig.BriefLog.LogLevelDefault = log.INFO
	defaultLogConfig.EventLog.LogLevelDefault = log.INFO
	logger.SetLogConfig(defaultLogConfig)
	ctl := gomock.NewController(t)

	chainConfig := &configpb.ChainConfig{
		Version: "3000000",
		Crypto: &configpb.CryptoConfig{
			Hash: crypto2.CRYPTO_ALGO_SHA256,
		},
		Contract: &configpb.ContractConfig{
			EnableSqlSupport: false,
		},
		Core: &configpb.CoreConfig{
			EnableOptimizeChargeGas: true,
		},
		AccountConfig: &configpb.GasAccountConfig{
			EnableGas: true,
		},
		AuthType: protocol.Identity,
		Vm: &configpb.Vm{
			AddrType: configpb.AddrType_CHAINMAKER,
		},
		Consensus: &configpb.ConsensusConfig{
			Type: consensus.ConsensusType_TBFT,
		},
	}

	// 设置 local config
	chainmakerConfig, err := prepareLocalConfig("./testdata/chainmaker.yml")
	assert.Nil(t, err)
	localconf.ChainMakerConfig = chainmakerConfig

	// 构造 Block
	block := prepareBlockV300()

	// 构造 TxScheduler
	scheduler := prepareTxScheduler(ctl, chainConfig, block)

	// 构造 txBatch
	contract, txBatch := prepareTxBatch(10)

	// 构造 snapshot
	snapshot := prepareSnapshot(ctl, chainConfig, contract, block, 10)

	txRWSets, txEvents, err := scheduler.Schedule(block, txBatch, snapshot)
	assert.Nil(t, err)
	assert.NotNil(t, txRWSets)
	assert.NotNil(t, txEvents)

	//fmt.Printf("block dag = %v \n", block.Dag)
	successNum := 0
	for i, tx := range block.Txs {
		fmt.Printf("%v) block tx => id = %v, result = %v \n",
			i, tx.Payload.TxId, tx.Result.ContractResult)

		if tx.Result.Code == commonPb.TxStatusCode_SUCCESS {
			successNum++
		}
	}
	assert.Equal(t, 7, successNum)
	fmt.Printf("block.Dag = %v \n", block.Dag)
}
