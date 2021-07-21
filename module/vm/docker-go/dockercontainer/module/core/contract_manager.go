package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var (
	mountDir string
)

type ContractManager struct {
	lock            sync.RWMutex
	getContractLock singleflight.Group
	contractsMap    map[string]string
	logger          *zap.SugaredLogger
	scheduler       protocol.Scheduler
}

func NewContractManager() *ContractManager {
	contractManager := &ContractManager{
		lock:            sync.RWMutex{},
		getContractLock: singleflight.Group{},
		contractsMap:    make(map[string]string),
		logger:          logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER),
	}

	mountDir = config.ContractBaseDir

	_ = contractManager.initialContractMap()
	return contractManager
}

// GetContract get contract path in volume,
// if it exists in volume, return path
// if not exist in volume, request from chain maker state library
func (cm *ContractManager) GetContract(txId, contractName string) (string, error) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	// get contract path from map
	contractPath, ok := cm.contractsMap[contractName]
	if ok {
		cm.logger.Debugf("get contract from memory [%s], path is [%s]", contractName, contractPath)
		return contractPath, nil
	}

	// get contract path from chain maker
	cPath, err, _ := cm.getContractLock.Do(contractName, func() (interface{}, error) {
		defer cm.getContractLock.Forget(contractName)

		return cm.lookupContractFromDB(txId, contractName)
	})
	if err != nil {
		return "", err
	}

	return cPath.(string), nil
}

func (cm *ContractManager) lookupContractFromDB(txId, contractName string) (string, error) {
	getByteCodeMsg := &protogo.CDMMessage{
		TxId:    txId,
		Type:    protogo.CDMType_CDM_TYPE_GET_BYTECODE,
		Payload: []byte(contractName),
	}

	// send request to chain maker
	responseChan := make(chan *protogo.CDMMessage)
	cm.scheduler.RegisterResponseCh(txId, responseChan)

	cm.scheduler.GetGetByteCodeReqCh() <- getByteCodeMsg

	<-responseChan

	// set contract mod
	contractPath := filepath.Join(mountDir, contractName)
	err := cm.setFileMod(contractPath)
	if err != nil {
		return "", err
	}

	// save contract file path to map
	cm.contractsMap[contractName] = contractPath
	cm.logger.Debugf("get contract disk [%s], path is [%s]", contractName, contractPath)

	return contractPath, nil
}

// SetFileRunnable make file runnable, file permission is 755
func (cm *ContractManager) setFileMod(filePath string) error {

	err := os.Chmod(filePath, 0755)
	if err != nil {
		return err
	}

	return nil
}

func (cm *ContractManager) initialContractMap() error {

	files, err := ioutil.ReadDir(mountDir)
	if err != nil {
		cm.logger.Errorf("fail to scan contract dir")
		return err
	}
	for _, f := range files {
		contractName := f.Name()
		contractPath := filepath.Join(mountDir, contractName)
		cm.contractsMap[contractName] = contractPath
	}

	cm.logger.Debugf("init contract map with size [%d]", len(cm.contractsMap))

	return nil
}
