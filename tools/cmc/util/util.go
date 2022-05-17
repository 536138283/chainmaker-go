// Copyright (C) BABEC. All rights reserved.
// Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0s

package util

import (
	"encoding/hex"
	"fmt"

	"chainmaker.org/chainmaker/common/v2/evmutils"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/hokaccha/go-prettyjson"
)

func MaxInt(i, j int) int {
	if j > i {
		return j
	}
	return i
}

func ConvertParameters(pars map[string]string) []*common.KeyValuePair {
	var kvp []*common.KeyValuePair
	for k, v := range pars {
		kvp = append(kvp, &common.KeyValuePair{
			Key:   k,
			Value: []byte(v),
		})
	}
	return kvp
}

func CalcEvmContractName(contractName string) string {
	return hex.EncodeToString(evmutils.Keccak256([]byte(contractName)))[24:]
}

func PrintPrettyJson(data interface{}) {
	output, err := prettyjson.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(output))
}
