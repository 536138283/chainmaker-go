/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmertest

import (
	"chainmaker.org/chainmaker-go/protocol"
	_ "net/http/pprof"
)

// Module 序列化后实例wasm
// 经测试证明 序列化反序列化方式Instantiate慢200倍
//func TestSerializationModuleSpendTest(t *testing.T) {
//	byteCode, _ := wasm.ReadBytes("../../../../test/wasm/rust-counter-1.2.0.wasm")
//	module1, _ := wasm.Compile(byteCode)
//
//	serialization, _ := module1.Serialize()
//	module1.Close()
//
//	start := time.Now().UnixNano() / 1e6
//	for i := 0; i < 10000; i++ {
//		module2, _ := wasm.DeserializeModule(serialization)
//		vm := wasmer.GetVmBridgeManager()
//		module2.InstantiateWithImports(vm.GetImports()) // 44832ms
//		module2.Close()
//	}
//
//	end := time.Now().UnixNano() / 1e6
//	println("【spend】", end-start)
//}

// Module 直接实例wasm
//func TestModuleSpendTest(t *testing.T) {
//	byteCode, _ := wasm.ReadBytes("../../../../test/wasm/rust-counter-1.2.0.wasm")
//	module1, _ := wasm.Compile(byteCode)
//
//	start := time.Now().UnixNano() / 1e6
//	for i := 0; i < 10000; i++ {
//		vm := wasmer.GetVmBridgeManager()
//		module1.InstantiateWithImports(vm.GetImports()) // 643ms
//	}
//
//	end := time.Now().UnixNano() / 1e6
//	println("【spend】", end-start)
//}

func baseParam(parameters map[string]string) {
	parameters[protocol.ContractTxIdParam] = "TX_ID"
	parameters[protocol.ContractCreatorOrgIdParam] = "CREATOR_ORG_ID"
	parameters[protocol.ContractCreatorRoleParam] = "CREATOR_ROLE"
	parameters[protocol.ContractCreatorPkParam] = "CREATOR_PK"
	parameters[protocol.ContractSenderOrgIdParam] = "SENDER_ORG_ID"
	parameters[protocol.ContractSenderRoleParam] = "SENDER_ROLE"
	parameters[protocol.ContractSenderPkParam] = "SENDER_PK"
	parameters[protocol.ContractBlockHeightParam] = "111"
}