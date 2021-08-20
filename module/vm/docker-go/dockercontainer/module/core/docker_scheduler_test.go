/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"testing"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
)

func getBlankDockerScheduler() *DockerScheduler {
	userController := &UsersManager{}
	handlerRegister := &HandlerRegister{}
	return NewDockerScheduler(userController, handlerRegister)
}

//func (s *DockerScheduler) RegisterResponseCh(txId string, responseCh chan *protogo.CDMMessage) {
func TestRegisterResponseCh(t *testing.T) {
	s := getBlankDockerScheduler()
	const txId = "txId1"
	responseChan := make(chan *protogo.CDMMessage)
	s.RegisterResponseCh(txId, responseChan)
	t.Run("test1", func(t *testing.T) {
		if s.getStateResponseChMap[txId] != responseChan {
			t.Errorf("test RegisterResponseCh failed.")
		}
	})
}

//func (s *DockerScheduler) GetResponseChByTxId(txId string) chan *protogo.CDMMessage {
func TestGetResponseChByTxId(t *testing.T) {
	s := getBlankDockerScheduler()
	txId := "txId1"
	responseChan := make(chan *protogo.CDMMessage)
	s.RegisterResponseCh(txId, responseChan)
	t.Run("test1", func(t *testing.T) {
		if s.GetResponseChByTxId(txId) != responseChan {
			t.Errorf("test RegisterResponseCh failed.")
		}
	})
}
