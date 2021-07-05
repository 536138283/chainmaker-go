package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"go.uber.org/zap"
	"sync"
)

type HandlerRegister struct {
	lock          sync.Mutex
	HandlersTable map[string]*Handler
	logger        *zap.SugaredLogger
}

func NewHandlerRegister() *HandlerRegister {

	return &HandlerRegister{
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
	delete(hr.HandlersTable, handlerName)
	hr.logger.Debugf("free [%s] handler", handlerName)
}

func (hr *HandlerRegister) GetHandlerByName(handlerName string) *Handler {
	hr.logger.Debugf("get [%s] handler", handlerName)
	return hr.HandlersTable[handlerName]
}
