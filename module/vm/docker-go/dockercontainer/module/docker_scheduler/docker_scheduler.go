package docker_scheduler

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/docker_handler"
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"fmt"
	"log"
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
	User   *security2.User
	Tx     *outside.TxRequest
}

type DockerScheduler struct {
	txCh           chan *outside.TxRequest
	txResultCh     chan *outside.ContractResult
	exitCh         chan ExitStatus
	userController protocol.UserController
	handlers       map[string]*docker_handler.DockerHandler
	logger         *log.Logger
}

func NewDockerScheduler(userController protocol.UserController) *DockerScheduler {
	txCh := make(chan *outside.TxRequest)
	txResultCh := make(chan *outside.ContractResult, 1)
	exitCh := make(chan ExitStatus, 1)

	scheduler := &DockerScheduler{
		txCh:           txCh,
		txResultCh:     txResultCh,
		exitCh:         exitCh,
		userController: userController,
		logger:         utils.NewLogger("Docker Scheduler"),
		handlers:       make(map[string]*docker_handler.DockerHandler),
	}

	return scheduler
}

func (s *DockerScheduler) Start() {

	s.logger.Println("start docker scheduler")

	go s.listenIncoming()

	go s.monitorSandBox()

}

func (s *DockerScheduler) Stop() {
	// close all channels:
	//close(s.txCh)
	//close(s.txResultCh)
	//close(s.exitCh)

	// stop listen

	// stop monitor
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
			//m.workerFinishCh <- true
		}

		// free current user
		s.userController.UpdateUserState(status.User.Uid, false)
		s.userController.ResetUserEnv(status.User)
	}
}

func (s *DockerScheduler) handleTx(tx *outside.TxRequest) error {

	s.logger.Println("Scheduler -- Begin handle tx")

	// set available user
	user := s.userController.GetAvailableUser()
	s.userController.UpdateUserState(user.Uid, true)

	// save bytes to executable file and set proper permission
	err := utils.ConvertBytesToRunnableFile(tx.ByteCode, user.BinPath, user.Uid)
	if err != nil {
		fmt.Println(1)
		log.Fatalln(err)
	}

	// register the new handler
	handler, err := docker_handler.NewDockerHandler(user, tx, s)
	if err != nil {
		fmt.Println(2)
		log.Fatalln(err)
	}
	s.handlers[tx.ContractName] = handler

	var wg sync.WaitGroup
	wg.Add(1)
	// begin handle sandbox
	go handler.UdsServer.StartServer(tx, &wg)

	err = s.startSandBox(user, tx, &wg)
	if err != nil {
		return err
	}

	s.FreeHandler(tx.ContractName)

	return nil
}

func (s *DockerScheduler) startSandBox(user *security2.User, tx *outside.TxRequest, wg *sync.WaitGroup) error {

	cmd := exec.Cmd{
		Path: user.BinPath,
		Args: []string{user.SockPath},
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

	// need to wait uds server finish the job
	wg.Wait()

	s.exitCh <- exitStatus

	return nil
}

func (s *DockerScheduler) FreeHandler(contractName string) {
	// free handler map
	delete(s.handlers, contractName)
	s.logger.Printf("free [%s] handler", contractName)
}
