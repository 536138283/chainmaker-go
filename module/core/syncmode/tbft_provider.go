/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package syncmode

import (
	"chainmaker.org/chainmaker-go/module/core/provider"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v3"
)

// ConsensusTypeTBFT consensys type tbft
const ConsensusTypeTBFT = "TBFT"

// NilTBFTProvider nil tbft provider
var NilTBFTProvider provider.CoreProvider = (*tbftProvider)(nil)

// tbftProvider tbft provider
type tbftProvider struct {
}

// NewCoreEngine by tbft provider
func (tp *tbftProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
