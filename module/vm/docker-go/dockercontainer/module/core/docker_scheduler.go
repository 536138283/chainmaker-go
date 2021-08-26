/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpc"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"go.uber.org/zap"
)

const (
	// ReqChanSize tx request chan size
	ReqChanSize = 1000
	// ResponseChanSize tx response chan size
	ResponseChanSize = 1000
	runtimePanic     = "runtime panic"
)

type DockerScheduler struct {
	lock           sync.Mutex
	logger         *zap.SugaredLogger
	userController protocol.UserController

	handlerRegister *HandlerRegister
	contractManager *ContractManager

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

}

func (s *DockerScheduler) StopScheduler() {
	s.logger.Infof("stop docker scheduler")
	close(s.txResponseCh)
	close(s.txReqCh)
	close(s.getStateReqCh)
	close(s.getByteCodeReqCh)
}

func (s *DockerScheduler) listenIncomingTxRequest() {
	s.logger.Debugf("start listen incoming tx request")

	for {
		txRequest := <-s.txReqCh
		go s.handleTx(txRequest)
	}
}

func (s *DockerScheduler) handleTx(txRequest *protogo.TxRequest) {

	startTime := time.Now()

	s.logger.Debugf("begin handle tx request: txid: [%s]", txRequest.TxId)

	// get contract from contract manager
	contractKey := s.constructContractKey(txRequest.ContractName, txRequest.ContractVersion)
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
	defer func(userController protocol.UserController, user *security.User) {
		err = userController.FreeUser(user)
		if err != nil {
			s.logger.Errorf("fail to free user: err [%s] -- user[%v] -- txId [%s]", err, user, txRequest.TxId)
		}
	}(s.userController, user)

	// register new handler
	handlerName := s.constructHandlerName(txRequest)
	dmsHandler, err := rpc.NewDMSHandler(user, txRequest, s, handlerName, txRequest.ContractName)
	if err != nil {
		s.logger.Errorf("fail to generate new handler: [%s] -- txId [%s]", err, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId, err)
		return
	}

	err = s.handlerRegister.RegisterNewHandler(handlerName, dmsHandler)
	if err != nil {
		s.logger.Errorf("fail to register handler: [%s] -- txId [%s]", err, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId, err)
		return
	}

	// start sandbox
	err = s.startSandBox(user, txRequest.TxId, txRequest.ContractName, handlerName, contractPath)
	if err != nil {
		s.logger.Errorf("faild to run contract : [%s] -- txId [%s]", err, txRequest.TxId)
		s.returnErrorTxResponse(txRequest.TxId, err)
	}

	s.logger.Debugf("cost time for running sandbox is: %s", time.Since(startTime))

}

func (s *DockerScheduler) startSandBox(user *security.User, txId,
	contractName, handlerName, contractPath string) error {
	var err error           // sandbox global error
	var stderr bytes.Buffer // used to capture the error message from sandbox
	var stdout bytes.Buffer

	cmd := exec.Cmd{
		Path: contractPath,
		Args: []string{user.SockPath, handlerName, contractName, config.SandBoxLogLevel},
	}
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	// set namespace, these settings just working in linux
	// but it doens't affect running, cause it will put into docker to run
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(user.Uid),
		},
		Cloneflags: syscall.CLONE_NEWPID,
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
		s.logger.Errorf("tx fail: txId [%s], err [%s]", txId, stderr.String())
		err = errors.New(runtimePanic)
	}
	s.logger.Debugf(stdout.String())

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
	handlerName := tx.ContractName + ":" + tx.ContractVersion + ":" + tx.TxId
	return handlerName
}

// constructContractKey contractKey: contractName:contractVersion
func (s *DockerScheduler) constructContractKey(contractName, contractVersion string) string {
	return contractName + "#" + contractVersion
}
