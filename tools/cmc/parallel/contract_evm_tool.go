/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package parallel

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"chainmaker.org/chainmaker-go/tools/cmc/util"
	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	"chainmaker.org/chainmaker/common/v2/json"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
)

var contractAbi *abi.ABI
var makeAbiOnce sync.Once

func makeInvokeEvmKvs(method string, abiPath string, pairs []*commonPb.KeyValuePair) ([]*commonPb.KeyValuePair, error) {
	makeAbiOnce.Do(func() {
		abiBytes, err := ioutil.ReadFile(abiPath)
		if err != nil {
			fmt.Printf("failed to ioutil.ReadFile(abiPath): %v", err)
			os.Exit(1)
		}

		contractAbi, err = abi.JSON(bytes.NewReader(abiBytes))
		if err != nil {
			fmt.Printf("failed to abi.JSON(bytes.NewReader(abiBytes)): %v", err)
			os.Exit(1)
		}
	})

	var evmParams []map[string]string
	for _, kv := range pairs {
		evmParams = append(evmParams, map[string]string{kv.Key: string(kv.Value)})
	}

	paramsJson, err := json.Marshal(evmParams)
	if err != nil {
		return nil, err
	}

	inputData, err := util.Pack(contractAbi, method, string(paramsJson))
	if err != nil {
		return nil, err
	}

	return []*commonPb.KeyValuePair{
		{
			Key:   "data",
			Value: []byte(hex.EncodeToString(inputData)),
		},
	}, nil
}
