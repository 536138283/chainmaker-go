package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpcserver"
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
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
	TxId   string
}

type ManagerImpl struct {
	dockerRpcServer *rpcserver.DockerRpcServer
	workerStatusCh  chan ExitStatus
	//workerFinishCh  chan bool
	// uid manager
	// child process manage

}

func NewManager() *ManagerImpl {

	// new docker rpc server
	server, err := rpcserver.NewDockerRpcServer(config.Port)
	if err != nil {
		log.Fatalln(err)
	}

	exitCh := make(chan ExitStatus)
	//finishCh := make(chan bool)

	manager := &ManagerImpl{
		dockerRpcServer: server,
		workerStatusCh:  exitCh,
		//workerFinishCh:  finishCh,
	}

	return manager
}

func (m *ManagerImpl) InitContainer() error {

	// start server
	go m.dockerRpcServer.StartServer()

	// init sandBox
	go security2.InitSandboxEnv()

	// init monitor
	go m.MonitorWorkers()

	// listen incoming txs
	go m.listenIncoming()

	return nil
}

func (m *ManagerImpl) listenIncoming() {
	fmt.Println("Manager -- Begin listen incoming")
	for {
		select {
		case tx := <-m.dockerRpcServer.TxCh:
			go m.HandleTx(tx)
		}
	}
	fmt.Println("Manager -- Stop listen incoming")
}

func (m *ManagerImpl) HandleTx(tx *outside.TxRequest) *outside.ContractResult {

	fmt.Println("Manager -- Begin handle tx")

	// set available uid

	// set file path for uid

	// store info for
	newFilePath := "/home/user10000/hello"
	err := utils.ConvertBytesToRunnableFile(tx.ByteCode, newFilePath, 10000)
	if err != nil {
		log.Fatalln(err)
	}

	return m.StartWorker(newFilePath, uint32(10000), tx.TxId)

}

func (m *ManagerImpl) StartWorker(command string, uid uint32, txId string) *outside.ContractResult {
	cmd := exec.Cmd{
		Path: command,
	}

	cmd.Stdout = os.Stdout

	//set namespace
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uid,
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

	fmt.Println("Add Pid ", cmd.Process.Pid, " to file ", config.ProcsFile)

	cmd.Wait()

	// capture result of current process
	contractResult := &outside.ContractResult{
		Code:    outside.ContractResultCode_OK,
		Result:  nil,
		Message: "testing",
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
		TxId: txId,
	}

	if status.Signaled() {
		exitStatus.Signal = status.Signal()
	}

	m.workerStatusCh <- exitStatus

	return contractResult
}

func (m *ManagerImpl) MonitorWorkers() {
	for {
		status := <-m.workerStatusCh

		switch status.Signal {
		case os.Kill:
			// means process run fail, todo
			fmt.Printf("process %d fail with code: %d, txId: %s\n", status.PID, status.Code, status.TxId)
		default:
			// means process run successful, return the value back
			fmt.Printf("process %d success with code: %d, txId: %s\n", status.PID, status.Code, status.TxId)

			contractResult := &outside.ContractResult{
				Code:    outside.ContractResultCode_OK,
				Result:  nil,
				Message: "testing",
			}

			m.dockerRpcServer.TxResultCh <- contractResult
			//m.workerFinishCh <- true
		}

	}
}
