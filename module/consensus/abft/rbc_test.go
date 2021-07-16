/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"bytes"
	"sort"
	"sync"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/logger"
	abftpb "chainmaker.org/chainmaker/pb-go/consensus/abft"
	"github.com/stretchr/testify/assert"
)

var unittestLogger *logger.CMLogger

var (
	one_node_cfg   *Config
	three_node_cfg *Config
	four_node_cfg  *Config
	seven_node_cfg *Config
)

func init() {
	unittestLogger = logger.GetLogger("unittest")
	one_node_cfg = &Config{
		logger: unittestLogger,
		height: 10,
		id:     "id",
		nodeID: "id",
		nodes:  []string{"id"},
	}
	one_node_cfg.fillWithDefaults()

	three_node_cfg = &Config{
		logger: unittestLogger,
		height: 10,
		id:     "id1",
		nodeID: "id1",
		nodes:  []string{"id1", "id2", "id3"},
	}
	three_node_cfg.fillWithDefaults()

	four_node_cfg = &Config{
		logger: unittestLogger,
		height: 10,
		id:     "id1",
		nodeID: "id1",
		nodes:  []string{"id1", "id2", "id3", "id4"},
	}
	four_node_cfg.fillWithDefaults()

	seven_node_cfg = &Config{
		logger: unittestLogger,
		height: 10,
		id:     "id1",
		nodeID: "id1",
		nodes:  []string{"id1", "id2", "id3", "id4", "id5", "id6", "id7"},
	}
	seven_node_cfg.fillWithDefaults()
}

func TestNewRBC(t *testing.T) {
	type args struct {
		cfg *Config
	}
	tests := []struct {
		name string
		args args
	}{
		{"1 node", args{cfg: one_node_cfg.clone()}},
		{"3 node", args{cfg: three_node_cfg.clone()}},
		{"4 node", args{cfg: four_node_cfg.clone()}},
		{"7 node", args{cfg: seven_node_cfg.clone()}},
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
		config  *Config
		args    args
		wantErr bool
	}{
		{"1 node", one_node_cfg.clone(), args{[]byte("test data")}, false},
		{"3 node", three_node_cfg.clone(), args{[]byte("test data")}, false},
		{"4 node", four_node_cfg.clone(), args{[]byte("test data")}, false},
		{"7 node", seven_node_cfg.clone(), args{[]byte("test data")}, false},
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

func TestRBC_Output(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		data   []byte
	}{
		{"1 node", one_node_cfg.clone(), []byte("test data")},
		{"3 node", three_node_cfg.clone(), []byte("test data")},
		{"4 node", four_node_cfg.clone(), []byte("test data")},
		{"7 node", seven_node_cfg.clone(), []byte("test data")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rbcs := make(map[string]*RBC)
			config := tt.config
			for i := range tt.config.nodes {
				cfg := config.clone()
				cfg.nodeID = cfg.nodes[i]
				rbc := NewRBC(cfg)
				rbcs[cfg.nodeID] = rbc
			}

			var wg sync.WaitGroup
			wg.Add(len(config.nodes))

			finishC := make(chan struct{})
			go func() {
				wg.Wait()
				close(finishC)
			}()

			for id, rbc := range rbcs {
				go func(id string, rbc *RBC) {
					defer wg.Done()
					for {
						msgs := rbc.Messages()
						for _, msg := range msgs {
							rbcs[msg.To].HandleMessage(msg.From, msg.Acs.Message.(*abftpb.ACSMessage_Rbc).Rbc)
						}

						if output := rbc.Output(); output != nil {
							assert.Equal(t, tt.data, output)
							return
						}
					}
				}(id, rbc)
			}

			rbcs[config.nodes[0]].Input(tt.data)

			select {
			case <-time.After(time.Second):
				assert.Fail(t, "timeout: RBC not finished")
			case <-finishC:
			}
		})
	}
}
