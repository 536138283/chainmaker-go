package coinbasemgr

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
	consensuspb "chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/protocol/v3"
)

// CheckCoinbaseEnable Check if coinbase is enabled
func CheckCoinbaseEnable(chainConf protocol.ChainConf) bool {

	return IsOptimizeChargeGasEnabled(chainConf) ||
		chainConf.ChainConfig().Consensus.Type == consensuspb.ConsensusType_DPOS
}

// IsOptimizeChargeGasEnabled is optimized charge gas enable
func IsOptimizeChargeGasEnabled(chainConf protocol.ChainConf) bool {
	enableGas := false
	enableOptimizeChargeGas := false
	if chainConf.ChainConfig() == nil || chainConf.ChainConfig().AccountConfig == nil {
		return false
	}

	if chainConf.ChainConfig() == nil || chainConf.ChainConfig().Core == nil {
		return false
	}

	enableGas = chainConf.ChainConfig().AccountConfig.EnableGas
	enableOptimizeChargeGas = chainConf.ChainConfig().Core.EnableOptimizeChargeGas

	return enableGas && enableOptimizeChargeGas
}

// IsCoinBaseTx Returns true if it is a coinbase transaction
func IsCoinBaseTx(tx *commonPb.Transaction) bool {
	if tx == nil || tx.Payload == nil ||
		tx.Payload.ContractName != syscontract.SystemContract_COINBASE.String() ||
		tx.Payload.Method == syscontract.CoinbaseFunction_RUN_COINBASE.String() {
		return false
	}

	return false
}
