package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sync"
)

const BaseDir = "/contracts"

type ContractManager struct {
	lock         sync.RWMutex
	contractsMap map[string]string
	logger       *zap.SugaredLogger
}

func NewContractManager() *ContractManager {
	return &ContractManager{
		lock:         sync.RWMutex{},
		contractsMap: make(map[string]string),
		logger:       logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER),
	}
}

func (cm *ContractManager) GetContract(contractName string) (string, bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	contractPath, ok := cm.contractsMap[contractName]
	if ok {
		return contractPath, true
	}

	return "", false
}

func (cm *ContractManager) SaveContract(contractName string, byteCode []byte) (string, error) {

	cm.lock.Lock()
	defer cm.lock.Unlock()

	contractFilePath := filepath.Join(BaseDir, contractName)

	// convert byte array to file
	err := cm.convertBytesToFile(byteCode, contractFilePath)
	if err != nil {
		cm.logger.Errorf("fail to convert bytes to file: %s", err)
		return "", err
	}

	// set file runnable with mod 755
	err = cm.setFileMod(contractFilePath)
	if err != nil {
		cm.logger.Errorf("fail to set file mod: %s", err)
		return "", err
	}

	// save contract file path to map
	cm.contractsMap[contractName] = contractFilePath

	return contractFilePath, nil
}

// ConvertBytesToFile convert byte array to file
func (cm *ContractManager) convertBytesToFile(bytes []byte, newFilePath string) error {

	f, err := os.Create(newFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		return err
	}

	return f.Sync()
}

// SetFileRunnable make file runnable, file permission is 700
func (cm *ContractManager) setFileMod(filePath string) error {

	err := os.Chmod(filePath, 0755)
	if err != nil {
		return err
	}

	return nil
}
