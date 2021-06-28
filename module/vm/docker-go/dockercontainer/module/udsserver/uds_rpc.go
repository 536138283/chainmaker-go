package udsserver

import (
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"contract-sdk-test1/pb_sdk/protogo"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	serverMinInterval  = time.Duration(1) * time.Minute
	connectionTimeout  = 5 * time.Second
	maxRecvMessageSize = 100 * 1024 * 1024 // 100 MiB
	maxSendMessageSize = 100 * 1024 * 1024 // 100 MiB
)

type UDSServer struct {
	Listener net.Listener
	Server   *grpc.Server
	logger   *log.Logger
	Handler  protocol.Handler
	FinishCh chan bool
}

func (s *UDSServer) Connect(stream protogo.Contract_ConnectServer) error {

	defer close(s.FinishCh)

	s.logger.Println("begin to handle stream....")

	s.Handler.SetStream(stream)

	type recvMsg struct {
		msg *protogo.ContractMessage
		err error
	}

	msgAvail := make(chan *recvMsg, 1)
	//defer close(msgAvail)

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
				s.logger.Println(11)
				s.logger.Println(rmsg.err)
				return errors.New("received EOF, ending contract stream")
			case rmsg.err != nil:
				s.logger.Println(12)
				s.logger.Println(rmsg.err)
				err := fmt.Errorf("receive failed: %s", rmsg.err)
				return err
			case rmsg.msg == nil:
				s.logger.Println(13)
				err := errors.New("received nil message, ending contract stream")
				return err
			default:
				err := s.Handler.HandleMessage(rmsg.msg)
				if err != nil {
					err = fmt.Errorf("error handling message: %s", err)
					return err
				}
			}

			go receiveMessage()

		case <-s.FinishCh:
			s.logger.Println("close stream")
			return nil

		}

	}

}

// NewUDSRpcServer build new uds server, current: each server in charge of one sandbox
func NewUDSRpcServer(user *security2.User, handler protocol.Handler) (*UDSServer, error) {

	if user.SockPath == "" {
		return nil, errors.New("server listen port not provided")
	}

	listenAddress, err := net.ResolveUnixAddr("unix", user.SockPath)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	listener, err := CreateUnixListener(listenAddress, user)
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
		logger:   utils.NewLogger("Docker UDS RPC Server"),
		Handler:  handler,
		FinishCh: make(chan bool, 1),
	}, nil
}

func CreateUnixListener(listenAddress *net.UnixAddr, user *security2.User) (*net.UnixListener, error) {
start:
	listener, err := net.ListenUnix("unix", listenAddress)
	if err != nil {
		fmt.Println("server -- unix domain socket create fail, try to re create")
		err = os.Remove(user.SockPath)
		if err != nil {
			fmt.Println("server -- delete socket file fail: ", err)
			return nil, err
		}
		goto start
	} else {
		if err := os.Chown(user.SockPath, user.Uid, user.Uid); err != nil {
			return nil, err
		}

		if err := os.Chmod(user.SockPath, 0700); err != nil {
			return nil, err
		}
		return listener, nil
	}
}

// 	Start the server
func (s *UDSServer) StartServer(tx *outside.TxRequest, wg *sync.WaitGroup) error {

	defer wg.Done()

	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterContractServer(s.Server, s)

	s.logger.Printf("Start server for contract [%s]", tx.ContractName)

	return s.Server.Serve(s.Listener)
}

// Stop the server
// Stop the server
func (s *UDSServer) Stop() {

	if s.Server != nil {
		s.Server.Stop()
	}

	s.logger.Println("stop server for contract")
}
