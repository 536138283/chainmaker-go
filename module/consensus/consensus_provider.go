/*

Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package consensus

import (
	utils "chainmaker.org/chainmaker/consensus-utils/v3"
	consensusPb "chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/protocol/v3"
)

// Provider ConsensusEngine provider
type Provider func(config *utils.ConsensusImplConfig) (protocol.ConsensusEngine, error)

var consensusProviders = make(map[consensusPb.ConsensusType]Provider)

// RegisterConsensusProvider register
// @param t
// @param f
func RegisterConsensusProvider(t consensusPb.ConsensusType, f Provider) {
	consensusProviders[t] = f
}

// GetConsensusProvider  get a provider by consensus type
// @param t
// @return Provider
func GetConsensusProvider(t consensusPb.ConsensusType) Provider {
	provider, ok := consensusProviders[t]
	if !ok {
		return nil
	}
	return provider
}
