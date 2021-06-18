package protocol

import "chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"

type Manager interface {
	HandleTx(tx *outside.TxRequest) *outside.ContractResult
}
