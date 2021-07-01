package protocol

import "chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"

type Scheduler interface {

	// GetTxCh get tx channel
	GetTxCh() chan *outside.TxRequest

	// GetTxResultCh get tx result channel
	GetTxResultCh() chan *outside.ContractResult
}
