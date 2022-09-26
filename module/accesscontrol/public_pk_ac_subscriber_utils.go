/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"fmt"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"

	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
)

func (p *pkACProvider) messageChainConfig(chainConfig *config.ChainConfig, fromMaxBFT bool) {
	p.hashType = chainConfig.GetCrypto().GetHash()

	updateTrustRootAndMemberFunc := func() {
		err := p.initAdminMembers(chainConfig.TrustRoots)
		if err != nil {
			err = fmt.Errorf("new public AC provider failed: %s", err.Error())
			p.log.Error(err)
		}

		err = p.initConsensusMember(chainConfig)
		if err != nil {
			err = fmt.Errorf("new public AC provider failed: %s", err.Error())
			p.log.Error(err)
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

func (p *pkACProvider) onMessageMaxbftChainconfigInEpoch(msg *msgbus.Message) {
	epochConfig, ok := msg.Payload.(*maxbft.GovernanceContract)
	if !ok {
		p.log.Error("payload is not *maxbft.GovernanceContract")
		return
	}

	//update chainconfig
	p.messageChainConfig(epochConfig.ChainConfig, true)
}
