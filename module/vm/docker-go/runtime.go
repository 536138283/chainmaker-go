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
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	MountContractDir = "contracts"
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
	//log.Println("method: ", method)

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_FAIL,
		Result:  nil,
		Message: "",
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

			getStateResponse := &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE,
				Payload: nil,
			}

			value, err := txSimContext.Get(contractId.ContractName, recvMsg.Payload)
			if err != nil {
				// if has error, return payload is nil
				r.Log.Errorf("failt to get state from sim context: %s", err)
				r.Client.GetStateResponseSendCh() <- getStateResponse
				continue
			}

			r.Log.Debug("get value: ", string(value))
			getStateResponse.Payload = value

			r.Client.GetStateResponseSendCh() <- getStateResponse

		case protogo.CDMType_CDM_TYPE_GET_BYTECODE:

			getBytecodeResponse := &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE,
				Payload: nil,
			}

			contractFullName := string(recvMsg.Payload)             // contract1#1.0.0
			contractName := strings.Split(contractFullName, "#")[0] // contract1

			dockerConfig := localconf.ChainMakerConfig.DockerConfig
			hostMountDir := dockerConfig.HostMountDir

			contractDir := filepath.Join(hostMountDir, MountContractDir)
			contractZipPath := filepath.Join(contractDir, fmt.Sprintf("%s.7z", contractName)) // contract1.7z
			contractPathWithoutVersion := filepath.Join(contractDir, contractName)
			contractPathWithVersion := filepath.Join(contractDir, contractFullName)

			// save bytecode to disk
			err := r.saveBytesToDisk(byteCode, contractZipPath)
			if err != nil {
				r.Log.Errorf("fail to save bytecode to disk: %s", err)
				r.Client.GetStateResponseSendCh() <- getBytecodeResponse
				continue
			}
			r.Log.Debug("write zip file: ", contractZipPath)

			// extract 7z file
			unzipCommand := fmt.Sprintf("7z e %s -o%s -y", contractZipPath, contractDir) // contract1
			err = r.runCmd(unzipCommand)
			if err != nil {
				r.Log.Errorf("fail to extract contract: %s", err)
				r.Client.GetStateResponseSendCh() <- getBytecodeResponse
				continue
			}
			r.Log.Debug("extract zip file: ", contractZipPath)

			// remove 7z file
			err = os.Remove(contractZipPath)
			if err != nil {
				return r.errorResult(contractResult, err, "fail to remove zipped file")
			}

			// replace contract name to contractName:version
			err = os.Rename(contractPathWithoutVersion, contractPathWithVersion)
			if err != nil {
				r.Log.Errorf("fail to rename original file name: %s, please make sure contract name should be same as zipped file", err)
				r.Client.GetStateResponseSendCh() <- getBytecodeResponse
				continue
			}

			getBytecodeResponse.Payload = []byte(contractFullName)

			r.Client.GetStateResponseSendCh() <- getBytecodeResponse

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

// RunCmd exec cmd
func (r *RuntimeInstance) runCmd(command string) error {
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...)

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}
