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

// ConsensusTypeDPOS consensus type dops
const ConsensusTypeDPOS = "DPOS"

// NilDPOSProvider nil dpos provider
var NilDPOSProvider provider.CoreProvider = (*dposProvider)(nil)

// dposProvider dpos provider
type dposProvider struct {
}

// NewCoreEngine new core engine by dpos provider
func (tp *dposProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
