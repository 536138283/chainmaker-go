/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abftmode

import (
	"chainmaker.org/chainmaker-go/module/core/provider"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v3"
)

// ConsensusTypeABFT consensus type ABFT
const ConsensusTypeABFT = "ABFT"

// NilABFTProvider nil variable provider
var NilABFTProvider provider.CoreProvider = (*abftProvider)(nil)

type abftProvider struct {
}

func (ap *abftProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
