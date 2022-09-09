/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
)

func Dispatch(client *sdk.ChainClient, contractName, method string, kvs []*common.KeyValuePair,
	evmMethod *ethabi.Method, limit *common.Limit) {
	var (
		wgSendReq sync.WaitGroup
	)

	for i := 0; i < concurrency; i++ {
		wgSendReq.Add(1)
		go runInvokeContract(client, contractName, method, kvs, &wgSendReq, evmMethod, limit)
	}

	wgSendReq.Wait()
}
func DispatchTimes(client *sdk.ChainClient, contractName, method string, kvs []*common.KeyValuePair,
	evmMethod *ethabi.Method) {
	var (
		wgSendReq sync.WaitGroup
	)
	times := util.MaxInt(1, sendTimes)
	wgSendReq.Add(times)
	for i := 0; i < times; i++ {
		go runInvokeContractOnce(client, contractName, method, kvs, &wgSendReq, evmMethod)
	}
	wgSendReq.Wait()
}

func runInvokeContract(client *sdk.ChainClient, contractName, method string, kvs []*common.KeyValuePair,
	wg *sync.WaitGroup, evmMethod *ethabi.Method, limit *common.Limit) {

	defer func() {
		wg.Done()
	}()

	for i := 0; i < totalCntPerGoroutine; i++ {
		invokeContract(client, contractName, method, "", kvs, evmMethod, limit)
	}
}

func runInvokeContractOnce(client *sdk.ChainClient, contractName, method string, kvs []*common.KeyValuePair,
	wg *sync.WaitGroup, evmMethod *ethabi.Method) {

	defer func() {
		wg.Done()
	}()

	invokeContract(client, contractName, method, "", kvs, evmMethod, nil)
}

func invokeContract(client *sdk.ChainClient, contractName, method, txId string, kvs []*common.KeyValuePair,
	evmMethod *ethabi.Method, limit *common.Limit) {
	adminKeys, adminCrts, adminOrgs, err := makeAdminInfo(client)
	if err != nil {
		fmt.Printf("makeAdminInfo failed, %s", err)
		return
	}
	payload := client.CreatePayload(txId, common.TxType_INVOKE_CONTRACT, contractName, method, kvs, 0, limit)
	endorsers, err := makeEndorsement(adminKeys, adminCrts, adminOrgs, client, payload)
	if err != nil {
		fmt.Printf("makeEndorsement failed, %s", err)
		return
	}
	req, err := client.GenerateTxRequest(payload, endorsers)
	if err != nil {
		fmt.Printf("GenerateTxRequest failed, %s", err)
		return
	}
	resp, err := client.SendTxRequest(req, timeout, syncResult)
	if err != nil {
		fmt.Printf("[ERROR] invoke contract failed, %s", err.Error())
		return
	}

	if evmMethod != nil && resp.ContractResult != nil && resp.ContractResult.Result != nil {
		output, err := util.DecodeOutputs(evmMethod, resp.ContractResult.Result)
		if err != nil {
			fmt.Println(err)
			return
		}
		resp.ContractResult.Result = []byte(fmt.Sprintf("%v", output))
	}

	util.PrintPrettyJson(resp)
}
