/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"errors"
	"sync"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpc"
	"go.uber.org/zap"
)

const (
	handlerAlreadyExist = "handler already exist"
	handlerNotExist     = "handler not exist"
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

func (hr *HandlerRegister) RegisterNewHandler(handlerName string, handler *rpc.DMSHandler) error {
	hr.lock.Lock()
	defer hr.lock.Unlock()

	_, ok := hr.HandlersTable[handlerName]
	if ok {
		return errors.New(handlerAlreadyExist)
	}

	hr.HandlersTable[handlerName] = handler
	hr.logger.Debugf("register handler: [%s]", handlerName)
	return nil
}

//func (hr *HandlerRegister) FreeHandler(handlerName string) {
//	hr.lock.Lock()
//	defer hr.lock.Unlock()
//	delete(hr.HandlersTable, handlerName)
//	hr.logger.Infof("free [%s] handler", handlerName)
//}

// GetHandlerByName return handler and delete from register table
func (hr *HandlerRegister) GetHandlerByName(handlerName string) (*rpc.DMSHandler, error) {
	hr.lock.Lock()
	defer hr.lock.Unlock()

	handler, ok := hr.HandlersTable[handlerName]
	if ok {
		hr.logger.Debugf("get [%s] handler", handlerName)
		delete(hr.HandlersTable, handlerName)
		return handler, nil
	}

	return nil, errors.New(handlerNotExist)

}
