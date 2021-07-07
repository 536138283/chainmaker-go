package security

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"go.uber.org/zap"
	"os"
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

	s.logger.Infof("successfully set tmp file mod")

	if err := SetCGroup(); err != nil {
		return err
	}
	s.logger.Infof("successfully set cgroup")

	if err := s.createContractDir(); err != nil {
		return err
	}
	s.logger.Infof("successfully create contract base dir")

	s.logger.Infof("init security env completed")

	return nil
}

func (s *SecurityEnv) setTmpMod() error {
	return os.Chmod("/tmp/", 0755)
}

func (s *SecurityEnv) createContractDir() error {
	return os.Mkdir("/contracts", 0755)
}
