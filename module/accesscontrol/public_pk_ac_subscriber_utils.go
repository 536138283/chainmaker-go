/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/pb-go/v3/consensus/maxbft"
	"chainmaker.org/chainmaker/pb-go/v3/syscontract"
	"chainmaker.org/chainmaker/vm-native/v3/dposmgr"
)

// messageChainConfig
//  @Description: handle chainconfig update message
//  @receiver p
//  @param chainConfig
//  @param fromMaxBFT
//
func (p *pkACProvider) messageChainConfig(chainConfig *config.ChainConfig, fromMaxBFT bool) {
	p.hashType = chainConfig.GetCrypto().GetHash()

	// inner func for update trust root and member
	updateTrustRootAndMemberFunc := func() {
		err := p.initAdminMembers(chainConfig.TrustRoots)
		if err != nil {
			err = fmt.Errorf("new public AC provider failed: %s", err.Error())
			p.log.Error(err)
		}

		if chainConfig.Consensus.Type != consensus.ConsensusType_DPOS {
			err = p.initConsensusMember(chainConfig)
			if err != nil {
				err = fmt.Errorf("new public AC provider failed: %s", err.Error())
				p.log.Error(err)
			}
		}

		p.memberCache.Clear()
	}

	//if consensus is maxbft, delay update
	if p.consensusType != consensus.ConsensusType_MAXBFT {
		updateTrustRootAndMemberFunc()
	} else {
		if fromMaxBFT {
			updateTrustRootAndMemberFunc()
		}
	}
}

func (p *pkACProvider) messageEpoch(epoch *syscontract.Epoch) {
	nodes := make([]*config.OrgConfig, 0)
	orgConfig := &config.OrgConfig{
		OrgId:  DposOrgId,
		NodeId: make([]string, 0, len(epoch.ProposerVector)),
	}
	for _, validator := range epoch.ProposerVector {
		nodeID, err := p.dataStore.ReadObject(syscontract.SystemContract_DPOS_STAKE.String(), dposmgr.ToNodeIDKey(validator))
		if err != nil || len(nodeID) == 0 {
			p.log.Errorf("read nodeID of the validator[%s] failed, reason: %s", validator, err)
			return
		}
		orgConfig.NodeId = append(orgConfig.NodeId, string(nodeID))
	}
	nodes = append(nodes, orgConfig)
	err := p.initDPoSMember(nodes)
	if err != nil {
		err = fmt.Errorf("update chainconfig error: %s", err.Error())
		p.log.Error(err)
	}

	// refresh memberCache because trust root maybe update
	p.memberCache.Clear()

}

func (p *pkACProvider) onMessageMaxbftChainconfigInEpoch(msg *msgbus.Message) {
	epochConfig, ok := msg.Payload.(*maxbft.GovernanceContract)
	if !ok {
		p.log.Error("payload is not *maxbft.GovernanceContract")
		return
	}

	//update chainconfig, delay it
	p.messageChainConfig(epochConfig.ChainConfig, true)
}
