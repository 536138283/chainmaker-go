package protocol

import "contract-sdk-test1/pb_sdk/protogo"

type Handler interface {
	HandleMessage(message *protogo.ContractMessage) error

	SetStream(stream protogo.Contract_ConnectServer)
}
