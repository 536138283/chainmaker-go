package parallel

import (
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
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

// 交易请求时延，用来记录成功发起交易的时间 key:txId
var txLatency sync.Map

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

// 节点的sdk客户端
var chainClients []*sdk.ChainClient

func initParallel() error {
	if nodeNum > threadNum {
		threadNum = nodeNum
	}
	if err := initChainClient(); err != nil {
		return err
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
			return err
		}
		signKey, err := asym.PrivateKeyFromPEM(file, nil)
		if err != nil {
			fmt.Printf("analysis node[%s] sign key err: %s\n", hosts[i], err.Error())
			return err
		}
		privateKeys = append(privateKeys, signKey)
	}
	txLatency = sync.Map{}
	return nil
}

func initChainClient() error {
	for i := range hosts {
		nodeConf := sdk.NewNodeConfig(
			// 节点地址，格式：127.0.0.1:12301
			sdk.WithNodeAddr(hosts[i]),
			// 节点连接数
			sdk.WithNodeConnCnt(threadNum),
			// 节点是否启用TLS认证
			sdk.WithNodeUseTLS(useTLS),
			// 根证书路径，支持多个
			sdk.WithNodeCAPaths(caPaths),
			// TLS Hostname
			sdk.WithNodeTLSHostName(hostnamesString),
		)
		opts := make([]sdk.ChainClientOption, 0)
		switch sdk.AuthType(authTypeUint32) {
		case sdk.Public:
			opts = append(opts, sdk.WithAuthType(sdk.AuthTypeToStringMap[sdk.AuthType(authTypeUint32)]))
			opts = append(opts, sdk.WithChainClientChainId(chainId))
			opts = append(opts, sdk.WithUserSignKeyFilePath(signKeyPaths[i]))
			opts = append(opts, sdk.WithCryptoConfig(sdk.NewCryptoConfig(sdk.WithHashAlgo(hashAlgo))))
			opts = append(opts, sdk.WithUserKeyFilePath(userKeyPaths[i]))
			opts = append(opts, sdk.WithUserSignCrtFilePath(caPaths[i]))
			if len(encCrtPaths) > 0 && len(encKeyPaths) > 0 {
				opts = append(opts, sdk.WithUserEncKeyBytes(encKeyBytes[i]))
				opts = append(opts, sdk.WithUserEncCrtBytes(encCrtBytes[i]))
			}
			opts = append(opts, sdk.AddChainClientNodeConfig(nodeConf))
		case sdk.PermissionedWithCert:
			opts = append(opts, sdk.WithAuthType(sdk.AuthTypeToStringMap[sdk.AuthType(authTypeUint32)]))
			opts = append(opts, sdk.WithChainClientOrgId(orgIDs[i]))
			opts = append(opts, sdk.WithChainClientChainId(chainId))
			opts = append(opts, sdk.WithUserKeyFilePath(userKeyPaths[i]))
			opts = append(opts, sdk.WithUserCrtFilePath(userCrtPaths[i]))
			opts = append(opts, sdk.WithUserSignCrtFilePath(signCrtPaths[i]))
			opts = append(opts, sdk.WithUserSignKeyFilePath(signKeyPaths[i]))
			opts = append(opts, sdk.AddChainClientNodeConfig(nodeConf))
			if len(encCrtPaths) > 0 && len(encKeyPaths) > 0 {
				opts = append(opts, sdk.WithUserEncKeyBytes(encKeyBytes[i]))
				opts = append(opts, sdk.WithUserEncCrtBytes(encCrtBytes[i]))
			}
		case sdk.PermissionedWithKey:
			opts = append(opts, sdk.WithAuthType(sdk.AuthTypeToStringMap[sdk.AuthType(authTypeUint32)]))
			opts = append(opts, sdk.WithChainClientChainId(chainId))
			opts = append(opts, sdk.WithUserSignKeyFilePath(signKeyPaths[i]))
			opts = append(opts, sdk.WithCryptoConfig(sdk.NewCryptoConfig(sdk.WithHashAlgo(hashAlgo))))
			opts = append(opts, sdk.WithChainClientOrgId(orgIDs[i]))
			opts = append(opts, sdk.AddChainClientNodeConfig(nodeConf))
		}
		sdkClient, err := sdk.NewChainClient(opts...)
		if err != nil {
			return err
		}
		chainClients = append(chainClients, sdkClient)
	}
	return nil
}

// initProductFactor 对半法加载生产因子
// 在一个chan内每个线程分配一个请求参数和一个预处理请求参数
func initProductFactor(factor int) {
	if factor/nodeNum > threadNum/nodeNum*2 {
		initProductFactor(factor / 2)
	} else {
		productFactor = factor
	}
}

func parallel(method string) error {
	switch method {
	case invokerMethod:
		return parallelInvoke(method)
	default:
		return parallelOthers(method)
	}
}

// 压力测试主方法
func parallelInvoke(method string) error {
	if err := initParallel(); err != nil {
		return err
	}
	timeoutChan := make(chan struct{}, threadNum)
	doneChan := make(chan struct{}, threadNum)
	statistician := getStatistician()
	// 开始生产请求参数
	go producer(invokerMethod)
	produceSignal <- -1
	// 创建线程对象
	threads, err := threadFactory(threadNum, doneChan, timeoutChan, statistician)
	if err != nil {
		return err
	}
	// 开启节点订阅
	go subNodes(statistician)
	// 开启结果收集
	go statistician.collect()
	// 订阅后记录当前时间
	statistician.startTime = time.Now()
	// 启动线程，并发请求
	go parallelStart(threads)
	// 定时打印结果
	printTicker := time.NewTicker(time.Duration(printTime) * time.Second)
	go printResult(printTicker, statistician)
	// 等待超时或请求执行完毕
	listenAndExit(timeoutChan, doneChan)
	finalPrint(statistician, printTicker)
	// 关闭client
	for _, t := range threads {
		t.stop()
	}
	return nil
}

// finalPrint函数用于在区块链高度不再增长时输出统计信息。
//
// 参数:
// statistician (*Statistician): 一个指向Statistician结构体的指针，该结构体包含了需要打印的统计详情。
//
// 功能描述:
// 1. 首先打印提示信息"final print :"。
// 2. 初始化一个变量lastHeight为0，用于存储上一次查询到的区块高度。
// 3. 进入无限循环，不断尝试获取最新的区块高度。
// 4. 调用getBlockHeight函数获取当前区块链高度。如果获取过程中发生错误，则打印错误信息并退出函数。
// 5. 比较当前高度与上一次的高度，如果两者相同，说明区块高度未发生变化，此时调用statistician.PrintDetails()方法输出统计详情，并结束循环。
// 6. 如果区块高度有变化，则更新lastHeight为当前高度，并让程序暂停一秒后继续下一次循环，以避免频繁查询。
func finalPrint(statistician *Statistician, printTicker *time.Ticker) {
	lastHeight := uint64(0)
	for {
		height, err := getBlockHeight()
		if err != nil {
			fmt.Printf("get block height err: %s\n", err.Error())
			return
		}
		if height == lastHeight {
			fmt.Println(height)
			printTicker.Stop()
			fmt.Println("all thread word done finish print")
			statistician.printDetails()
			return
		} else {
			lastHeight = height
			time.Sleep(time.Second * time.Duration(checkInterval))
		}
	}
}

// 函数负责定时输出统计信息
// 参数:
// printTicker (*time.Ticker): 一个时间ticker，每隔一定周期发送一个信号。
// statistician (*Statistician): 一个指向Statistician结构体的指针，封装了统计数据及其打印方法。
func printResult(printTicker *time.Ticker, statistician *Statistician) {
	for {
		select {
		case <-printTicker.C:
			go statistician.printDetails()
		}
	}
}

// PrintDetails方法用于打印Statistician的统计详情。
// 成员变量修改:
// endTime (time.Time): 更新为当前时间，表示统计结束的时间点。
// elapsedSeconds (float32): 计算从开始到结束的持续时间（以秒为单位）。
func (s *Statistician) printDetails() {
	m := make(map[string]interface{})
	s.endTime = time.Now()
	s.elapsedSeconds = float32(time.Now().Sub(s.startTime).Seconds())
	s.run(m, s.usualPrint(), s.chainPrint(), s.rpcPrint())
	jsonChainByte, err := json.Marshal(m)
	if err != nil {
		return
	}
	fmt.Println("result set: ", string(jsonChainByte))
	fmt.Println()
}

// 定义打印选项类型为一个函数，该函数接收一个map[string]interface{}作为参数
type printOpt func(map[string]interface{})

// run方法遍历并执行所有提供的打印选项函数。
// 每个选项函数会修改传入的map，向其中添加或更新键值对，以收集各类统计信息。
func (s *Statistician) run(m map[string]interface{}, opts ...printOpt) {
	for _, opt := range opts {
		opt(m)
	}
}

// usualPrint 返回一个printOpt函数，用于填充常规统计信息到给定的映射中
// 包括线程数、循环次数以及统计的开始和结束时间
func (s *Statistician) usualPrint() printOpt {
	return func(m map[string]interface{}) {
		m["loopNum"] = loopNum
		m["startTime"] = s.startTime.Format("2006-01-02 15:04:05")
		m["endTime"] = s.endTime.Format("2006-01-02 15:04:05")
		m["elapsed"] = s.elapsedSeconds
	}
}

// chainPrint返回一个printOpt函数，它负责在映射中添加区块链相关的统计结果。
// 这包括调用outBlockInfo和outNodeBlockInfo方法获取数据，并存储在一个ChainResultSet结构中。
func (s *Statistician) chainPrint() printOpt {
	return func(m map[string]interface{}) {
		chainResult := &ChainResultSet{}
		s.outBlockInfo(chainResult)
		s.outNodeBlockInfo(chainResult)
		m["chainResult"] = *chainResult
	}
}

// rpcPrint返回一个printOpt函数，用于向映射中添加RPC调用的统计结果。
// 具体操作是调用outRpcInfo方法填充RpcResultSet结构，并将其存入映射中。
func (s *Statistician) rpcPrint() printOpt {
	return func(m map[string]interface{}) {
		rpcResult := &RpcResultSet{Nodes: make(map[string]*RpcInfo)}
		s.outRpcInfo(rpcResult)
		m["rpcResult"] = *rpcResult
	}
}
