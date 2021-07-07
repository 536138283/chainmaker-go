package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"go.uber.org/zap"
	"sync"
)

type HandlerRegister struct {
	lock          sync.RWMutex
	HandlersTable map[string]*Handler
	logger        *zap.SugaredLogger
}

func NewHandlerRegister() *HandlerRegister {

	return &HandlerRegister{
		lock:          sync.RWMutex{},
		HandlersTable: make(map[string]*Handler),
		logger:        logger.NewDockerLogger(logger.MODULE_HANDLER_REGISTER),
	}
}

func (hr *HandlerRegister) RegisterNewHandler(handlerName string, handler *Handler) {
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

func (hr *HandlerRegister) GetHandlerByName(handlerName string) *Handler {
	hr.lock.RLock()
	defer hr.lock.RUnlock()
	hr.logger.Debugf("get [%s] handler", handlerName)
	return hr.HandlersTable[handlerName]
}
