package docker_go

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/groupcache/lru"
	"strings"
)

type CDMClient interface {
	GetTxSendCh() chan *protogo.CDMMessage

	RegisterRecvChan(txId string, recvCh chan *protogo.CDMMessage)
}

type TmpCache interface {
	TmpAdd(key lru.Key, value interface{})

	TmpGet(key lru.Key) (value interface{}, ok bool)
}

// RuntimeInstance evm runtime
type RuntimeInstance struct {
	ContainerName string
	ChainId       string // chain id

	Client CDMClient

	TmpCache TmpCache
}

func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {
	txId := txSimContext.GetTx().GetHeader().TxId

	//log.Println("--------------------------------------")
	//log.Println("Start to run contract in docker")

	// split args from parameters
	argsMap := make(map[string]string)
	for key, value := range parameters {
		if strings.Contains(key, "arg") {
			argsMap[key] = value
		}
	}

	// construct cdm message
	txRequest := &protogo.TxRequest{
		TxId:            txId,
		ContractName:    contractId.ContractName,
		ContractVersion: contractId.ContractVersion,
		Method:          method,
		ByteCode:        nil,
		Parameters:      argsMap,
	}

	// lru test
	cacheKey := r.ConstructContractKey(contractId.ContractName, contractId.ContractVersion)
	_, ok := r.TmpCache.TmpGet(cacheKey)
	if !ok {
		r.TmpCache.TmpAdd(cacheKey, true)
		txRequest.ByteCode = byteCode
	} else {
		txRequest.ByteCode = nil
	}

	txPayload, _ := proto.Marshal(txRequest)

	cdmMessage := &protogo.CDMMessage{
		TxId:    txId,
		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
		Payload: txPayload,
	}

	// register result chan
	responseCh := make(chan *protogo.CDMMessage)
	r.Client.RegisterRecvChan(txId, responseCh)

	// send message to tx chan
	r.Client.GetTxSendCh() <- cdmMessage

	// wait this chan
	recvMsg := <-responseCh

	// construct response
	var txResponse protogo.TxResponse
	err := proto.UnmarshalMerge(recvMsg.Payload, &txResponse)
	if err != nil {
		return nil
	}

	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode(txResponse.Code),
		Result:  txResponse.Result,
		Message: txResponse.Message,
	}

	//fmt.Println(contractResult)
	// merge the sim context

	//log.Println("-----------------------------------------------")
	//log.Println("End to run contract in docker")

	return contractResult
}

func (r *RuntimeInstance) ConstructContractKey(contractName, contractVersion string) string {
	return contractName + ":" + contractVersion
}
