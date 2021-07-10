package rpc

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"errors"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"sync"
)

type CDMApi struct {
	logger    *zap.SugaredLogger
	scheduler protocol.Scheduler
	stream    protogo.CDMRpc_CDMCommunicateServer
	stop      chan bool
	wg        *sync.WaitGroup
}

func NewCDMApi(scheduler protocol.Scheduler) *CDMApi {
	return &CDMApi{
		scheduler: scheduler,
		logger:    logger.NewDockerLogger(logger.MODULE_CDM_API),
		stream:    nil,
		stop:      make(chan bool),
		wg:        new(sync.WaitGroup),
	}
}

// CDMCommunicate docker manager stream function
func (cdm *CDMApi) CDMCommunicate(stream protogo.CDMRpc_CDMCommunicateServer) error {

	cdm.stream = stream

	cdm.wg.Add(2)

	go cdm.recvMsgRoutine()

	go cdm.sendMsgRoutine()

	cdm.wg.Wait()

	return nil
}

func (cdm *CDMApi) closeConnection() {
	close(cdm.stop)
}

// recv three types of msg
// type1: txRequest
// type2: get_state response
// type3: get_bytecode response or path
func (cdm *CDMApi) recvMsgRoutine() {

	cdm.logger.Infof("start receiving cdm message ")

	var err error

	for {

		select {
		case <-cdm.stop:
			cdm.wg.Done()
			cdm.logger.Infof("stop receiving cdm message ")
			return
		default:
			recvMsg, errRecv := cdm.stream.Recv()
			cdm.logger.Debugf("receive msg: [%s]", recvMsg.Type)
			if errRecv != nil {
				err = errRecv
			}

			cdm.logger.Debugf("recv msg [%s]", recvMsg.TxId[:5])

			switch recvMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_REQUEST:
				err = cdm.handleTxRequest(recvMsg)
			case protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE:
				err = cdm.handleGetStateResponse(recvMsg)
			case protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE:
				err = cdm.handleGetByteCodeResponse(recvMsg)
			default:
				err = errors.New("unknown message type")
			}

			if err != nil {
				cdm.logger.Errorf("fail to recv msg: [%s]", err)
			}

		}

	}
}

func (cdm *CDMApi) sendMsgRoutine() {

	cdm.logger.Infof("start sending cdm message")

	var err error

	for {
		select {
		case txResponseMsg := <-cdm.scheduler.GetTxResponseCh():
			cdmMsg := cdm.constructCDMMessage(txResponseMsg)
			err = cdm.sendMessage(cdmMsg)
		case <-cdm.stop:
			cdm.wg.Done()
			return
		}

		if err != nil {
			cdm.logger.Error(err)
		}

	}

}

func (cdm *CDMApi) constructCDMMessage(txResponseMsg *protogo.TxResponse) *protogo.CDMMessage {

	payload, _ := txResponseMsg.Marshal()

	cdmMsg := &protogo.CDMMessage{
		TxId:    txResponseMsg.TxId,
		Type:    protogo.CDMType_CDM_TYPE_TX_RESPONSE,
		Payload: payload,
	}

	return cdmMsg
}

func (cdm *CDMApi) sendMessage(msg *protogo.CDMMessage) error {
	cdm.logger.Debugf("send message [%s]", msg)
	return cdm.stream.Send(msg)
}

func (cdm *CDMApi) handleTxRequest(cdmMessage *protogo.CDMMessage) error {

	var txRequest protogo.TxRequest
	err := proto.Unmarshal(cdmMessage.Payload, &txRequest)
	if err != nil {
		return err
	}

	cdm.scheduler.GetTxReqCh() <- &txRequest

	return nil
}

func (cdm *CDMApi) handleGetStateResponse(cdmMessage *protogo.CDMMessage) error {
	return nil
}

func (cdm *CDMApi) handleGetByteCodeResponse(cdmMessage *protogo.CDMMessage) error {
	return nil
}
