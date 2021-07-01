package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"log"
)

type HandlerRegister struct {
	HandlersTable map[string]*Handler
	logger        *log.Logger
}

func NewHandlerRegister() *HandlerRegister {

	return &HandlerRegister{
		HandlersTable: make(map[string]*Handler),
		logger:        utils.NewLogger("Handler Register"),
	}
}

func (hr *HandlerRegister) RegisterNewHandler(handlerName string, handler *Handler) {
	hr.HandlersTable[handlerName] = handler
	hr.logger.Println("register handler: ", handlerName)
}

func (hr *HandlerRegister) FreeHandler(handlerName string) {
	delete(hr.HandlersTable, handlerName)
	hr.logger.Printf("free [%s] handler", handlerName)
}

func (hr *HandlerRegister) GetHandlerByName(handlerName string) *Handler {
	hr.logger.Printf("get [%s] handler", handlerName)
	return hr.HandlersTable[handlerName]
}
