package security

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strconv"
)

type SecurityEnv struct {
	logger *zap.SugaredLogger
}

func NewSecurityEnv() *SecurityEnv {
	return &SecurityEnv{
		logger: logger.NewDockerLogger(logger.MODULE_SECURITY_ENV),
	}
}

func (s *SecurityEnv) InitSecurityEnv() error {
	if err := s.setTmpMod(); err != nil {
		return err
	}

	if err := SetCGroup(); err != nil {
		return err
	}

	s.logger.Infof("init security env completed")

	return nil
}

func (s *SecurityEnv) InitConfig() error {

	var err error

	// set mount dir mod
	mountDir := os.Getenv("DockerMountDir")

	// set mount sub directory: contracts, share, sock
	contractDir := filepath.Join(mountDir, config.ContractsDir)
	config.ContractBaseDir = contractDir

	shareDir := filepath.Join(mountDir, config.ShareDir)
	config.ShareBaseDir = shareDir

	sockDir := filepath.Join(mountDir, config.SockDir)
	config.SockBaseDir = sockDir

	// set timeout
	timeLimitConfig := os.Getenv("TimeLimit")
	timeLimit, err := strconv.Atoi(timeLimitConfig)
	if err != nil {
		timeLimit = 2
	}
	config.SandBoxTimeout = timeLimit

	// set dms directory
	if err = s.setDMSDir(); err != nil {
		return err
	}
	s.logger.Debug("set dms dir: ", config.DMSDir)

	return nil

}

func (s *SecurityEnv) setDMSDir() error {
	return os.Mkdir(config.DMSDir, 0755)
}

func (s *SecurityEnv) setTmpMod() error {
	return os.Chmod("/tmp/", 0755)
}
