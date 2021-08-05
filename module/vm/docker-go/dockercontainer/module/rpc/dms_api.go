package rpc

import (
	"chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
)

type HandlerRegisterInterface interface {
	GetHandlerByName(handlerName string) *DMSHandler
}

type DMSApi struct {
	logger          *zap.SugaredLogger
	handlerRegister HandlerRegisterInterface
}

func NewDMSApi(handlerRegister HandlerRegisterInterface) *DMSApi {
	return &DMSApi{
		logger:          logger.NewDockerLogger(logger.MODULE_CDM_SERVER),
		handlerRegister: handlerRegister,
	}
}

func (s *DMSApi) DMSCommunicate(stream protogo.DMSRpc_DMSCommunicateServer) error {

	s.logger.Debugf("begin to handle stream....")

	// get handler from handler_register
	registerMsg, err := stream.Recv()
	if err != nil {
		s.logger.Errorf("fail to get handler from handler_register")
		return err
	}

	handlerName := string(registerMsg.Payload)
	handler := s.handlerRegister.GetHandlerByName(handlerName)

	if handler == nil {
		// todo
		s.logger.Errorf("no handler found")
	}

	handler.SetStream(stream)
	s.logger.Debugf("get handler: %s", handlerName)

	err = handler.HandleMessage(registerMsg)
	if err != nil {
		s.logger.Errorf("fail to handle register msg: [%s] -- msg: [%s]", err, registerMsg)
		return err
	}

	// begin loop to receive msg
	type recvMsg struct {
		msg *protogo.DMSMessage
		err error
	}

	msgAvail := make(chan *recvMsg, 1)
	defer close(msgAvail)

	receiveMessage := func() {
		in, err := stream.Recv()
		msgAvail <- &recvMsg{in, err}
	}

	go receiveMessage()

	for {
		select {
		case rmsg := <-msgAvail:
			switch {
			case rmsg.err == io.EOF:
				s.logger.Debugf("received EOF, ending contract stream")
				return nil
			case rmsg.err != nil:
				err := fmt.Errorf("receive failed: %s", rmsg.err)
				return err
			case rmsg.msg == nil:
				err := errors.New("received nil message, ending contract stream")
				return err
			default:
				err := handler.HandleMessage(rmsg.msg)
				if err != nil {
					err = fmt.Errorf("error handling message: %s", err)
					return err
				}
			}

			go receiveMessage()
		}

	}

}
