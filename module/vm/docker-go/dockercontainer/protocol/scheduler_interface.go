package protocol

import "chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"

type Scheduler interface {
	GetTxReqCh() chan *protogo.TxRequest

	GetTxResponseCh() chan *protogo.TxResponse

	GetGetStateReqCh() chan *protogo.CDMMessage

	RegisterResponseCh(txId string, responseCh chan *protogo.CDMMessage)

	GetResponseChByTxId(txId string) chan *protogo.CDMMessage
}
