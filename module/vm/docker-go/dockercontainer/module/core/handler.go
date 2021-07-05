package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"contract-sdk-test1/pb_sdk/protogo"
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

// Handler used to handle each sandbox's message
// to deal with each contract message
type Handler struct {
	user        *helper.User
	logger      *zap.SugaredLogger
	tx          *outside.TxRequest
	state       state
	handlerName string

	stream    protogo.Contract_ContactServer
	scheduler protocol.Scheduler

	contractResult *outside.ContractResult
}

func NewHandler(user *helper.User, tx *outside.TxRequest, scheduler protocol.Scheduler, handlerName string) (*Handler, error) {

	loggerName := "[Docker Handler " + handlerName + " ]"

	handler := &Handler{
		logger:         logger.NewDockerLogger(loggerName),
		tx:             tx,
		user:           user,
		state:          created,
		scheduler:      scheduler,
		handlerName:    handlerName,
		contractResult: nil,
	}

	//fmt.Println("------------------")
	//fmt.Println("------------------")
	//fmt.Println(tx.TxId)
	//fmt.Println(tx.Method)
	//fmt.Println(len(tx.ByteCode))
	//fmt.Println(tx.ContractName)
	//fmt.Println(tx.Parameters)
	//udsServer, err := rpcserver.NewUDSRpcServer(user, handler)
	//if err != nil {
	//	return nil, err
	//}

	//handler.UdsServer = udsServer
	handler.logger.Debugf("Handler created for tx: [%s]\n", tx.TxId[:5])
	return handler, nil
}

func (h *Handler) SetStream(stream protogo.Contract_ContactServer) {
	h.stream = stream
}

func (h *Handler) sendMessage(msg *protogo.ContractMessage) error {
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
func (h *Handler) HandleMessage(msg *protogo.ContractMessage) error {
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

func (h *Handler) handleCreated(registerMsg *protogo.ContractMessage) error {
	if registerMsg.Type != protogo.Type_REGISTER {
		return fmt.Errorf("handler [%s] cannot handle message (%s) while in state: %s", registerMsg.HandlerName, registerMsg.Type, h.state)
	}

	registeredMsg := &protogo.ContractMessage{
		Type:        protogo.Type_REGISTERED,
		HandlerName: registerMsg.HandlerName,
		Payload:     nil,
	}

	h.sendMessage(registeredMsg)
	h.state = prepare

	return h.afterRegistered()
}

func (h *Handler) afterRegistered() error {
	if h.state != prepare {
		return fmt.Errorf("contract [%s] handler cannot send prepare message while in state: %s", h.tx.ContractName, h.state)
	}

	prepareMsg := &protogo.ContractMessage{
		Type:        protogo.Type_PREPARE,
		HandlerName: h.handlerName,
		Payload:     nil,
	}

	return h.sendMessage(prepareMsg)
}

// handlePrepare when sandbox send fist ready to server
func (h *Handler) handlePrepare(readyMsg *protogo.ContractMessage) error {
	h.state = ready

	return h.afterFirstReady()
}

func (h *Handler) afterFirstReady() error {

	switch h.tx.Method {
	case "init_contract":
		return h.sendInit()
	case "invoke_contract":
		return h.sendInvoke()
	default:
		return fmt.Errorf("contract [%s] handler cannot send such method: %s", h.tx.ContractName, h.tx.Method)
	}
}

func (h *Handler) sendInit() error {
	initMsg := &protogo.ContractMessage{
		Type:        protogo.Type_INIT,
		HandlerName: h.handlerName,
		Payload:     nil, // put some parameters
	}

	return h.sendMessage(initMsg)
}

func (h *Handler) sendInvoke() error {

	// send args

	argsMap := make(map[string]string)

	for key, value := range h.tx.Parameters {
		argsMap[key] = value
	}

	input := &protogo.Input{Args: argsMap}

	inputPayload, _ := proto.Marshal(input)
	invokeMsg := &protogo.ContractMessage{
		Type:        protogo.Type_INVOKE,
		HandlerName: h.handlerName,
		Payload:     inputPayload, // put some parameters
	}

	return h.sendMessage(invokeMsg)
}

func (h *Handler) handleReady(readyMsg *protogo.ContractMessage) error {
	//if h.state != prepare {
	//	return fmt.Errorf("contract [%s] handler cannot handle ready message (%s) while in state: %s", h.tx.ContractName, readyMsg.Type, h.state)
	//}

	switch readyMsg.Type {
	case protogo.Type_GET_STATE:
		return h.handleGetState(readyMsg)
	case protogo.Type_COMPLETED:
		return h.handleCompleted(readyMsg)
	default:
		return fmt.Errorf("contract [%s] handler cannot handle ready message (%s) while in state: %s", h.tx.ContractName, readyMsg.Type, h.state)
	}

}

func (h *Handler) handleGetState(getStateMsg *protogo.ContractMessage) error {

	// get data from snapshot

	// get data from node

	responseMsg := &protogo.ContractMessage{
		Type:        protogo.Type_RESPONSE,
		HandlerName: h.handlerName,
		Payload:     nil,
	}

	return h.sendMessage(responseMsg)
}

func (h *Handler) handleCompleted(completedMsg *protogo.ContractMessage) error {

	// handle result
	//resultMessage := string(completedMsg.Payload)
	//h.logger.Println("-------------------------")
	//h.logger.Println(resultMessage)

	var response protogo.Response
	err := proto.Unmarshal(completedMsg.Payload, &response)

	if err != nil {
		return err
	}

	contractResult := &outside.ContractResult{}
	if response.Status == 200 {
		contractResult.Code = outside.ContractResultCode_OK
		contractResult.Result = response.Payload
		contractResult.Message = "Success"
	} else {
		contractResult.Code = outside.ContractResultCode_FAIL
		contractResult.Result = response.Payload
		contractResult.Message = "Fail"
	}

	//h.logger.Println("in complete: ", contractResult)

	// give back result to scheduler  -- for multiple tx incoming
	//h.scheduler.GetTxResultCh() <- contractResult

	// save contract result into handler
	h.contractResult = contractResult

	return h.afterCompleted()
}

// afterCompleted send completed to client, client end stream
func (h *Handler) afterCompleted() error {

	responseMsg := &protogo.ContractMessage{
		Type:        protogo.Type_COMPLETED,
		HandlerName: h.handlerName,
		Payload:     nil,
	}

	return h.sendMessage(responseMsg)

}
