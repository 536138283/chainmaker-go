package rpcserver

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/core"
	"contract-sdk-test1/pb_sdk/protogo"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type UDSServer struct {
	Listener net.Listener
	Server   *grpc.Server
	logger   *zap.SugaredLogger

	handlerRegister *core.HandlerRegister
}

func (s *UDSServer) Contact(stream protogo.Contract_ContactServer) error {

	s.logger.Debugf("begin to handle stream....")

	// get handler from handler_register
	registerMsg, err := stream.Recv()
	if err != nil {
		return err
	}

	handlerName := registerMsg.HandlerName
	//fmt.Println(handlerName)
	handler := s.handlerRegister.GetHandlerByName(handlerName)

	if handler == nil {
		// todo
		fmt.Println("no handler")
	}

	handler.SetStream(stream)
	s.logger.Debugf("get handler: %s", registerMsg.HandlerName)

	err = handler.HandleMessage(registerMsg)
	if err != nil {
		return err
	}

	// begin loop to receive msg
	type recvMsg struct {
		msg *protogo.ContractMessage
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

// NewUDSRpcServer build new uds server, current: each server in charge of one sandbox
func NewUDSRpcServer(handlerRegister *core.HandlerRegister) (*UDSServer, error) {

	listenAddress, err := net.ResolveUnixAddr("unix", config.SockPath)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	listener, err := CreateUnixListener(listenAddress, config.SockPath)
	if err != nil {
		log.Fatalf("Failed to listen1: %v", err)
	}

	//set up server options for keepalive and TLS
	var serverOpts []grpc.ServerOption

	// add keepalive
	serverKeepAliveParameters := keepalive.ServerParameters{
		Time:    1 * time.Minute,
		Timeout: 20 * time.Second,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(serverKeepAliveParameters))

	//set enforcement policy
	kep := keepalive.EnforcementPolicy{
		MinTime: serverMinInterval,
		// allow keepalive w/o rpc
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))

	//set default connection timeout
	serverOpts = append(serverOpts, grpc.ConnectionTimeout(connectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(maxSendMessageSize))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(maxRecvMessageSize))

	server := grpc.NewServer(serverOpts...)

	return &UDSServer{
		Listener: listener,
		Server:   server,
		logger:   logger.NewDockerLogger(logger.MODULE_UDS_SERVER),

		handlerRegister: handlerRegister,
	}, nil
}

func CreateUnixListener(listenAddress *net.UnixAddr, sockPath string) (*net.UnixListener, error) {
start:
	listener, err := net.ListenUnix("unix", listenAddress)
	if err != nil {
		fmt.Println("server -- unix domain socket create fail, try to re create")
		err = os.Remove(sockPath)
		if err != nil {
			fmt.Println("server -- delete socket file fail: ", err)
			return nil, err
		}
		goto start
	} else {
		//if err := os.Chown(user.SockPath, user.Uid, user.Uid); err != nil {
		//	return nil, err
		//}

		if err := os.Chmod(sockPath, 0777); err != nil {
			return nil, err
		}
		return listener, nil
	}
}

// 	Start the server
func (s *UDSServer) StartServer() error {

	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterContractServer(s.Server, s)

	s.logger.Infof("start uds server")

	return s.Server.Serve(s.Listener)
}

// Stop the server
// Stop the server
func (s *UDSServer) Stop() {

	if s.Server != nil {
		s.Server.Stop()
	}

	s.logger.Infof("stop uds server")
}
