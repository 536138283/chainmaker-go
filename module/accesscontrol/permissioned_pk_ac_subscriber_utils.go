/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"

	"chainmaker.org/chainmaker/pb-go/v2/config"
)

func (pp *permissionedPkACProvider) messageChainConfig(chainConfig *config.ChainConfig, fromMaxBFT bool) {
	pp.acService.hashType = chainConfig.GetCrypto().GetHash()
	pp.acService.initResourcePolicy(chainConfig.ResourcePolicies, pp.localOrg)

	updateTrustRootAndMemberFunc := func() {
		err := pp.initAdminMembers(chainConfig.TrustRoots)
		if err != nil {
			err = fmt.Errorf("update chainconfig error: %s", err.Error())
			pp.acService.log.Error(err)
		}

		err = pp.initConsensusMember(chainConfig.Consensus.Nodes)
		if err != nil {
			err = fmt.Errorf("update chainconfig error: %s", err.Error())
			pp.acService.log.Error(err)
		}

		pp.acService.memberCache.Clear()
	}
	//if consensus is maxbft, delay update
	if pp.consensusType != consensus.ConsensusType_MAXBFT {
		updateTrustRootAndMemberFunc()
	} else {
		if fromMaxBFT {
			updateTrustRootAndMemberFunc()
		}
	}
}

func (pp *permissionedPkACProvider) onMessageMaxbftChainconfigInEpoch(msg *msgbus.Message) {
	epochConfig, ok := msg.Payload.(*maxbft.GovernanceContract)
	if !ok {
		pp.acService.log.Error("payload is not *maxbft.GovernanceContract")
		return
	}
	//update chainconfig
	pp.messageChainConfig(epochConfig.ChainConfig, true)
}
