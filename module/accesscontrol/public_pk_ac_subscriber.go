package accesscontrol

import "chainmaker.org/chainmaker/common/v2/msgbus"

var _ msgbus.Subscriber = (*pkACProvider)(nil)

// OnMessage contract event data is a []string, hexToString(proto.Marshal(data))
func (p *pkACProvider) OnMessage(msg *msgbus.Message) {
	// TODO  implement
	switch msg.Topic {
	case msgbus.ChainConfig:
		//case msgbus.CertManageCertsDelete:
	}

}

func (p *pkACProvider) OnQuit() {

}
