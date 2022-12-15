package cutover

import (
	"encoding/hex"

	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/protocol/v3"
	"github.com/gogo/protobuf/proto"
)

//ConsensusCarrier carryies the consensus algorithm which can operate consensus algorithm switching
type ConsensusCarrier interface {
	//SwitchConsensus do consensus switching
	//ConsensusConfig is the consensus config data used by consensus algorithm to switch
	SwitchConsensus(*config.ConsensusConfig) error
}

//ConsensusSwitchSubscriber listen for consensus algorithm type changing
type ConsensusSwitchSubscriber struct {
	consensusCarrier ConsensusCarrier
	consensusConfig  config.ConsensusConfig
	log              protocol.Logger
}

//NewConsensusSwitchSubscriber create a new ConsensusSwitchSubscriber instance
//carrier: implement consensus algorithm switching function
//cf: current consensus configuration information，ConsensusSwitchSubscriber will save a copy of the
//consensus config to prevent the modification of the upper layer from affecting the logic judgment
//log used to output log information
func NewConsensusSwitchSubscriber(
	carrier ConsensusCarrier,
	cf *config.ConsensusConfig,
	log protocol.Logger) *ConsensusSwitchSubscriber {
	return &ConsensusSwitchSubscriber{
		consensusCarrier: carrier,
		consensusConfig:  *cf,
		log:              log,
	}
}

//OnMessage obtain chain configuration change data and compare with the old consensus type
//if the consensus type of chain configuration data does't match he old consensus type
//do consensus switching with consensusCarrier and update the consensus config
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
			_ = cs.consensusCarrier.SwitchConsensus(chainConfig.GetConsensus())
			cs.consensusConfig = *chainConfig.GetConsensus()
		}
	}
}

//OnQuit nothing to do
func (cs *ConsensusSwitchSubscriber) OnQuit() {
	// nothing for implement interface msgbus.Subscriber
}
