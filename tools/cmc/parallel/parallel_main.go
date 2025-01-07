package parallel

import (
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"fmt"
	"math"
	"os"
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

// print test report
func printResult(printTicker *time.Ticker, statistician *Statistician) {
	for {
		select {
		case <-printTicker.C:
			go statistician.PrintDetails(false)
		}
	}
}
