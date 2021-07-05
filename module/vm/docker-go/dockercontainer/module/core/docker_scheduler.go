package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
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

	lru *Cache
}

func NewDockerScheduler(userController protocol.UserController, handlerRegister *HandlerRegister) *DockerScheduler {

	scheduler := &DockerScheduler{
		txCh:            make(chan *outside.TxRequest, 2),
		txResultCh:      make(chan *outside.ContractResult, 2),
		exitCh:          make(chan ExitStatus, 2),
		userController:  userController,
		logger:          logger.NewDockerLogger(logger.MODULE_SCHEDULER),
		handlerRegister: handlerRegister,
		lru:             New(5),
	}

	return scheduler
}

func (s *DockerScheduler) StartScheduler() {

	s.logger.Infof("start docker scheduler")

	go s.listenIncoming()

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

	s.logger.Debugf("begin handle tx")

	// lru test
	contractKey := s.ConstructContractKey(tx.ContractName, tx.ContractVersion)
	v, ok := s.lru.Get(contractKey)

	if ok {
		s.logger.Debugf("get bytecode from cache [%s]", contractKey)
		tx.ByteCode = v.([]byte)
	} else {
		s.logger.Debugf("add [%s] to cache", contractKey)
		s.lru.Add(contractKey, tx.ByteCode)
	}

	// set available user
	user := s.userController.GetAvailableUser()
	s.userController.UpdateUserState(user.Uid, true)

	// save bytes to executable file and set proper permission
	err := utils.ConvertBytesToRunnableFile(tx.ByteCode, user.BinPath, user.Uid)
	if err != nil {
		fmt.Println(1)
		log.Fatalln(err)
	}

	// todo change contractName to other name
	handlerName := s.constructHandlerName(tx)

	// register the new handler
	handler, err := NewHandler(user, tx, s, handlerName)
	if err != nil {
		fmt.Println(2)
		log.Fatalln(err)
	}

	s.handlerRegister.RegisterNewHandler(handlerName, handler)

	err = s.startSandBox(user, tx, handlerName)
	if err != nil {
		return nil, err
	}

	// todo using txResultChan to handle multiple incoming txs

	// free handler
	s.handlerRegister.FreeHandler(handlerName)

	// free current user
	s.userController.UpdateUserState(user.Uid, false)
	s.userController.ResetUserEnv(user)

	// return result -- for one tx incoming
	result := handler.contractResult
	s.logger.Debugf("result is: [%s]", result.Result)

	return result, nil
}

func (s *DockerScheduler) startSandBox(user *helper.User, tx *outside.TxRequest, handlerName string) error {

	cmd := exec.Cmd{
		Path: user.BinPath,
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
		log.Panicln(err)
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
