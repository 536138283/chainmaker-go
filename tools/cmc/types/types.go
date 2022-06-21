/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/store"
)

// BlockHeader extends of *common.BlockHeader
type BlockHeader struct {
	*common.BlockHeader
	BlockHash string `json:"block_hash,omitempty"`
}

// Block extends of *common.Block
type Block struct {
	*common.Block
	Header *BlockHeader `json:"header,omitempty"`
}

// BlockWithRWSet extends of *store.BlockWithRWSet
type BlockWithRWSet struct {
	*store.BlockWithRWSet
	Block *Block `json:"block,omitempty"`
}

// CreateUpgradeContractTxResponse extends of *common.TxResponse when upgrade contract
type CreateUpgradeContractTxResponse struct {
	*common.TxResponse
	ContractResult *CreateUpgradeContractContractResult `json:"contract_result"`
}

// CreateUpgradeContractContractResult extends of *common.ContractResult when upgrade contract
type CreateUpgradeContractContractResult struct {
	*common.ContractResult
	Result *common.Contract `json:"result"`
}

// EvmTxResponse extends of *common.TxResponse when tx is evm kind
type EvmTxResponse struct {
	*common.TxResponse
	ContractResult *EvmContractResult `json:"contract_result"`
}

// EvmContractResult extends of *common.ContractResult when tx is evm kind
type EvmContractResult struct {
	*common.ContractResult
	Result string `json:"result"`
}

// TxResponse extends of *common.TxResponse
type TxResponse struct {
	*common.TxResponse
	ContractResult *ContractResult `json:"contract_result"`
}

// ContractResult extends of *common.ContractResult
type ContractResult struct {
	*common.ContractResult
	Result *common.Contract `json:"result"`
}
