package rpcserver

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/api"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"io"
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
	TxCh       chan *outside.TxRequest
}

func (s *DockerRpcServer) RunContracts(stream api.DockerRpc_RunContractsServer) error {

	for {
		tx, err := stream.Recv()
		if err == io.EOF {

			txResult := &outside.TxResult{
				Code: 0,
				ContractResult: &outside.ContractResult{
					Code:    0,
					Result:  nil,
					Message: "testing",
				},
				RwSetHash: nil,
			}
			fmt.Println("server receive all")
			return stream.SendAndClose(txResult)
		}

		if err != nil {
			return err
		}
		// handle each incoming tx
		fmt.Println("Server receive tx")
		fmt.Println("bytes length: ", len(tx.ByteCode))
		//
		s.TxCh <- tx
	}

	return nil
}

// NewDockerRpcServer build new rpc server
func NewDockerRpcServer(port string) (*DockerRpcServer, error) {

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
	return &DockerRpcServer{
		Listener:   listener,
		Server:     server,
		isShutdown: true,
		TxCh:       txCh,
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
func (s *DockerRpcServer) Stop() {
	if s.Server != nil {
		s.Server.Stop()
	}
}
