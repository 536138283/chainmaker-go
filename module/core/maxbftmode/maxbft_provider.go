/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package maxbftmode

import (
	"chainmaker.org/chainmaker-go/module/core/provider"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v3"
)

// ConsensusTypeMAXBFT consensus type max bft
const ConsensusTypeMAXBFT = "MAXBFT"

// NilTMAXBFTProvider nil max bft provider
var NilTMAXBFTProvider provider.CoreProvider = (*maxbftProvider)(nil)

// maxbftProvider max bft provider
type maxbftProvider struct {
}

// NewCoreEngine new core engine by max bft provider return core engine, error
func (hp *maxbftProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
