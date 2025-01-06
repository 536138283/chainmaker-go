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
	reqStatC          chan *reqStat
	minSuccessElapsed int64
	maxSuccessElapsed int64
	sumSuccessElapsed int64
	totalCount        int32
	successCount      int

	lastIndex      int
	lastStartTime  time.Time
	startTime      time.Time // 开始时间
	endTime        time.Time // 结束时间
	preTime        time.Time // 上次的统计结束的时间用来计算时间间隔
	preBlockHeight uint64
	preBlockTime   int64

	elapsedSeconds float32 // 统计的时间间隔
	// Classify by node id
	cReqStatC             chan *cReqStat
	nodeMinSuccessElapsed []int64
	nodeMaxSuccessElapsed []int64
	nodeSumSuccessElapsed []int64
	nodeSuccessReqCount   []int
	nodeTotalReqCount     []int

	// block chain standards
	blockNum             int64
	nodeBlockNum         []int64
	nodeTxTotal          []int64
	nodeMaxTxBlockHeight []uint64
	nodeMaxTxBlockCount  []uint32
	nodeMinTxBlockHeight []uint64
	nodeMinTxBlockCount  []uint32
	nodeFirstBlockHeight []uint64
	nodeLastBlockHeight  []uint64
	maxTxBlockHeight     uint64
	maxTxBlockCount      uint32
	minTxBlockHeight     uint64
	minTxBlockCount      uint32
	firstBlockTime       int64
	firstBlockHeight     uint64
	lastBlockTime        int64
	lastBlockHeight      uint64
	nodeFirstBlockTime   []int64
	nodeLastBlockTime    []int64
	txTotal              uint32
	blockTotal           uint32
}

func (s *Statistician) outBlockInfo(resultSet *ResultSet) {
	if s.blockTotal == 0 {
		fmt.Println("no block")
		return
	}
	blockInfo := &BlockInfo{}
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
	blockInfo.BlockOutAvg = float32(s.lastBlockHeight-s.preBlockHeight+addNum) / float32(s.elapsedSeconds)
	// 区块数量
	blockInfo.BlockNum = s.lastBlockHeight - s.preBlockHeight + addNum
	// 第一个区块的出块时间, 高度
	blockInfo.FirstBlockTime = time.Unix(s.preBlockTime, 0).Format("2006-01-02 15:04:05.000")
	blockInfo.FirstBlockHeight = s.preBlockHeight
	// 最后一个区块的出块时间，高度
	blockInfo.LastBlockHeight = s.lastBlockHeight
	blockInfo.LastBlockTime = time.Unix(s.lastBlockTime, 0).Format("2006-01-02 15:04:05.000")
	// 计算ctps
	blockInfo.CTps = float32(s.txTotal) / float32(s.elapsedSeconds)
	// 计算区块内平均的交易数
	blockInfo.BlockTxNumAvg = float32(s.txTotal) / float32(s.blockTotal)
	resultSet.BlockInfo = blockInfo
	// 成功上链交易数量
	resultSet.SuccessCount = int64(s.txTotal)
	// 未上链交易数量
	resultSet.FailCount = requestId - int64(s.txTotal)
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
	// 计算完毕 以当前最后一个区块为准计算下一次的平均区块产出
	s.preBlockHeight = s.lastBlockHeight
	s.preBlockTime = s.lastBlockTime
}

func (s *Statistician) outNodeBlockInfo(resultSet *ResultSet) {
	for i, _ := range hosts {
		blockInfo := &BlockInfo{}
		resultSet.MaxTxBlock.BlockHeight = s.nodeMaxTxBlockHeight[i]
		resultSet.MaxTxBlock.TxCount = s.nodeMaxTxBlockCount[i]
		resultSet.MinTxBlock.BlockHeight = s.nodeMinTxBlockHeight[i]
		resultSet.MinTxBlock.TxCount = s.nodeMinTxBlockCount[i]
		blockInfo.BlockOutAvg = float32(s.nodeMaxTxBlockHeight[i]-s.nodeMinTxBlockHeight[i]) / s.elapsedSeconds
		blockInfo.BlockNum = s.lastBlockHeight - s.firstBlockHeight
		blockInfo.FirstBlockHeight = s.nodeFirstBlockHeight[i]
		blockInfo.LastBlockHeight = s.nodeLastBlockHeight[i]
		blockInfo.LastBlockTime = time.Unix(s.nodeLastBlockTime[i], 0).Format("2006-01-02 15:04:05")
		blockInfo.CTps = float32(s.txTotal) / s.elapsedSeconds
		resultSet.Nodes = append(resultSet.Nodes, &NodeInfo{BlockInfo: blockInfo})
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
			// 统计交易最多的区块高度，块交易数量
			if s.maxTxBlockCount < stat.blockHeader.TxCount {
				s.maxTxBlockHeight = stat.blockHeader.BlockHeight
				s.maxTxBlockCount = stat.blockHeader.TxCount
			}
			// 统计节点交易最多的区块高度，块交易数量
			if s.nodeMaxTxBlockCount[stat.nodeId] < stat.blockHeader.TxCount {
				s.nodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
				s.nodeMaxTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
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
			// 统计节点交易最少的区块高度，块交易数量
			if s.nodeMinTxBlockCount[stat.nodeId] == 0 {
				s.nodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
				s.nodeMinTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
			}
			if s.nodeMinTxBlockCount[stat.nodeId] > stat.blockHeader.TxCount {
				s.nodeMaxTxBlockHeight[stat.nodeId] = stat.blockHeader.BlockHeight
				s.nodeMinTxBlockCount[stat.nodeId] = stat.blockHeader.TxCount
			}
			// 统计第一次出块时间和最后一次出块时间
			if s.firstBlockTime == 0 {
				s.firstBlockTime = stat.blockHeader.BlockTimestamp
			}
			if s.firstBlockHeight == 0 {
				s.firstBlockHeight = stat.blockHeader.BlockHeight
			}
			s.txTotal += stat.blockHeader.TxCount
			s.blockTotal++
			s.lastBlockTime = stat.blockHeader.BlockTimestamp
			s.lastBlockHeight = stat.blockHeader.BlockHeight
			// 统计节点第一个出块时间和最后一次区块高度
			if s.nodeFirstBlockTime[stat.nodeId] == 0 {
				s.nodeFirstBlockTime[stat.nodeId] = stat.blockHeader.BlockTimestamp
			}
			s.nodeLastBlockTime[stat.nodeId] = stat.blockHeader.BlockTimestamp
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
	BlockInfo  *BlockInfo `json:"blockInfo"`
	MaxTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"`
		TxCount     uint32 `json:"txCount"`
	} `json:"maxTxBlock"`
	MinTxBlock struct {
		BlockHeight uint64 `json:"blockHeight"`
		TxCount     uint32 `json:"txCount"`
	} `json:"minTxBlock"`
	SuccessCount int64 `json:"successCount"`
	FailCount    int64 `json:"failCount"`
	DealMax      int   `json:"dealMax"`
	DealMin      int   `json:"dealMin"`
	//CTPs         float64     `json:"ctps"`
	QTPs      float64     `json:"qtps"`
	ThreadNum int         `json:"threadNum"`
	LoopNum   int         `json:"loopNum"`
	StartTime string      `json:"startTime"`
	EndTime   string      `json:"endTime"`
	Nodes     []*NodeInfo `json:"nodes"`
}

type NodeInfo struct {
	BlockInfo    *BlockInfo `json:"blockInfo"`
	Ctps         float64    `json:"ctps"`
	Qtps         float64    `json:"qtps"`
	SuccessCount int        `json:"successCount"`
	FailCount    int        `json:"failCount"`
	DealMax      int        `json:"dealMax"`
	DealMin      int        `json:"dealMin"`
}
