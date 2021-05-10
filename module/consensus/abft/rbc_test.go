/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"bytes"
	"sort"
	"testing"

	"chainmaker.org/chainmaker-go/logger"
	abftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
)

var utLogger *logger.CMLogger

var (
	one_node_cfg  Config
	four_node_cfg Config
)

func init() {
	utLogger = logger.GetLogger("ut")
	one_node_cfg = Config{
		logger: utLogger,
		height: 10,
		id:     "id",
		nodeID: "id",
		nodes:  []string{"id"},
	}

	four_node_cfg = Config{
		logger: utLogger,
		height: 10,
		id:     "id1",
		nodeID: "id1",
		nodes:  []string{"id1", "id2", "id3", "id4"},
	}
	one_node_cfg.fillWithDefaults()
	four_node_cfg.fillWithDefaults()
}

func TestNewRBC(t *testing.T) {
	type args struct {
		cfg Config
	}
	tests := []struct {
		name string
		args args
	}{
		{"1 node", args{cfg: one_node_cfg}},
		{"4 node", args{cfg: four_node_cfg}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRBC(tt.args.cfg); got == nil {
				t.Errorf("NewRBC() = %v", got)
			}
		})
	}
}

func TestRBC_Input(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		config  Config
		args    args
		wantErr bool
	}{
		{"1 node", one_node_cfg, args{[]byte("test data")}, false},
		{"4 node", four_node_cfg, args{[]byte("test data")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rbc := NewRBC(tt.config)
			if err := rbc.Input(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("RBC.Input() error = %v, wantErr %v", err, tt.wantErr)
			}

			msgs := rbc.Messages()
			var prfs proofs
			for _, msg := range msgs {
				prf := msg.Acs.Message.(*abftpb.ACSMessage_Rbc).Rbc.Message.(*abftpb.RBCRequest_ProofRequest).ProofRequest
				prfs = append(prfs, prf)
			}
			sort.Sort(prfs)

			shards := make([][]byte, rbc.nodesNum)
			for _, p := range prfs {
				shards[p.Index] = p.Proof[0]
			}
			if err := rbc.enc.Reconstruct(shards); err != nil {
				t.Errorf("rbc.enc.Reconstruct err: %v", err)
			}

			var output []byte
			for _, data := range shards[:rbc.faultsNum+1] {
				output = append(output, data...)
			}

			output = output[:len(tt.args.data)]
			if !bytes.Equal(output, tt.args.data) {
				t.Errorf("RBC.Input() output = %s len: %v, want: %s len: %v", output, len(output), tt.args.data, len(tt.args.data))
			}
		})
	}
}

/*
func TestRBC_Output(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		data   []byte
	}{
		{"1 node", one_node_cfg, []byte("test data")},
		{"4 node", four_node_cfg, []byte("test data")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rbcs []*RBC
			config := tt.config
			for i := range tt.config.nodes {
				config.nodeID = config.nodes[i]
				rbc := NewRBC(config)
				rbcs = append(rbcs, rbc)
			}
			rbcs[0].Input(tt.data)
			msgs := rbcs[0].Messages()

			for _, msg := range msgs {
				for _, rbc := range rbcs {
					if msg.To == rbc.nodeID {
						rbc.HandleMessage(msg.From, msg.Acs.Message.(*abftpb.ACSMessage_Rbc).Rbc)
					}
				}
			}

			var wg sync.WaitGroup
			wg.Add(len(config.nodes))

			for _, rbc := range rbcs {
				if got := rbc.Output(); !reflect.DeepEqual(got, tt.data) {
					t.Errorf("RBC.Output() = %v, data %v", got, tt.data)
				}
			}
		})
	}
}
*/
