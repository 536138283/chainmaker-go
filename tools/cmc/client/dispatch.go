/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker-go/tools/cmc/types"
	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	utils "chainmaker.org/chainmaker/utils/v2"
)

// Dispatch dispatch tx
func Dispatch(client *sdk.ChainClient, contractName, methodName string, kvs []*common.KeyValuePair,
	abi *abi.ABI, limit *common.Limit) {
	var (
		wgSendReq sync.WaitGroup
	)

	for i := 0; i < concurrency; i++ {
		wgSendReq.Add(1)
		go runInvokeContract(client, contractName, methodName, kvs, &wgSendReq, abi, limit)
	}

	wgSendReq.Wait()
}

// DispatchTimes dispatch tx in times
func DispatchTimes(client *sdk.ChainClient, contractName, method string, kvs []*common.KeyValuePair,
	evmMethod *abi.ABI) {
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

func runInvokeContract(client *sdk.ChainClient, contractName, methodName string,
	kvs []*common.KeyValuePair, wg *sync.WaitGroup, abi *abi.ABI, limit *common.Limit) {

	defer func() {
		wg.Done()
	}()

	for i := 0; i < totalCntPerGoroutine; i++ {
		if client.IsEnableNormalKey() {
			txId = utils.GetRandTxId()
		} else {
			txId = utils.GetTimestampTxId()
		}

		resp, err := client.InvokeContractWithLimit(contractName, methodName, txId, kvs, timeout, syncResult, limit)
		if err != nil {
			fmt.Printf("[ERROR] invoke contract failed, %s", err.Error())
			return
		}

		if resp.Code != common.TxStatusCode_SUCCESS {
			util.PrintPrettyJson(resp)
			return
		}

		if abi != nil && resp.ContractResult != nil {
			output, err := abi.Unpack(methodName, resp.ContractResult.Result)
			if err != nil {
				fmt.Println(err)
				return
			}
			util.PrintPrettyJson(types.EvmTxResponse{
				TxResponse: resp,
				ContractResult: &types.EvmContractResult{
					ContractResult: resp.ContractResult,
					Result:         fmt.Sprintf("%v", output),
				},
			})
		} else {
			util.PrintPrettyJson(resp)
		}
	}
}

func runInvokeContractOnce(client *sdk.ChainClient, contractName, method string, kvs []*common.KeyValuePair,
	wg *sync.WaitGroup, evmMethod *abi.ABI) {

	defer func() {
		wg.Done()
	}()

	txId := sdkutils.GetTimestampTxId()
	resp, err := client.InvokeContract(contractName, method, txId, kvs, timeout, syncResult)
	if err != nil {
		fmt.Printf("[ERROR] invoke contract failed, %s", err.Error())
		return
	}

	if resp.Code != common.TxStatusCode_SUCCESS {
		fmt.Printf("[ERROR] invoke contract failed, [code:%d]/[msg:%s]/[txId:%s]\n", resp.Code, resp.Message, txId)
		return
	}

	if evmMethod != nil && resp.ContractResult != nil {
		output, err := evmMethod.Unpack(method, resp.ContractResult.Result)
		if err != nil {
			fmt.Println(err)
			return
		}
		resp.ContractResult.Result = []byte(fmt.Sprintf("%v", output))
	}

	fmt.Printf("INVOKE contract resp, [code:%d]/[msg:%s]/[contractResult:%+v]/[txId:%s]\n", resp.Code, resp.Message,
		resp.ContractResult, txId)
}

func invokeContract(client *sdk.ChainClient, contractName, methodName, txId string,
	kvs []*common.KeyValuePair, abi *abi.ABI, limit *common.Limit) {

	resp, err := client.InvokeContractWithLimit(contractName, methodName, txId, kvs, timeout, syncResult, limit)
	if err != nil {
		fmt.Printf("[ERROR] invoke contract failed, %s", err.Error())
		return
	}

	if resp.Code != common.TxStatusCode_SUCCESS {
		util.PrintPrettyJson(resp)
		return
	}

	if abi != nil && resp.ContractResult != nil {
		output, err := abi.Unpack(methodName, resp.ContractResult.Result)
		if err != nil {
			fmt.Println(err)
			return
		}
		util.PrintPrettyJson(types.EvmTxResponse{
			TxResponse: resp,
			ContractResult: &types.EvmContractResult{
				ContractResult: resp.ContractResult,
				Result:         fmt.Sprintf("%v", output),
			},
		})
	} else {
		util.PrintPrettyJson(resp)
	}
}
