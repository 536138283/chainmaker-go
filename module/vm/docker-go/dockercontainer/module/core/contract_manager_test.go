/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"os"
	"testing"
)

//func (cm *ContractManager) GetContract(txId, contractName string) (string, error) {
func TestGetContract(t *testing.T) {
	c1Name := "c1"
	c1Code := "dummy contract1 for unit test"
	cm := NewContractManager()
	cm.contractsMap[c1Name] = c1Code

	tests := []struct {
		name string
		want string
		wantErr interface{}
	}{
		{
			name: "test1",
			want: c1Code,
			wantErr: nil,
		},
	}

	for _, test := range tests{
		t.Run(test.name, func(t *testing.T){
			contract, err := cm.GetContract("", c1Name)

			if contract != test.want || err != test.wantErr {
				t.Errorf("GetContract error = %v", err)
			}
		})
	}
}

//func (cm *ContractManager) lookupContractFromDB(txId, contractName string) (string, error) {
func TestLookupContractFromDB(t *testing.T) {
	//userController := NewUsersManager()
	userController := &UsersManager{}
	handlerRegister := &HandlerRegister{}
	cm := NewContractManager()
	cm.scheduler = NewDockerScheduler(userController, handlerRegister)

	txId := "tx1"
	contractName := "DockerContract1"
	go func() {
		byteCodeCh := cm.scheduler.GetGetByteCodeReqCh()
		<- byteCodeCh

		os.Create(contractName)
		msg := &protogo.CDMMessage{}
		resCh := cm.scheduler.GetResponseChByTxId(txId)
		resCh <- msg
	}()

	t.Run("test1", func(t *testing.T){
		path, err := cm.lookupContractFromDB(txId, contractName)

		if path != contractName || err != nil {
			t.Errorf("cm.lookupContractFromDB error = %v", err)
		}
	})

}
//func (cm *ContractManager) setFileMod(filePath string) error {
func TestSetFileMod(t *testing.T) {
	contractName := "DockerContract1"
	os.Create(contractName)
	cm := NewContractManager()

	t.Run("test1", func(t *testing.T) {
		err := cm.setFileMod(contractName)
		if err != nil{
			t.Errorf("cm.setFileMod error = %v", err)
		}
	})
}

//func (cm *ContractManager) initialContractMap() error {
func  TestInitialContractMap(t *testing.T) {
	cm := NewContractManager()
	mountDir, _ = os.Getwd()


	t.Run("test1", func(t *testing.T) {
		err := cm.initialContractMap()
		if err != nil{
			t.Errorf("cm.initialContractMap error = %v", err)
		}
	})
}
