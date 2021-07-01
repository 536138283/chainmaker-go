package docker_go

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/api"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"context"
	"google.golang.org/grpc"
	"log"
	"strings"
	"time"
)

// RuntimeInstance evm runtime
type RuntimeInstance struct {
	ContainerName string
	ChainId       string // chain id
}

const (
	dialTimeout        = 10 * time.Second
	maxRecvMessageSize = 100 * 1024 * 1024 // 100 MiB
	maxSendMessageSize = 100 * 1024 * 1024 // 100 MiB
)

// Invoke contract by call vm, send tx to docker and get result after all txs finished
func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {
	txId := txSimContext.GetTx().GetHeader().TxId

	//log.Println("--------------------------------------")
	//log.Println("Start to run contract in docker")

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    commonPb.ContractResultCode_FAIL,
		Result:  nil,
		Message: "",
	}

	// split args from parameters
	argsMap := make(map[string]string)

	for key, value := range parameters {
		if strings.Contains(key, "arg") {
			argsMap[key] = value
		}
	}

	// construct tx request and send to docker rpc
	txRequest := &outside.TxRequest{
		TxId:            txId,
		ContractName:    contractId.ContractName,
		ContractVersion: contractId.ContractVersion,
		Method:          method,
		ByteCode:        byteCode,
		Parameters:      argsMap,
	}

	result, err := r.RpcCall(txRequest)

	contractResult.Message = result.Message
	contractResult.Result = result.Result

	if err != nil {
		return contractResult
	}

	contractResult.Code = commonPb.ContractResultCode_OK

	//log.Println("-----------------------------------------------")
	//log.Println("End to run contract in docker")
	//log.Println(result)

	return contractResult
}

func (r *RuntimeInstance) initRpcConnection() (*grpc.ClientConn, error) {
	Port := "12355"
	conn, err := grpc.Dial(":"+Port, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(maxRecvMessageSize),
		grpc.MaxCallSendMsgSize(maxSendMessageSize),
	))

	if err != nil {
		log.Fatalf("err when dial the server %v", err)
	}

	return conn, nil
}

// RpcCall later change to stream send txs, and return whole result once
func (r *RuntimeInstance) RpcCall(request *outside.TxRequest) (*outside.ContractResult, error) {

	conn, err := r.initRpcConnection()

	defer conn.Close()

	client := api.NewDockerRpcClient(conn)

	result, err := client.RunContracts(context.Background(), request)

	if err != nil {
		return nil, err
	}

	return result, nil

}
