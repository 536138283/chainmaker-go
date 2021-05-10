/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"fmt"

	"chainmaker.org/chainmaker-go/logger"
)

type Config struct {
	logger    *logger.CMLogger
	height    int64
	id        string
	nodeID    string
	nodes     []string
	nodesNum  int
	faultsNum int
}

func (c *Config) fillWithDefaults() {
	if c.nodesNum == 0 {
		c.nodesNum = len(c.nodes)
	}
	if c.faultsNum == 0 {
		c.faultsNum = (c.nodesNum - 1) / 3
	}
}

func (c *Config) String() string {
	return fmt.Sprintf("Config height: %v, id: %v, nodeID: %v, nodes: %v", c.height, c.id, c.nodeID, c.nodes)
}
