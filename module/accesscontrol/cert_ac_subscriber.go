package accesscontrol

import (
	"crypto/x509"
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"github.com/gogo/protobuf/proto"
)

var _ msgbus.Subscriber = (*certACProvider)(nil)

// OnMessage contract event data is a []string, hexToString(proto.Marshal(data))
func (cp *certACProvider) OnMessage(msg *msgbus.Message) {
	switch msg.Topic {
	case msgbus.ChainConfig:
		cp.onMessageChainConfig(msg)
	case msgbus.CertManageCertsFreeze:
		cp.onMessageCertFreeze(msg)
	case msgbus.CertManageCertsUnfreeze:
		cp.onMessageCertUnFreeze(msg)
	case msgbus.CertManageCertsRevoke:

		// TODO  alias delete
		//case msgbus.CertManageCertsDelete:
	}

}

func (cp *certACProvider) OnQuit() {

}

func (cp *certACProvider) onMessageChainConfig(msg *msgbus.Message) {
	dataStr := msg.Payload.([]string)
	dataBytes, err := hex.DecodeString(dataStr[0])
	if err != nil {
		cp.acService.log.Error(err)
		return
	}
	chainConfig := &config.ChainConfig{}
	proto.Unmarshal(dataBytes, chainConfig)

	cp.acService.hashType = chainConfig.GetCrypto().GetHash()
	cp.acService.authType = chainConfig.AuthType
	err = cp.initTrustRootsForUpdatingChainConfig(chainConfig, cp.localOrg.id)
	if err != nil {
		cp.acService.log.Error(err)
		return
	}

	cp.acService.initResourcePolicy(chainConfig.ResourcePolicies, cp.localOrg.id)

	cp.opts.KeyUsages = make([]x509.ExtKeyUsage, 1)
	cp.opts.KeyUsages[0] = x509.ExtKeyUsageAny

	cp.acService.memberCache.Clear()
	cp.certCache.Clear()
	err = cp.initTrustMembers(chainConfig.TrustMembers)
	if err != nil {
		cp.acService.log.Error(err)
		return
	}
}

func (cp *certACProvider) onMessageCertFreeze(msg *msgbus.Message) {
	// full cert(s)
	data := msg.Payload.([]string)
	certs := data[0]
	// TODO parse cert list
	fmt.Println(certs)
	//certList := strings.Replace(certs, ",", "\n", -1)
	//certBlock, rest := pem.Decode([]byte(certList))
	//for certBlock != nil {
	//	cp.frozenList.Store(string(certBlock.Bytes), true)
	//	certBlock, rest = pem.Decode(rest)
	//}
}

func (cp *certACProvider) onMessageCertUnFreeze(msg *msgbus.Message) {
	// full or hash cert
	data := msg.Payload.([]string)
	certs := data[0]
	hashes := data[1]

	fmt.Println(hashes, certs)
	// TODO parse cert list
	//certList := strings.Replace(certs, ",", "\n", -1)
	//certBlock, rest := pem.Decode([]byte(certList))
	//for certBlock != nil {
	//	_, ok := cp.frozenList.Load(string(certBlock.Bytes))
	//	if ok {
	//		cp.frozenList.Delete(string(certBlock.Bytes))
	//	}
	//	certBlock, rest = pem.Decode(rest)
	//}
}

func (cp *certACProvider) onMessageCertRevoke(msg *msgbus.Message) {
	// full cert(s)
	crl := msg.Payload.([]string)[0]
	// TODO parse cert list
	fmt.Println(crl)
}
