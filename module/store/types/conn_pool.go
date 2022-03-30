/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package types

import (
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/golang/groupcache/lru"
)

type ConnectionPoolDBHandle struct {
	c   *lru.Cache
	log protocol.Logger
}

func NewConnectionPoolDBHandle(size int, log protocol.Logger) *ConnectionPoolDBHandle {
	cache := lru.New(size)
	cache.OnEvicted = func(key lru.Key, value interface{}) {
		handle, _ := value.(protocol.SqlDBHandle)
		log.Infof("close state sql db for contract:%s", key)
		handle.Close()
	}
	return &ConnectionPoolDBHandle{c: cache, log: log}
}
func (c *ConnectionPoolDBHandle) GetDBHandle(contractName string) (protocol.SqlDBHandle, bool) {
	handle, ok := c.c.Get(contractName)
	if ok {
		return handle.(protocol.SqlDBHandle), true
	}
	return nil, false
}
func (c *ConnectionPoolDBHandle) SetDBHandle(contractName string, handle protocol.SqlDBHandle) {
	c.c.Add(contractName, handle)
}
func (c *ConnectionPoolDBHandle) Clear() {
	c.c.Clear()
}
