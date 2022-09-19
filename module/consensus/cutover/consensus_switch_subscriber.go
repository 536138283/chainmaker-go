package cutover

import (
	"encoding/hex"

	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/gogo/protobuf/proto"
)

type ConsensusCarrier interface {
	SwitchConsensus(*config.ConsensusConfig) error
}

type ConsensusSwitchSubscriber struct {
	consensusCarrier ConsensusCarrier
	consensusConfig  config.ConsensusConfig
	log              protocol.Logger
}

func NewConsensusSwitchSubscriber(carrier ConsensusCarrier, cf *config.ConsensusConfig, log protocol.Logger) *ConsensusSwitchSubscriber {
	return &ConsensusSwitchSubscriber{
		consensusCarrier: carrier,
		consensusConfig:  *cf,
		log:              log,
	}
}

func (cs *ConsensusSwitchSubscriber) OnMessage(msg *msgbus.Message) {
	switch msg.Topic {
	case msgbus.ChainConfig:
		dataStr, _ := msg.Payload.([]string)
		dataBytes, err := hex.DecodeString(dataStr[0])
		if err != nil {
			cs.log.Error(err)
			return
		}
		chainConfig := &config.ChainConfig{}
		err = proto.Unmarshal(dataBytes, chainConfig)
		if err != nil {
			cs.log.Error(err)
			return
		}
		if chainConfig.GetConsensus().Type != cs.consensusConfig.Type {
			cs.consensusCarrier.SwitchConsensus(chainConfig.GetConsensus())
			cs.consensusConfig = *chainConfig.GetConsensus()
		}
	}
}

func (cs *ConsensusSwitchSubscriber) OnQuit() {
	// nothing for implement interface msgbus.Subscriber
}
