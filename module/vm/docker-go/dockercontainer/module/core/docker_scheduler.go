package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"go.uber.org/zap"
	"log"
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
	Tx     *outside.TxRequest
}

type DockerScheduler struct {
	txCh       chan *outside.TxRequest
	txResultCh chan *outside.ContractResult
	exitCh     chan ExitStatus
	logger     *zap.SugaredLogger

	userController  protocol.UserController
	handlerRegister *HandlerRegister

	contractManager *ContractManager
}

func NewDockerScheduler(userController protocol.UserController, handlerRegister *HandlerRegister) *DockerScheduler {

	scheduler := &DockerScheduler{
		txCh:            make(chan *outside.TxRequest, 2),
		txResultCh:      make(chan *outside.ContractResult, 2),
		exitCh:          make(chan ExitStatus, 2),
		userController:  userController,
		logger:          logger.NewDockerLogger(logger.MODULE_SCHEDULER),
		handlerRegister: handlerRegister,
		contractManager: NewContractManager(),
	}

	return scheduler
}

func (s *DockerScheduler) StartScheduler() {

	s.logger.Infof("start docker scheduler")

	//go s.listenIncoming()

	go s.monitorSandBox()

	//go s.listenResult()

}

func (s *DockerScheduler) StopScheduler() {

}

func (s *DockerScheduler) GetTxCh() chan *outside.TxRequest {
	return s.txCh
}

func (s *DockerScheduler) GetTxResultCh() chan *outside.ContractResult {
	return s.txResultCh
}

// listenIncoming for now, doesn't use it, later change handle multiple txs, use this func
func (s *DockerScheduler) listenIncoming() {
	s.logger.Infof("Begin listen incoming")
	for {
		select {
		case tx := <-s.txCh:
			go s.HandleTx(tx)
		}
	}
	s.logger.Infof("Stop listen incoming")
}

func (s *DockerScheduler) monitorSandBox() {
	for {
		status := <-s.exitCh

		switch status.Signal {
		case os.Kill:
			// means process run fail, todo
			s.logger.Debugf("process %d fail with code: %d, txId: %s\n", status.PID, status.Code, status.Tx.TxId)
		default:
			// means process run successful, return the value back
			s.logger.Debugf("process %d success with code: %d, txId: %s\n", status.PID, status.Code, status.Tx.TxId)
		}

	}
}

func (s *DockerScheduler) listenResult() {
	//for result := range s.txResultCh {
	//	//todo
	//	// rwset conflict analysis
	//}
}

func (s *DockerScheduler) HandleTx(tx *outside.TxRequest) (*outside.ContractResult, error) {

	startTime := time.Now()

	s.logger.Debugf("begin handle tx")

	// get contract from contract manager
	contractKey := s.ConstructContractKey(tx.ContractName, tx.ContractVersion)

	var contractPath string

	contractPath, ok := s.contractManager.GetContract(contractKey)

	if ok {
		s.logger.Debugf("get contrac from memory [%s]", contractPath)
	} else {
		// todo change using single flight
		newContractPath, err := s.contractManager.SaveContract(contractKey, tx.ByteCode)
		if err != nil {
			return nil, err
		}
		s.logger.Debugf("save [%s] to disk", newContractPath)
		contractPath = newContractPath
	}

	// set available user
	user, err := s.userController.GetAvailableUser()
	if err != nil {
		s.logger.Errorf("fail to get a user: [%s]", err)
		return nil, err
	}

	// register new handler
	handlerName := s.constructHandlerName(tx)

	handler, err := NewHandler(user, tx, s, handlerName)
	if err != nil {
		s.logger.Errorf("fail to generate new handler: %s", err)
		return nil, err
	}

	s.handlerRegister.RegisterNewHandler(handlerName, handler)

	// start sand box
	err = s.startSandBox(user, tx, handlerName, contractPath)
	if err != nil {
		return nil, err
	}

	// todo using txResultChan to handle multiple incoming txs

	// free handler
	s.handlerRegister.FreeHandler(handlerName)

	// free current user
	if err = s.userController.FreeUser(user); err != nil {
		return nil, err
	}
	//if err = s.userController.ResetUserEnv(user); err != nil {
	//	return nil, err
	//}

	// return result -- for one tx incoming
	result := handler.contractResult
	s.logger.Debugf("cost time for running sandbox is: %s", time.Since(startTime))

	return result, nil
}

func (s *DockerScheduler) startSandBox(user *helper.User, tx *outside.TxRequest, handlerName, contractPath string) error {

	cmd := exec.Cmd{
		Path: contractPath,
		Args: []string{user.SockPath, handlerName},
	}

	cmd.Stdout = os.Stdout

	//set namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(user.Uid),
		},
		Cloneflags: syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNET,
	}

	// start app
	if err := cmd.Start(); err != nil {
		s.logger.Errorf("fail to run child process [%v]", err)
		log.Fatalln(err)
	}

	// set timeout
	timer := time.AfterFunc(config.TimeLimit*time.Second, func() {
		cmd.Process.Kill()
	})

	// set cgroup procs id
	memoryPath := filepath.Join(config.CGroupRoot, config.ProcsFile)
	utils.WriteToFile(memoryPath, cmd.Process.Pid)

	s.logger.Debugf("Add Pid [%d] to file [%s]", cmd.Process.Pid, config.ProcsFile)

	cmd.Wait()

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
		Tx:   tx,
	}

	if status.Signaled() {
		exitStatus.Signal = status.Signal()
	}

	s.exitCh <- exitStatus

	return nil
}

func (s *DockerScheduler) constructHandlerName(tx *outside.TxRequest) string {
	handlerName := tx.ContractName + ":" + tx.ContractVersion + ":" + tx.TxId[:10]
	return handlerName
}

func (s *DockerScheduler) ConstructContractKey(contractName, contractVersion string) string {
	return contractName + ":" + contractVersion
}
