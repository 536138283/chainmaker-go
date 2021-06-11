package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpcserver"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/security"
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
}

type Manager struct {
	dockerRpcServer *rpcserver.DockerRpcServer
	existCh         chan ExitStatus
	// uid manager
	// child process manage

}

func NewManager() *Manager {

	// new docker rpc server
	server, err := rpcserver.NewDockerRpcServer("12355")
	if err != nil {
		log.Fatalln(err)
	}

	exitCh := make(chan ExitStatus)

	manager := &Manager{
		dockerRpcServer: server,
		existCh:         exitCh,
	}

	return manager
}

func (m *Manager) InitContainer() error {

	// start server
	go m.dockerRpcServer.StartServer()

	// init sandBox
	go security.InitSandboxEnv()

	// init monitor
	go m.MonitorWorkers()

	// listen incoming txs
	go m.listenIncoming()

	return nil
}

func (m *Manager) StartWorker(command string, uid uint32) {
	cmd := exec.Cmd{
		Path: command,
	}

	cmd.Stdout = os.Stdout

	//set namespace
	//cmd.SysProcAttr = &syscall.SysProcAttr{
	//	Credential: &syscall.Credential{
	//		Uid: uid,
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

	fmt.Println("Add Pid ", cmd.Process.Pid, " to file ", config.ProcsFile)

	if err := cmd.Wait(); err != nil {
		fmt.Println("cmd return with error: ", err)
	}

	timer.Stop()

	status := cmd.ProcessState.Sys().(syscall.WaitStatus)

	options := ExitStatus{
		Code: status.ExitStatus(),
		PID:  cmd.Process.Pid,
	}

	if status.Signaled() {
		options.Signal = status.Signal()
	}

	cmd.Process.Kill()

	m.existCh <- options
}

func (m *Manager) MonitorWorkers() {
	for {
		status := <-m.existCh

		switch status.Signal {
		case os.Kill:
			fmt.Printf("process %d is killed by system\n", status.PID)
		default:
			fmt.Printf("process %d exit with code: \n", status.Code)
		}

	}
}

func (m *Manager) handleTx(tx *outside.TxRequest) {

	fmt.Println("begin handle tx")

	newFilePath := "/home/user10000/hello"
	err := utils.ConvertBytesToFile(tx.ByteCode, newFilePath)
	if err != nil {
		log.Fatalln("err in convert to file ", err)
	}

	err = utils.SetFileRunnable(newFilePath, 10000)
	if err != nil {
		log.Fatalln("err in set file runnable ", err)
	}

	m.StartWorker(newFilePath, uint32(10000))

}

func (m *Manager) listenIncoming() {
	fmt.Println("begin listen incoming")
	for i := 0; i < 1; i++ {
		select {
		case tx := <-m.dockerRpcServer.TxCh:
			go m.handleTx(tx)
		}
	}
	fmt.Println("begin listen incoming  -- finished")
}
