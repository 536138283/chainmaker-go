/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package filtercommon

import (
	bn "chainmaker.org/chainmaker/common/v2/birdsnest"
	sbn "chainmaker.org/chainmaker/common/v2/shardingbirdsnest"
)

type TxFilterType int32

const (
	TxFilterTypeDefault           TxFilterType = 0
	TxFilterTypeBirdsNest         TxFilterType = 1
	TxFilterTypeMap               TxFilterType = 2
	TxFilterTypeShardingBirdsNest TxFilterType = 3
)

// TxFilterConfig transaction filter config
type TxFilterConfig struct {
	// Transaction filter type
	Type TxFilterType `json:"type,omitempty"`
	// Bird's nest configuration
	BirdsNest *bn.BirdsNestConfig `json:"birds_nest,omitempty"`
	// Sharding bird's nest configuration
	ShardingBirdsNest *sbn.ShardingBirdsNestConfig `json:"sharding_birds_nest,omitempty"`
}
