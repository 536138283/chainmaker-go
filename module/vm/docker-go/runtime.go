package docker_go

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/api"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo/outside"
	commonPb "chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/protocol"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
)

// RuntimeInstance evm runtime
type RuntimeInstance struct {
	ContainerName string
	ChainId       string // chain id
}

// Invoke contract by call vm, send tx to docker and get result after all txs finished
func (r *RuntimeInstance) Invoke(contractId *commonPb.ContractId, method string, byteCode []byte, parameters map[string]string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (contractResult *commonPb.ContractResult) {
	//txId := txSimContext.GetTx().GetHeader().TxId

	//startTime := utils.CurrentTimeMillisSeconds()
	// return whole snapshot

	// split the whole snapshot to contractResult

	// only return the runtimeContractResult

	// construct tx request and send to docker rpc
	txRequest := &outside.TxRequest{
		ContractId: nil,
		Method:     method,
		ByteCode:   byteCode,
		Parameters: parameters,
	}

	result := r.startRpcClient(txRequest)

	contractResult = &commonPb.ContractResult{
		Code:          0,
		Result:        result.ContractResult.Result,
		Message:       result.ContractResult.Message,
		GasUsed:       0,
		ContractEvent: nil,
	}

	fmt.Println("-----------------------------------------------")
	fmt.Println(result)

	return contractResult
}

func (r *RuntimeInstance) startRpcClient(request *outside.TxRequest) *outside.TxResult {
	Port := "12355"
	conn, err := grpc.Dial(":"+Port, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("err when dial the server %v", err)
	}
	defer conn.Close()

	client := api.NewDockerRpcClient(conn)

	stream, err := client.RunContracts(context.Background())
	stream.Send(request)

	result, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalln(err)
	}

	return result

}
