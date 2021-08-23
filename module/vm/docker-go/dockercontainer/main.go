package main

import (
	"log"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module"
	"go.uber.org/zap"
)

var managerLogger *zap.SugaredLogger

func main() {

	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER)

	// pprof
	enablePprof, _ := strconv.ParseBool(os.Getenv("PProfEnabled"))
	if enablePprof {
		startPprof()
	}

	manager, err := module.NewManager(managerLogger)
	if err != nil {
		managerLogger.Errorf("Err in creating docker manager: %s", err)
		return
	}

	managerLogger.Infof("docker manager created")

	go manager.InitContainer()

	managerLogger.Infof("docker manager init...")

	// infinite loop
	for i := 0; ; i++ {
		time.Sleep(time.Hour)
	}
}

func startPprof() {

	cpuProfileFile := "/mount/share/cpu_pprof"

	f, err := os.Create(cpuProfileFile)
	if err != nil {
		log.Fatal(err)
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		return
	}
	managerLogger.Infof("start pprof")

	time.AfterFunc(30*time.Minute, func() {
		pprof.StopCPUProfile()
		managerLogger.Infof("finish pprof")
	})

}
