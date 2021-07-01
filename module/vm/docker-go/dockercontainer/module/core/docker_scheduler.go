package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/helper"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
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
	logger     *log.Logger

	userController  protocol.UserController
	handlerRegister *HandlerRegister
}

func NewDockerScheduler(userController protocol.UserController, handlerRegister *HandlerRegister) *DockerScheduler {

	scheduler := &DockerScheduler{
		txCh:            make(chan *outside.TxRequest, 2),
		txResultCh:      make(chan *outside.ContractResult, 2),
		exitCh:          make(chan ExitStatus, 2),
		userController:  userController,
		logger:          utils.NewLogger("Docker Scheduler"),
		handlerRegister: handlerRegister,
	}

	return scheduler
}

func (s *DockerScheduler) StartScheduler() {

	s.logger.Println("start docker scheduler")

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

func (s *DockerScheduler) listenIncoming() {
	s.logger.Println("Begin listen incoming")
	for {
		select {
		case tx := <-s.txCh:
			s.logger.Println("receive tx, begin to handle")
			go s.handleTx(tx)
		}
	}
	s.logger.Println("Stop listen incoming")
}

func (s *DockerScheduler) monitorSandBox() {
	for {
		status := <-s.exitCh

		switch status.Signal {
		case os.Kill:
			// means process run fail, todo
			s.logger.Printf("process %d fail with code: %d, txId: %s\n", status.PID, status.Code, status.Tx.TxId)
		default:
			// means process run successful, return the value back
			s.logger.Printf("process %d success with code: %d, txId: %s\n", status.PID, status.Code, status.Tx.TxId)
		}

	}
}

func (s *DockerScheduler) listenResult() {
	//for result := range s.txResultCh {
	//	//todo
	//	// rwset conflict analysis
	//}
}

func (s *DockerScheduler) handleTx(tx *outside.TxRequest) error {

	s.logger.Println("Scheduler -- Begin handle tx")

	startTime := time.Now()

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
		return err
	}

	// free handler
	s.handlerRegister.FreeHandler(handlerName)

	// free current user
	s.userController.UpdateUserState(user.Uid, false)
	s.userController.ResetUserEnv(user)

	s.logger.Println("running time is:", time.Since(startTime))

	return nil
}

func (s *DockerScheduler) startSandBox(user *helper.User, tx *outside.TxRequest, handlerName string) error {

	cmd := exec.Cmd{
		Path: user.BinPath,
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
		log.Panicln(err)
	}

	// set timeout
	timer := time.AfterFunc(config.TimeLimit*time.Second, func() {
		cmd.Process.Kill()
	})

	// set cgroup procs id
	memoryPath := filepath.Join(config.CGroupRoot, config.ProcsFile)
	utils.WriteToFile(memoryPath, cmd.Process.Pid)

	s.logger.Println("Add Pid ", cmd.Process.Pid, " to file ", config.ProcsFile)

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

	s.logger.Println("--------- after wait")

	s.exitCh <- exitStatus

	return nil
}

func (s *DockerScheduler) constructHandlerName(tx *outside.TxRequest) string {

	handlerName := tx.ContractName + ":" + tx.ContractVersion + ":" + tx.TxId[:5]
	return handlerName
}
