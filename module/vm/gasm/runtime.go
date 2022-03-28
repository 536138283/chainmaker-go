/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package gasm

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"sync"

	"chainmaker.org/chainmaker-go/common/serialize"
	"chainmaker.org/chainmaker-go/gasm/gasm-go/hostfunc"
	"chainmaker.org/chainmaker-go/gasm/gasm-go/waci"
	"chainmaker.org/chainmaker-go/gasm/gasm-go/wasi"
	"chainmaker.org/chainmaker-go/gasm/gasm-go/wasm"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/golang/groupcache/lru"
)

const (
	LruCacheSize = 64
)

type wasmModMap struct {
	modCache *lru.Cache
}

var inst *wasmModMap
var mu sync.Mutex

func putContractDecodedMod(chainId string, contractId *commonPb.ContractId, mod *wasm.Module) {
	mu.Lock()
	defer mu.Unlock()

	if inst == nil {
		inst = &wasmModMap{
			modCache: lru.New(LruCacheSize),
		}
	}
	modName := chainId + contractId.ContractName + protocol.ContractStoreSeparator + contractId.ContractVersion
	inst.modCache.Add(modName, mod)
}

func getContractDecodedMod(chainId string, contractId *commonPb.ContractId) *wasm.Module {
	mu.Lock()
	defer mu.Unlock()

	if inst == nil {
		inst = &wasmModMap{
			modCache: lru.New(LruCacheSize),
		}
	}

	modName := chainId + contractId.ContractName + protocol.ContractStoreSeparator + contractId.ContractVersion
	if mod, ok := inst.modCache.Get(modName); ok {
		return mod.(*wasm.Module)
	}
	return nil
}

func removeContractDecodedMod(chainId string, contractId *commonPb.ContractId) *wasm.Module {
	mu.Lock()
	defer mu.Unlock()

	if inst == nil {
		inst = &wasmModMap{
			modCache: lru.New(LruCacheSize),
		}
	}

	modName := chainId + contractId.ContractName + protocol.ContractStoreSeparator + contractId.ContractVersion
	inst.modCache.Remove(modName)
	return nil
}

// RuntimeInstance gasm runtime
type RuntimeInstance struct {
	ChainId string
	Log     *logger.CMLogger
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte,
	parameters map[string]string, txContext protocol.TxSimContext, gasUsed uint64) (
	contractResult *commonPb.ContractResult, specialTxType protocol.ExecOrderTxType) {
	tx := txContext.GetTx()

	defer func() {
		if err := recover(); err != nil {
			r.Log.Errorf("failed to invoke gasm, tx id:%s, error:%s", tx.Header.TxId, err)
			// if panic, set return value
			contractResult.Code = commonPb.ContractResultCode_FAIL
			if e, ok := err.(error); ok {
				contractResult.Message = e.Error()
			} else if e, ok := err.(string); ok {
				contractResult.Message = e
			}
			specialTxType = protocol.ExecOrderTxTypeNormal
			debug.PrintStack()
		}
	}()

	// set default return value
	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_OK,
		Result:  nil,
		Message: "",
	}
	specialTxType = protocol.ExecOrderTxTypeNormal

	var vm *wasm.VirtualMachine
	var mod *wasm.Module
	var err error
	waciInstance := &waci.WaciInstance{
		TxSimContext:   txContext,
		ContractId:     contractId,
		ContractResult: contractResult,
		Log:            r.Log,
		ChainId:        r.ChainId,
		Method:         method,
		SpecialTxType:  protocol.ExecOrderTxTypeNormal,
	}
	wasiInstance := &wasi.WasiInstance{}
	builder := newBuilder(wasiInstance, waciInstance)
	externalMods := builder.Done()

	baseMod := getContractDecodedMod(r.ChainId, contractId)
	if baseMod == nil {
		if baseMod, err = wasm.DecodeModule(bytes.NewBuffer(byteCode)); err != nil {
			contractResult.Code = commonPb.ContractResultCode_FAIL
			contractResult.Message = err.Error()
			r.Log.Errorf("invoke gasm, tx id:%s, error= %s, bytecode len=%d",
				tx.GetHeader().TxId, err.Error(), len(byteCode))
			return
		}

		if err = baseMod.BuildIndexSpaces(externalMods); err != nil {
			contractResult.Code = commonPb.ContractResultCode_FAIL
			contractResult.Message = err.Error()
			r.Log.Errorf("invoke gasm, failed to build wasm index space, tx id:%s, error= %s, bytecode len=%d",
				tx.GetHeader().TxId, err.Error(), len(byteCode))
			return
		}
		putContractDecodedMod(r.ChainId, contractId, baseMod)
		mod = baseMod
	} else {
		mod = &wasm.Module{
			SecTypes:     baseMod.SecTypes,
			SecImports:   baseMod.SecImports,
			SecFunctions: baseMod.SecFunctions,
			SecTables:    baseMod.SecTables,
			SecMemory:    baseMod.SecMemory,
			SecGlobals:   baseMod.SecGlobals,
			SecExports:   baseMod.SecExports,
			SecStart:     baseMod.SecStart,
			SecElements:  baseMod.SecElements,
			SecCodes:     baseMod.SecCodes,
			SecData:      baseMod.SecData,
		}
		if err = mod.BuildIndexSpacesUsingOldNativeFunction(externalMods, baseMod.IndexSpace.Function); err != nil {
			contractResult.Code = commonPb.ContractResultCode_FAIL
			contractResult.Message = err.Error()
			r.Log.Errorf("invoke gasm, failed to build wasm index space using old native function, tx id:%s, "+
				"error= %s, bytecode len=%d", tx.GetHeader().TxId, err.Error(), len(byteCode))
			return
		}
	}

	if vm, err = wasm.NewVM(mod, gasUsed, protocol.GasLimit, protocol.TimeLimit); err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		contractResult.Message = err.Error()
		r.Log.Errorf("invoke gasm,tx id:%s, error= %s", tx.GetHeader().TxId, err.Error())
		r.removeModByMethod(method, contractId)
		return
	}
	var paramMarshalBytes []byte
	var runtimeSdkType []uint64
	if runtimeSdkType, _, err = vm.ExecExportedFunction(protocol.ContractRuntimeTypeMethod); err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		contractResult.Message = err.Error()
		r.Log.Errorf("invoke gasm,tx id:%s, failed to call args(), error=", tx.GetHeader().TxId, err.Error())
		r.removeModByMethod(method, contractId)
		return
	}

	parameters[protocol.ContractContextPtrParam] = "0" // 兼容rust
	if uint64(commonPb.RuntimeType_GASM) == runtimeSdkType[0] {
		ec := serialize.NewEasyCodecWithMap(parameters)
		paramMarshalBytes = ec.Marshal()
	} else {
		r.runtimeTypeError(contractId, runtimeSdkType, contractResult)
		return
	}

	var allocateSize = uint64(len(paramMarshalBytes))
	var allocatePtr []uint64
	if allocatePtr, _, err = vm.ExecExportedFunction(protocol.ContractAllocateMethod, allocateSize); err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		contractResult.Message = err.Error()
		r.Log.Errorf("invoke gasm, tx id:%s,failed to allocate, error=", tx.GetHeader().TxId, err.Error())
		r.removeModByMethod(method, contractId)
		return
	}
	copy(vm.Memory[allocatePtr[0]:allocatePtr[0]+allocateSize], paramMarshalBytes)

	// run invoke method may modify waciInstance's ExecOrderTxType
	if ret, retTypes, err := vm.ExecExportedFunction(method); err != nil {
		contractResult.Code = commonPb.ContractResultCode_FAIL
		contractResult.Message = err.Error()
		r.Log.Errorf("invoke gasm, tx id:%s,error=%+v", tx.GetHeader().TxId, err.Error())
		r.removeModByMethod(method, contractId)
	} else {
		contractResult.ContractEvent = waciInstance.ContractEvent
		r.Log.Debugf("invoke gasm success, tx id:%s, gas cost %+v,[IGNORE: ret %+v, retTypes %+v]",
			tx.GetHeader().TxId, vm.Gas, ret, retTypes)
	}
	specialTxType = waciInstance.SpecialTxType

	//// gasm 无需释放内存, 借助golang自动回收
	//if false {
	//	if ret, retTypes, err := vm.ExecExportedFunction(protocol.ContractDeallocateMethod); err != nil {
	//		contractResult.Code = commonPb.ContractResultCode_FAIL
	//		contractResult.Message = err.Error()
	//		r.Log.Errorf("invoke gasm, tx id:%s,error=%+v", tx.GetHeader().TxId, err.Error())
	//	} else {
	//		r.Log.Debugf("invoke gasm deallocate success,tx id:%s, gas cost %+v,[IGNORE: ret %+v, "+
	//			"retTypes %+v]", tx.GetHeader().TxId, vm.Gas, ret, retTypes)
	//	}
	//}
	contractResult.GasUsed = int64(vm.Gas)
	return
}

func (r *RuntimeInstance) runtimeTypeError(contractId *commonPb.ContractId, runtimeSdkType []uint64,
	contractResult *commonPb.ContractResult) {
	msg := fmt.Sprintf("runtime type error, expect gasm:%d, but got %d", uint64(commonPb.RuntimeType_GASM),
		runtimeSdkType[0])
	contractResult.Code = commonPb.ContractResultCode_FAIL
	contractResult.Message = msg
	r.Log.Errorf(msg)
	removeContractDecodedMod(r.ChainId, contractId)
}

// removeModByMethod
func (r *RuntimeInstance) removeModByMethod(method string, contractId *commonPb.ContractId) {
	if method == commonPb.ManageUserContractFunction_INIT_CONTRACT.String() {
		removeContractDecodedMod(r.ChainId, contractId)
	}
}

func newBuilder(wasiInstance *wasi.WasiInstance, waciInstance *waci.WaciInstance) *hostfunc.ModuleBuilder {
	builder := hostfunc.NewModuleBuilder()
	builder.MustSetFunction(wasi.WasiUnstableModuleName, "fd_write", wasiInstance.FdWrite)
	builder.MustSetFunction(wasi.WasiModuleName, "fd_write", wasiInstance.FdWrite)
	builder.MustSetFunction(wasi.WasiModuleName, "fd_read", wasiInstance.FdRead)
	builder.MustSetFunction(wasi.WasiModuleName, "fd_close", wasiInstance.FdClose)
	builder.MustSetFunction(wasi.WasiModuleName, "fd_seek", wasiInstance.FdSeek)
	builder.MustSetFunction(wasi.WasiModuleName, "proc_exit", wasiInstance.ProcExit)

	builder.MustSetFunction(waci.WaciModuleName, "sys_call", waciInstance.SysCall)
	builder.MustSetFunction(waci.WaciModuleName, "log_message", waciInstance.LogMsg)

	//builder.MustSetFunction(waci.WaciModuleName, "get_state_len_from_chain", waciInstance.GetStateLen)
	//builder.MustSetFunction(waci.WaciModuleName, "get_state_from_chain", waciInstance.GetState)
	//builder.MustSetFunction(waci.WaciModuleName, "put_state_to_chain", waciInstance.PutState)
	//builder.MustSetFunction(waci.WaciModuleName, "delete_state_from_chain", waciInstance.DeleteState)
	//builder.MustSetFunction(waci.WaciModuleName, "success_result_to_chain", waciInstance.SuccessResult)
	//builder.MustSetFunction(waci.WaciModuleName, "error_result_to_chain", waciInstance.ErrorResult)
	//builder.MustSetFunction(waci.WaciModuleName, "call_contract_len_from_chain", waciInstance.CallContractLen)
	//builder.MustSetFunction(waci.WaciModuleName, "call_contract_from_chain", waciInstance.CallContract)
	return builder
}
