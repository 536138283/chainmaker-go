package cutover

import (
	"encoding/hex"
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/common/v3/msgbus"
	"chainmaker.org/chainmaker/logger/v3"
	"chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/pb-go/v3/consensus"
	"chainmaker.org/chainmaker/protocol/v3"
	"github.com/gogo/protobuf/proto"
)

type testConsensusCarrier struct {
}

func (t *testConsensusCarrier) SwitchConsensus(c *config.ConsensusConfig) error {
	fmt.Println("switch consensus to", c.Type.String())
	return nil
}

func TestConsensusSwitchSubscriber_OnMessage(t *testing.T) {
	type fields struct {
		consensusCarrier ConsensusCarrier
		consensusConfig  config.ConsensusConfig
		log              protocol.Logger
	}
	type args struct {
		msg *msgbus.Message
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test",
			fields: fields{
				consensusCarrier: &testConsensusCarrier{},
				consensusConfig: config.ConsensusConfig{
					Type: consensus.ConsensusType_TBFT,
				},
				log: newMockLogger(),
			},
			args: args{
				msg: &msgbus.Message{
					Topic:   msgbus.ChainConfig,
					Payload: genPayload(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ConsensusSwitchSubscriber{
				consensusCarrier: tt.fields.consensusCarrier,
				consensusConfig:  tt.fields.consensusConfig,
				log:              tt.fields.log,
			}
			cs.OnMessage(tt.args.msg)
		})
	}
}

func newMockLogger() protocol.Logger {
	return logger.GetLoggerByChain(logger.MODULE_CONSENSUS, "test_chain_id")
}

func genPayload() []string {
	jsonBz, _ := proto.Marshal(&config.ChainConfig{
		Consensus: &config.ConsensusConfig{
			Type: consensus.ConsensusType_RAFT,
		},
	})
	str := hex.EncodeToString(jsonBz)
	return []string{str}
}
