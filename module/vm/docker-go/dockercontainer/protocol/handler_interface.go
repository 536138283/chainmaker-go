package protocol

import "chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"

type Handler interface {
	GetTxCh() chan *outside.TxRequest

	GetTxResultCh() chan *outside.ContractResult

	Start()

	Stop()
}
