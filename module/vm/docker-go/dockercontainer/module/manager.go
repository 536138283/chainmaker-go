package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/core"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpc"
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
)

type ManagerImpl struct {
	cdmRpcServer    *rpc.CDMServer
	dmsRpcServer    *rpc.DMSServer
	scheduler       *core.DockerScheduler
	userController  *core.UsersManager
	securityEnv     *security2.SecurityEnv
	handlerRegister *core.HandlerRegister
}

func NewManager() (*ManagerImpl, error) {

	// new users controller
	userController := core.NewUsersManager()

	// new handler register
	handlerRegister := core.NewHandlerRegister()

	// new scheduler
	scheduler := core.NewDockerScheduler(userController, handlerRegister)

	// new docker manager to sandbox server
	dmsRpcServer, err := rpc.NewDMSServer(config.SockPath)
	if err != nil {
		return nil, err
	}

	// new chain maker to docker manager server
	cdmRpcServer, err := rpc.NewCDMServer(config.Port)
	if err != nil {
		return nil, err
	}

	manager := &ManagerImpl{
		cdmRpcServer:    cdmRpcServer,
		dmsRpcServer:    dmsRpcServer,
		scheduler:       scheduler,
		userController:  userController,
		securityEnv:     security2.NewSecurityEnv(),
		handlerRegister: handlerRegister,
	}

	return manager, nil
}

func (m *ManagerImpl) InitContainer() {

	// start cdm server
	cdmApiInstance := rpc.NewCDMApi(m.scheduler)
	go m.cdmRpcServer.StartCDMServer(cdmApiInstance)

	// start dms server
	dmsApiInstance := rpc.NewDMSApi(m.handlerRegister)
	go m.dmsRpcServer.StartDMSServer(dmsApiInstance)

	// init sandBox
	go m.securityEnv.InitSecurityEnv()

	// create new users
	go m.userController.CreateNewUsers(config.UserNum)

	// start scheduler
	go m.scheduler.StartScheduler()

}

func (m *ManagerImpl) StopManager() {

}
