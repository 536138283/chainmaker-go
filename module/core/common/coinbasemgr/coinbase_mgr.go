package coinbasemgr

import (
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
)

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