package coinbasemgr

import (
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/protocol/v2"
)

func CheckCoinbaseEnable(chainConf protocol.ChainConf) bool {

	if chainConf.ChainConfig().AccountConfig == nil {
		return false
	}

	return chainConf.ChainConfig().AccountConfig.EnableGas ||
		chainConf.ChainConfig().Consensus.Type == consensuspb.ConsensusType_DPOS
}
