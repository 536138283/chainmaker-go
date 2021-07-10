package rpc

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	SDKProtogo "contract-sdk-test1/pb_sdk/protogo"
	"fmt"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type state string

const (
	created state = "created"

	prepare state = "prepare"

	ready state = "ready"
)

// DMSHandler used to handle each sandbox's message
// to deal with each contract message
type DMSHandler struct {
	handlerName string
	state       state
	logger      *zap.SugaredLogger

	user      *helper.User
	txRequest *protogo.TxRequest

	stream    SDKProtogo.Contract_ContactServer
	scheduler protocol.Scheduler
}

func NewDMSHandler(user *helper.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler, handlerName string) (*DMSHandler, error) {

	loggerName := "[DMS Handler " + handlerName + " ]"

	handler := &DMSHandler{
		logger:      logger.NewDockerLogger(loggerName),
		txRequest:   txRequest,
		user:        user,
		state:       created,
		scheduler:   scheduler,
		handlerName: handlerName,
	}

	handler.logger.Debugf("Handler created for tx: [%s]\n", txRequest.TxId[:5])
	return handler, nil
}

func (h *DMSHandler) SetStream(stream SDKProtogo.Contract_ContactServer) {
	h.stream = stream
}

func (h *DMSHandler) sendMessage(msg *SDKProtogo.ContractMessage) error {
	h.logger.Debugf("send message [%s]", msg)
	return h.stream.Send(msg)
}

// HandleMessage handle incoming message from sandbox
// the sequence is:
// 1. sandbox send register to server  --> server send back registered
// 2. server send prepare(SimContext) to client --> client send back ready  --> server become ready
// 3. server send parameters to client --> client invoke relative function
// 4. client send get_state to server --> server send back get_state with payload
// 5. client send result to server and close --> server receive result and give result to scheduler
func (h *DMSHandler) HandleMessage(msg *SDKProtogo.ContractMessage) error {
	h.logger.Debugf("handle msg [%s]\n", msg)
	//var err error
	switch h.state {
	case created:
		return h.handleCreated(msg)
	case prepare:
		return h.handlePrepare(msg)
	case ready:
		return h.handleReady(msg)
	}
	return nil
}

func (h *DMSHandler) handleCreated(registerMsg *SDKProtogo.ContractMessage) error {
	if registerMsg.Type != SDKProtogo.Type_REGISTER {
		return fmt.Errorf("handler [%s] cannot handle message (%s) while in state: %s", registerMsg.HandlerName, registerMsg.Type, h.state)
	}

	registeredMsg := &SDKProtogo.ContractMessage{
		Type:        SDKProtogo.Type_REGISTERED,
		HandlerName: registerMsg.HandlerName,
		Payload:     nil,
	}

	if err := h.sendMessage(registeredMsg); err != nil {
		return err
	}
	h.state = prepare

	return h.afterRegistered()
}

func (h *DMSHandler) afterRegistered() error {
	if h.state != prepare {
		return fmt.Errorf("contract [%s] handler cannot send prepare message while in state: %s", h.txRequest.ContractName, h.state)
	}

	prepareMsg := &SDKProtogo.ContractMessage{
		Type:        SDKProtogo.Type_PREPARE,
		HandlerName: h.handlerName,
		Payload:     nil,
	}

	return h.sendMessage(prepareMsg)
}

// handlePrepare when sandbox send fist ready to server
func (h *DMSHandler) handlePrepare(readyMsg *SDKProtogo.ContractMessage) error {
	h.state = ready

	return h.afterFirstReady()
}

func (h *DMSHandler) afterFirstReady() error {

	switch h.txRequest.Method {
	case "init_contract":
		return h.sendInit()
	case "invoke_contract":
		return h.sendInvoke()
	default:
		return fmt.Errorf("contract [%s] handler cannot send such method: %s", h.txRequest.ContractName, h.txRequest.Method)
	}
}

func (h *DMSHandler) sendInit() error {
	initMsg := &SDKProtogo.ContractMessage{
		Type:        SDKProtogo.Type_INIT,
		HandlerName: h.handlerName,
		Payload:     nil, // put some parameters
	}

	return h.sendMessage(initMsg)
}

func (h *DMSHandler) sendInvoke() error {

	// send args

	argsMap := make(map[string]string)

	for key, value := range h.txRequest.Parameters {
		argsMap[key] = value
	}

	input := &SDKProtogo.Input{Args: argsMap}

	inputPayload, _ := proto.Marshal(input)
	invokeMsg := &SDKProtogo.ContractMessage{
		Type:        SDKProtogo.Type_INVOKE,
		HandlerName: h.handlerName,
		Payload:     inputPayload, // put some parameters
	}

	return h.sendMessage(invokeMsg)
}

func (h *DMSHandler) handleReady(readyMsg *SDKProtogo.ContractMessage) error {
	//if h.state != prepare {
	//	return fmt.Errorf("contract [%s] handler cannot handle ready message (%s) while in state: %s", h.tx.ContractName, readyMsg.Type, h.state)
	//}

	switch readyMsg.Type {
	case SDKProtogo.Type_GET_STATE:
		return h.handleGetState(readyMsg)
	case SDKProtogo.Type_COMPLETED:
		return h.handleCompleted(readyMsg)
	default:
		return fmt.Errorf("contract [%s] handler cannot handle ready message (%s) while in state: %s", h.txRequest.ContractName, readyMsg.Type, h.state)
	}

}

func (h *DMSHandler) handleGetState(getStateMsg *SDKProtogo.ContractMessage) error {

	// get data from snapshot

	// get data from node

	responseMsg := &SDKProtogo.ContractMessage{
		Type:        SDKProtogo.Type_RESPONSE,
		HandlerName: h.handlerName,
		Payload:     nil,
	}

	return h.sendMessage(responseMsg)
}

func (h *DMSHandler) handleCompleted(completedMsg *SDKProtogo.ContractMessage) error {

	var response SDKProtogo.Response
	err := proto.Unmarshal(completedMsg.Payload, &response)

	if err != nil {
		return err
	}

	txResponse := &protogo.TxResponse{
		TxId: h.txRequest.TxId,
	}
	if response.Status == 200 {
		txResponse.Code = protogo.ContractResultCode_OK
		txResponse.Result = response.Payload
		txResponse.Message = "Success"
	} else {
		txResponse.Code = protogo.ContractResultCode_FAIL
		txResponse.Result = response.Payload
		txResponse.Message = "Fail"
	}

	// give back result to scheduler  -- for multiple tx incoming
	h.scheduler.GetTxResponseCh() <- txResponse

	return h.afterCompleted()
}

// afterCompleted send completed to client, client end stream
func (h *DMSHandler) afterCompleted() error {

	responseMsg := &SDKProtogo.ContractMessage{
		Type:        SDKProtogo.Type_COMPLETED,
		HandlerName: h.handlerName,
		Payload:     nil,
	}

	return h.sendMessage(responseMsg)

}
