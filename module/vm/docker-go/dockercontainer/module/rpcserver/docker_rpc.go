package rpcserver

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/api"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
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

type DockerRpcServer struct {
	Listener   net.Listener
	Server     *grpc.Server
	isShutdown bool
	handler    protocol.Handler
	TxCh       chan *outside.TxRequest
	TxResultCh chan *outside.ContractResult
}

func (s *DockerRpcServer) RunContracts(ctx context.Context, txRequest *outside.TxRequest) (*outside.ContractResult, error) {

	s.handler.GetTxCh() <- txRequest

	for {

		contractResult := <-s.handler.GetTxResultCh()

		return contractResult, nil

	}

}

// NewDockerRpcServer build new rpc server
func NewDockerRpcServer(port string, handler protocol.Handler) (*DockerRpcServer, error) {

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

	server := grpc.NewServer(serverOpts...)

	log.Println("Server created successfully")

	txCh := make(chan *outside.TxRequest)
	txResCh := make(chan *outside.ContractResult)

	return &DockerRpcServer{
		Listener:   listener,
		Server:     server,
		isShutdown: true,
		TxCh:       txCh,
		TxResultCh: txResCh,
		handler:    handler,
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

	log.Println("Start server ..... ")
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
