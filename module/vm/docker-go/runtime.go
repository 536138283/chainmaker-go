package docker_go

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/groupcache/lru"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type CDMClient interface {
	GetTxSendCh() chan *protogo.CDMMessage

	GetStateSendCh() chan *protogo.CDMMessage

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
}

const (
	ContractDir = "C:\\Users\\jiana\\Desktop\\mount\\"
)

func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {
	txId := txSimContext.GetTx().GetHeader().TxId

	log.Println("--------------------------------------")
	log.Println("Start to run contract in docker")

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
	//todo set timeout
	for {
		recvMsg := <-responseCh

		switch recvMsg.Type {
		case protogo.CDMType_CDM_TYPE_GET_STATE:
			value, err := txSimContext.Get(contractId.ContractName, recvMsg.Payload)
			if err != nil {
				log.Println("fail to get state from sim context ", err)
				return
			}
			log.Println("get value: ", string(value))

			r.Client.GetStateSendCh() <- &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE,
				Payload: value,
			}

		case protogo.CDMType_CDM_TYPE_GET_BYTECODE:
			contractName := string(recvMsg.Payload)
			contractPath := filepath.Join(ContractDir, contractName)

			err := r.saveBytesToDisk(byteCode, contractPath)
			if err != nil {
				log.Println("fail to save bytecode to disk ", err)
				return
			}
			log.Println("get contract path: ", contractPath)

			r.Client.GetStateSendCh() <- &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE,
				Payload: nil,
			}

		case protogo.CDMType_CDM_TYPE_TX_RESPONSE:

			// construct response
			var txResponse protogo.TxResponse
			_ = proto.UnmarshalMerge(recvMsg.Payload, &txResponse)

			contractResult = &commonPb.ContractResult{
				Code:    commonPb.ContractResultCode(txResponse.Code),
				Result:  txResponse.Result,
				Message: txResponse.Message,
			}

			// merge the sim context write map
			for key, value := range txResponse.WriteMap {
				err := txSimContext.Put(contractId.ContractName, []byte(key), value)
				if err != nil {
					log.Println("fail to put in sim context: ", err)
					return nil
				}
			}

			close(responseCh)

			log.Println("-----------------------------------------------")
			log.Println("End to run contract in docker")
			log.Println(contractResult)

			return contractResult
		default:
			//todo error
			log.Println("unknown type:", recvMsg.Type)

		}

	}

}

func (r *RuntimeInstance) ConstructContractKey(contractName, contractVersion string) string {
	return contractName + ":" + contractVersion
}

func (r *RuntimeInstance) saveBytesToDisk(bytes []byte, newFilePath string) error {

	f, err := os.Create(newFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		return err
	}

	return f.Sync()
}
