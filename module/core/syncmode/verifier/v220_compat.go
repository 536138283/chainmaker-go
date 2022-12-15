/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0

This file is for version compatibility
*/

package verifier

import (
	chainConfConfig "chainmaker.org/chainmaker/pb-go/v3/config"
	"chainmaker.org/chainmaker/protocol/v3"
)

var _ protocol.Watcher = (*BlockVerifierImpl)(nil)

// Module return module name core
func (v *BlockVerifierImpl) Module() string {
	return ModuleNameCore
}

// Watch set the chainConf block by chain config block
func (v *BlockVerifierImpl) Watch(chainConfig *chainConfConfig.ChainConfig) error {
	v.chainConf.ChainConfig().Block = chainConfig.Block
	v.log.Infof("update chainconf,blockverify[%v]", v.chainConf.ChainConfig().Block)
	return nil
}
