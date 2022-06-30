/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package syncmode

import (
	"chainmaker.org/chainmaker-go/module/core/provider"
	"chainmaker.org/chainmaker-go/module/core/provider/conf"
	"chainmaker.org/chainmaker/protocol/v2"
)

// ConsensusTypeSOLO consensus type solo
const ConsensusTypeSOLO = "SOLO"

// NilSOLOProvider nil solo provider
var NilSOLOProvider provider.CoreProvider = (*soloProvider)(nil)

// soloProvider solo provider
type soloProvider struct {
}

// NewCoreEngine by solo provider
func (sp *soloProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
