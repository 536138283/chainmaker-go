package module

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/logger"
	"context"
	"google.golang.org/grpc"
	"sync"
)

const (
	maxRecvMessageSize = 100 * 1024 * 1024 // 100 MiB
	maxSendMessageSize = 100 * 1024 * 1024 // 100 MiB
	Port               = ":12355"
	ChanSize           = 1000
	StateChanSize      = 1000
)

type CDMClient struct {
	txSendCh    chan *protogo.CDMMessage // channel receive tx from docker-go instance
	stateSendCh chan *protogo.CDMMessage // channel receive state response

	lock      sync.Mutex
	recvChMap map[string]chan *protogo.CDMMessage // store tx_id to chan, retrieve chan to send tx response back to docker-go instance

	stream protogo.CDMRpc_CDMCommunicateClient

	logger *logger.CMLogger

	stop chan bool
}

func NewCDMClient(chainId string) *CDMClient {

	return &CDMClient{
		txSendCh:    make(chan *protogo.CDMMessage, ChanSize),
		stateSendCh: make(chan *protogo.CDMMessage, StateChanSize),
		recvChMap:   make(map[string]chan *protogo.CDMMessage),
		lock:        sync.Mutex{},
		stream:      nil,
		logger:      logger.GetLoggerByChain("[CDM Client]", chainId),
		stop:        make(chan bool),
	}
}

func (c *CDMClient) GetTxSendCh() chan *protogo.CDMMessage {
	return c.txSendCh
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
	// listen two chan:
	// txCh: used to send tx to docker manager
	// stateCh: used to send get state response or bytecode response to docker manager

	var err error

	for {
		select {
		case txMsg := <-c.txSendCh:
			err = c.sendCDMMsg(txMsg)
		case stateMsg := <-c.stateSendCh:
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
			c.logger.Infof("close recv cdm msg")
			return
		default:
			recvMsg, _ := c.stream.Recv()

			switch recvMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
				waitCh := c.recvChMap[recvMsg.TxId]
				waitCh <- recvMsg
				c.deleteRecvChan(recvMsg.TxId)
			case protogo.CDMType_CDM_TYPE_GET_STATE:
				err = c.handleGetState(recvMsg)
			case protogo.CDMType_CDM_TYPE_GET_BYTECODE:
				err = c.handleGetByteCode(recvMsg)
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
	c.logger.Debugf("send message: [%s]", msg.Type)
	return c.stream.Send(msg)
}

func (c *CDMClient) handleGetState(recvMsg *protogo.CDMMessage) error {

	return nil
}

func (c *CDMClient) handleGetByteCode(recvMsg *protogo.CDMMessage) error {

	return nil
}

// NewClientConn create client connection
func NewClientConn() (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMessageSize),
			grpc.MaxCallSendMsgSize(maxSendMessageSize),
		),
	}

	return grpc.Dial(Port, dialOpts...)
}

// GetCDMClientStream get rpc stream
func GetCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
}
