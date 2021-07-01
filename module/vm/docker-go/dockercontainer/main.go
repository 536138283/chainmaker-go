package main

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module"
	"fmt"
	"go.uber.org/zap"
	"log"
	"time"
)

var ManagerLogger *zap.SugaredLogger

func main() {

	ManagerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER)

	manager, err := module.NewManager()
	if err != nil {
		log.Fatalf("Err in creating manager: %s", err)
	}

	ManagerLogger.Infof("docker manager created")

	manager.InitContainer()

	ManagerLogger.Infof("docker manager init...")

	// infinite loop
	// todo wait node send stop
	for i := 0; ; i++ {
		fmt.Println("in main process -- ", i)
		time.Sleep(time.Minute)
	}
}
