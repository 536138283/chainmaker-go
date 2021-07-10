package protocol

import "chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"

type Scheduler interface {
	GetTxReqCh() chan *protogo.TxRequest

	GetTxResponseCh() chan *protogo.TxResponse
}
