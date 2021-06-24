package txhandler

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	security2 "chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
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
	TxId   string
	User   *security2.User
}

type Handler struct {
	txCh           chan *outside.TxRequest
	txResultCh     chan *outside.ContractResult
	exitCh         chan ExitStatus
	userController protocol.UserController
}

func NewHandler(userController protocol.UserController) *Handler {
	txCh := make(chan *outside.TxRequest)
	txResultCh := make(chan *outside.ContractResult)
	exitCh := make(chan ExitStatus)

	handler := &Handler{
		txCh:           txCh,
		txResultCh:     txResultCh,
		exitCh:         exitCh,
		userController: userController,
	}

	return handler
}

func (h *Handler) Start() {

	go h.listenIncoming()

	go h.monitorSandBox()

}

func (h *Handler) Stop() {
	// close all channels:
}

func (h *Handler) GetTxCh() chan *outside.TxRequest {
	return h.txCh
}

func (h *Handler) GetTxResultCh() chan *outside.ContractResult {
	return h.txResultCh
}

func (h *Handler) listenIncoming() {
	fmt.Println("Handler -- Begin listen incoming")
	for {
		select {
		case tx := <-h.txCh:
			go h.handleTx(tx)
		}
	}
	fmt.Println("Handler -- Stop listen incoming")
}

func (h *Handler) monitorSandBox() {
	for {
		status := <-h.exitCh

		switch status.Signal {
		case os.Kill:
			// means process run fail, todo
			fmt.Printf("process %d fail with code: %d, txId: %s\n", status.PID, status.Code, status.TxId)
		default:
			// means process run successful, return the value back
			fmt.Printf("process %d success with code: %d, txId: %s\n", status.PID, status.Code, status.TxId)
			//m.workerFinishCh <- true
		}

		// free current user
		h.userController.UpdateUserState(status.User.Uid, false)
		h.userController.ResetUserEnv(status.User)
	}
}

func (h *Handler) handleTx(tx *outside.TxRequest) *outside.ContractResult {

	fmt.Println("Handler -- Begin handle tx")

	// set available uid
	user := h.userController.GetAvailableUser()
	h.userController.UpdateUserState(user.Uid, true)

	// save bytes to executable file and set proper permission
	err := utils.ConvertBytesToRunnableFile(tx.ByteCode, user.BinPath, user.Uid)
	if err != nil {
		log.Fatalln(err)
	}

	return h.startSandBox(user, tx.TxId)
}

func (h *Handler) startSandBox(user *security2.User, txId string) *outside.ContractResult {

	cmd := exec.Cmd{
		Path: user.BinPath,
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

	fmt.Println("Add Pid ", cmd.Process.Pid, " to file ", config.ProcsFile)

	cmd.Wait()

	// capture result of current process
	// todo: unix domain socket receive the result
	contractResult := &outside.ContractResult{
		Code:    outside.ContractResultCode_OK,
		Result:  nil,
		Message: "testing",
	}
	h.txResultCh <- contractResult

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
		User: user,
	}

	if status.Signaled() {
		exitStatus.Signal = status.Signal()
	}

	h.exitCh <- exitStatus

	return contractResult
}
