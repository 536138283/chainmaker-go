package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"context"
	"google.golang.org/grpc"
	"io"
	"strconv"
	"sync"
)

type CDMClient struct {
	txSendCh            chan *protogo.CDMMessage // channel receive tx from docker-go instance
	stateResponseSendCh chan *protogo.CDMMessage // channel receive state response

	lock      sync.Mutex
	recvChMap map[string]chan *protogo.CDMMessage // store tx_id to chan, retrieve chan to send tx response back to docker-go instance

	stream protogo.CDMRpc_CDMCommunicateClient

	logger *logger.CMLogger

	stop chan struct{}
}

func NewCDMClient(chainId string) *CDMClient {

	dockerConfig := localconf.ChainMakerConfig.DockerConfig

	return &CDMClient{
		txSendCh:            make(chan *protogo.CDMMessage, dockerConfig.DockerVmConfig.TxSize),   // tx request
		stateResponseSendCh: make(chan *protogo.CDMMessage, dockerConfig.DockerVmConfig.TxSize*8), // get_state response and bytecode response
		recvChMap:           make(map[string]chan *protogo.CDMMessage),
		lock:                sync.Mutex{},
		stream:              nil,
		logger:              logger.GetLoggerByChain("[CDM Client]", chainId),
		stop:                nil,
	}
}

func (c *CDMClient) GetTxSendCh() chan *protogo.CDMMessage {
	return c.txSendCh
}

func (c *CDMClient) GetStateResponseSendCh() chan *protogo.CDMMessage {
	return c.stateResponseSendCh
}

func (c *CDMClient) RegisterRecvChan(txId string, recvCh chan *protogo.CDMMessage) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.logger.Debugf("register recv chan [%s]", txId[:5])
	c.recvChMap[txId] = recvCh
}

func (c *CDMClient) deleteRecvChan(txId string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.logger.Debugf("delete recv chan [%s]", txId[:5])
	delete(c.recvChMap, txId)
}

func (c *CDMClient) StartClient() bool {

	c.logger.Debugf("start cdm client..")
	conn, err := NewClientConn()
	if err != nil {
		c.logger.Errorf("fail to create connection: %s", err)
		return false
	}

	stream, err := GetCDMClientStream(conn)
	if err != nil {
		c.logger.Errorf("fail to get connection stream: %s", err)
		return false
	}

	c.stream = stream
	c.stop = make(chan struct{})

	go c.sendMsgRoutine()

	go c.recvMsgRoutine()

	return true
}

func (c *CDMClient) closeConnection() {
	// close two goroutine
	close(c.stop)
	// close stream
	err := c.stream.CloseSend()
	if err != nil {
		return
	}
}

func (c *CDMClient) sendMsgRoutine() {

	c.logger.Debugf("start sending cdm message ")
	// listen three chan:
	// txSendCh: used to send tx to docker manager
	// stateResponseSendCh: used to send get state response or bytecode response to docker manager
	// stopCh

	var err error

	for {
		select {
		case txMsg := <-c.txSendCh:
			err = c.sendCDMMsg(txMsg)
		case stateMsg := <-c.stateResponseSendCh:
			err = c.sendCDMMsg(stateMsg)
		case <-c.stop:
			c.logger.Debugf("close send cdm msg")
			return
		}

		if err != nil {
			c.logger.Errorf("fail to send msg: %s", err)
		}
	}

}

func (c *CDMClient) recvMsgRoutine() {

	c.logger.Debugf("start receiving cdm message ")

	var err error

	for {

		select {
		case <-c.stop:
			c.logger.Debugf("close recv cdm msg")
			return
		default:
			recvMsg, recvErr := c.stream.Recv()

			if recvErr == io.EOF {
				c.closeConnection()
				continue
			}

			if recvErr != nil {
				c.logger.Errorf("fail to recv msg in client: %v", recvErr)
				c.closeConnection()
				continue
			}

			switch recvMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
				waitCh := c.recvChMap[recvMsg.TxId]
				waitCh <- recvMsg
				c.deleteRecvChan(recvMsg.TxId)
			case protogo.CDMType_CDM_TYPE_GET_STATE:
				waitCh := c.recvChMap[recvMsg.TxId]
				waitCh <- recvMsg
			case protogo.CDMType_CDM_TYPE_GET_BYTECODE:
				waitCh := c.recvChMap[recvMsg.TxId]
				waitCh <- recvMsg
			default:
				c.logger.Errorf("unknown message type")
			}

			if err != nil {
				c.logger.Error(err)
			}

		}
	}

}

func (c *CDMClient) sendCDMMsg(msg *protogo.CDMMessage) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("send message: [%s]", msg.Type)
	return c.stream.Send(msg)
}

// NewClientConn create client connection
func NewClientConn() (*grpc.ClientConn, error) {

	dockerConfig := localconf.ChainMakerConfig.DockerConfig

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(int(dockerConfig.DockerRpcConfig.MaxRecvMessageSize*1024*1024)),
			grpc.MaxCallSendMsgSize(int(dockerConfig.DockerRpcConfig.MaxSendMessageSize*1024*1024)),
		),
	}

	port := ":" + strconv.Itoa(int(dockerConfig.DockerRpcConfig.Port))

	return grpc.Dial(port, dialOpts...)
}

// GetCDMClientStream get rpc stream
func GetCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
}
