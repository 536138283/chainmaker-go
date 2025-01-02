/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package parallel 并发处理，主要用于压测的场景
package parallel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/logger/v2"
	apiPb "chainmaker.org/chainmaker/pb-go/v2/api"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var log = logger.GetLogger(logger.MODULE_CLI)

var (
	threadNum      int
	loopNum        int
	timeout        int
	printTime      int
	sleepTime      int
	climbTime      int
	requestTimeout int
	runTime        int32
	checkResult    bool
	recordLog      bool
	showKey        bool
	useTLS         bool
	useShortCrt    bool

	hostsString        string
	hostnamesString    string
	statisticalType    string
	signCrtPathsString string
	signKeyPathsString string
	userCrtPathsString string
	userKeyPathsString string
	orgIDsString       string
	hashAlgo           string
	caPathsString      string
	pairsFile          string
	pairsString        string
	globalPairs        []*KeyValuePair
	abiPath            string
	method             string
	orgIds             string // 组织列表(多个用逗号隔开)
	adminSignKeys      string // 管理员私钥列表(多个用逗号隔开)
	adminSignCrts      string // 管理员证书列表(多个用逗号隔开)
	chainId            string
	contractName       string
	version            string
	wasmPath           string

	caPaths      []string
	hosts        []string
	hostnames    []string
	signCrtPaths []string
	signKeyPaths []string
	userCrtPaths []string
	userKeyPaths []string
	orgIDs       []string

	nodeNum int

	fileCache = NewFileCacheReader()
	certCache = NewCertFileCacheReader()

	abiCache     = NewFileCacheReader()
	outputResult bool

	authTypeUint32 uint32
	authType       sdk.AuthType
	gasLimit       uint64
)

type KeyValuePair struct {
	Key        string `json:"key,omitempty"`
	Value      string `json:"value,omitempty"`
	Unique     bool   `json:"unique,omitempty"`
	RandomRate int64  `json:"randomRate,omitempty"`
	Increase   bool   `json:"increase"`
	Decrease   bool   `json:"decrease"`
	// mu protect IntValue in Increase/Decrease scene.
	mu       sync.Mutex
	IntValue int64 `json:"-"`
}

type Detail struct {
	TPS          float32                `json:"tps"`
	SuccessCount int                    `json:"successCount"`
	FailCount    int                    `json:"failCount"`
	Count        int                    `json:"count"`
	MinTime      int64                  `json:"minTime"`
	MaxTime      int64                  `json:"maxTime"`
	AvgTime      float32                `json:"avgTime"`
	StartTime    string                 `json:"startTime"`
	EndTime      string                 `json:"endTime"`
	Elapsed      float32                `json:"elapsed"`
	ThreadNum    int                    `json:"threadNum"`
	LoopNum      int                    `json:"loopNum"`
	Nodes        map[string]interface{} `json:"nodes"`
}

type reqStat struct {
	success bool
	elapsed int64
	nodeId  int
}

type cReqStat struct {
	blockHeader *commonPb.BlockHeader
	nodeId      int
	elapsed     int64
}

type Statistician struct {
	// rpc standards
	reqStatC          chan *reqStat
	minSuccessElapsed int64
	maxSuccessElapsed int64
	sumSuccessElapsed int64
	totalCount        int32
	successCount      int

	lastIndex     int
	lastStartTime time.Time

	startTime time.Time
	endTime   time.Time

	// Classify by node id
	cReqStatC             chan *cReqStat
	nodeMinSuccessElapsed []int64
	nodeMaxSuccessElapsed []int64
	nodeSumSuccessElapsed []int64
	nodeSuccessReqCount   []int
	nodeTotalReqCount     []int

	// block chain standards
	BlockNum             int64
	TxTotal              int64
	NodeBlockNum         []int64
	NodeTxTotal          []int64
	NodeMaxTxBlockHeight []uint64
	NodeMaxTxBlockCount  []uint32
	NodeMinTxBlockHeight []uint64
	NodeMinTxBlockCount  []uint32
	MaxTxBlockHeight     []int64
	MinTxBlockCount      []int64
	firstBlockTime       int64
	firstBlockHeight     uint64
	lastBlockTime        time.Time
	lastBlockHeight      uint64
}

// ParallelCMD parallel sub command
// @return *cobra.Command
func ParallelCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parallel",
		Short: "Parallel",
		Long:  "Parallel",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			authType = sdk.AuthType(authTypeUint32)
			caPaths = strings.Split(caPathsString, ",")
			hosts = strings.Split(hostsString, ",")
			hostnames = strings.Split(hostnamesString, ",")
			signCrtPaths = strings.Split(signCrtPathsString, ",")
			signKeyPaths = strings.Split(signKeyPathsString, ",")
			userCrtPaths = strings.Split(userCrtPathsString, ",")
			userKeyPaths = strings.Split(userKeyPathsString, ",")
			orgIDs = strings.Split(orgIDsString, ",")

			if authType == sdk.Public {
				if len(hosts) != len(signKeyPaths) {
					panic(fmt.Sprintf("hosts[%d], sign-keys[%d] length invalid", len(hosts), len(signKeyPaths)))
				}
			} else if authType == sdk.PermissionedWithKey {
				if len(hosts) != len(signKeyPaths) || len(hosts) != len(orgIDs) {
					panic(fmt.Sprintf("hosts[%d], sign-keys[%d], orgIDs[%d] length invalid",
						len(hosts), len(signKeyPaths), len(orgIDs)))
				}
			} else {
				if len(hosts) != len(signCrtPaths) || len(hosts) != len(signKeyPaths) || len(hosts) != len(caPaths) || len(hosts) != len(orgIDs) {
					panic(fmt.Sprintf("hosts[%d], sign-crts[%d], sign-keys[%d], ca-path[%d], orgIDs[%d] length invalid",
						len(hosts), len(signCrtPaths), len(signKeyPaths), len(caPaths), len(orgIDs)))
				}
			}

			if useTLS && (len(hosts) != len(userCrtPaths) || len(hosts) != len(userKeyPaths)) {
				panic(fmt.Sprintf("use tls, but hosts[%d], user-crts[%d], user-keys[%d] length invalid",
					len(hosts), len(userCrtPaths), len(userKeyPaths)))
			}

			nodeNum = len(hosts)
			if len(pairsFile) != 0 {
				bytes, err := ioutil.ReadFile(pairsFile)
				if err != nil {
					panic(err)
				}
				pairsString = string(bytes)
			}
			var err error
			globalPairs, err = getPairInfos()
			if err != nil {
				panic(err)
			}
			fmt.Println("tx content: ", pairsString)
		},
	}

	flags := cmd.PersistentFlags()
	flags.IntVarP(&threadNum, "threadNum", "N", 10, "specify thread number")
	flags.IntVarP(&loopNum, "loopNum", "l", 1000, "specify loop number")
	flags.IntVarP(&timeout, "timeout", "T", 2, "specify timeout(unit: s)")
	flags.IntVarP(&printTime, "printTime", "r", 1, "specify print time(unit: s)")
	flags.IntVarP(&sleepTime, "sleepTime", "S", 100, "specify sleep time(unit: ms)")
	flags.IntVarP(&climbTime, "climbTime", "L", 10, "specify climb time(unit: s)")
	flags.StringVarP(&hostsString, "hosts", "H", "localhost:17988,localhost:27988", "specify hosts")
	flags.StringVarP(&signCrtPathsString, "sign-crts", "K", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.sign.crt", "specify user crt path")
	flags.StringVarP(&signKeyPathsString, "sign-keys", "u", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.sign.key", "specify user key path")
	flags.StringVar(&userCrtPathsString, "user-crts", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.crt", "specify tls crt path")
	flags.StringVar(&userKeyPathsString, "user-keys", "../../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.tls.key,../../config/crypto-config/wx-org2.chainmaker.org/user/client1/client1.tls.key", "specify tls key path")
	flags.StringVarP(&orgIDsString, "org-IDs", "I", "wx-org1,wx-org2", "specify user key path")
	flags.BoolVarP(&checkResult, "check-result", "Y", false, "specify whether check result")
	flags.BoolVarP(&recordLog, "record-log", "g", false, "specify whether record log")
	flags.BoolVarP(&outputResult, "output-result", "", false, "output rpc result, eg: txid")
	flags.BoolVarP(&showKey, "showKey", "", false, "bool")
	flags.StringVar(&hashAlgo, "hash-algorithm", "SHA256", "hash algorithm set in chain configuration")
	flags.StringVarP(&caPathsString, "ca-path", "P", "../../config/crypto-config/wx-org1.chainmaker.org/ca,../../config/crypto-config/wx-org2.chainmaker.org/ca", "specify ca path")
	flags.BoolVarP(&useTLS, "use-tls", "t", false, "specify whether use tls")
	flags.StringVar(&orgIds, "org-ids", "wx-org1,wx-org2,wx-org3,wx-org4", "orgIds of admin")
	flags.StringVar(&adminSignKeys, "admin-sign-keys", "../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.key", "adminSignKeys of admin")
	flags.StringVar(&adminSignCrts, "admin-sign-crts", "../../config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt,../../config/crypto-config/wx-org4.chainmaker.org/user/admin1/admin1.sign.crt", "adminSignCrts of admin")
	flags.StringVarP(&chainId, "chain-id", "C", "chain1", "specify chain id")
	flags.StringVarP(&contractName, "contract-name", "n", "contract1", "specify contract name")
	flags.BoolVar(&useShortCrt, "use-short-crt", false, "use compressed certificate in transactions")
	flags.IntVar(&requestTimeout, "requestTimeout", 5, "specify request timeout(unit: s)")
	flags.Uint32Var(&authTypeUint32, "auth-type", 1, "chainmaker auth type. PermissionedWithCert:1,PermissionedWithKey:2,Public:3")
	flags.Uint64Var(&gasLimit, "gas-limit", 0, "gas limit in uint64 type")
	flags.StringVarP(&hostnamesString, "tls-host-names", "", "", "specify hostname, the sequence is the same as --hosts")
	flags.StringVarP(&statisticalType, "statistical-type", "", "default", "normal statistical type or block based statistical type, input normal or block default:normal ")
	cmd.AddCommand(invokeCMD())
	cmd.AddCommand(queryCMD())
	cmd.AddCommand(createContractCMD())
	cmd.AddCommand(upgradeContractCMD())
	return cmd
}

const (
	invokerMethod      = "invoke"
	queryMethod        = "query"
	createContractStr  = "createContract"
	upgradeContractStr = "upgradeContract"
)

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
	for i := 0; i < nodeNum; i++ {
		statistician.nodeMinSuccessElapsed[i] = math.MaxInt16
	}
	go statistician.Start()
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

	listenAndExitParallel(timeoutChan, doneChan, printTicker)
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
		NodeBlockNum:          make([]int64, nodeNum),

		cReqStatC:            make(chan *cReqStat, threadNum),
		NodeTxTotal:          make([]int64, nodeNum),
		NodeMaxTxBlockHeight: make([]uint64, nodeNum),
		NodeMaxTxBlockCount:  make([]uint32, nodeNum),
		NodeMinTxBlockHeight: make([]uint64, nodeNum),
		NodeMinTxBlockCount:  make([]uint32, nodeNum),
		MaxTxBlockHeight:     make([]int64, nodeNum),
		MinTxBlockCount:      make([]int64, nodeNum),
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
func listenAndExitParallel(timeoutChan, doneChan chan struct{}, printTicker *time.Ticker) {
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

func (s *Statistician) Start() {
	for {
		select {
		case stat := <-s.reqStatC:
			if stat.success {
				if stat.elapsed < s.minSuccessElapsed {
					s.minSuccessElapsed = stat.elapsed
				}
				if stat.elapsed > s.maxSuccessElapsed {
					s.maxSuccessElapsed = stat.elapsed
				}

				if stat.elapsed < s.nodeMinSuccessElapsed[stat.nodeId] {
					s.nodeMinSuccessElapsed[stat.nodeId] = stat.elapsed
				}
				if stat.elapsed > s.nodeMaxSuccessElapsed[stat.nodeId] {
					s.nodeMaxSuccessElapsed[stat.nodeId] = stat.elapsed
				}

				s.successCount++
				s.sumSuccessElapsed += stat.elapsed

				s.nodeSuccessReqCount[stat.nodeId]++
				s.nodeSumSuccessElapsed[stat.nodeId] += stat.elapsed
			}
			s.nodeTotalReqCount[stat.nodeId]++
		case stat := <-s.cReqStatC:
			// 统计节点最大区块高度
			if s.NodeMaxTxBlockHeight[stat.nodeId] < stat.blockHeader.BlockHeight {
				s.NodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
			}
			// 最大交易数量
			if s.NodeMaxTxBlockCount[stat.nodeId] < stat.blockHeader.TxCount {
				s.NodeMaxTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
			}
			// 最小区块高度
			if s.NodeMinTxBlockHeight[stat.nodeId] > stat.blockHeader.BlockHeight {
				s.NodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
			}
			// 最小交易数量
			if s.NodeMinTxBlockCount[stat.nodeId] > stat.blockHeader.TxCount {
				s.NodeMinTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
			}
			if s.firstBlockHeight == 0 {
				s.firstBlockHeight = stat.blockHeader.BlockHeight
			}
			if s.firstBlockTime != 0 {
				s.firstBlockTime = stat.blockHeader.BlockTimestamp
			}
			s.lastBlockHeight = stat.blockHeader.BlockHeight
		}
	}
}

// PrintDetails print statistics results
// @param all
func (s *Statistician) PrintDetails(all bool) {
	nowCount := atomic.LoadInt32(&s.totalCount)
	nowTime := time.Now()

	detail := s.statisticsResults(&numberResults{count: int(s.totalCount), successCount: s.successCount,
		max: s.maxSuccessElapsed, min: s.minSuccessElapsed, sum: s.sumSuccessElapsed,
		nodeSuccessCount: s.nodeSuccessReqCount,
		nodeCount:        s.nodeTotalReqCount, nodeMin: s.nodeMinSuccessElapsed,
		nodeMax: s.nodeMaxSuccessElapsed, nodeSum: s.nodeSumSuccessElapsed}, all, nowTime)
	s.lastIndex = int(nowCount)
	s.lastStartTime = time.Now()

	bytes, err := json.Marshal(detail)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(bytes))
	fmt.Println()
}

type numberResults struct {
	count, successCount         int
	min, max, sum               int64
	nodeSuccessCount, nodeCount []int
	nodeMin, nodeMax, nodeSum   []int64
}

func (s *Statistician) statisticsResults(ret *numberResults, all bool, nowTime time.Time) (detail *Detail) {
	detail = &Detail{
		Nodes: make(map[string]interface{}),
	}
	if ret.count > 0 {
		detail = &Detail{
			SuccessCount: ret.successCount,
			FailCount:    ret.count - ret.successCount,
			Count:        ret.count,
			MinTime:      ret.min,
			MaxTime:      ret.max,
			AvgTime:      float32(ret.sum) / float32(ret.count),
			ThreadNum:    threadNum,
			LoopNum:      loopNum,
			Nodes:        make(map[string]interface{}),
		}
		for i := 0; i < nodeNum; i++ {
			detail.Nodes[fmt.Sprintf("node%d_successCount", i)] = ret.nodeSuccessCount[i]
			detail.Nodes[fmt.Sprintf("node%d_failCount", i)] = ret.nodeCount[i] - ret.nodeSuccessCount[i]
			detail.Nodes[fmt.Sprintf("node%d_count", i)] = ret.nodeCount[i]
			detail.Nodes[fmt.Sprintf("node%d_minTime", i)] = ret.nodeMin[i]
			detail.Nodes[fmt.Sprintf("node%d_maxTime", i)] = ret.nodeMax[i]
			detail.Nodes[fmt.Sprintf("node%d_avgTime", i)] = float32(ret.nodeSum[i]) / float32(ret.nodeCount[i])
		}
	}
	if all {
		detail.StartTime = s.startTime.Format("2006-01-02 15:04:05.000")
		detail.EndTime = s.endTime.Format("2006-01-02 15:04:05.000")
		detail.Elapsed = float32(s.endTime.Sub(s.startTime).Milliseconds()) / 1000
		detail.TPS = float32(ret.successCount) / float32(s.endTime.Sub(s.startTime).Seconds())
		for i := 0; i < nodeNum; i++ {
			detail.Nodes[fmt.Sprintf("node%d_tps", i)] = float32(ret.nodeSuccessCount[i]) / float32(s.endTime.Sub(s.startTime).Seconds())
		}
	} else {
		detail.StartTime = s.lastStartTime.Format("2006-01-02 15:04:05.000")
		detail.EndTime = nowTime.Format("2006-01-02 15:04:05.000")
		detail.Elapsed = float32(nowTime.Sub(s.lastStartTime).Milliseconds()) / 1000
		detail.TPS = float32(ret.successCount) / float32(nowTime.Sub(s.startTime).Seconds())
		for i := 0; i < nodeNum; i++ {
			detail.Nodes[fmt.Sprintf("node%d_tps", i)] = float32(ret.nodeSuccessCount[i]) / float32(nowTime.Sub(s.startTime).Seconds())
		}
	}
	return detail
}

// Thread for multi-thread object
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

func getPairInfos() ([]*KeyValuePair, error) {
	var ps []*KeyValuePair
	err := json.Unmarshal([]byte(pairsString), &ps)
	if err != nil {
		log.Errorf("unmarshal pair content failed, origin content: %s, err: %s", pairsString, err)
		return nil, err
	}

	for _, p := range ps {
		if p.Decrease || p.Increase {
			p.IntValue, err = strconv.ParseInt(p.Value, 10, 64)
			if err != nil {
				return nil, err
			}
		}
	}

	return ps, nil
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

// Handler do multi-thread operation action
//type Handler interface {
//	handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string, userCrtPath string, loopId int) error
//}

// invokeHandler contract invoke handler
//type invokeHandler struct {
//	threadId int
//}

var (
	respStr     = "proposalRequest error, resp: %+v"
	templateStr = "%s_%d_%d_%d"
	resultStr   = "exec result, orgid: %s, loop_id: %d, method1: %s, txid: %s, resp: %+v"
)

var totalSentTxs int64
var totalRandomSentTxs int64
var resp *commonPb.TxResponse

/*
type InvokerMsg struct {
	txType       commonPb.TxType
	chainId      string
	txId         string
	method       string
	contractName string
	oldSeq       uint64
	pairs        []*commonPb.KeyValuePair
}

func (h *invokeHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string,
	userCrtPath string, loopId int) error {
	txId := utils.GetTimestampTxId()

	// 构造Payload
	pairs := makeKvs(11111)
	if showKey {
		j, err := json.Marshal(pairs)
		if err != nil {
			fmt.Println(err)
		}
		rate := totalRandomSentTxs * 100 / totalSentTxs
		fmt.Printf("totalSentTxs:%d\t totalRandomSentTxs:%d\t randomRate:%d \t param:%s\t \n",
			totalSentTxs, totalRandomSentTxs, rate, string(j))
	}

	// 支持evm
	//var err error
	var abiData *[]byte
	if abiPath != "" {
		abiData = abiCache.Read(abiPath)
		runTime = 5 //evm
	}

	method1, pairs1, err := makePairs(method, abiPath, pairs, commonPb.RuntimeType(runTime), abiData)

	//fmt.Println("[exec_handle]orgId: ", orgId, ", userCrtPath: ", userCrtPath, ", loopId: ", loopId, ", method1: ", method1, ", pairs1: ", pairs1, ", method: ", method, ", pairs: ", pairs)
	payloadBytes, err := constructInvokePayload(chainId, contractName, method1, pairs1, gasLimit)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payloadBytes, nil)
	if err != nil {
		return err
	}

	if outputResult {
		msg := fmt.Sprintf(resultStr, orgId, loopId, method1, txId, resp)
		fmt.Println(msg)
	}

	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

// queryHandler contract query handler
type queryHandler struct {
	threadId int
}

func (h *queryHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string,
	userCrtPath string, loopId int) error {
	txId := utils.GetTimestampTxId()

	// 构造Payload
	pairs := makeKvs(1111)
	if showKey {
		j, err := json.Marshal(pairs)
		if err != nil {
			fmt.Println(err)
		}
		rate := totalRandomSentTxs * 100 / totalSentTxs
		fmt.Printf("totalSentTxs:%d\t totalRandomSentTxs:%d\t randomRate:%d \t param:%s\t \n",
			totalSentTxs, totalRandomSentTxs, rate, string(j))
	}

	payloadBytes, err := constructQueryPayload(chainId, contractName, method, pairs, gasLimit)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_QUERY_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payloadBytes, nil)
	if err != nil {
		return err
	}

	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

// createContractHandler create contract handler
type createContractHandler struct {
	threadId int
}

func (h *createContractHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string,
	userCrtPath string, loopId int) error {
	txId := utils.GetTimestampTxId()

	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return err
	}
	var pairs []*commonPb.KeyValuePair
	payload, _ := utils.GenerateInstallContractPayload(fmt.Sprintf(templateStr, contractName, h.threadId, loopId, time.Now().Unix()),
		"1.0.0", commonPb.RuntimeType(runTime), wasmBin, pairs)
	// gas limit
	if gasLimit > 0 {
		var limit = &commonPb.Limit{GasLimit: gasLimit}
		payload.Limit = limit
	}

	//
	//method := syscontract.ContractManageFunction_INIT_CONTRACT.String()
	//
	//payload := &commonPb.Payload{
	//	ChainId: chainId,
	//	Contract: &commonPb.Contract{
	//		ContractName:    fmt.Sprintf(templateStr, contractName, h.threadId, loopId, time.Now().Unix()),
	//		ContractVersion: "1.0.0",
	//		RuntimeType:     commonPb.RuntimeType(runTime),
	//	},
	//	Method:      method,
	//	Parameters:  pairs,
	//	ByteCode:    wasmBin,
	//	Endorsement: nil,
	//}

	endorsement, err := acSign(payload)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payload, endorsement)
	if err != nil {
		return err
	}
	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

// upgradeContractHandler upgrade contract handler
type upgradeContractHandler struct {
	threadId int
}

func (h *upgradeContractHandler) handle(client apiPb.RpcNodeClient, sk3 crypto.PrivateKey, orgId string,
	userCrtPath string, loopId int) error {
	txId := utils.GetTimestampTxId()

	wasmBin, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return err
	}

	var pairs []*commonPb.KeyValuePair
	payload, _ := GenerateUpgradeContractPayload(fmt.Sprintf(templateStr, contractName, h.threadId, loopId, time.Now().Unix()),
		version, commonPb.RuntimeType(runTime), wasmBin, pairs)
	// gas limit
	if gasLimit > 0 {
		var limit = &commonPb.Limit{GasLimit: gasLimit}
		payload.Limit = limit
	}
	payload.TxId = txId
	payload.ChainId = chainId
	payload.Timestamp = time.Now().Unix()
	endorsement, err := acSign(payload)
	if err != nil {
		return err
	}

	resp, err = sendRequest(sk3, client, &InvokerMsg{txType: commonPb.TxType_INVOKE_CONTRACT,
		txId: txId, chainId: chainId}, orgId, userCrtPath, payload, endorsement)
	if err != nil {
		return err
	}

	if checkResult && resp.Code != commonPb.TxStatusCode_SUCCESS {
		return fmt.Errorf(respStr, resp)
	}

	return nil
}

func GenerateUpgradeContractPayload(contractName, version string, runtimeType commonPb.RuntimeType, bytecode []byte,
	initParameters []*commonPb.KeyValuePair) (*commonPb.Payload, error) {
	payload, err := utils.GenerateInstallContractPayload(contractName, version, runtimeType, bytecode, initParameters)
	if err != nil {
		return nil, err
	}
	payload.Method = syscontract.ContractManageFunction_UPGRADE_CONTRACT.String()
	return payload, nil
}


*/

/*
func sendRequest(sk3 crypto.PrivateKey, client apiPb.RpcNodeClient, msg *InvokerMsg, orgId, userCrtPath string,

	payload *commonPb.Payload, endorsers []*commonPb.EndorsementEntry) (*commonPb.TxResponse, error) {

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(requestTimeout)*time.Second))
	defer cancel()

	// 构造Sender
	var sender *acPb.Member
	if authType == sdk.Public {
		pubKey := sk3.PublicKey()
		memberInfo, err := pubKey.String()
		if err != nil {
			return nil, err
		}
		sender = &acPb.Member{
			OrgId:      "",
			MemberInfo: []byte(memberInfo),
			MemberType: acPb.MemberType_PUBLIC_KEY,
		}
	} else if authType == sdk.PermissionedWithKey {
		pubKey := sk3.PublicKey()
		memberInfo, err := pubKey.String()
		if err != nil {
			return nil, err
		}
		sender = &acPb.Member{
			OrgId:      orgId,
			MemberInfo: []byte(memberInfo),
			MemberType: acPb.MemberType_PUBLIC_KEY,
		}
	} else {
		file := fileCache.Read(userCrtPath)
		if useShortCrt {
			certId, err := certCache.Read(userCrtPath, *file, hashAlgo)
			if err != nil {
				return nil, fmt.Errorf("fail to compute the identity for certificate [%v]", err)
			}
			sender = &acPb.Member{
				OrgId:      orgId,
				MemberInfo: *certId,
				MemberType: acPb.MemberType_CERT_HASH,
			}
		} else {
			sender = &acPb.Member{
				OrgId:      orgId,
				MemberInfo: *file,
				//IsFullCert: true,
			}
		}
	}

	// 构造Header
	req := &commonPb.TxRequest{
		Payload: payload,
		Sender: &commonPb.EndorsementEntry{
			Signer: sender,
		},
	}
	if len(endorsers) > 0 {
		req.Endorsers = endorsers
	}
	// 拼接后，计算Hash，对hash计算签名
	rawTxBytes, err := utils.CalcUnsignedTxRequestBytes(req)
	if err != nil {
		return nil, err
	}

	hashType, err := getHashType(hashAlgo)
	if err != nil {
		return nil, err
	}

	signBytes, err := sdkutils.SignPayloadBytesWithHashType(sk3, hashType, rawTxBytes)
	if err != nil {
		return nil, err
	}

	req.Sender.Signature = signBytes

	result, err := client.SendRequest(ctx, req)
	if err != nil {
		if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.DeadlineExceeded {
			return nil, fmt.Errorf("client.call err: deadline\n")
		}
		return nil, fmt.Errorf("client.call err: %v\n", err)
	}
	return result, nil
}
*/

type outerFunc func(statistician *Statistician)

func rPrint(s *Statistician, of ...outerFunc) {
	for _, o := range of {
		o(s)
	}
}

func outA() outerFunc {
	return func(statistician *Statistician) {
		fmt.Println("")
	}
}

func outB() outerFunc {
	return func(statistician *Statistician) {

	}
}
