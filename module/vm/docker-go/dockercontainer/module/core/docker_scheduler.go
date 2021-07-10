package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpc"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type ExitStatus struct {
	Signal os.Signal
	Code   int
	PID    int
	User   *helper.User
	TxId   string
}

const (
	ReqChanSize      = 1000
	ResponseChanSize = 1000
)

type DockerScheduler struct {
	exitCh chan ExitStatus
	logger *zap.SugaredLogger

	userController  protocol.UserController
	handlerRegister *HandlerRegister

	contractManager *ContractManager

	txReqCh      chan *protogo.TxRequest
	txResponseCh chan *protogo.TxResponse
}

func NewDockerScheduler(userController protocol.UserController, handlerRegister *HandlerRegister) *DockerScheduler {

	scheduler := &DockerScheduler{
		exitCh:          make(chan ExitStatus, 2),
		userController:  userController,
		logger:          logger.NewDockerLogger(logger.MODULE_SCHEDULER),
		handlerRegister: handlerRegister,
		contractManager: NewContractManager(),

		txReqCh:      make(chan *protogo.TxRequest, ReqChanSize),
		txResponseCh: make(chan *protogo.TxResponse, ResponseChanSize),
	}

	return scheduler
}

func (s *DockerScheduler) GetTxReqCh() chan *protogo.TxRequest {
	return s.txReqCh
}

func (s *DockerScheduler) GetTxResponseCh() chan *protogo.TxResponse {
	return s.txResponseCh
}

// StartScheduler three goroutines lifecycle is same as docker vm
func (s *DockerScheduler) StartScheduler() {

	s.logger.Infof("start docker scheduler")

	go s.listenIncomingTxRequest()

	go s.listenIncomingTxResponse()

	go s.monitorSandBox()

}

func (s *DockerScheduler) StopScheduler() {
	s.logger.Infof("stop docker scheduler")
	close(s.txResponseCh)
	close(s.txReqCh)
	close(s.exitCh)
}

func (s *DockerScheduler) listenIncomingTxResponse() {

	s.logger.Infof("start listen tx response")

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

	var contractPath string

	contractPath, ok := s.contractManager.GetContract(contractKey)
	if !ok {
		// todo change using single flight
		newContractPath, err := s.contractManager.SaveContract(contractKey, txRequest.ByteCode)
		if err != nil {
			s.logger.Errorf("fail to save contract err: [%s] -- contract [%s], with txId [%s]", err, contractKey, txRequest.TxId)
			s.returnErrorTxResponse(txRequest.TxId)
			return
		}
		s.logger.Debugf("save [%s] to disk and get new contract path", newContractPath)
		contractPath = newContractPath
	}

	s.logger.Debugf("get contract path from disk [%s]", contractPath)

	// set available user
	user, err := s.userController.GetAvailableUser()
	if err != nil {
		s.logger.Errorf("fail to get a user: [%s] -- txId [%s]", err, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId)
		return
	}

	// register new handler
	handlerName := s.constructHandlerName(txRequest)

	dmsHandler, err := rpc.NewDMSHandler(user, txRequest, s, handlerName)
	if err != nil {
		s.logger.Errorf("fail to generate new handler: %s -- txId [%s]", err, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId)
		return
	}

	s.handlerRegister.RegisterNewHandler(handlerName, dmsHandler)

	// start sand box
	err = s.startSandBox(user, txRequest.TxId, handlerName, contractPath)
	if err != nil {
		s.returnErrorTxResponse(txRequest.TxId)
		return
	}

	// free handler
	s.handlerRegister.FreeHandler(handlerName)

	// free current user
	if err = s.userController.FreeUser(user); err != nil {
		s.logger.Errorf("fail to free user: err [%s] -- user[%v] -- txId [%s]", err, user, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId)
		return
	}

	// return result -- for one tx incoming
	//result := handler.contractResult
	//tx.ContractResult = result

	s.logger.Debugf("cost time for running sandbox is: %s", time.Since(startTime))

}

func (s *DockerScheduler) startSandBox(user *helper.User, txId, handlerName, contractPath string) error {

	cmd := exec.Cmd{
		Path: contractPath,
		Args: []string{user.SockPath, handlerName},
	}

	cmd.Stdout = os.Stdout

	//set namespace
	//cmd.SysProcAttr = &syscall.SysProcAttr{
	//	Credential: &syscall.Credential{
	//		Uid: uint32(user.Uid),
	//	},
	//	Cloneflags: syscall.CLONE_NEWIPC |
	//		syscall.CLONE_NEWPID |
	//		syscall.CLONE_NEWNET,
	//}

	// start app
	if err := cmd.Start(); err != nil {
		s.logger.Errorf("fail to run child process [%s] -- txId [%s]", err, txId)
		return err
	}

	// set timeout
	timer := time.AfterFunc(config.TimeLimit*time.Second, func() {
		err := cmd.Process.Kill()
		if err != nil {
			s.logger.Errorf("fail to kill process [%s] -- txId [%s]", err, txId)
			return
		}
	})

	// set cgroup procs id
	memoryPath := filepath.Join(config.CGroupRoot, config.ProcsFile)
	err := utils.WriteToFile(memoryPath, cmd.Process.Pid)
	if err != nil {
		s.logger.Errorf("fail to add cgroup [%s] -- txId [%s]", err, txId)
		return err
	}

	s.logger.Debugf("Add Pid [%d] to file [%s]", cmd.Process.Pid, config.ProcsFile)

	err = cmd.Wait()
	if err != nil {
		s.logger.Errorf("fail to wait child process end [%s] -- txId [%s]", err, txId)
		return err
	}

	// timeout, stop the process
	timer.Stop()

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

	if status.Signaled() {
		exitStatus.Signal = status.Signal()
		s.returnErrorTxResponse(txId)
	}

	s.exitCh <- exitStatus

	return nil
}

func (s *DockerScheduler) returnErrorTxResponse(txId string) {
	errTxResponse := s.constructErrorResponse(txId)
	s.txResponseCh <- errTxResponse
}

// handlerName: contractName:contractVersion:txId[:10]
func (s *DockerScheduler) constructHandlerName(tx *protogo.TxRequest) string {
	handlerName := tx.ContractName + ":" + tx.ContractVersion + ":" + tx.TxId[:10]
	return handlerName
}

// ConstructContractKey contractKey: contractName:contractVersion
func (s *DockerScheduler) ConstructContractKey(contractName, contractVersion string) string {
	return contractName + ":" + contractVersion
}

func (s *DockerScheduler) constructErrorResponse(txId string) *protogo.TxResponse {
	return &protogo.TxResponse{
		TxId:    txId,
		Code:    protogo.ContractResultCode_FAIL,
		Result:  nil,
		Message: "fail",
	}
}
