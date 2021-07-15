package rpc

import (
	"chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"os"
	"time"
)

type DMSServer struct {
	Listener net.Listener
	Server   *grpc.Server
	logger   *zap.SugaredLogger
}

// NewDMSServer build new docker manager to sandbox server, current: each server in charge of one sandbox
func NewDMSServer(sockPath string) (*DMSServer, error) {

	listenAddress, err := net.ResolveUnixAddr("unix", sockPath)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	listener, err := CreateUnixListener(listenAddress, sockPath)
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
		MinTime: ServerMinInterval,
		// allow keepalive w/o rpc
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))

	//set default connection timeout
	serverOpts = append(serverOpts, grpc.ConnectionTimeout(ConnectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(MaxSendMessageSize))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(MaxRecvMessageSize))

	server := grpc.NewServer(serverOpts...)

	return &DMSServer{
		Listener: listener,
		Server:   server,
		logger:   logger.NewDockerLogger(logger.MODULE_DMS_SERVER),
	}, nil
}

func CreateUnixListener(listenAddress *net.UnixAddr, sockPath string) (*net.UnixListener, error) {
start:
	listener, err := net.ListenUnix("unix", listenAddress)
	if err != nil {
		err = os.Remove(sockPath)
		if err != nil {
			return nil, err
		}
		goto start
	} else {

		// todo change 777: limit delete for user
		if err := os.Chmod(sockPath, 0777); err != nil {
			return nil, err
		}
		return listener, nil
	}
}

// 	Start the server
func (dms *DMSServer) StartDMSServer(dmsApi *DMSApi) error {

	if dms.Listener == nil {
		return errors.New("nil listener")
	}

	if dms.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDMSRpcServer(dms.Server, dmsApi)

	dms.logger.Infof("start dms server")

	go func() {
		err := dms.Server.Serve(dms.Listener)
		if err != nil {
			dms.logger.Errorf("dms server fail to start: %s", err)
		}
	}()

	return nil
}

// StopDMSServer Stop the server
func (dms *DMSServer) StopDMSServer() {

	dms.logger.Infof("stop dms server")

	if dms.Server != nil {
		dms.Server.Stop()
	}
}
