package consensus

import "chainmaker.org/chainmaker/protocol/v2"

type ConsensusStateBundle struct {
	protocol.ConsensusState
}

func NewConsensusStateWrapper() protocol.ConsensusStateWrapper {
	return &ConsensusStateBundle{}
}

func (c *ConsensusStateBundle) Wrap(cs protocol.ConsensusState) {
	c.ConsensusState = cs
}

func (c *ConsensusStateBundle) IsVaild() bool {
	return c.ConsensusState != nil
}

func (c *ConsensusStateBundle) GetAllNodeInfos() []protocol.ConsensusNodeInfo {
	if !c.IsVaild() {
		return nil
	}
	return c.ConsensusState.GetAllNodeInfos()
}
