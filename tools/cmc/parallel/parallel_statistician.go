package parallel

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"fmt"
	"time"
)

// 用于记录单次rpc请求的统计信息对象
type reqStat struct {
	success bool  // 标记请求是否成功。true表示成功，false表示失败
	elapsed int64 // 记录请求的耗时，单位为毫秒。从请求开始到接收到响应的总时间
	nodeId  int   // 发起请求的目标节点ID，用于区分不同节点的请求统计
}

// 用来记录链上交易情况的统计信息对象
type cReqStat struct {
	blockHeader *commonPb.BlockHeader // 区块头信息
	nodeId      int                   // 发起请求的目标节点ID，用于区分不同节点的请求统计
}

type Statistician struct {
	rpcStatistician // rpc统计对象
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

// 统计rpc请求指标
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

// 统计链上交易的性能指标
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

// 统计链上交易的性能指标(节点)
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

// 计算链上处理交易的速度，单位笔/秒
func computeSpeed(stat *cReqStat, s *Statistician) {
	s.temporaryTxSpeed += stat.blockHeader.TxCount
	s.nodeTemporaryTxSpeed[stat.nodeId] += stat.blockHeader.TxCount
	for {
		select {
		case <-s.txDealSpeedTicker.C:
			// 非节点
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
			// 节点
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

// BlockInfo 区块信息
type BlockInfo struct {
	FirstBlockHeight uint64  `json:"firstBlockHeight"` // 链上的第一个出块的区块高度
	LastBlockHeight  uint64  `json:"lastBlockHeight"`  // 链上的会后一个出块的区块高度
	FirstBlockTime   string  `json:"firstBlockTime"`   // 链上的第一个出块的出块时间
	LastBlockTime    string  `json:"lastBlockTime"`    // 链上的最后一次出块的出块时间
	BlockOutAvg      float32 `json:"blockOutAvg"`      // 平均出块时间 单位：区块数/秒
	BlockNum         uint64  `json:"blockNum"`         // 链上出块总数
	BlockTxNumAvg    float32 `json:"blockTxNumAvg"`    // 区块平均交易数
	CTps             float32 `json:"ctps"`             // 区块链的吞吐量，用来衡量链上交易的处理能力 单位：交易数/秒
}

// ChainResultSet 统计结果集
type ChainResultSet struct {
	BlockInfo  // 区块信息
	MaxTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"` // 该区块的高度。
		TxCount     uint32 `json:"txCount"`     // 该区块中的交易数量
	} `json:"maxTxBlock"` // 结构体表示交易数量最多的区块信息
	MinTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"` // 该区块的高度。
		TxCount     uint32 `json:"txCount"`     // 该区块中的交易数量
	} `json:"minTxBlock"` // 结构体表示交易数量最少的区块信息
	SuccessCount uint32               `json:"successCount"` // 上链的交易数
	DealMax      uint32               `json:"dealMax"`      // 处理能力的最大值，可能指最大交易处理量等单位：笔/秒
	DealMin      uint32               `json:"dealMin"`      // 处理能力的最小值，与DealMax相对应
	Nodes        map[string]*NodeInfo `json:"nodes"`        // 字符串键映射到NodeInfo指针的字典，用于存储节点的区块信息
}

// NodeInfo 节点信息
type NodeInfo struct {
	BlockInfo           // 节点的区块信息
	SuccessCount uint32 `json:"successCount"` // 上链的交易数
	DealMax      uint32 `json:"dealMax"`      // 处理能力的最大值，可能指最大交易处理量等单位：笔/秒
	DealMin      uint32 `json:"dealMin"`      // 处理能力的最小值，与DealMax相对应
}

// RpcResultSet 结构体用于汇总RPC请求的统计结果，主要关注于性能指标和请求成功率。
type RpcResultSet struct {
	TPS          float32                `json:"tps"`          // 每秒处理事务数
	SuccessCount uint32                 `json:"successCount"` // 成功请求的计数，表示在统计周期内有多少RPC调用成功
	FailCount    uint32                 `json:"failCount"`    // 失败请求的计数，反映调用失败的次数
	Count        uint32                 `json:"count"`        // 总请求计数，即成功和失败请求的总和
	MinTime      int64                  `json:"minTime"`      // 所有请求中耗时最短的时间，单位通常是毫秒
	MaxTime      int64                  `json:"maxTime"`      // 所有请求中耗时最长的时间，单位通常是毫秒
	AvgTime      float32                `json:"avgTime"`      // 平均响应时间，所有请求耗时的平均值，单位通常是毫秒
	StartTime    string                 `json:"startTime"`    // 统计周期的开始时间，格式依据实际应用场景
	EndTime      string                 `json:"endTime"`      // 统计周期的结束时间，格式与StartTime对应
	Elapsed      float32                `json:"elapsed"`      // 统计周期的总时长，单位通常是秒
	Nodes        map[string]interface{} `json:"nodes"`        // 存储与各节点相关的数据，键为节点标识
}
