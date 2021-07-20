package security

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
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

	if err := SetCGroup(); err != nil {
		return err
	}

	if err := s.createContractDir(); err != nil {
		return err
	}

	if err := s.setDMSDir(); err != nil {
		return err
	}

	s.logger.Infof("init security env completed")

	return nil
}

func (s *SecurityEnv) setDMSDir() error {

	return os.Mkdir(config.DMSDir, 755)
}

func (s *SecurityEnv) setMountDir() error {
	// set mount directory mod as 755
	mountDir := os.Getenv("DockerMountDir")
	err := os.Chmod(mountDir, 755)
	if err != nil {
		return err
	}

	return nil
}

func (s *SecurityEnv) setTmpMod() error {
	return os.Chmod("/tmp/", 0755)
}

func (s *SecurityEnv) createContractDir() error {
	return os.Mkdir("/contracts", 0755)
}
