package wasmertest

import (
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"chainmaker.org/chainmaker-go/vm/test"
	"chainmaker.org/chainmaker-go/wasmer"
	wasm "chainmaker.org/chainmaker-go/wasmer/wasmer-go"
	"fmt"
	"testing"
	"time"
)

func TestEncryptDataFunc(t *testing.T) {
	fmt.Println("\n\n\n =========== Encrypt Test ================")
	contractName := "ContractCounter"
	contractVersion := "1.0.0"
	wasmFile := "../../../../test/wasm/encrypt_data.wasm"

	contractId, txContext, bytes := test.InitContextTest(contractName, contractVersion, wasmFile, commonPb.RuntimeType_WASMER)

	bytes, _ = wasm.ReadBytes(wasmFile)
	println("bytes len", len(bytes))

	pool := test.GetVmPoolManager()
	println("start")
	start := time.Now().UnixNano() / 1e6

	invokeSaveAuthData(contractId, txContext, pool, bytes)
	//invokeLoopForTestOutOfGas(contractId, txContext, pool, bytes)

	end := time.Now().UnixNano() / 1e6
	println("end 【spend】", end-start)
	//time.Sleep(time.Second * 5)
}


func invokeSaveAuthData(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "save_auth_data"
	parameters := make(map[string]string)
	baseParam(parameters)
	parameters["value"] = "567124123"
	parameters["sign"] = "MEUCIEGi5PH4Sum9v4AL5ob+lq4jiwRseWtYi4gEtjnSb0BFAiEAip7z7UJE/clX9gX2ndNJopSVDNyyRKfIeoi1LQte7aM="
	parameters["originalData"] = "052cce07a3c544558a29f4d6062b4f00"
	parameters["publicKeyXy"] = "BEm4d5Cdy3oF79O6gwLI/n3N0jClaKnHUHKzWo8Gas4Y/J8wBiOPP92Uii/rumYt5my+xKZuCRlgjJ+o8W0CUAE="
	parameters["forever"] = "true"
	parameters["contract_name"] = "taifu_contract"

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}

func invokeLoopForTestOutOfGas(contractId *commonPb.ContractId, txContext protocol.TxSimContext, pool *wasmer.VmPoolManager, byteCode []byte) {
	method := "loop_for_test_out_of_gas"
	parameters := make(map[string]string)
	baseParam(parameters)

	runtime, _ := pool.NewRuntimeInstance(contractId, byteCode)
	runtime.Invoke(contractId, method, byteCode, parameters, txContext, 0)
}