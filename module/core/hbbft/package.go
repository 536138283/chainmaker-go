/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hbbft

import (
	"chainmaker.org/chainmaker-go/pb/protogo/consensus/hbbft"
	"chainmaker.org/chainmaker-go/protocol"
)

type Packager struct {
	packagedSignal *hbbft.PackagedSignal
	packageStatus  bool
	txPool         protocol.TxPool
	ledgerCache    protocol.LedgerCache
}

func (p *Packager) Package() error {

	//判断当前高度
	//判断打包状态
	return nil
}
