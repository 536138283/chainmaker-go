package rpc

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"net"
	"os"
	"strconv"
	"time"
)

type CDMServer struct {
	Listener net.Listener
	Server   *grpc.Server
	logger   *zap.SugaredLogger
}

// NewCDMServer build new chainmaker to docker manager rpc server
func NewCDMServer() (*CDMServer, error) {

	port := os.Getenv("Port")

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
		MinTime: ServerMinInterval,
		// allow keepalive w/o rpc
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))

	//set default connection timeout
	maxSendSizeConfig := os.Getenv("MaxSendMessageSize")
	maxRecvSizeConfig := os.Getenv("MaxRecvMessageSize")

	maxSendSize, _ := strconv.Atoi(maxSendSizeConfig)
	maxRecvSize, _ := strconv.Atoi(maxRecvSizeConfig)

	serverOpts = append(serverOpts, grpc.ConnectionTimeout(ConnectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(maxSendSize*1024*1024))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(maxRecvSize*1024*1024))

	server := grpc.NewServer(serverOpts...)

	return &CDMServer{
		Listener: listener,
		Server:   server,
		logger:   logger.NewDockerLogger(logger.MODULE_CDM_SERVER),
	}, nil
}

// 	Start the server
func (cdm *CDMServer) StartCDMServer(apiInstance *CDMApi) error {

	var err error

	if cdm.Listener == nil {
		return errors.New("nil listener")
	}

	if cdm.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterCDMRpcServer(cdm.Server, apiInstance)

	cdm.logger.Infof("start cdm server")

	go func() {
		err = cdm.Server.Serve(cdm.Listener)
		if err != nil {
			cdm.logger.Errorf("cdm server fail to start: %s", err)
		}
	}()

	return nil
}

// StopCDMServer Stop the server
func (cdm *CDMServer) StopCDMServer() {
	cdm.logger.Infof("stop cdm server")
	if cdm.Server != nil {
		cdm.Server.Stop()
	}
}
