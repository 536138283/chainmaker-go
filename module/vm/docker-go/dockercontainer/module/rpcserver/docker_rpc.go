package rpcserver

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/api"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"time"
)

const (
	serverMinInterval = time.Duration(1) * time.Minute
	connectionTimeout = 5 * time.Second
)

const (
	dialTimeout        = 10 * time.Second
	maxRecvMessageSize = 100 * 1024 * 1024 // 100 MiB
	maxSendMessageSize = 100 * 1024 * 1024 // 100 MiB
)

type DockerRpcServer struct {
	Listener   net.Listener
	Server     *grpc.Server
	isShutdown bool
	scheduler  protocol.Scheduler
	TxCh       chan *outside.TxRequest
	TxResultCh chan *outside.ContractResult
	logger     *log.Logger
}

func (s *DockerRpcServer) RunContracts(ctx context.Context, txRequest *outside.TxRequest) (*outside.ContractResult, error) {

	s.scheduler.GetTxCh() <- txRequest

	for {

		// todo check if result is for current request
		contractResult := <-s.scheduler.GetTxResultCh()
		s.logger.Println(contractResult)
		if contractResult.Result != nil {
			return contractResult, nil

		}

	}

}

// NewDockerRpcServer build new rpc server
func NewDockerRpcServer(port string, scheduler protocol.Scheduler) (*DockerRpcServer, error) {

	if port == "" {
		return nil, errors.New("server listen port not provided")
	}

	//create listener
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
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

	txCh := make(chan *outside.TxRequest)
	txResCh := make(chan *outside.ContractResult)

	return &DockerRpcServer{
		Listener:   listener,
		Server:     server,
		isShutdown: true,
		TxCh:       txCh,
		TxResultCh: txResCh,
		scheduler:  scheduler,
		logger:     utils.NewLogger("Docker RPC Server"),
	}, nil
}

// 	Start the server
func (s *DockerRpcServer) StartServer() error {

	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.Server == nil {
		return errors.New("nil server")
	}

	api.RegisterDockerRpcServer(s.Server, s)

	s.logger.Println("Start server ..... ")
	s.isShutdown = true
	s.Server.Serve(s.Listener)
	return nil
}

// Stop the server
// Stop the server
func (s *DockerRpcServer) Stop() {
	if s.Server != nil {
		s.isShutdown = false
		s.Server.Stop()
	}
}
