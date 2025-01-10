package parallel

import (
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"encoding/json"
	"fmt"
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

// 中断信号 不可逆中断状态，一旦检测到中断信号为true，停止所有的生产行为
var interruptSignal bool

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
	if factor/nodeNum > threadNum/nodeNum*5 {
		initProductFactor(factor / 2)
	} else {
		productFactor = factor
	}
}

func parallel(parallelMethod string) error {
	initParallel()
	timeoutChan := make(chan struct{}, threadNum)
	doneChan := make(chan struct{}, threadNum)
	statistician := getStatistician()
	threads, err := threadFactory(threadNum, parallelMethod, doneChan, timeoutChan, statistician)
	if err != nil {
		return err
	}
	go subNodes(statistician)
	go statistician.collect()
	// 订阅后记录当前时间
	statistician.startTime = time.Now()
	go parallelStart(threads)
	printTicker := time.NewTicker(time.Duration(printTime) * time.Second)
	go printResult(printTicker, statistician)
	listenAndExit(timeoutChan, doneChan, printTicker)
	// last once statistics
	fmt.Println("Statistics for the entire test")
	//statistician.endTime = time.Now()
	statistician.PrintDetails()
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
			go statistician.PrintDetails()
		}
	}
}

// PrintDetails print statistics results
// @param all
func (s *Statistician) PrintDetails() {
	m := make(map[string]interface{})
	s.endTime = time.Now()
	s.elapsedSeconds = float32(time.Now().Sub(s.startTime).Seconds())
	fmt.Printf("当前时间与上一次统计时间的时间间隔为: %.3f 秒\n", s.elapsedSeconds)
	s.run(m, s.usualPrint(), s.chainPrint(), s.rpcPrint())
	jsonChainByte, err := json.Marshal(m)
	if err != nil {
		fmt.Println("e: ", err)
		return
	}
	fmt.Println("result set: ", string(jsonChainByte))
}

type printOpt func(map[string]interface{})

func (s *Statistician) run(m map[string]interface{}, opts ...printOpt) {
	for _, opt := range opts {
		opt(m)
	}
}

func (s *Statistician) usualPrint() printOpt {
	return func(m map[string]interface{}) {
		m["threadNum"] = threadNum
		m["loopNum"] = loopNum
		m["startTime"] = s.startTime.Format("2006-01-02 15:04:05")
		m["endTime"] = s.endTime.Format("2006-01-02 15:04:05")
	}
}

func (s *Statistician) chainPrint() printOpt {
	return func(m map[string]interface{}) {
		chainResult := &ChainResultSet{}
		s.outBlockInfo(chainResult)
		s.outNodeBlockInfo(chainResult)
		m["chainResult"] = *chainResult
	}
}

func (s *Statistician) rpcPrint() printOpt {
	return func(m map[string]interface{}) {
		rpcResult := &RpcResultSet{}
		s.outRpcInfo(rpcResult)
		m["rpcResult"] = *rpcResult
	}
}
