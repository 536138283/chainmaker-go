package parallel

import (
	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	//respStr     = "proposalRequest error, resp: %+v"
	templateStr = "%s_%d_%d_%d"
	resultStr   = "exec result, orgid: %s, loop_id: %d, method1: %s, txid: %s, resp: %+v"
)

const (
	invokerMethod      = "invoke"
	queryMethod        = "query"
	createContractStr  = "createContract"
	upgradeContractStr = "upgradeContract"
)

var totalSentTxs int64
var totalRandomSentTxs int64

type RequestParam struct {
	Param     *commonPb.TxRequest
	RequestId int64
}

// 请求参数队列，存放请求参数，队列的数量为节点的数量
// 队列的index为队列的下标
var paramQueues []chan RequestParam

var privateKeys []crypto.PrivateKey

// 每一个请求参数的唯一id
var requestId int64

// 生产信号，当该chan接收到数据时，开始生产请求参数
var produceSignal chan int

// 生产因子，用来控制生产消息的数量
var productFactor int

func parallel(parallelMethod string) error {
	initParallel()
	// 开始生产请求参数
	go producer(invokerMethod)
	// produce request param
	produceSignal <- -1
	// 定义退出channel
	timeoutChan := make(chan struct{}, threadNum)
	doneChan := make(chan struct{}, threadNum)
	statistician := getStatistician()
	statistician.startTime = time.Now()
	statistician.preTime = time.Now()
	for i := 0; i < nodeNum; i++ {
		statistician.nodeMinSuccessElapsed[i] = math.MaxInt16
	}
	go statistician.Collect()
	go subNodes(statistician)
	threads, err := threadFactory(threadNum, parallelMethod, doneChan, timeoutChan, statistician)
	if err != nil {
		return err
	}
	statistician.startTime = time.Now()
	statistician.lastStartTime = time.Now()
	fmt.Println(time.Now())
	go parallelStart(threads)
	printTicker := time.NewTicker(time.Duration(printTime) * time.Second)
	go printResult(printTicker, statistician)

	listenAndExit(timeoutChan, doneChan, printTicker)
	// last once statistics
	fmt.Println("Statistics for the entire test")
	statistician.endTime = time.Now()
	statistician.PrintDetails(true)
	// close client conn
	for _, t := range threads {
		t.Stop()
	}
	return nil
}

func getStatistician() *Statistician {
	return &Statistician{
		reqStatC:              make(chan *reqStat, threadNum),
		nodeMinSuccessElapsed: make([]int64, nodeNum),
		nodeMaxSuccessElapsed: make([]int64, nodeNum),
		nodeSumSuccessElapsed: make([]int64, nodeNum),
		nodeSuccessReqCount:   make([]int, nodeNum),
		nodeTotalReqCount:     make([]int, nodeNum),
		nodeBlockNum:          make([]int64, nodeNum),

		cReqStatC:            make(chan *cReqStat, threadNum),
		nodeTxTotal:          make([]int64, nodeNum),
		nodeMaxTxBlockHeight: make([]uint64, nodeNum),
		nodeMaxTxBlockCount:  make([]uint32, nodeNum),
		nodeMinTxBlockHeight: make([]uint64, nodeNum),
		nodeMinTxBlockCount:  make([]uint32, nodeNum),
		nodeFirstBlockHeight: make([]uint64, nodeNum),
		nodeLastBlockHeight:  make([]uint64, nodeNum),
		nodeFirstBlockTime:   make([]int64, nodeNum),
		nodeLastBlockTime:    make([]int64, nodeNum),
	}
}

// print test report
func printResult(printTicker *time.Ticker, statistician *Statistician) {
	for {
		select {
		case <-printTicker.C:
			go statistician.PrintDetails(false)
		}
	}
}

func threadFactory(number int, method string, doneChan,
	timeoutChan chan struct{}, statistician *Statistician) ([]*Thread, error) {
	threads := make([]*Thread, number)
	var err error
	for i := 0; i < number; i++ {
		t := &Thread{id: i, loopNum: loopNum, doneChan: doneChan, timeoutChan: timeoutChan, statistician: statistician}
		t.index = t.id % len(hosts)
		t.conn, err = t.initGRPCConnect(useTLS, t.index)
		if err != nil {
			return nil, err
		}
		t.client = apiPb.NewRpcNodeClient(t.conn)
		switch method {
		case invokerMethod:
			t.operationName = invokerMethod
			//t.handler = &invokeHandler{threadId: i}
		case queryMethod:
			t.operationName = queryMethod
			//t.handler = &queryHandler{threadId: i}
		case createContractStr:
			t.operationName = createContractStr
			//t.handler = &createContractHandler{threadId: i}
		case upgradeContractStr:
			t.operationName = upgradeContractStr
			//t.handler = &upgradeContractHandler{threadId: i}
		}
		threads[i] = t
	}
	return threads, nil
}

// listen channel ,when arrive some condition exit
// 1、exit when arrive user set timeout value (second)
// 2、exit when all goroutine done work
func listenAndExit(timeoutChan, doneChan chan struct{}, printTicker *time.Ticker) {
	doneCount := 0
	timeoutTicker := time.NewTicker(time.Duration(timeout) * time.Second)
	timeoutOnce := sync.Once{}
	for {
		if doneCount >= threadNum {
			break
		}
		select {
		case <-doneChan:
			doneCount++
		case <-timeoutTicker.C:
			go func() {
				timeoutOnce.Do(func() {
					for i := 0; i < threadNum; i++ {
						timeoutChan <- struct{}{}
					}
				})
			}()
		}
	}
	close(timeoutChan)
	close(doneChan)
	printTicker.Stop()
	timeoutTicker.Stop()
}

func initParallel() {
	if nodeNum > threadNum {
		threadNum = nodeNum
	}
	initProductFactor(threadNum * loopNum)
	produceSignal = make(chan int, 1)
	paramQueues = make([]chan RequestParam, nodeNum)
	for i := 0; i < nodeNum; i++ {
		paramQueues[i] = make(chan RequestParam, productFactor)
		// 解析签名私钥
		file, err := os.ReadFile(signKeyPaths[i])
		if err != nil {
			fmt.Printf("read node[%s] sign key err: %s\n", hosts[i], err.Error())
			return
		}
		signKey, err := asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			fmt.Printf("analysis node[%s] sign key err: %s\n", hosts[i], err.Error())
			return
		}
		privateKeys = append(privateKeys, signKey)
	}
}

// 对半法加载生产因子
// 在一个chan内每个线程分配一个请求参数和一个预处理请求参数
func initProductFactor(factor int) {
	if factor/nodeNum > threadNum/nodeNum*2 {
		initProductFactor(factor / 2)
	} else {
		productFactor = factor
	}
}

// 构建请求参数
// param：
// @method：方法接收器，接受不同的cmd命令构建不同的方法调用所需要的请求参数
// 该方法用于批量生成请求参数，生成个数由生产因子控制，当index为-1的时候像每个chan成产数据，否则像指定chan生产数据
func producer(method string) {
	var builder StressBuilder
	switch method {
	case invokerMethod:
		builder = Invoke{}
	case queryMethod:
		builder = Query{}
	case createContractStr:
		builder = Create{}
	case upgradeContractStr:
		builder = Upgrade{}
	}
	for {
		if requestId >= int64(threadNum*loopNum) {
			close(produceSignal)
			return
		}
		select {
		case index := <-produceSignal:
			if index == -1 {
				for i := 0; i < productFactor; i++ {
					for nodeIndex := 0; nodeIndex < nodeNum; nodeIndex++ {
						atomic.AddInt64(&requestId, 1)
						param, err := builder.Build(requestId, nodeIndex)
						if err != nil {
							fmt.Printf("producer err: %s\n", err.Error())
						}
						paramQueues[nodeIndex] <- RequestParam{
							Param:     param,
							RequestId: requestId,
						}
					}

				}
			} else {
				go func() {
					for i := 0; i < productFactor; i++ {
						atomic.AddInt64(&requestId, 1)
						param, err := builder.Build(requestId, index)
						if err != nil {
							fmt.Printf("producer err: %s\n", err.Error())
						}
						paramQueues[index] <- RequestParam{
							Param:     param,
							RequestId: requestId,
						}
					}
				}()
			}

		}
	}

}

type Thread struct {
	id          int
	loopNum     int
	doneChan    chan struct{}
	timeoutChan chan struct{}
	//handler       Handler
	statistician  *Statistician
	operationName string

	conn   *grpc.ClientConn
	client apiPb.RpcNodeClient
	index  int
}

// Start thread start
func (t *Thread) Start() {
	for i := 0; i < t.loopNum; i++ {
		select {
		case <-t.timeoutChan:
			t.doneChan <- struct{}{}
			return
		case req, ok := <-paramQueues[t.index]:
			// 如果chan 关闭，被分配到该chan的线程也一起关闭
			if !ok {
				return
			}
			if len(paramQueues[t.index]) < productFactor/2 {
				produceSignal <- t.index
			}
			start := time.Now()
			var err error
			err = sendRequest(t.client, orgIDs[t.index], i, req.Param)
			// 结果进入结果集
			atomic.AddInt32(&t.statistician.totalCount, 1)
			t.statistician.reqStatC <- &reqStat{
				success: err == nil,
				elapsed: time.Since(start).Milliseconds(),
				nodeId:  t.index,
			}
			if recordLog && err != nil {
				log.Errorf("threadId: %d, loopId: %d, nodeId: %d, err: %s", t.id, i, t.index, err)
			}
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
	}
	t.doneChan <- struct{}{}
}

// Thread for multi-thread object
// Stop thread stop
func (t *Thread) Stop() {
	err := t.conn.Close()
	if err != nil {
		return
	}
}

func (t *Thread) initGRPCConnect(useTLS bool, index int) (*grpc.ClientConn, error) {
	url := hosts[index]

	if useTLS {
		var serverName string
		if hostnamesString == "" {
			serverName = "chainmaker.org"
		} else {
			if len(hosts) != len(hostnames) {
				return nil, errors.New("required len(hosts) == len(hostnames)")
			}
			serverName = hostnames[index]
		}
		tlsClient := ca.CAClient{
			ServerName: serverName,
			CaPaths:    []string{caPaths[index]},
			CertFile:   userCrtPaths[index],
			KeyFile:    userKeyPaths[index],
		}

		c, err := tlsClient.GetCredentialsByCA()
		if err != nil {
			return nil, err
		}
		return grpc.Dial(url, grpc.WithTransportCredentials(*c))
	} else {
		return grpc.Dial(url, grpc.WithInsecure())
	}
}

func parallelStart(threads []*Thread) {
	count := threadNum / 10
	if count == 0 {
		count = 1
	}
	interval := time.Duration(climbTime/count) * time.Second
	for index := 0; index < threadNum; {
		for j := 0; j < 10; j++ {
			go threads[index].Start()
			index++
			if index >= threadNum {
				break
			}
		}
		time.Sleep(interval)
	}
}
