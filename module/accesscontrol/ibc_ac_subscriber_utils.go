/*
Copyright (C) Beijing Advanced Innovation Center for Future Blockchain and Privacy Computing (未来区块链与隐
私计算⾼精尖创新中⼼). All rights reserved.
SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/json"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/maxbft"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"github.com/gogo/protobuf/proto"
)

func (ip *ibcACProvider) messageChainConfig(chainConfig *config.ChainConfig, fromMaxBFT bool) {
	ip.acService.hashType = chainConfig.GetCrypto().GetHash()
	ip.acService.initResourcePolicy(chainConfig.ResourcePolicies, ip.localOrg.id)

	updateTrustRootAndMemberFunc := func() {
		err := ip.initTrustRootsForUpdatingChainConfig(chainConfig, ip.localOrg.id)
		if err != nil {
			ip.acService.log.Error(err)
			return
		}

		ip.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
		ip.opts.KeyUsages[0] = x509.ExtKeyUsageAny

		ip.acService.memberCache.Clear()

		if err := ip.initIBCMasterKey(chainConfig.IbcMasterKeys); err != nil {
			ip.acService.log.Error(err)
		}
	}

	//if consensus is maxbft, delay update
	if ip.consensusType != consensus.ConsensusType_MAXBFT {
		updateTrustRootAndMemberFunc()
	} else {
		if fromMaxBFT {
			updateTrustRootAndMemberFunc()
		}
	}
}

//loadChainConfigFromGovernance used to load config from system contract, only for maxbft
func (ip *ibcACProvider) loadChainConfigFromGovernance() (*maxbft.GovernanceContract, error) {
	contractName := syscontract.SystemContract_GOVERNANCE.String()
	bz, err := ip.store.ReadObject(contractName, []byte(contractName))
	if err != nil {
		return nil, fmt.Errorf("get contractName=%s from db failed, reason: %s", contractName, err)
	}
	if bz == nil {
		return nil, nil
	}
	cfg := &maxbft.GovernanceContract{}
	if err = proto.Unmarshal(bz, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal contractName=%s failed, reason: %s", contractName, err)
	}
	return cfg, nil
}

// onMessageMaxbftChainconfigInEpoch update ac for maxbft
/*
	1. if not maxbft, return
	2. update consensusType if change (not effective now, need consensus module support)
	3. refresh trustroots
	4. refresh trustmembers
	5. refresh freezeList
	6. refresh crlList
*/
func (ip *ibcACProvider) onMessageMaxbftChainconfigInEpoch(msg *msgbus.Message) {
	epochConfig, ok := msg.Payload.(*maxbft.GovernanceContract)
	if !ok {
		ip.acService.log.Error("payload is not *maxbft.GovernanceContract")
		return
	}

	//update chainconfig
	ip.messageChainConfig(epochConfig.ChainConfig, true)
	if err := ip.updateFrozenAndCRL(epochConfig); err != nil {
		ip.acService.log.Errorf("fail to updateFrozenAndCRL: %s", err.Error())
	}
}

func (ip *ibcACProvider) updateFrozenAndCRL(epochConfig *maxbft.GovernanceContract) error {
	//update frozenList
	if len(epochConfig.CertFrozenList) != 0 {
		var certIDs []string
		if err := json.Unmarshal(epochConfig.CertFrozenList, &certIDs); err != nil {
			return fmt.Errorf("unmarshal frozen certificate list failed: %v", err)
		}
		for _, certID := range certIDs {
			certBytes, err := ip.acService.dataStore.
				ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certID))
			if err != nil {
				return fmt.Errorf("load frozen certificate failed: %s", certID)
			}
			if certBytes == nil {
				return fmt.Errorf("load frozen certificate failed: empty certificate [%s]", certID)
			}

			certBlock, _ := pem.Decode(certBytes)
			ip.frozenList.Store(string(certBlock.Bytes), true)
		}
	}

	//update crl
	if len(epochConfig.CRL) != 0 {
		var crlAKIs []string
		if err := json.Unmarshal(epochConfig.CRL, &crlAKIs); err != nil {
			return fmt.Errorf("fail to Unmarshal CRL list: %v", err)
		}
		if err := ip.storeCrls(crlAKIs); err != nil {
			return fmt.Errorf("fail to update CRL list: %v", err)
		}
	}
	return nil
}
