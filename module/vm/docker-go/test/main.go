package main

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker/common/random/uuid"
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

const (
	maxRecvMessageSize = 100 * 1024 * 1024 // 100 MiB
	maxSendMessageSize = 100 * 1024 * 1024 // 100 MiB
	port               = ":12355"
	chanSize           = 1000
	//stateChanSize      = 1000
)

var (
	ContractPath  = ""
	ContractName  = ""
	MountSockPath = ""
)

func InitTest() {
	ContractPath = "/home/jianan/Documents/workspace/chainmaker-go/test/wasm/docker-go-contract20_big"
	ContractName = "contract1p2"
	MountSockPath = "/home/jianan/Documents/workspace/chainmaker-go/module/vm/docker-go/mount/sock/cdm.sock"
}

func main() {

	fmt.Println("start test")

	InitTest()

	createContract := false

	stream, _ := initGRPCConnect(true)
	client := NewCDMClient(stream)

	// 1) 合约创建
	if createContract {
		testDeployContract(client)
		time.Sleep(5 * time.Second)
	}

	// 2) 批量测试
	txNum := 10

	for i := 0; i < 50; i++ {
		testPerformance(client, txNum, i)
		time.Sleep(50 * time.Microsecond)
	}

	err := client.stream.CloseSend()
	if err != nil {
		fmt.Println("err in close send")
		return
	}

	fmt.Println("end test")

}

func testDeployContract(client *CDMClient) {

	txId := GetRandTxId()

	fmt.Printf("\n============ create contract %s [%s] ============\n", ContractName, txId)

	cdmMsg := contractCreateMsg(txId)

	startTime := time.Now()
	err := client.stream.Send(cdmMsg)
	if err != nil {
		return
	}
	fmt.Println("deploy time is: ", time.Since(startTime))

	recvMsg, _ := client.stream.Recv()
	var result protogo.TxResponse
	_ = proto.Unmarshal(recvMsg.Payload, &result)
	fmt.Printf("\n============ create contract result ============\n [%s]\n", result.String())

}

func testPerformance(client *CDMClient, txNum, batchSeq int) {

	fmt.Printf("\n============ test performance, tx number [%d] for [%d] ============\n", txNum, batchSeq)

	startTime := time.Now()

	waitc := make(chan struct{})
	recvNum := 0

	go func() {
		for {
			_, err := client.stream.Recv()

			if err == io.EOF {
				fmt.Println("recv eof")
				close(waitc)
				return
			}

			if err != nil {
				fmt.Println(err)
				return
			}

			recvNum++
			if recvNum >= txNum {
				fmt.Printf("[%d] tx running time is: [%s]\n", txNum, time.Since(startTime))
				close(waitc)
				return
			}

		}
	}()

	for i := 0; i < txNum; i++ {
		reqMsg := contractInvokeMsg()
		err := client.stream.Send(reqMsg)
		if err != nil {
			fmt.Println("fail to send req msg: ", err)
			return
		}
	}

	<-waitc

}

func contractCreateMsg(txId string) *protogo.CDMMessage {

	// construct cdm message
	params := make(map[string]string)
	contractBin, _ := ioutil.ReadFile(ContractPath)

	txRequest := &protogo.TxRequest{
		TxId:            txId,
		ContractName:    ContractName,
		ContractVersion: "1.0.0",
		Method:          "init_contract",
		ByteCode:        contractBin,
		Parameters:      params,
	}

	txPayload, _ := proto.Marshal(txRequest)

	cdmMessage := &protogo.CDMMessage{
		TxId:    txId,
		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
		Payload: txPayload,
	}
	return cdmMessage
}

func contractInvokeMsg() *protogo.CDMMessage {

	txId := GetRandTxId()

	// construct cdm message
	params := make(map[string]string)
	params["arg0"] = "sum"
	params["arg1"] = "1"
	params["arg2"] = "2"

	txRequest := &protogo.TxRequest{
		TxId:            txId,
		ContractName:    ContractName,
		ContractVersion: "1.2.1",
		Method:          "invoke_contract",
		ByteCode:        nil,
		Parameters:      params,
	}

	txPayload, _ := proto.Marshal(txRequest)

	cdmMessage := &protogo.CDMMessage{
		TxId:    txId,
		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
		Payload: txPayload,
	}
	return cdmMessage
}

// NewClientConn create client connection
func NewClientConn(udsOpen bool) (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMessageSize),
			grpc.MaxCallSendMsgSize(maxSendMessageSize),
		),
	}

	if udsOpen {

		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
			unixAddress, _ := net.ResolveUnixAddr("unix", sock)
			conn, err := net.DialUnix("unix", nil, unixAddress)
			return conn, err
		}))

		sockAddress := MountSockPath

		return grpc.DialContext(context.Background(), sockAddress, dialOpts...)

	}

	return grpc.Dial(port, dialOpts...)

}

// GetCDMClientStream get rpc stream
func GetCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
}

func initGRPCConnect(udsOpen bool) (protogo.CDMRpc_CDMCommunicateClient, error) {
	conn, err := NewClientConn(udsOpen)
	if err != nil {
		fmt.Println("fail to create connection: ", err)
		return nil, err
	}

	stream, err := GetCDMClientStream(conn)
	if err != nil {
		fmt.Println("fail to get connection stream: ", err)
		return nil, err
	}

	return stream, nil
}

type CDMClient struct {
	txSendCh    chan *protogo.CDMMessage // channel receive tx from docker-go instance
	stateSendCh chan *protogo.CDMMessage // channel receive state response

	lock      sync.Mutex
	recvChMap map[string]chan *protogo.CDMMessage // store tx_id to chan, retrieve chan to send tx response back to docker-go instance

	stream protogo.CDMRpc_CDMCommunicateClient

	stop chan bool
}

func NewCDMClient(stream protogo.CDMRpc_CDMCommunicateClient) *CDMClient {

	return &CDMClient{
		txSendCh:  make(chan *protogo.CDMMessage, chanSize),
		recvChMap: make(map[string]chan *protogo.CDMMessage),
		lock:      sync.Mutex{},
		stream:    stream,
		stop:      make(chan bool),
	}
}

func GetRandTxId() string {
	return uuid.GetUUID() + uuid.GetUUID()
}
