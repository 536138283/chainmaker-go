package parallel

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"fmt"
	"time"
)

type reqStat struct {
	success bool  // 请求是否成功
	elapsed int64 // 单次请求结束用时 单位：毫秒
	nodeId  int   // 节点id
}

type cReqStat struct {
	blockHeader *commonPb.BlockHeader
	nodeId      int
	elapsed     int64
}

type Statistician struct {
	rpcStatistician
	chainStatistician
	// 通用统计数据
	nodeTotalReqCount []uint32 // 发起 交易/请求总数
	totalCount        uint32   // 各个节点发起 交易/请求 总数
	elapsedSeconds    float32  // 统计的时间间隔
}

// rpc统计对象
type rpcStatistician struct {
	// rpc standards
	reqStatC              chan *reqStat // 用来接收rpc响应的chan，从该chan中读取请求响应信息并进行统计
	minSuccessElapsed     int64         // 最短成功响应时长 单位：ms
	maxSuccessElapsed     int64         // 最大成功响应时长 单位：ms
	sumSuccessElapsed     int64         // 成功请求总延时 单位：ms
	successCount          uint32        // 成功请求数
	startTime             time.Time     // 开始时间
	endTime               time.Time     // 结束时间
	nodeMinSuccessElapsed []int64       // 节点最短请求响应成功时间
	nodeMaxSuccessElapsed []int64       // 节点最大请求响应成功时间
	nodeSumSuccessElapsed []int64       // 节点总响应成功时间
	nodeSuccessCount      []uint32      // 节点成功请求数
}

// 区块链统计对象
type chainStatistician struct {
	cReqStatC chan *cReqStat // 用来接收订阅的节点返回的区块的chan,从该chan中读取区块信息并进行统计
	blockStatistician
	nodeBlockStatistician
}

// 整链子区块统计对象
type blockStatistician struct {
	temporaryTxSpeed  uint32       // 链上临时交易处理速度
	txDealSpeedTicker *time.Ticker // 时间定时器
	MaxTxDealSpeed    uint32       // 链上最块交易处理速度
	MinTxDealSpeed    uint32       // 链上最慢交易处理速度
	blockNum          int64        // 链上的出块总数
	maxTxBlockHeight  uint64       // 链上包含最多交易的区块的区块高度
	maxTxBlockCount   uint32       // 链上包含最多交易的区块的交易数量
	minTxBlockHeight  uint64       // 链上包含最少交易的区块的区块高度
	minTxBlockCount   uint32       // 链上包含最少交易的区块的交易数量
	firstBlockTime    int64        // 链上第一次出块的时间
	firstBlockHeight  uint64       // 链上第一次出块的区块高度
	lastBlockTime     int64        // 链上最后一次出块的时间
	lastBlockHeight   uint64       // 链上最后一次出块的区块高度
	txTotal           uint32       // 链上交易总数
	blockTotal        uint32       // 链上出块总数
}

// 节点区块统计对象
type nodeBlockStatistician struct {
	nodeTemporaryTxSpeed []uint32 // 各个节点的临时交易处理速度，每秒统计一次统计后清空
	nodeMaxTxDealSpeed   []uint32 // 各个节点的最大交易处理速度，单位 笔/秒
	nodeMinTxDealSpeed   []uint32 // 各个节点的最小交易处理速度，单位 笔/秒
	nodeTxTotal          []uint32 // 各个节点的交易总数
	nodeMaxTxBlockHeight []uint64 // 各个节点包含最多交易的区块的高度
	nodeMaxTxBlockCount  []uint32 // 各个节点包含最多交易的区块的交易数
	nodeMinTxBlockHeight []uint64 // 各个节点包含最少交易的区块高度
	nodeMinTxBlockCount  []uint32 // 各个节点包含最少交易的区块的交易数
	nodeFirstBlockHeight []uint64 // 各个节点第一个区块的区块高度
	nodeLastBlockHeight  []uint64 // 各个节点最后一个区块的区块高度
	nodeFirstBlockTime   []int64  // 各个节点出块第一个出块时间
	nodeLastBlockTime    []int64  // 节点最后一次出块的时间
}

// 初始化默认统计对象
func getStatistician() *Statistician {
	s := &Statistician{}
	s.reqStatC = make(chan *reqStat, threadNum)
	s.nodeMinSuccessElapsed = make([]int64, nodeNum)
	s.nodeMaxSuccessElapsed = make([]int64, nodeNum)
	s.nodeSumSuccessElapsed = make([]int64, nodeNum)
	s.nodeSuccessCount = make([]uint32, nodeNum)
	s.nodeTotalReqCount = make([]uint32, nodeNum)
	s.cReqStatC = make(chan *cReqStat, threadNum)
	s.nodeTxTotal = make([]uint32, nodeNum)
	s.nodeMaxTxBlockHeight = make([]uint64, nodeNum)
	s.nodeMaxTxBlockCount = make([]uint32, nodeNum)
	s.nodeMinTxBlockHeight = make([]uint64, nodeNum)
	s.nodeMinTxBlockCount = make([]uint32, nodeNum)
	s.nodeFirstBlockHeight = make([]uint64, nodeNum)
	s.nodeLastBlockHeight = make([]uint64, nodeNum)
	s.nodeFirstBlockTime = make([]int64, nodeNum)
	s.nodeLastBlockTime = make([]int64, nodeNum)
	s.txDealSpeedTicker = time.NewTicker(time.Second)
	s.nodeTemporaryTxSpeed = make([]uint32, nodeNum)
	s.nodeMaxTxDealSpeed = make([]uint32, nodeNum)
	s.nodeMinTxDealSpeed = make([]uint32, nodeNum)
	s.startTime = time.Now()
	return s
}

// 将结果输出到result set结果集
func (s *Statistician) outBlockInfo(resultSet *ChainResultSet) {
	if s.blockTotal == 0 {
		fmt.Println("no block")
		return
	}
	// 区块数量
	resultSet.BlockNum = s.lastBlockHeight - s.firstBlockHeight + 1
	// 计算平均出块时间
	resultSet.BlockOutAvg = float32(resultSet.BlockNum) / float32(s.elapsedSeconds)
	// 第一个区块的出块时间, 高度
	resultSet.FirstBlockTime = time.Unix(s.firstBlockTime, 0).Format("2006-01-02 15:04:05.000")
	resultSet.FirstBlockHeight = s.firstBlockHeight
	// 最后一个区块的出块时间，高度
	resultSet.LastBlockHeight = s.lastBlockHeight
	resultSet.LastBlockTime = time.Unix(s.lastBlockTime, 0).Format("2006-01-02 15:04:05.000")
	// 计算ctps
	resultSet.CTps = float32(s.txTotal) / float32(s.elapsedSeconds)
	// 计算区块内平均的交易数
	resultSet.BlockTxNumAvg = float32(s.txTotal) / float32(resultSet.BlockNum)
	// 成功上链交易数量
	resultSet.SuccessCount = s.txTotal
	// 获取包含最大最小交易数的区块的区块高度和交易数量
	resultSet.MaxTxBlock.BlockHeight = s.maxTxBlockHeight
	resultSet.MaxTxBlock.TxCount = s.maxTxBlockCount
	resultSet.MinTxBlock.BlockHeight = s.minTxBlockHeight
	resultSet.MinTxBlock.TxCount = s.minTxBlockCount
	// 获取到处理速度
	resultSet.DealMax = s.MaxTxDealSpeed
	resultSet.DealMin = s.MinTxDealSpeed
}

func (s *Statistician) outNodeBlockInfo(resultSet *ChainResultSet) {
	resultSet.Nodes = make(map[string]*NodeInfo)
	for i, _ := range hosts {
		// 第一次计算是以第一个区块的高度为准，所以这里定义一个加数防止少计算一个区块
		nodeInfo := &NodeInfo{}
		// 节点的区块数量
		nodeInfo.BlockNum = s.nodeLastBlockHeight[i] - s.nodeFirstBlockHeight[i] + 1
		// 节点的平均区出块时间
		nodeInfo.BlockOutAvg = float32(nodeInfo.BlockNum) / s.elapsedSeconds
		// 第一个区块的出块时间, 高度
		nodeInfo.FirstBlockTime = time.Unix(s.nodeFirstBlockTime[i], 0).Format("2006-01-02 15:04:05.000")
		nodeInfo.FirstBlockHeight = s.nodeFirstBlockHeight[i]
		// 节点最后一个区块的出块时间，高度
		nodeInfo.LastBlockHeight = s.nodeLastBlockHeight[i]
		nodeInfo.LastBlockTime = time.Unix(s.nodeLastBlockTime[i], 0).Format("2006-01-02 15:04:05.000")
		// 计算节点的ctps
		nodeInfo.CTps = float32(s.nodeTxTotal[i]) / float32(s.elapsedSeconds)
		// 计算区块内平均的交易数
		nodeInfo.BlockTxNumAvg = float32(s.nodeTxTotal[i]) / float32(nodeInfo.BlockNum)
		// 统计节点的成功上链的交易数量
		nodeInfo.SuccessCount = s.nodeTxTotal[i]
		nodeInfo.DealMax = s.nodeMaxTxDealSpeed[i]
		nodeInfo.DealMin = s.nodeMinTxDealSpeed[i]
		// 添加到节点的结果集信息统计
		resultSet.Nodes[fmt.Sprintf("node%d", i)] = nodeInfo
	}
}

func (s *Statistician) outRpcInfo(resultSet *RpcResultSet) {
	resultSet.Nodes = make(map[string]interface{})
	if s.totalCount > 0 {
		resultSet.SuccessCount = s.successCount
		resultSet.FailCount = s.totalCount - s.successCount
		resultSet.Count = s.totalCount
		resultSet.MinTime = s.minSuccessElapsed
		resultSet.MaxTime = s.maxSuccessElapsed
		resultSet.AvgTime = float32(s.sumSuccessElapsed) / float32(s.totalCount)
		for i := 0; i < nodeNum; i++ {
			resultSet.Nodes[fmt.Sprintf("node%d_successCount", i)] = s.nodeSuccessCount[i]
			resultSet.Nodes[fmt.Sprintf("node%d_failCount", i)] = s.nodeTotalReqCount[i] - s.nodeSuccessCount[i]
			resultSet.Nodes[fmt.Sprintf("node%d_count", i)] = s.nodeTotalReqCount[i]
			resultSet.Nodes[fmt.Sprintf("node%d_minTime", i)] = s.nodeMinSuccessElapsed[i]
			resultSet.Nodes[fmt.Sprintf("node%d_maxTime", i)] = s.nodeMaxSuccessElapsed[i]
			resultSet.Nodes[fmt.Sprintf("node%d_avgTime", i)] = float32(s.nodeSumSuccessElapsed[i]) / float32(s.nodeTotalReqCount[i])
		}
	}
	resultSet.StartTime = s.startTime.Format("2006-01-02 15:04:05.000")
	resultSet.EndTime = s.endTime.Format("2006-01-02 15:04:05.000")
	resultSet.Elapsed = s.elapsedSeconds
	resultSet.TPS = float32(s.successCount) / float32(s.endTime.Sub(s.startTime).Seconds())
	for i := 0; i < nodeNum; i++ {
		resultSet.Nodes[fmt.Sprintf("node%d_tps", i)] = float32(s.nodeSuccessCount[i]) / float32(s.endTime.Sub(s.startTime).Seconds())
	}
}

// 收集参数
func (s *Statistician) collect() {
	flag := true
	for {
		select {
		case stat := <-s.reqStatC:
			// 统计rpc压测指标
			s.statisticianRpc(stat)
		case stat := <-s.cReqStatC:
			// 第一个区块为上一次执行时的最后一个区块，所以跳过第一个区块
			if flag {
				flag = false
				continue
			}
			// 统计区块信息（非节点）
			s.statisticianTxBlock(stat)
			// 统计节点区块信息（节点）
			s.statisticianNodeTxBlock(stat)
			// 计算交易处理速度
			computeSpeed(stat, s)
		}
	}
}

func (s *Statistician) statisticianRpc(stat *reqStat) {
	// 统计rpc结果到Statistician对象
	if stat.success {
		// 初始化最长最短的响应时长
		if s.minSuccessElapsed == 0 || s.maxSuccessElapsed == 0 {
			s.minSuccessElapsed = stat.elapsed
			s.maxSuccessElapsed = stat.elapsed
		}
		// 统计最长最短的成功响应时长
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
		s.nodeSuccessCount[stat.nodeId]++
		s.nodeSumSuccessElapsed[stat.nodeId] += stat.elapsed
	}
	s.nodeTotalReqCount[stat.nodeId]++
}

func (s *Statistician) statisticianTxBlock(stat *cReqStat) {
	// 统计交易最多的区块高度，块交易数量
	if s.maxTxBlockCount < stat.blockHeader.TxCount {
		s.maxTxBlockHeight = stat.blockHeader.BlockHeight
		s.maxTxBlockCount = stat.blockHeader.TxCount
	}
	// 统计交易最少的区块高度，块交易数量
	if s.minTxBlockCount == 0 {
		s.minTxBlockHeight = stat.blockHeader.BlockHeight
		s.minTxBlockCount = stat.blockHeader.TxCount
	}
	if s.minTxBlockCount > stat.blockHeader.TxCount {
		s.minTxBlockHeight = stat.blockHeader.BlockHeight
		s.minTxBlockCount = stat.blockHeader.TxCount
	}
	// 统计第一次出块时间和最后一次出块时间
	if s.firstBlockTime == 0 {
		s.firstBlockTime = stat.blockHeader.BlockTimestamp
	}
	if s.firstBlockHeight == 0 {
		s.firstBlockHeight = stat.blockHeader.BlockHeight
	}
	// 记录交易总数，区块总数
	s.txTotal += stat.blockHeader.TxCount
	s.blockTotal++
	// 更新最后一次出块的时间，区块高度
	s.lastBlockTime = stat.blockHeader.BlockTimestamp
	s.lastBlockHeight = stat.blockHeader.BlockHeight
}

func (s *Statistician) statisticianNodeTxBlock(stat *cReqStat) {
	// 统计节点交易最多的区块高度，块交易数量
	if s.nodeMaxTxBlockCount[stat.nodeId] < stat.blockHeader.TxCount {
		s.nodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
		s.nodeMaxTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
	}
	// 统计节点交易最少的区块高度，块交易数量
	if s.nodeMinTxBlockCount[stat.nodeId] == 0 {
		s.nodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
		s.nodeMinTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
	}
	if s.nodeMinTxBlockCount[stat.nodeId] > stat.blockHeader.TxCount {
		s.nodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
		s.nodeMinTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
	}
	// 统计节点第一个出块时间和最后一次区块高度
	if s.nodeFirstBlockTime[stat.nodeId] == 0 {
		s.nodeFirstBlockTime[stat.nodeId] = stat.blockHeader.BlockTimestamp
		s.nodeFirstBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
	}
	// 更新节点处理交易的总数
	s.nodeTxTotal[stat.nodeId] += stat.blockHeader.TxCount
	// 更新节点最后一次出块时间,区块高度
	s.nodeLastBlockTime[stat.nodeId] = stat.blockHeader.BlockTimestamp
	s.nodeLastBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
}

func computeSpeed(stat *cReqStat, s *Statistician) {
	s.temporaryTxSpeed += stat.blockHeader.TxCount
	s.nodeTemporaryTxSpeed[stat.nodeId] += stat.blockHeader.TxCount
	for {
		select {
		case <-s.txDealSpeedTicker.C:
			if s.MinTxDealSpeed == 0 || s.MaxTxDealSpeed == 0 {
				s.MaxTxDealSpeed = s.temporaryTxSpeed
				s.MinTxDealSpeed = s.temporaryTxSpeed
			}
			if s.temporaryTxSpeed > s.MaxTxDealSpeed {
				s.MaxTxDealSpeed = s.temporaryTxSpeed
			}
			if s.temporaryTxSpeed < s.MinTxDealSpeed {
				s.MinTxDealSpeed = s.temporaryTxSpeed
			}
			s.temporaryTxSpeed = 0
			if s.nodeMinTxDealSpeed[stat.nodeId] == 0 || s.nodeMaxTxDealSpeed[stat.nodeId] == 0 {
				s.nodeMaxTxDealSpeed[stat.nodeId] = s.nodeTemporaryTxSpeed[stat.nodeId]
				s.nodeMinTxDealSpeed[stat.nodeId] = s.nodeTemporaryTxSpeed[stat.nodeId]
			}
			if s.nodeTemporaryTxSpeed[stat.nodeId] > s.nodeMaxTxDealSpeed[stat.nodeId] {
				s.nodeMaxTxDealSpeed[stat.nodeId] = s.nodeTemporaryTxSpeed[stat.nodeId]
			}
			if s.nodeTemporaryTxSpeed[stat.nodeId] < s.nodeMinTxDealSpeed[stat.nodeId] {
				s.nodeMinTxDealSpeed[stat.nodeId] = s.nodeTemporaryTxSpeed[stat.nodeId]
			}
			s.nodeTemporaryTxSpeed[stat.nodeId] = 0
		default:
			return
		}
	}
}

type numberResults struct {
	count, successCount         int
	min, max, sum               int64
	nodeSuccessCount, nodeCount []int
	nodeMin, nodeMax, nodeSum   []int64
}

type BlockInfo struct {
	FirstBlockHeight uint64  `json:"firstBlockHeight"`
	LastBlockHeight  uint64  `json:"lastBlockHeight"`
	FirstBlockTime   string  `json:"firstBlockTime"`
	LastBlockTime    string  `json:"lastBlockTime"`
	BlockOutAvg      float32 `json:"blockOutAvg"`
	BlockNum         uint64  `json:"blockNum"`
	BlockTxNumAvg    float32 `json:"blockTxNumAvg"`
	CTps             float32 `json:"ctps"`
}

type ChainResultSet struct {
	BlockInfo
	MaxTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"`
		TxCount     uint32 `json:"txCount"`
	} `json:"maxTxBlock"`
	MinTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"`
		TxCount     uint32 `json:"txCount"`
	} `json:"minTxBlock"`
	SuccessCount uint32               `json:"successCount"`
	DealMax      uint32               `json:"dealMax"`
	DealMin      uint32               `json:"dealMin"`
	Nodes        map[string]*NodeInfo `json:"nodes"`
}

type NodeInfo struct {
	BlockInfo
	SuccessCount uint32 `json:"successCount"`
	DealMax      uint32 `json:"dealMax"`
	DealMin      uint32 `json:"dealMin"`
}

type RpcResultSet struct {
	TPS          float32                `json:"tps"`
	SuccessCount uint32                 `json:"successCount"`
	FailCount    uint32                 `json:"failCount"`
	Count        uint32                 `json:"count"`
	MinTime      int64                  `json:"minTime"`
	MaxTime      int64                  `json:"maxTime"`
	AvgTime      float32                `json:"avgTime"`
	StartTime    string                 `json:"startTime"`
	EndTime      string                 `json:"endTime"`
	Elapsed      float32                `json:"elapsed"`
	Nodes        map[string]interface{} `json:"nodes"`
}
