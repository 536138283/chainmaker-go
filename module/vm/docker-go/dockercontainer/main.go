package main

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module"
	"fmt"
	"go.uber.org/zap"
	"time"
)

var managerLogger *zap.SugaredLogger

func main() {

	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER)

	manager, err := module.NewManager(managerLogger)
	if err != nil {
		managerLogger.Errorf("Err in creating docker manager: %s", err)
		return
	}

	managerLogger.Infof("docker manager created")

	manager.InitContainer()

	managerLogger.Infof("docker manager init...")

	fmt.Println("testing")

	// infinite loop
	// todo wait node send stop
	for i := 0; ; i++ {
		fmt.Println("in main process -- ", i)
		time.Sleep(time.Minute)
	}
}
