package parallel

import (
	"chainmaker.org/chainmaker/common/v2/ca"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
	"time"
)

// 构建请求参数
// param：
// @method：方法接收器，接受不同的cmd命令构建不同的方法调用所需要的请求参数
// 该方法用于批量生成请求参数，生成个数由生产因子控制，当index为-1的时候向每个chan生产数据，否则像指定chan生产数据
func producer(method string) {
	builder := stressBuilderFactory(method)
	for {
		if requestId >= int64(threadNum*loopNum) {
			interruptSignal = true
			return
		}
		select {
		case index := <-produceSignal:
			if index == -1 {
				for nodeIndex := 0; nodeIndex < nodeNum; nodeIndex++ {
					produce(builder, nodeIndex)
				}
			} else {
				go func() {
					produce(builder, index)
				}()
			}
		}
	}
}

func stressBuilderFactory(method string) (builder StressBuilder) {
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
	return
}

func produce(builder StressBuilder, index int) {
	for i := 0; i < productFactor; i++ {
		if interruptSignal {
			return
		}
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
}

func parallelStart(threads []*Thread) {
	count := threadNum / 10
	if count == 0 {
		count = 1
	}
	interval := time.Duration(climbTime/count) * time.Second
	for index := 0; index < threadNum; {
		for j := 0; j < 10; j++ {
			go threads[index].consume()
			index++
			if index >= threadNum {
				break
			}
		}
		time.Sleep(interval)
	}
}

// listen channel ,when arrive some condition exit
// 1、exit when arrive user set timeout value (second)
// 2、exit when all goroutine done work
func listenAndExit(timeoutChan, doneChan chan struct{}, printTicker *time.Ticker) {
	fmt.Println("start listenAndExit")
	doneCount := 0
	timeoutTicker := time.NewTicker(time.Duration(timeout) * time.Second)
	once := sync.Once{}
	for {
		if doneCount >= threadNum {
			interruptSignal = true
			break
		}
		select {
		case <-doneChan:
			doneCount++
		case <-timeoutTicker.C:
			once.Do(func() {
				for i := 0; i < threadNum; i++ {
					timeoutChan <- struct{}{}
				}
				fmt.Println("complete timeout")
			})
		}
	}
	printTicker.Stop()
	close(timeoutChan)
	close(doneChan)
	for i := 0; i < nodeNum; i++ {
		close(paramQueues[i])
	}
	close(produceSignal)
	timeoutTicker.Stop()
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

// 初始化线程对象，分配grpc连接，返回thead数组
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

// Start thread start
func (t *Thread) consume() {
	for i := 0; i < t.loopNum; i++ {
		select {
		case <-t.timeoutChan:
			t.doneChan <- struct{}{}
			return
		default:
			select {
			case req, ok := <-paramQueues[t.index]:
				// 如果chan 关闭，被分配到该chan的线程也一起关闭
				if !ok {
					t.doneChan <- struct{}{}
					return
				}
				if len(paramQueues[t.index]) < productFactor && !interruptSignal {
					produceSignal <- t.index
				}
				start := time.Now()
				var err error
				err = sendTx(t.client, orgIDs[t.index], i, req.Param)
				// 结果进入结果集
				atomic.AddUint32(&t.statistician.totalCount, 1)
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
	}
	t.doneChan <- struct{}{}
}

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
