package rpc

import (
	SDKProtogo "chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"fmt"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type state string

const (
	created state = "created"

	prepared state = "prepared"

	ready state = "ready"
)

// DMSHandler used to handle each sandbox's message
// to deal with each contract message
type DMSHandler struct {
	handlerName  string
	contractName string
	state        state
	logger       *zap.SugaredLogger

	user      *security.User
	txRequest *protogo.TxRequest

	stream    SDKProtogo.DMSRpc_DMSCommunicateServer
	scheduler protocol.Scheduler
}

func NewDMSHandler(user *security.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler, handlerName, contractName string) (*DMSHandler, error) {

	loggerName := "[DMS Handler " + handlerName + " ]"

	handler := &DMSHandler{
		logger:       logger.NewDockerLogger(loggerName),
		txRequest:    txRequest,
		user:         user,
		state:        created,
		scheduler:    scheduler,
		handlerName:  handlerName,
		contractName: contractName,
	}

	handler.logger.Debugf("Handler created for tx: [%s]\n", txRequest.TxId[:5])
	return handler, nil
}

func (h *DMSHandler) SetStream(stream SDKProtogo.DMSRpc_DMSCommunicateServer) {
	h.stream = stream
}

func (h *DMSHandler) sendMessage(msg *SDKProtogo.DMSMessage) error {
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
func (h *DMSHandler) HandleMessage(msg *SDKProtogo.DMSMessage) error {
	h.logger.Debugf("handle msg [%s]\n", msg)
	//var err error
	switch h.state {
	case created:
		return h.handleCreated(msg)
	case prepared:
		return h.handlePrepare(msg)
	case ready:
		return h.handleReady(msg)
	}
	return nil
}

func (h *DMSHandler) handleCreated(registerMsg *SDKProtogo.DMSMessage) error {
	if registerMsg.Type != SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_REGISTER {
		return fmt.Errorf("handler [%s] cannot handle message (%s) while in state: %s", h.handlerName, registerMsg.Type, h.state)
	}

	registeredMsg := &SDKProtogo.DMSMessage{
		Type:         SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_REGISTERED,
		ContractName: registerMsg.ContractName,
		Payload:      nil,
	}

	if err := h.sendMessage(registeredMsg); err != nil {
		h.logger.Errorf("fail to send message : [%v]", registeredMsg)
		return err
	}
	h.state = prepared

	return nil
}

// handlePrepare when sandbox send fist ready to server
func (h *DMSHandler) handlePrepare(readyMsg *SDKProtogo.DMSMessage) error {
	if readyMsg.Type != SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_READY {
		return fmt.Errorf("type not right")
	}
	h.state = ready

	return h.afterFirstReady()
}

func (h *DMSHandler) afterFirstReady() error {

	//todo change with pb
	switch h.txRequest.Method {
	case "init_contract":
		return h.sendInit()
	case "upgrade":
		return h.sendInit()
	case "invoke_contract":
		return h.sendInvoke()
	default:
		return fmt.Errorf("contract [%s] handler cannot send such method: %s", h.txRequest.ContractName, h.txRequest.Method)
	}
}

func (h *DMSHandler) sendInit() error {

	argsMap := make(map[string]string)
	for key, value := range h.txRequest.Parameters {
		argsMap[key] = value
	}

	input := &SDKProtogo.Input{Args: argsMap}
	inputPayload, _ := proto.Marshal(input)

	initMsg := &SDKProtogo.DMSMessage{
		Type:         SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_INIT,
		ContractName: h.contractName,
		Payload:      inputPayload,
	}

	return h.sendMessage(initMsg)
}

func (h *DMSHandler) sendInvoke() error {

	argsMap := make(map[string]string)
	for key, value := range h.txRequest.Parameters {
		argsMap[key] = value
	}

	input := &SDKProtogo.Input{Args: argsMap}

	inputPayload, _ := proto.Marshal(input)
	invokeMsg := &SDKProtogo.DMSMessage{
		Type:         SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_INVOKE,
		ContractName: h.contractName,
		Payload:      inputPayload,
	}

	return h.sendMessage(invokeMsg)
}

func (h *DMSHandler) handleReady(readyMsg *SDKProtogo.DMSMessage) error {

	switch readyMsg.Type {
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_GET_STATE:
		return h.handleGetState(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_COMPLETED:
		return h.handleCompleted(readyMsg)
	default:
		return fmt.Errorf("contract [%s] handler cannot handle ready message (%s) while in state: %s", h.txRequest.ContractName, readyMsg.Type, h.state)
	}

}

func (h *DMSHandler) handleGetState(getStateMsg *SDKProtogo.DMSMessage) error {

	// get data from chain maker
	key := getStateMsg.Payload

	getStateReqMsg := &protogo.CDMMessage{
		TxId:    h.txRequest.TxId,
		Type:    protogo.CDMType_CDM_TYPE_GET_STATE,
		Payload: key,
	}
	getStateResponseCh := make(chan *protogo.CDMMessage)
	h.scheduler.RegisterResponseCh(h.txRequest.TxId, getStateResponseCh)

	// wait to get state response
	h.scheduler.GetGetStateReqCh() <- getStateReqMsg

	getStateResponse := <-getStateResponseCh

	responseMsg := &SDKProtogo.DMSMessage{
		Type:         SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_RESPONSE,
		ContractName: h.contractName,
		Payload:      getStateResponse.Payload,
	}

	return h.sendMessage(responseMsg)
}

func (h *DMSHandler) handleCompleted(completedMsg *SDKProtogo.DMSMessage) error {

	var responseWithWriteMap SDKProtogo.ResponseWithWriteMap
	_ = proto.Unmarshal(completedMsg.Payload, &responseWithWriteMap)

	//merge write map
	txResponse := &protogo.TxResponse{
		TxId: h.txRequest.TxId,
	}

	if responseWithWriteMap.Response.Status == 200 {
		txResponse.Code = protogo.ContractResultCode_OK
		txResponse.Result = []byte(responseWithWriteMap.Response.Message)
		txResponse.Message = "Success"
		txResponse.WriteMap = responseWithWriteMap.WriteMap
	} else {
		txResponse.Code = protogo.ContractResultCode_FAIL
		txResponse.Result = []byte(responseWithWriteMap.Response.Message)
		txResponse.Message = "Fail"
		txResponse.WriteMap = nil
	}

	// give back result to scheduler  -- for multiple tx incoming
	h.scheduler.GetTxResponseCh() <- txResponse

	return h.afterCompleted()
}

// afterCompleted send completed to client, client end stream
func (h *DMSHandler) afterCompleted() error {

	responseMsg := &SDKProtogo.DMSMessage{
		Type:         SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_COMPLETED,
		ContractName: h.contractName,
		Payload:      nil,
	}

	return h.sendMessage(responseMsg)

}
