package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/core"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpcserver"
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
)

type ManagerImpl struct {
	dockerRpcServer *rpcserver.DockerRpcServer
	udsRpcServer    *rpcserver.UDSServer
	scheduler       *core.DockerScheduler
	userController  *core.UsersController
}

func NewManager() (*ManagerImpl, error) {

	// new users controller
	userController := core.NewUsersController()

	// new handler register
	handlerRegister := core.NewHandlerRegister()

	// new scheduler
	scheduler := core.NewDockerScheduler(userController, handlerRegister)

	// new uds rpc server
	udsServer, err := rpcserver.NewUDSRpcServer(handlerRegister)
	if err != nil {
		return nil, err
	}

	// new docker rpc server
	server, err := rpcserver.NewDockerRpcServer(config.Port, scheduler)
	if err != nil {
		return nil, err
	}

	manager := &ManagerImpl{
		dockerRpcServer: server,
		udsRpcServer:    udsServer,
		scheduler:       scheduler,
		userController:  userController,
	}

	return manager, nil
}

func (m *ManagerImpl) InitContainer() {

	// start server
	go m.dockerRpcServer.StartServer()

	// start uds server
	go m.udsRpcServer.StartServer()

	// init sandBox
	go security2.InitSecurityEnv()

	// create new users
	go m.userController.CreateNewUsers(config.UserNum)

	// start scheduler
	go m.scheduler.StartScheduler()

}

func (m *ManagerImpl) StopManager() {

}
