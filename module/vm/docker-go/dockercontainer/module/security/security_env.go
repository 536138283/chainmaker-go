package security

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"go.uber.org/zap"
	"os"
	"path/filepath"
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

func (s *SecurityEnv) InitDirectory() error {

	var err error

	// set mount dir mod
	mountDir := os.Getenv("DockerMountDir")
	err = os.Chmod(mountDir, 0755)
	if err != nil {
		return err
	}
	s.logger.Debug("set mount dir: ", mountDir)

	// create sub directory: contracts, share, sock
	contractDir := filepath.Join(mountDir, config.ContractsDir)
	err = s.createSubDir(contractDir)
	if err != nil {
		return err
	}
	config.ContractBaseDir = contractDir
	s.logger.Debug("set contract dir: ", contractDir)

	shareDir := filepath.Join(mountDir, config.ShareDir)
	err = s.createSubDir(shareDir)
	if err != nil {
		return err
	}
	config.ShareBaseDir = shareDir
	s.logger.Debug("set share dir: ", shareDir)

	sockDir := filepath.Join(mountDir, config.SockDir)
	err = s.createSubDir(sockDir)
	if err != nil {
		return err
	}
	config.SockBaseDir = sockDir
	s.logger.Debug("set sock dir: ", sockDir)

	// set dms directory
	if err = s.setDMSDir(); err != nil {
		return err
	}
	s.logger.Debug("set dms dir: ", config.DMSDir)

	return nil

}

func (s *SecurityEnv) createSubDir(subDir string) error {
	exist, err := s.exists(subDir)
	if err != nil {
		return err
	}

	if !exist {
		err := os.Mkdir(subDir, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

// exists returns whether the given file or directory exists
func (s *SecurityEnv) exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *SecurityEnv) setDMSDir() error {
	return os.Mkdir(config.DMSDir, 0755)
}

func (s *SecurityEnv) setTmpMod() error {
	return os.Chmod("/tmp/", 0755)
}
