package protocol

import "chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"

type Scheduler interface {

	// Start start tx and tx result channel
	Start()

	// Stop stop tx and tx result channel
	Stop()

	// GetTxCh get tx channel
	GetTxCh() chan *outside.TxRequest

	// GetTxResultCh get tx result channel
	GetTxResultCh() chan *outside.ContractResult

	// FreeHandler remove handler key in scheduler
	FreeHandler(contractName string)
}
