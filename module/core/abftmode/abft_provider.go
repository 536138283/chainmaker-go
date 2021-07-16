package abftmode

import (
	"chainmaker.org/chainmaker-go/core/provider"
	"chainmaker.org/chainmaker-go/core/provider/conf"
	"chainmaker.org/chainmaker/protocol"
)

const ConsensusTypeABFT = "ABFT"

var NilABFTProvider provider.CoreProvider = (*abftProvider)(nil)

type abftProvider struct {
}

func (ap *abftProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
