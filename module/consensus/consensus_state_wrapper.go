/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package consensus

import "chainmaker.org/chainmaker/protocol/v3"

// ConsensusStateBundle a bundle with consensus state instance
type ConsensusStateBundle struct {
	protocol.ConsensusState
}

// NewConsensusStateWrapper create a consensusStateBundle instance
func NewConsensusStateWrapper() protocol.ConsensusStateWrapper {
	return &ConsensusStateBundle{}
}

// Wrap wrap a consensus state instance in the bundle
func (c *ConsensusStateBundle) Wrap(cs protocol.ConsensusState) {
	c.ConsensusState = cs
}

// IsValid check if there is a valid consensus state instance wrapped in it
func (c *ConsensusStateBundle) IsValid() bool {
	return c.ConsensusState != nil
}

// GetAllNodeInfos get state information of all consensus nodes
// if consensus state instance is not in it return nil
func (c *ConsensusStateBundle) GetAllNodeInfos() []protocol.ConsensusNodeInfo {
	if !c.IsValid() {
		return nil
	}
	return c.ConsensusState.GetAllNodeInfos()
}
