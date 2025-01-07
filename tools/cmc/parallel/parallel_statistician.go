package parallel

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"fmt"
	"time"
)

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
	reqStatC           chan *reqStat
	minSuccessElapsed  int64
	maxSuccessElapsed  int64
	sumSuccessElapsed  int64
	totalCount         uint32
	successCount       int
	lastIndex          int
	lastStartTime      time.Time
	startTime          time.Time // 开始时间
	endTime            time.Time // 结束时间
	preTime            time.Time // 上次的统计结束的时间用来计算时间间隔
	preBlockHeight     uint64
	preBlockTime       int64
	nodePreBlockHeight []uint64
	nodePreBlockTime   []int64
	elapsedSeconds     float32 // 统计的时间间隔
	// Classify by node id
	cReqStatC             chan *cReqStat
	nodeMinSuccessElapsed []int64
	nodeMaxSuccessElapsed []int64
	nodeSumSuccessElapsed []int64
	nodeSuccessReqCount   []int
	nodeTotalReqCount     []int
	// block chain standards
	temporaryTxTotal     uint32
	nodeTemporaryTxTotal []uint32
	nodeRequestTotal     []uint32
	txDealSpeedTicker    *time.Ticker
	TemporaryTxSpeed     uint32
	MaxTxDealSpeed       uint32
	MinTxDealSpeed       uint32
	blockNum             int64
	nodeBlockNum         []int64
	nodeTxTotal          []uint32
	nodeMaxTxBlockHeight []uint64
	nodeMaxTxBlockCount  []uint32
	nodeMinTxBlockHeight []uint64
	nodeMinTxBlockCount  []uint32
	nodeFirstBlockHeight []uint64
	nodeLastBlockHeight  []uint64
	nodeFirstBlockTime   []int64
	nodeLastBlockTime    []int64
	maxTxBlockHeight     uint64
	maxTxBlockCount      uint32
	minTxBlockHeight     uint64
	minTxBlockCount      uint32
	firstBlockTime       int64
	firstBlockHeight     uint64
	lastBlockTime        int64
	lastBlockHeight      uint64
	txTotal              uint32
	blockTotal           uint32
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
		nodeTemporaryTxTotal:  make([]uint32, nodeNum),
		nodePreBlockTime:      make([]int64, nodeNum),
		nodePreBlockHeight:    make([]uint64, nodeNum),
		cReqStatC:             make(chan *cReqStat, threadNum),
		nodeTxTotal:           make([]uint32, nodeNum),
		nodeMaxTxBlockHeight:  make([]uint64, nodeNum),
		nodeMaxTxBlockCount:   make([]uint32, nodeNum),
		nodeMinTxBlockHeight:  make([]uint64, nodeNum),
		nodeMinTxBlockCount:   make([]uint32, nodeNum),
		nodeFirstBlockHeight:  make([]uint64, nodeNum),
		nodeLastBlockHeight:   make([]uint64, nodeNum),
		nodeFirstBlockTime:    make([]int64, nodeNum),
		nodeLastBlockTime:     make([]int64, nodeNum),
		nodeRequestTotal:      make([]uint32, nodeNum),
		txDealSpeedTicker:     time.NewTicker(time.Second),
	}
}

func (s *Statistician) outBlockInfo(resultSet *ResultSet) {
	if s.blockTotal == 0 {
		fmt.Println("no block")
		return
	}
	// 第一次计算是以第一个区块的高度为准，所以这里定义一个加数防止少计算一个区块
	var addNum uint64
	if s.preBlockHeight == 0 {
		s.preBlockHeight = s.firstBlockHeight
		s.preBlockTime = s.firstBlockTime
		addNum = 1
	} else {
		addNum = 0
	}
	// 计算平均出块时间
	resultSet.BlockOutAvg = float32(s.lastBlockHeight-s.preBlockHeight+addNum) / float32(s.elapsedSeconds)
	// 区块数量
	resultSet.BlockNum = s.lastBlockHeight - s.preBlockHeight + addNum
	// 第一个区块的出块时间, 高度
	resultSet.FirstBlockTime = time.Unix(s.preBlockTime, 0).Format("2006-01-02 15:04:05.000")
	resultSet.FirstBlockHeight = s.preBlockHeight
	// 最后一个区块的出块时间，高度
	resultSet.LastBlockHeight = s.lastBlockHeight
	resultSet.LastBlockTime = time.Unix(s.lastBlockTime, 0).Format("2006-01-02 15:04:05.000")
	// 计算ctps
	resultSet.CTps = float32(s.temporaryTxTotal) / float32(s.elapsedSeconds)
	// 计算区块内平均的交易数
	resultSet.BlockTxNumAvg = float32(s.temporaryTxTotal) / float32(resultSet.BlockNum)
	// 成功上链交易数量
	resultSet.SuccessCount = s.txTotal
	// 未上链交易数量
	resultSet.FailCount = s.totalCount - s.txTotal
	// 开始结束时间
	resultSet.StartTime = s.preTime.Format("2006-01-02 15:04:05.000")
	resultSet.EndTime = time.Now().Format("2006-01-02 15:04:05.000")
	resultSet.ThreadNum = threadNum
	resultSet.LoopNum = loopNum
	// 获取包含最大最小交易数的区块的区块高度和交易数量
	resultSet.MaxTxBlock.BlockHeight = s.maxTxBlockHeight
	resultSet.MaxTxBlock.TxCount = s.maxTxBlockCount
	resultSet.MinTxBlock.BlockHeight = s.minTxBlockHeight
	resultSet.MinTxBlock.TxCount = s.minTxBlockCount
	// 获取到处理速度
	resultSet.DealMax = s.MaxTxDealSpeed
	resultSet.DealMin = s.MinTxDealSpeed
	// 计算完毕 以当前最后一个区块为准计算下一次的平均区块产出
	s.preBlockHeight = s.lastBlockHeight
	s.preBlockTime = s.lastBlockTime
	s.temporaryTxTotal = 0
}

func (s *Statistician) outNodeBlockInfo(resultSet *ResultSet) {
	for i, _ := range hosts {
		// 第一次计算是以第一个区块的高度为准，所以这里定义一个加数防止少计算一个区块
		var addNum uint64
		if s.nodePreBlockHeight[i] == 0 {
			s.nodePreBlockHeight[i] = s.firstBlockHeight
			s.nodePreBlockTime[i] = s.firstBlockTime
			addNum = 1
		} else {
			addNum = 0
		}
		nodeInfo := &NodeInfo{}
		// 节点的平均区出块时间
		nodeInfo.BlockOutAvg = float32(s.nodeLastBlockHeight[i]-s.nodePreBlockHeight[i]+addNum) / s.elapsedSeconds
		// 节点的区块数量
		nodeInfo.BlockNum = s.nodeLastBlockHeight[i] - s.nodePreBlockHeight[i] + addNum
		// 第一个区块的出块时间, 高度
		nodeInfo.FirstBlockTime = time.Unix(s.nodePreBlockTime[i], 0).Format("2006-01-02 15:04:05.000")
		nodeInfo.FirstBlockHeight = s.nodePreBlockHeight[i]
		// 节点最后一个区块的出块时间，高度
		nodeInfo.LastBlockHeight = s.nodeLastBlockHeight[i]
		nodeInfo.LastBlockTime = time.Unix(s.nodeLastBlockTime[i], 0).Format("2006-01-02 15:04:05.000")
		// 计算节点的ctps
		nodeInfo.CTps = float32(s.nodeTemporaryTxTotal[i]) / float32(s.elapsedSeconds)
		// 计算区块内平均的交易数
		nodeInfo.BlockTxNumAvg = float32(s.nodeTemporaryTxTotal[i]) / float32(nodeInfo.BlockNum)
		// 统计节点的成功上链的交易数量与请求数量
		nodeInfo.SuccessCount = s.nodeTxTotal[i]
		nodeInfo.FailCount = s.nodeRequestTotal[i] - s.nodeTxTotal[i]
		// 添加到节点的结果集信息统计
		resultSet.Nodes = append(resultSet.Nodes, nodeInfo)
		// 计算完毕 以当前最后一个区块为准计算下一次的平均区块产出
		s.nodePreBlockHeight[i] = s.nodeLastBlockHeight[i]
		s.nodePreBlockTime[i] = s.nodeLastBlockTime[i]
		s.nodeTemporaryTxTotal[i] = 0
	}
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

// 收集参数
func (s *Statistician) Collect() {
	flag := true
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
	// 记录临时的交易处理数量
	s.temporaryTxTotal += stat.blockHeader.TxCount
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
	// 记录节点临时的交易处理数量
	s.nodeTemporaryTxTotal[stat.nodeId] += stat.blockHeader.TxCount
	// 统计节点第一个出块时间和最后一次区块高度
	if s.nodeFirstBlockTime[stat.nodeId] == 0 {
		s.nodeFirstBlockTime[stat.nodeId] = stat.blockHeader.BlockTimestamp
	}
	// 更新节点处理交易的总数
	s.nodeTxTotal[stat.nodeId] += stat.blockHeader.TxCount
	// 更新节点最后一次出块时间,区块高度
	s.nodeLastBlockTime[stat.nodeId] = stat.blockHeader.BlockTimestamp
	s.nodeLastBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
}

func computeSpeed(stat *cReqStat, s *Statistician) {
	s.TemporaryTxSpeed += stat.blockHeader.TxCount
	for {
		select {
		case <-s.txDealSpeedTicker.C:
			if s.MinTxDealSpeed == 0 || s.MaxTxDealSpeed == 0 {
				s.MaxTxDealSpeed = s.TemporaryTxSpeed
				s.MinTxDealSpeed = s.TemporaryTxSpeed
			}
			if s.TemporaryTxSpeed > s.MaxTxDealSpeed {
				s.MaxTxDealSpeed = s.TemporaryTxSpeed
			}
			if s.TemporaryTxSpeed < s.MinTxDealSpeed {
				s.MinTxDealSpeed = s.TemporaryTxSpeed
			}
			s.TemporaryTxSpeed = 0
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
type ResultSet struct {
	BlockInfo
	MaxTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"`
		TxCount     uint32 `json:"txCount"`
	} `json:"maxTxBlock"`
	MinTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"`
		TxCount     uint32 `json:"txCount"`
	} `json:"minTxBlock"`
	SuccessCount uint32 `json:"successCount"`
	FailCount    uint32 `json:"failCount"`
	DealMax      uint32 `json:"dealMax"`
	DealMin      uint32 `json:"dealMin"`
	//CTPs         float64     `json:"ctps"`
	QTPs      float64     `json:"qtps"`
	ThreadNum int         `json:"threadNum"`
	LoopNum   int         `json:"loopNum"`
	StartTime string      `json:"startTime"`
	EndTime   string      `json:"endTime"`
	Nodes     []*NodeInfo `json:"nodes"`
}

type NodeInfo struct {
	BlockInfo
	Qtps         float64 `json:"qtps"`
	SuccessCount uint32  `json:"successCount"`
	FailCount    uint32  `json:"failCount"`
	DealMax      int     `json:"dealMax"`
	DealMin      int     `json:"dealMin"`
}
