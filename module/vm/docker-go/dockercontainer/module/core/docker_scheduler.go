package core

import (
	"bytes"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpc"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"errors"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type ExitStatus struct {
	Signal os.Signal
	Code   int
	PID    int
	User   *security.User
	TxId   string
}

const (
	ReqChanSize      = 1000
	ResponseChanSize = 1000 //todo how to set number
)

type DockerScheduler struct {
	lock           sync.Mutex
	logger         *zap.SugaredLogger
	userController protocol.UserController

	handlerRegister *HandlerRegister
	contractManager *ContractManager

	exitCh                chan ExitStatus
	txReqCh               chan *protogo.TxRequest
	txResponseCh          chan *protogo.TxResponse
	getStateReqCh         chan *protogo.CDMMessage
	getByteCodeReqCh      chan *protogo.CDMMessage
	getStateResponseChMap map[string]chan *protogo.CDMMessage
}

func NewDockerScheduler(userController protocol.UserController, handlerRegister *HandlerRegister) *DockerScheduler {

	contractManager := NewContractManager()

	scheduler := &DockerScheduler{
		lock:            sync.Mutex{},
		userController:  userController,
		logger:          logger.NewDockerLogger(logger.MODULE_SCHEDULER),
		handlerRegister: handlerRegister,
		contractManager: contractManager,

		exitCh:                make(chan ExitStatus, ReqChanSize),
		txReqCh:               make(chan *protogo.TxRequest, ReqChanSize),
		txResponseCh:          make(chan *protogo.TxResponse, ResponseChanSize),
		getStateReqCh:         make(chan *protogo.CDMMessage, ReqChanSize*8),
		getByteCodeReqCh:      make(chan *protogo.CDMMessage, ReqChanSize),
		getStateResponseChMap: make(map[string]chan *protogo.CDMMessage),
	}

	contractManager.scheduler = scheduler

	return scheduler
}

func (s *DockerScheduler) GetTxReqCh() chan *protogo.TxRequest {
	return s.txReqCh
}

func (s *DockerScheduler) GetTxResponseCh() chan *protogo.TxResponse {
	return s.txResponseCh
}

func (s *DockerScheduler) GetGetStateReqCh() chan *protogo.CDMMessage {
	return s.getStateReqCh
}

func (s *DockerScheduler) GetGetByteCodeReqCh() chan *protogo.CDMMessage {
	return s.getByteCodeReqCh
}

func (s *DockerScheduler) RegisterResponseCh(txId string, responseCh chan *protogo.CDMMessage) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.getStateResponseChMap[txId] = responseCh
}

func (s *DockerScheduler) GetResponseChByTxId(txId string) chan *protogo.CDMMessage {
	s.lock.Lock()
	defer s.lock.Unlock()

	responseCh := s.getStateResponseChMap[txId]
	delete(s.getStateResponseChMap, txId)
	return responseCh
}

// StartScheduler three goroutines lifecycle is same as docker vm
func (s *DockerScheduler) StartScheduler() {

	s.logger.Infof("start docker scheduler")

	go s.listenIncomingTxRequest()

	go s.monitorSandBox()

}

func (s *DockerScheduler) StopScheduler() {
	s.logger.Infof("stop docker scheduler")
	close(s.txResponseCh)
	close(s.txReqCh)
	close(s.exitCh)
	close(s.getStateReqCh)
	close(s.getByteCodeReqCh)
}

func (s *DockerScheduler) listenIncomingTxRequest() {
	s.logger.Infof("start listen incoming tx request")

	for {
		txRequest := <-s.txReqCh
		go s.handleTx(txRequest)
	}
}

func (s *DockerScheduler) monitorSandBox() {
	for {
		status := <-s.exitCh

		switch status.Signal {
		case os.Kill:
			// means process run fail, todo
			s.logger.Debugf("process %d fail with code: %d, txId: %s\n", status.PID, status.Code, status.TxId)
		default:
			// means process run successful, return the value back
			s.logger.Debugf("process %d success with code: %d, txId: %s\n", status.PID, status.Code, status.TxId)
		}

	}
}

func (s *DockerScheduler) handleTx(txRequest *protogo.TxRequest) {

	startTime := time.Now()

	s.logger.Debugf("begin handle tx request")

	// get contract from contract manager
	contractKey := s.ConstructContractKey(txRequest.ContractName, txRequest.ContractVersion)
	contractPath, err := s.contractManager.GetContract(txRequest.TxId, contractKey)
	if err != nil || len(contractPath) == 0 {
		s.logger.Errorf("fail to get contract path -- contractName is [%s], err is [%s]", contractKey, err)
		s.returnErrorTxResponse(txRequest.TxId, err)
		return
	}
	s.logger.Debugf("get contract path [%s]", contractPath)

	// set available user
	user, err := s.userController.GetAvailableUser()
	if err != nil {
		s.logger.Errorf("fail to get a user: [%s] -- txId [%s]", err, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId, err)
		return
	}

	// register new handler
	handlerName := s.constructHandlerName(txRequest)
	dmsHandler, err := rpc.NewDMSHandler(user, txRequest, s, handlerName, txRequest.ContractName)
	if err != nil {
		s.logger.Errorf("fail to generate new handler: [%s] -- txId [%s]", err, txRequest.TxId)
		_ = s.userController.FreeUser(user)
		s.returnErrorTxResponse(txRequest.TxId, err)
		return
	}

	s.handlerRegister.RegisterNewHandler(handlerName, dmsHandler)

	// start sand box
	err = s.startSandBox(user, txRequest.TxId, txRequest.ContractName, handlerName, contractPath)
	if err != nil {
		s.logger.Errorf("faild to start sand box : [%s] -- txId [%s]",err,txRequest.TxId)
		s.handlerRegister.FreeHandler(handlerName)
		_ = s.userController.FreeUser(user)
		s.returnErrorTxResponse(txRequest.TxId, err)
		return
	}

	// free handler
	s.handlerRegister.FreeHandler(handlerName)

	// free current user
	if err = s.userController.FreeUser(user); err != nil {
		s.logger.Errorf("fail to free user: err [%s] -- user[%v] -- txId [%s]", err, user, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId, err)
		return
	}

	s.logger.Debugf("cost time for running sandbox is: %s", time.Since(startTime))

}

func (s *DockerScheduler) startSandBox(user *security.User, txId, contractName, handlerName, contractPath string) error {
	var err error           // sandbox global error
	var stderr bytes.Buffer // used to capture the error message from sandbox

	cmd := exec.Cmd{
		Path: contractPath,
		Args: []string{user.SockPath, handlerName, contractName},
	}
	cmd.Stderr = &stderr

	// set namespace, these settings just working in linux
	// but it doens't affect running, cause it will put into docker to run
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(user.Uid),
		},
		Cloneflags: syscall.CLONE_NEWPID,
		//Cloneflags: syscall.CLONE_NEWIPC |
	}

	// start sandbox
	if err = cmd.Start(); err != nil {
		s.logger.Errorf("fail to start tx: txId [%s], err [%s]", txId, err)
		return err
	}

	// set timeout
	timer := time.AfterFunc(time.Duration(config.SandBoxTimeout)*time.Second, func() {
		_ = cmd.Process.Kill()
		s.logger.Errorf("timeout: kill tx: txId [%s]", txId)
		return
	})
	defer timer.Stop()

	// add control group
	memoryPath := filepath.Join(config.CGroupRoot, config.ProcsFile)
	if err = utils.WriteToFile(memoryPath, cmd.Process.Pid); err != nil {
		s.logger.Errorf("fail to add cgroup [%s] -- txId [%s]", err, txId)
		return err
	}
	s.logger.Debugf("Add Pid [%d] to file [%s]", cmd.Process.Pid, config.ProcsFile)

	// wait sandbox end
	if err = cmd.Wait(); err != nil {
		s.logger.Errorf("fail to wait tx end: txId [%s], err [%s]", txId, err)
		s.logger.Errorf("tx error: [%s]", stderr.String())
		err = errors.New(stderr.String())
	}

	// capture current process exit status
	// code : 0 : process run successfully
	// code : -1 : process run fail, maybe timeout, maybe memory out
	status := cmd.ProcessState.Sys().(syscall.WaitStatus)

	exitStatus := ExitStatus{
		Code: status.ExitStatus(),
		PID:  cmd.Process.Pid,
		User: user,
		TxId: txId,
	}

	s.exitCh <- exitStatus

	//if status.Signaled() {
	//	exitStatus.Signal = status.Signal()
	//	return fmt.Errorf("fail to run child process with kill signal")
	//}

	return err
}

func (s *DockerScheduler) returnErrorTxResponse(txId string, err error) {
	errTxResponse := s.constructErrorResponse(txId, err)
	s.txResponseCh <- errTxResponse
}

func (s *DockerScheduler) constructErrorResponse(txId string, err error) *protogo.TxResponse {
	return &protogo.TxResponse{
		TxId:    txId,
		Code:    protogo.ContractResultCode_FAIL,
		Result:  nil,
		Message: err.Error(),
	}
}

// handlerName: contractName:contractVersion:txId[:10]
func (s *DockerScheduler) constructHandlerName(tx *protogo.TxRequest) string {
	handlerName := tx.ContractName + ":" + tx.ContractVersion + ":" + tx.TxId[:10]
	return handlerName
}

// ConstructContractKey contractKey: contractName:contractVersion
func (s *DockerScheduler) ConstructContractKey(contractName, contractVersion string) string {
	return contractName + "#" + contractVersion
}
