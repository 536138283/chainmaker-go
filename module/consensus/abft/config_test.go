/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_clone(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want *Config
	}{
		{"1 node", one_node_cfg.clone(), one_node_cfg},
		{"3 node", three_node_cfg.clone(), three_node_cfg},
		{"4 node", four_node_cfg.clone(), four_node_cfg},
		{"7 node", seven_node_cfg.clone(), seven_node_cfg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.clone()
			assert.Equal(t, tt.want.height, got.height)
			assert.Equal(t, tt.want.id, got.id)
			assert.Equal(t, tt.want.nodeID, got.nodeID)
			assert.Equal(t, tt.want.nodes, got.nodes)
			assert.Equal(t, tt.want.nodesNum, got.nodesNum)
			assert.Equal(t, tt.want.faultsNum, got.faultsNum)
		})
	}
}

func TestConfig_fillWithDefaults(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *Config
		wantNodesNum  int
		wantFaultsNum int
	}{
		{"1 node", one_node_cfg.clone(), 1, 0},
		{"3 node", three_node_cfg.clone(), 3, 0},
		{"4 node", four_node_cfg.clone(), 4, 1},
		{"7 node", seven_node_cfg.clone(), 7, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.fillWithDefaults()
			assert.Equal(t, tt.wantNodesNum, tt.cfg.nodesNum)
			assert.Equal(t, tt.wantFaultsNum, tt.cfg.faultsNum)
		})
	}
}
