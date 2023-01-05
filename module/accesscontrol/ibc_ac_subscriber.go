/*
Copyright (C) Beijing Advanced Innovation Center for Future Blockchain and Privacy Computing (未来区块链与隐
私计算⾼精尖创新中⼼). All rights reserved.
SPDX-License-Identifier: Apache-2.0
*/

package accesscontrol

import (
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/consensus"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"github.com/gogo/protobuf/proto"
)

var _ msgbus.Subscriber = (*ibcACProvider)(nil)

// OnMessage contract event data is a []string, hexToString(proto.Marshal(data))
func (ip *ibcACProvider) OnMessage(msg *msgbus.Message) {
	ip.acService.log.Infof("[AC] receive msg, topic: %s", msg.Topic.String())
	switch msg.Topic {
	case msgbus.ChainConfig:
		ip.onMessageChainConfig(msg)
	case msgbus.CertManageCertsFreeze:
		ip.onMessageCertFreeze(msg)
	case msgbus.CertManageCertsUnfreeze:
		ip.onMessageCertUnFreeze(msg)
	case msgbus.CertManageCertsRevoke:
		ip.onMessageCertRevoke(msg)
	case msgbus.MaxbftEpochConf:
		ip.onMessageMaxbftChainconfigInEpoch(msg)
	}

}

func (ip *ibcACProvider) OnQuit() {
	// nothing
}

func (ip *ibcACProvider) onMessageChainConfig(msg *msgbus.Message) {
	dataStr, _ := msg.Payload.([]string)
	dataBytes, err := hex.DecodeString(dataStr[0])
	if err != nil {
		ip.acService.log.Error(err)
		return
	}
	chainConfig := &config.ChainConfig{}
	_ = proto.Unmarshal(dataBytes, chainConfig)

	ip.messageChainConfig(chainConfig, false)
}

func (ip *ibcACProvider) onMessageCertFreeze(msg *msgbus.Message) {
	data, _ := msg.Payload.([]string)
	certs := data[0]

	certList := strings.Replace(certs, ",", "\n", -1)
	ip.acService.log.Debugf("freeze certs: %s", certList)
	certBlock, rest := pem.Decode([]byte(certList))
	for certBlock != nil {
		if ip.consensusType == consensus.ConsensusType_MAXBFT && isConsensusCert(certBlock.Bytes) {
			ip.acService.log.Debugf("freeze certs delay for maxbft in epoch: %s")
			continue
		}
		ip.frozenList.Store(string(certBlock.Bytes), true)
		certBlock, rest = pem.Decode(rest)
	}
}

func (ip *ibcACProvider) onMessageCertUnFreeze(msg *msgbus.Message) {
	// full or hash cert
	data, _ := msg.Payload.([]string)
	certs := data[0]
	hashes := data[1]

	certList := strings.Replace(certs, ",", "\n", -1)
	ip.acService.log.Debugf("unfreeze cert hashes: %s, certs: %s", hashes, certList)
	certBlock, rest := pem.Decode([]byte(certList))
	for certBlock != nil {
		if ip.consensusType == consensus.ConsensusType_MAXBFT && isConsensusCert(certBlock.Bytes) {
			ip.acService.log.Debugf("unfreeze cert delay for maxbft in epoch: %s")
			continue
		}
		_, ok := ip.frozenList.Load(string(certBlock.Bytes))
		if ok {
			ip.frozenList.Delete(string(certBlock.Bytes))
		}
		certBlock, rest = pem.Decode(rest)
	}

	if hashes != "" {
		certHashes := strings.Split(hashes, ",")
		for _, hash := range certHashes {
			cert, err := ip.acService.dataStore.ReadObject(syscontract.SystemContract_CERT_MANAGE.String(), []byte(hash))
			if err != nil {
				ip.acService.log.Errorf("fail to load compressed certificate from local storage [%s]", hash)
				continue
			}
			if cert == nil {
				ip.acService.log.Warnf("cert id [%s] does not exist in local storage", hash)
				continue
			}
			_, ok := ip.frozenList.Load(string(cert))
			if ok {
				ip.frozenList.Delete(string(cert))
			}
		}
	}

}

func (ip *ibcACProvider) onMessageCertRevoke(msg *msgbus.Message) {
	crl := msg.Payload.([]string)[0]
	crl = strings.Replace(crl, ",", "\n", -1)
	ip.acService.log.Debugf("revoke cert crl: %s", crl)
	crls, err := ip.ValidateCRL([]byte(crl))
	if err != nil {
		err = fmt.Errorf("update CRL failed, invalid CRLS: %v", err)
		ip.acService.log.Error(err)
	}
	for _, crl := range crls {
		aki, _, err := bcx509.GetAKIFromExtensions(crl.TBSCertList.Extensions)
		if err != nil {
			err = fmt.Errorf("update CRL failed: %v", err)
			ip.acService.log.Error(err)
		}
		ip.crl.Store(string(aki), crl)
	}
}
