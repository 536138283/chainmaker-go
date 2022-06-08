/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package model

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
)

// NewBlockEvent - define new block event object
type NewBlockEvent struct {
	BlockInfo *commonPb.BlockInfo
}

// NewContractEvent new contract event
type NewContractEvent struct {
	ContractEventInfoList *commonPb.ContractEventInfoList
}
