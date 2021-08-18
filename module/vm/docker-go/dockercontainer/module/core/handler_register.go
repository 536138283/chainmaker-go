/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpc"
	"go.uber.org/zap"
)

type HandlerRegister struct {
	lock          sync.RWMutex
	HandlersTable map[string]*rpc.DMSHandler
	logger        *zap.SugaredLogger
}

func NewHandlerRegister() *HandlerRegister {

	return &HandlerRegister{
		lock:          sync.RWMutex{},
		HandlersTable: make(map[string]*rpc.DMSHandler),
		logger:        logger.NewDockerLogger(logger.MODULE_HANDLER_REGISTER),
	}
}

func (hr *HandlerRegister) RegisterNewHandler(handlerName string, handler *rpc.DMSHandler) {
	hr.lock.Lock()
	defer hr.lock.Unlock()
	hr.HandlersTable[handlerName] = handler
	hr.logger.Debugf("register handler: [%s]", handlerName)
}

func (hr *HandlerRegister) FreeHandler(handlerName string) {
	hr.lock.Lock()
	defer hr.lock.Unlock()
	delete(hr.HandlersTable, handlerName)
	hr.logger.Debugf("free [%s] handler", handlerName)
}

func (hr *HandlerRegister) GetHandlerByName(handlerName string) *rpc.DMSHandler {
	hr.lock.RLock()
	defer hr.lock.RUnlock()
	hr.logger.Debugf("get [%s] handler", handlerName)
	return hr.HandlersTable[handlerName]
}
