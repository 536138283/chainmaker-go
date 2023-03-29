package scheduler

import (
	"testing"

	"chainmaker.org/chainmaker/logger/v2"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCalcInvokeTxGasUsedWithEmptyParameters(t *testing.T) {
	ctl := gomock.NewController(t)

	logger := logger.GetLogger("test")
	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ContractName: "test-contract-1",
			Method:       "test-method-1",
			TxId:         "test-transaction-id-12345",
		},
	}
	chainConfig := &config.ChainConfig{
		AccountConfig: &config.GasAccountConfig{
			EnableGas:       true,
			DefaultGas:      uint64(1000),
			DefaultGasPrice: float32(1),
			InstallBaseGas:  uint64(1000000),
			InstallGasPrice: float32(0.1),
		},
	}
	txSimContext := mock.NewMockTxSimContext(ctl)
	txSimContext.EXPECT().GetBlockVersion().Return(uint32(2030102)).AnyTimes()
	txSimContext.EXPECT().GetLastChainConfig().Return(chainConfig).AnyTimes()
	txSimContext.EXPECT().GetTx().Return(tx).AnyTimes()

	gasUsed, err := calcTxGasUsed(txSimContext, logger)
	assert.Nil(t, err)
	assert.Equal(t, gasUsed, uint64(1000))
}

func TestCalcInvokeTxGasUsed(t *testing.T) {
	ctl := gomock.NewController(t)

	logger := logger.GetLogger("test")
	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ContractName: "test-contract-1",
			Method:       "test-method-1",
			TxId:         "test-transaction-id-12345",
			Parameters: []*commonPb.KeyValuePair{
				{Key: "Key-1", Value: []byte("value-1")},
				{Key: "Key-2", Value: []byte("value-2")},
			},
		},
	}
	chainConfig := &config.ChainConfig{
		AccountConfig: &config.GasAccountConfig{
			EnableGas:       true,
			DefaultGas:      uint64(1000),
			DefaultGasPrice: float32(2),
			InstallBaseGas:  uint64(1000000),
			InstallGasPrice: float32(1),
		},
	}
	txSimContext := mock.NewMockTxSimContext(ctl)
	txSimContext.EXPECT().GetBlockVersion().Return(uint32(2030102)).AnyTimes()
	txSimContext.EXPECT().GetLastChainConfig().Return(chainConfig).AnyTimes()
	txSimContext.EXPECT().GetTx().Return(tx).AnyTimes()

	gasUsed, err := calcTxGasUsed(txSimContext, logger)
	assert.Nil(t, err)
	assert.Equal(t, gasUsed, uint64(1048))
}

func TestCalcInstallTxGasUsed(t *testing.T) {
	ctl := gomock.NewController(t)

	logger := logger.GetLogger("test")
	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ContractName: syscontract.SystemContract_CONTRACT_MANAGE.String(),
			Method:       syscontract.ContractManageFunction_INIT_CONTRACT.String(),
			TxId:         "test-transaction-id-12345",
			Parameters: []*commonPb.KeyValuePair{
				{Key: "Key-1", Value: []byte("value-1")},
				{Key: "Key-2", Value: []byte("value-2")},
			},
		},
	}
	chainConfig := &config.ChainConfig{
		AccountConfig: &config.GasAccountConfig{
			EnableGas:       true,
			DefaultGas:      uint64(1000),
			DefaultGasPrice: float32(2),
			InstallBaseGas:  uint64(1000000),
			InstallGasPrice: float32(1),
		},
	}
	txSimContext := mock.NewMockTxSimContext(ctl)
	txSimContext.EXPECT().GetBlockVersion().Return(uint32(2030102)).AnyTimes()
	txSimContext.EXPECT().GetLastChainConfig().Return(chainConfig).AnyTimes()
	txSimContext.EXPECT().GetTx().Return(tx).AnyTimes()

	gasUsed, err := calcTxGasUsed(txSimContext, logger)
	assert.Nil(t, err)
	assert.Equal(t, gasUsed, uint64(1000024))
}

func TestCalcMultiSignTxGasUsed(t *testing.T) {
	ctl := gomock.NewController(t)

	logger := logger.GetLogger("test")
	bytecode := [3000]byte{}
	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ContractName: syscontract.SystemContract_MULTI_SIGN.String(),
			Method:       syscontract.MultiSignFunction_REQ.String(),
			TxId:         "test-transaction-id-12345",
			Parameters: []*commonPb.KeyValuePair{
				// 32 bytes
				{
					Key:   "SYS_CONTRACT_NAME",
					Value: []byte("CONTRACT_MANAGE"),
				},
				// 23 bytes
				{
					Key:   "SYS_METHOD",
					Value: []byte("INIT_CONTRACT"),
				},
				// 31 bytes
				{
					Key:   "CONTRACT_NAME",
					Value: []byte("test-contract-name"),
				},
				// 23 bytes
				{
					Key:   "CONTRACT_VERSION",
					Value: []byte("2030102"),
				},
				// 3017 bytes
				{
					Key:   "CONTRACT_BYTECODE",
					Value: bytecode[:],
				},
				// 30 bytes
				{
					Key:   "CONTRACT_RUNTIME_TYPE",
					Value: []byte(commonPb.RuntimeType_DOCKER_GO.String()),
				},
			},
		},
	}
	chainConfig := &config.ChainConfig{
		AccountConfig: &config.GasAccountConfig{
			EnableGas:       true,
			DefaultGas:      uint64(1000),
			DefaultGasPrice: float32(2),
			InstallBaseGas:  uint64(1000000),
			InstallGasPrice: float32(1),
		},
	}
	txSimContext := mock.NewMockTxSimContext(ctl)
	txSimContext.EXPECT().GetBlockVersion().Return(uint32(2030102)).AnyTimes()
	txSimContext.EXPECT().GetLastChainConfig().Return(chainConfig).AnyTimes()
	txSimContext.EXPECT().GetTx().Return(tx).AnyTimes()

	gasUsed, err := calcTxGasUsed(txSimContext, logger)
	assert.Nil(t, err)
	assert.Equal(t, gasUsed, uint64(1003156))
}
