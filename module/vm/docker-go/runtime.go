package docker_go

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"os"
	"path/filepath"
	"strings"
)

type CDMClient interface {
	GetTxSendCh() chan *protogo.CDMMessage

	GetStateResponseSendCh() chan *protogo.CDMMessage

	RegisterRecvChan(txId string, recvCh chan *protogo.CDMMessage)
}

// RuntimeInstance evm runtime
type RuntimeInstance struct {
	ContainerName string
	ChainId       string // chain id
	Client        CDMClient
	Log           *logger.CMLogger
}

func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {
	txId := txSimContext.GetTx().GetHeader().TxId

	//log.Println("-----------")
	//log.Println("start contract")

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_FAIL,
		Result:  nil,
		Message: "",
	}

	// split args from parameters
	// todo is ok?
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
		Parameters:      parameters,
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
				return r.errorResult(contractResult, err, "fail to get state from sim context")
			}
			r.Log.Debug("get value: ", string(value))

			r.Client.GetStateResponseSendCh() <- &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE,
				Payload: value,
			}

		case protogo.CDMType_CDM_TYPE_GET_BYTECODE:
			contractName := string(recvMsg.Payload)

			dockerConfig := localconf.ChainMakerConfig.DockerConfig
			hostMountDir := dockerConfig.HostMountDir
			contractPath := filepath.Join(hostMountDir, "contracts", contractName)

			err := r.saveBytesToDisk(byteCode, contractPath)
			if err != nil {
				return r.errorResult(contractResult, err, "fail to save bytecode to disk")
			}
			r.Log.Debug("get contract path: ", contractPath)

			r.Client.GetStateResponseSendCh() <- &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE,
				Payload: nil,
			}

		case protogo.CDMType_CDM_TYPE_TX_RESPONSE:

			// construct response
			var txResponse protogo.TxResponse
			_ = proto.UnmarshalMerge(recvMsg.Payload, &txResponse)

			contractResult.Code = commonPb.ContractResultCode(txResponse.Code)
			contractResult.Result = txResponse.Result
			contractResult.Message = txResponse.Message

			// merge the sim context write map
			for key, value := range txResponse.WriteMap {
				err := txSimContext.Put(contractId.ContractName, []byte(key), value)
				if err != nil {
					return r.errorResult(contractResult, err, "fail to put in sim context")
				}
			}

			close(responseCh)

			//log.Println("----------------------------")
			//log.Println(contractResult)

			return contractResult
		default:
			err := fmt.Errorf("unknow type")
			return r.errorResult(contractResult, err, "fail to receive request")

		}

	}

}

func (r *RuntimeInstance) errorResult(contractResult *commonPb.ContractResult, err error, errMsg string) *commonPb.ContractResult {
	contractResult.Code = commonPb.ContractResultCode_FAIL
	if err != nil {
		errMsg += ", " + err.Error()
	}
	contractResult.Message = errMsg
	r.Log.Error(errMsg)
	return contractResult
}

func (r *RuntimeInstance) ConstructContractKey(contractName, contractVersion string) string {
	return contractName + ":" + contractVersion
}

func (r *RuntimeInstance) saveBytesToDisk(bytes []byte, newFilePath string) error {

	f, err := os.Create(newFilePath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)

	_, err = f.Write(bytes)
	if err != nil {
		return err
	}

	return f.Sync()
}
