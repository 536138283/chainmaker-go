package accesscontrol

import (
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"github.com/gogo/protobuf/proto"
)

var _ msgbus.Subscriber = (*pkACProvider)(nil)

// OnMessage contract event data is a []string, hexToString(proto.Marshal(data))
func (p *pkACProvider) OnMessage(msg *msgbus.Message) {
	// TODO  implement
	switch msg.Topic {
	case msgbus.ChainConfig:
		p.onMessageChainConfig(msg)
	}

}

func (p *pkACProvider) OnQuit() {

}

func (p *pkACProvider) onMessageChainConfig(msg *msgbus.Message) {
	dataStr := msg.Payload.([]string)
	dataBytes, err := hex.DecodeString(dataStr[0])
	if err != nil {
		p.log.Error(err)
		return
	}
	chainConfig := &config.ChainConfig{}
	proto.Unmarshal(dataBytes, chainConfig)

	p.hashType = chainConfig.GetCrypto().GetHash()
	err = p.initAdminMembers(chainConfig.TrustRoots)
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
