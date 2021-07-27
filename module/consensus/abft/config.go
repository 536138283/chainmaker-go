/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"chainmaker.org/chainmaker/protocol"
	"fmt"
	"sync"
)

type Config struct {
	logger protocol.Logger
	sync.Mutex
	height    uint64   // height
	id        string   // id of the RBC or BBA instance
	nodeID    string   // nodeID of current node
	nodes     []string // the list of nodes
	nodesNum  int
	faultsNum int
}

func (c *Config) clone() *Config {
	c.Lock()
	defer c.Unlock()
	cfg := &Config{
		logger:    c.logger,
		height:    c.height,
		id:        c.id,
		nodeID:    c.nodeID,
		nodes:     append([]string(nil), c.nodes...),
		nodesNum:  c.nodesNum,
		faultsNum: c.faultsNum,
	}
	return cfg
}

func (c *Config) fillWithDefaults() {
	c.Lock()
	defer c.Unlock()
	if c.nodesNum == 0 {
		c.nodesNum = len(c.nodes)
	}
	if c.faultsNum == 0 {
		c.faultsNum = (c.nodesNum - 1) / 3
	}
}

func (c *Config) String() string {
	c.Lock()
	defer c.Unlock()
	return fmt.Sprintf("Config height: %v, id: %v, nodeID: %v, nodes: %v", c.height, c.id, c.nodeID, c.nodes)
}
