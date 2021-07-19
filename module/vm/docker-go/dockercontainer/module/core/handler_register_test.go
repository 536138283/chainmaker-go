/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/rpc"
	"testing"
)

//func (hr *HandlerRegister) RegisterNewHandler(handlerName string, handler *rpc.DMSHandler) {
func  TestRegisterNewHandler(t *testing.T) {
	name := "handlerName1"
	hd := &rpc.DMSHandler{}
	hr := NewHandlerRegister()
	hr.RegisterNewHandler(name, hd)

	t.Run("TestRegisterNewHandler", func(t *testing.T) {
		if hr.HandlersTable[name] != hd {
			t.Errorf("RegisterNewHandler error, RegisterNewHandler failed.")
		}

	})
}

//func (hr *HandlerRegister) FreeHandler(handlerName string) {
func TestFreeHandler(t *testing.T) {
	name := "handlerName1"
	hd := &rpc.DMSHandler{}
	hr := NewHandlerRegister()
	hr.RegisterNewHandler(name, hd)
	hr.FreeHandler(name)

	t.Run("TestFreeHandler", func(t *testing.T) {
		_, ok := hr.HandlersTable[name]
		if ok {
			t.Errorf("FreeHandler error, FreeHandler failed.")
		}
	})
}

//func (hr *HandlerRegister) GetHandlerByName(handlerName string) *rpc.DMSHandler {
func TestGetHandlerByName(t *testing.T) {
	name := "handlerName1"
	hd := &rpc.DMSHandler{}
	hr := NewHandlerRegister()
	hr.RegisterNewHandler(name, hd)

	t.Run("TestRegisterNewHandler", func(t *testing.T) {
		if hr.GetHandlerByName(name) != hd {
			t.Errorf("RegisterNewHandler error, RegisterNewHandler failed.")
		}
	})
}

