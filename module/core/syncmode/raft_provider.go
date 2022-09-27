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

// ConsensusTypeRAFT consensus type raft
const ConsensusTypeRAFT = "RAFT"

// NilRAFTProvider nil raft provider
var NilRAFTProvider provider.CoreProvider = (*raftProvider)(nil)

// raft provider
type raftProvider struct {
}

// NewCoreEngine by raft provider
func (rp *raftProvider) NewCoreEngine(config *conf.CoreEngineConfig) (protocol.CoreEngine, error) {
	return NewCoreEngine(config)
}
