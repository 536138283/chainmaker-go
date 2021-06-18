package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpcserver"
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/txhandler"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"log"
	"os"
)

type ExitStatus struct {
	Signal os.Signal
	Code   int
	PID    int
	TxId   string
}

type ManagerImpl struct {
	dockerRpcServer *rpcserver.DockerRpcServer
	handler         protocol.Handler
	userController  protocol.UserController
	//workerFinishCh  chan bool
	// uid manager
	// child process manage

}

func NewManager() *ManagerImpl {

	// new users controller
	userController := security2.NewUsersController()

	// new handler
	handler := txhandler.NewHandler(userController)

	// new docker rpc server
	server, err := rpcserver.NewDockerRpcServer(config.Port, handler)
	if err != nil {
		log.Fatalln(err)
	}

	manager := &ManagerImpl{
		dockerRpcServer: server,
		handler:         handler,
		userController:  userController,
	}

	return manager
}

func (m *ManagerImpl) InitContainer() error {

	// start server
	go m.dockerRpcServer.StartServer()

	// init sandBox
	go security2.InitSandboxEnv()

	// create new users
	go m.userController.CreateNewUsers(config.UserNum)

	// start handler
	go m.handler.Start()

	return nil
}
