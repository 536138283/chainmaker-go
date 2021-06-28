package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/docker_scheduler"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpcserver"
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
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
	scheduler       protocol.Scheduler
	userController  protocol.UserController
}

func NewManager() *ManagerImpl {

	// new users controller
	userController := security2.NewUsersController()

	// new handler
	scheduler := docker_scheduler.NewDockerScheduler(userController)

	// new docker rpc server
	server, err := rpcserver.NewDockerRpcServer(config.Port, scheduler)
	if err != nil {
		log.Fatalln(err)
	}

	manager := &ManagerImpl{
		dockerRpcServer: server,
		scheduler:       scheduler,
		userController:  userController,
	}

	return manager
}

func (m *ManagerImpl) InitContainer() {

	// start server
	go m.dockerRpcServer.StartServer()

	// init sandBox
	go security2.InitSandboxEnv()

	// create new users
	go m.userController.CreateNewUsers(config.UserNum)

	// start handler
	go m.scheduler.Start()

}
