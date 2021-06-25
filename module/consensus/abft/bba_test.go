/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"sync"
	"testing"
	"time"

	abftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
)

func TestBBA_AcceptInput(t *testing.T) {
	bba := NewBBA(one_node_cfg.clone())
	assert.True(t, bba.AcceptInput())

	bba.epoch = 1
	assert.False(t, bba.AcceptInput())

	bba.epoch = 0
	bba.estimated = true
	assert.False(t, bba.AcceptInput())
}

func TestBBA_Input(t *testing.T) {
	type args struct {
		val bool
	}
	tests := []struct {
		name    string
		config  *Config
		args    args
		wantErr bool
	}{
		{"1 node input true", one_node_cfg.clone(), args{val: true}, false},
		{"1 node input false", one_node_cfg.clone(), args{val: false}, false},
		{"3 node input false", three_node_cfg.clone(), args{val: false}, false},
		{"3 node input false", three_node_cfg.clone(), args{val: false}, false},
		{"4 node input false", four_node_cfg.clone(), args{val: false}, false},
		{"4 node input false", four_node_cfg.clone(), args{val: false}, false},
		{"7 node input false", seven_node_cfg.clone(), args{val: false}, false},
		{"7 node input false", seven_node_cfg.clone(), args{val: false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bba := NewBBA(tt.config)
			if err := bba.Input(tt.args.val); (err != nil) != tt.wantErr {
				t.Errorf("BBA.Input() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.True(t, bba.estimated)
			assert.Equal(t, tt.args.val, bba.estimation)
			msgs := bba.Messages()
			assert.Equal(t, len(tt.config.nodes), len(msgs))
			index := 0
			funk.ForEach(msgs, func(msg *abftpb.ABFTMessageReq) {
				assert.Equal(t, tt.config.height, msg.Height)
				assert.Equal(t, tt.config.nodeID, msg.From)
				assert.Equal(t, tt.config.nodes[index], msg.To)
				index += 1
				assert.Equal(t, tt.config.id, msg.Id)

				bbaReq, ok := msg.Acs.Message.(*abftpb.ACSMessage_Bba)
				assert.True(t, ok)

				bbaReqBval, ok := bbaReq.Bba.Message.(*abftpb.BBARequest_Bval)
				assert.True(t, ok)
				assert.Equal(t, uint32(0), bbaReqBval.Bval.Epoch)
				assert.Equal(t, tt.args.val, bbaReqBval.Bval.Value)
			})
		})
	}
}

func TestBBA_Output(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		val    bool
	}{
		{"1 node input true", one_node_cfg.clone(), true},
		{"1 node input false", one_node_cfg.clone(), false},
		{"3 node input false", three_node_cfg.clone(), false},
		{"3 node input false", three_node_cfg.clone(), false},
		{"4 node input false", four_node_cfg.clone(), false},
		{"4 node input false", four_node_cfg.clone(), false},
		{"7 node input false", seven_node_cfg.clone(), false},
		{"7 node input false", seven_node_cfg.clone(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bbas := make(map[string]*BBA)
			config := tt.config
			for i := range tt.config.nodes {
				cfg := config.clone()
				cfg.nodeID = cfg.nodes[i]
				bba := NewBBA(cfg)
				bbas[cfg.nodeID] = bba
			}

			var wg sync.WaitGroup
			wg.Add(len(config.nodes))

			finishC := make(chan struct{})
			go func() {
				wg.Wait()
				close(finishC)
			}()

			for id, bba := range bbas {
				go func(id string, bba *BBA) {
					defer wg.Done()
					for {
						msgs := bba.Messages()
						for _, msg := range msgs {
							bbas[msg.To].HandleMessage(msg.From, msg.Acs.Message.(*abftpb.ACSMessage_Bba).Bba)
						}

						if outputted, output := bba.Output(); outputted {
							assert.Equal(t, tt.val, output)
							return
						}
					}
				}(id, bba)
			}

			for _, bba := range bbas {
				bba.Input(tt.val)
			}

			select {
			case <-time.After(2 * time.Second):
				assert.Fail(t, "timeout: BBA not finished")
			case <-finishC:
			}
		})
	}
}
