/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package demo

import (
	"log"

	"chainmaker.org/chainmaker/contract-sdk-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sandbox"
	"chainmaker.org/chainmaker/contract-sdk-go/v2/sdk"
)

type ContractFactCut struct {
}

func (c *ContractFactCut) InitContract() protogo.Response {
	return sdk.Success([]byte("Init success"))
}

func (c *ContractFactCut) UpgradeContract() protogo.Response {
	return sdk.Success([]byte("Upgrade success"))
}

func (c *ContractFactCut) InvokeContract(method string) protogo.Response {

	switch method {
	case "save":
		return c.Save()
	case "findByFileHash":
		return c.FindByFileHash()
	default:
		return sdk.Error("invalid method")
	}
}

func (c *ContractFactCut) Save() protogo.Response {
	key := string(sdk.Instance.GetArgs()["file_key"])
	name := sdk.Instance.GetArgs()["file_name"]

	err := sdk.Instance.PutStateByte(key, "", name)
	if err != nil {
		return sdk.Error("fail to save")
	}
	return sdk.Success([]byte("success"))
}

func (c *ContractFactCut) FindByFileHash() protogo.Response {
	key := string(sdk.Instance.GetArgs()["file_key"])

	_, err := sdk.Instance.GetStateByte(key, "")
	if err != nil {
		return sdk.Error("fail to find")
	}
	return sdk.Success([]byte(""))
}

func main() {
	err := sandbox.Start(new(ContractFactCut))
	if err != nil {
		log.Fatal(err)
	}
}
