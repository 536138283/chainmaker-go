/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import "chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"

type Scheduler interface {
	// GetTxReqCh get tx req chan
	GetTxReqCh() chan *protogo.TxRequest

	// GetTxResponseCh get tx response chan
	GetTxResponseCh() chan *protogo.TxResponse

	// GetGetStateReqCh get get_state request chan
	GetGetStateReqCh() chan *protogo.CDMMessage

	// RegisterResponseCh register response chan
	RegisterResponseCh(txId string, responseCh chan *protogo.CDMMessage)

	// GetResponseChByTxId get response chan
	GetResponseChByTxId(txId string) chan *protogo.CDMMessage

	// GetGetByteCodeReqCh get get_bytecode request chan
	GetGetByteCodeReqCh() chan *protogo.CDMMessage
}
