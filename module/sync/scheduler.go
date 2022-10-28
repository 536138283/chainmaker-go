/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sync

import (
	"fmt"
	"math"
	"sort"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/config"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/localconf/v2"
	syncPb "chainmaker.org/chainmaker/pb-go/v2/sync"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/Workiva/go-datastructures/queue"
	"github.com/gogo/protobuf/proto"
)

type syncSender interface {
	broadcastMsg(msgType syncPb.SyncMsg_MsgType, msg []byte) error
	sendMsg(msgType syncPb.SyncMsg_MsgType, msg []byte, to string) error
}

// scheduler Retrieves block data of specified height from different nodes
type scheduler struct {
	peers             map[string]uint64     // The state of the connected nodes
	blockStates       map[uint64]blockState // Block state for each height (New, Pending, Received)
	pendingTime       map[uint64]time.Time  // The time to send a request for a block data of the specified height
	pendingBlocks     map[uint64]string     // Which the block data of the specified height is being fetched from the node
	receivedBlocks    map[uint64]string     // Block data has been received from the node
	lastRequest       time.Time             // The last time which block request was sent
	pendingRecvHeight uint64                // The next block to be processed, all smaller blocks have been processed

	// the maximum number of blocks allowed to be processed simultaneously
	// (including: New, Pending, Received)
	maxPendingBlocks    uint64
	BatchesizeInEachReq uint64        // Number of blocks requested per request
	peerReqTimeout      time.Duration // The maximum timeout for a node response
	// When the difference between the height of the node and
	// the latest height of peers is 1, the time interval for requesting
	reqTimeThreshold time.Duration

	log    protocol.Logger
	sender syncSender
	ledger protocol.LedgerCache

	// the time when the service starts
	startTime         time.Time
	minLagReachC      chan struct{}
	lastCheckMajority time.Time
	// collect more node status for a specified period of time
	thresholdTime time.Duration
	// The remaining blocks to be synchronized are carried out by the consensus module
	thresholdBlocks uint64
	// indicate stop syncing block function
	stopSyncBlock bool
	chainConf     protocol.ChainConf
}

func newScheduler(
	sender syncSender,
	ledger protocol.LedgerCache, maxNum uint64,
	timeOut, reqTimeThreshold time.Duration,
	batchesize uint64, log protocol.Logger,
	reachC chan struct{}, minLagThreshold uint64,
	minLagThresholdTime time.Duration, conf protocol.ChainConf) *scheduler {

	currHeight, err := ledger.CurrentHeight()
	if err != nil {
		return nil
	}
	return &scheduler{
		log:    log,
		ledger: ledger,
		sender: sender,

		peerReqTimeout:      timeOut,
		maxPendingBlocks:    maxNum,
		BatchesizeInEachReq: batchesize,
		reqTimeThreshold:    reqTimeThreshold,

		peers:             make(map[string]uint64),
		blockStates:       make(map[uint64]blockState),
		pendingBlocks:     make(map[uint64]string),
		pendingTime:       make(map[uint64]time.Time),
		receivedBlocks:    make(map[uint64]string),
		pendingRecvHeight: currHeight + 1,

		startTime:       time.Now(),
		thresholdTime:   minLagThresholdTime,
		thresholdBlocks: minLagThreshold,
		minLagReachC:    reachC,
		chainConf:       conf,
	}
}

func (sch *scheduler) handler(event queue.Item) (queue.Item, error) {
	switch msg := event.(type) {
	case *NodeStatusMsg:
		sch.log.Debug("receive [NodeStatusMsg] msg, start handle...")
		sch.handleNodeStatus(msg)
	case *LivenessMsg:
		sch.log.Debug("receive [LivenessMsg] msg, start handle...")
		sch.handleLivinessMsg()
	case *SchedulerMsg:
		//sch.log.Debug("receive [SchedulerMsg] msg, start handle...")
		return sch.handleScheduleMsg()
	case *SyncedBlockMsg:
		sch.log.Info("receive [SyncedBlockMsg] msg, start handle...")
		return sch.handleSyncedBlockMsg(msg)
	case *ProcessedBlockResp:
		sch.log.Debug("receive [ProcessedBlockResp] msg, start handle...")
		return sch.handleProcessedBlockResp(msg)
	case *DataDetection:
		sch.log.Debug("receive [DataDetection] msg, start handle...")
		sch.handleDataDetection()
	case *StopSyncMsg:
		sch.log.Debug("receive [StopSyncMsg] msg, start handle...")
		sch.handleStopSyncMsg()
	case *StartSyncMsg:
		sch.handleStartSyncMsg()
	}
	return nil, nil
}

//update the node corresponding to the node id state information includes BlockHeight and ArchivedHeight
//if peer's ArchivedHeight is gather than local block height,indicates that node cant sync block from this peer,
//otherwise according to the own block height, mark the block that needs to be synchronized as "newblock"
func (sch *scheduler) handleNodeStatus(msg *NodeStatusMsg) {
	localCurrBlk := sch.ledger.GetLastCommittedBlock()
	if old, exist := sch.peers[msg.from]; exist {
		if old > msg.msg.BlockHeight || sch.isPeerArchivedTooHeight(localCurrBlk.Header.BlockHeight,
			msg.msg.GetArchivedHeight()) {
			delete(sch.peers, msg.from)
			return
		}
	}
	if sch.isPeerArchivedTooHeight(localCurrBlk.Header.BlockHeight, msg.msg.GetArchivedHeight()) {
		sch.log.Debugf("coming node[%s], status[height: %d, archivedHeight: %d], archived too height to sync, will ignore it",
			msg.from, msg.msg.BlockHeight, msg.msg.GetArchivedHeight())
		return
	}
	sch.log.Debugf("add node[%s], status[height: %d, archivedHeight: %d]", msg.from, msg.msg.BlockHeight,
		msg.msg.ArchivedHeight)
	sch.peers[msg.from] = msg.msg.BlockHeight
	sch.addPendingBlocksAndUpdatePendingHeight(msg.msg.BlockHeight)
}

func getNodes(conf *config.ChainConfig) []string {
	nodes := make([]string, 0, 4)
	for _, org := range conf.Consensus.Nodes {
		nodes = append(nodes, org.NodeId...)
	}
	return nodes
}

// receiveMajorityBlocks Check that most blocks are synchronized.
// currTime - startTime > thresholdTime && maxHeight - localHeight <= thresholdBlocks
func (sch *scheduler) receiveMajorityBlocks() {
	// 内部控制检测频率，避免上层调用间隔太短，造成频繁查询节点
	// 状态、获取共识节点数导致任务时间消耗变长
	if time.Since(sch.lastCheckMajority) < sch.thresholdTime/5 {
		return
	}
	sch.lastCheckMajority = time.Now()

	// 当节点刚启动后，需要一段时间同步其它节点的状态
	// 在thresholdTime时间范围内，当共识节点数量大于1，且获取到的节点
	// 状态信息小于总量的1/2时， 继续与其它节点进行状态同步
	nodeNum := len(getNodes(sch.chainConf.ChainConfig()))
	if time.Since(sch.startTime) < sch.thresholdTime && nodeNum > 1 && len(sch.peers) < nodeNum/2 {
		return
	}

	// 如果同步服务已经停止，不用再发送信号
	if sch.stopSyncBlock {
		return
	}

	// 查看是否已达到区块同步的阈值范围
	maxHeight := sch.maxHeight()
	currBlockHeight, _ := sch.ledger.CurrentHeight()
	if maxHeight > currBlockHeight && maxHeight-currBlockHeight > sch.thresholdBlocks {
		return
	}

	//达到阈值的同步范围
	select {
	case sch.minLagReachC <- struct{}{}:
		sch.log.Infof("has receive majorityBlocks, local node"+
			" block: %d, max height with peers: %d", currBlockHeight, maxHeight)
	default:
	}
}

//addPendingBlocksAndUpdatePendingHeight check if the local block height is lower than this peerHeight,
//if so, the state corresponding to the block height needs to be marked as newBlock
//only the block height of the newBlock state will be synchronized
func (sch *scheduler) addPendingBlocksAndUpdatePendingHeight(peerHeight uint64) {
	// 收集节点状态阶段 `handleNodeStatus` 添加 `blockStates` 长度检查和状态检查，保证最多发出 `bufferSize` 个区块数据请求
	// change '>' to '>=' indicate full range check
	if uint64(len(sch.blockStates)) >= sch.maxPendingBlocks {
		return
	}
	blk := sch.ledger.GetLastCommittedBlock()
	if blk.Header.BlockHeight >= peerHeight {
		return
	}
	//maxPendingBlocks is the upper limit of blocks waiting to be synchronized and being synchronized
	//Therefore, the quantity is required to ensure that it does not exceed this value
	for i := sch.pendingRecvHeight; i <= peerHeight && i < sch.pendingRecvHeight+sch.maxPendingBlocks; i++ {
		if _, exist := sch.blockStates[i]; !exist {
			// add blockState length check
			if len(sch.blockStates) > int(sch.maxPendingBlocks) {
				break
			}
			sch.blockStates[i] = newBlock
		}
	}
}

//handleDataDetection eliminate invalid data from the maintained data list
func (sch *scheduler) handleDataDetection() {
	blk := sch.ledger.GetLastCommittedBlock()
	for height := range sch.blockStates {
		if height < blk.Header.BlockHeight {
			delete(sch.blockStates, height)
			delete(sch.pendingBlocks, height)
			delete(sch.receivedBlocks, height)
			delete(sch.pendingTime, height)
		}
	}

	sch.pendingRecvHeight = blk.Header.BlockHeight + 1
	// `DataDetection` 中不对 `pendingRecvHeight` 高度的状态做状态重置，防止发起重复的数据请求，这部分逻辑由活性检查处理
	// match pendingRecvHeight to (local commit block height + 1)
	if _, exists := sch.blockStates[sch.pendingRecvHeight]; !exists {
		sch.blockStates[sch.pendingRecvHeight] = newBlock
	}
}

//handleLivinessMsg reset the response timeout data
func (sch *scheduler) handleLivinessMsg() {
	reqTime, exist := sch.pendingTime[sch.pendingRecvHeight]
	if exist && time.Since(reqTime) > sch.peerReqTimeout {
		id := sch.pendingBlocks[sch.pendingRecvHeight]
		sch.log.Debugf("block request [height: %d] time out from node[%s]", sch.pendingRecvHeight, id)
		if currBlk := sch.ledger.GetLastCommittedBlock(); currBlk != nil &&
			currBlk.Header.BlockHeight < sch.pendingRecvHeight {
			sch.blockStates[sch.pendingRecvHeight] = newBlock
		}
		delete(sch.peers, id)
		delete(sch.pendingTime, sch.pendingRecvHeight)
		delete(sch.pendingBlocks, sch.pendingRecvHeight)
	}

	// Note 共识单节点启动时，没有节点可以进行状态同步，需要通过该任务触发信号，
	// 通知共识模块开始运行
	sch.receiveMajorityBlocks()
}

//handleScheduleMsg find the starting block height that needs to be synchronized an a appropriate peer
//if ok, then send a sync request to peer to get the blocks data
//with height in[pendingHeight, pendingHeight+sch.BatchesizeInEachReq)
//localconf.ChainMakerConfig.NodeConfig.FastSyncConfig.Enable used to determine
//whether the response data of the request needs to have a read-write set
func (sch *scheduler) handleScheduleMsg() (queue.Item, error) {
	var (
		peer          string
		pendingHeight uint64
	)

	if !sch.isNeedSync() {
		//sch.log.Debugf("no need to sync block")
		return nil, nil
	}
	if pendingHeight = sch.nextHeightToReq(); pendingHeight == math.MaxUint64 {
		sch.log.Debugf("pendingHeight: %d, block status %v", pendingHeight, sch.blockStates)
		return nil, nil
	}
	if peer = sch.selectPeer(pendingHeight); len(peer) == 0 {
		sch.log.Debugf("no peers have block [%d] ", pendingHeight)
		return nil, nil
	}
	sch.lastRequest = time.Now()
	for i := pendingHeight; i <= sch.peers[peer] && i < sch.BatchesizeInEachReq+pendingHeight; i++ {
		sch.blockStates[i] = pendingBlock
		sch.pendingTime[i] = sch.lastRequest
		sch.pendingBlocks[i] = peer
	}
	if err := sch.sendSyncBlockRequest(peer, pendingHeight, sch.BatchesizeInEachReq,
		localconf.ChainMakerConfig.NodeConfig.FastSyncConfig.Enable); err != nil {
		return nil, err
	}
	return nil, nil
}

func (sch *scheduler) sendSyncBlockRequest(toPeer string, fromHeight, batch uint64, withRwset bool) error {
	req := syncPb.BlockSyncReq{
		BlockHeight: fromHeight,
		BatchSize:   batch,
		WithRwset:   withRwset,
	}
	bz, err := proto.Marshal(&req)
	if err != nil {
		return err
	}
	sch.log.Debugf("request block[height: %d] from node [%s], BatchesSizeInReq: %d", fromHeight, toPeer, batch)
	return sch.sender.sendMsg(syncPb.SyncMsg_BLOCK_SYNC_REQ, bz, toPeer)
}

//handleStopSyncMsg mark stop sync block and clean up records
func (sch *scheduler) handleStopSyncMsg() {
	sch.stopSyncBlock = true
	sch.peers = make(map[string]uint64)
	sch.blockStates = make(map[uint64]blockState)
	sch.pendingTime = make(map[uint64]time.Time)
	sch.pendingBlocks = make(map[uint64]string)
	sch.receivedBlocks = make(map[uint64]string)
}

func (sch *scheduler) handleStartSyncMsg() {
	// 1. 避免channel为空时发生阻塞
	select {
	case <-sch.minLagReachC:
	default:
	}
	sch.stopSyncBlock = false
}

//find the minimum block height marked as newBlock in blockStates
//return math.MaxUint64 to indicate that no block required to be requested for synchronizing
func (sch *scheduler) nextHeightToReq() uint64 {
	var min uint64 = math.MaxUint64
	for height, status := range sch.blockStates {
		if min > height && status == newBlock {
			min = height
		}
	}
	if min == math.MaxUint64 || min < sch.pendingRecvHeight {
		delete(sch.blockStates, min)
		return math.MaxUint64
	}
	return min
}

func (sch *scheduler) maxHeight() uint64 {
	var max uint64
	for _, height := range sch.peers {
		if max < height {
			max = height
		}
	}
	return max
}

//isNeedSync determine if synchronization is required
//it required if stopSyncBlock is false and local block height lags behind other nodes
//notes: if only one block behind, the time interval for synchronization needs to meet reqTimeThreshold
func (sch *scheduler) isNeedSync() bool {
	if sch.stopSyncBlock {
		return false
	}
	currHeight, err := sch.ledger.CurrentHeight()
	if err != nil {
		panic(err)
	}
	max := sch.maxHeight()
	// The reason for the interval of 1 block is that the block to
	// be synchronized is being processed by the consensus module.
	return currHeight+1 < max || (currHeight+1 == max && time.Since(sch.lastRequest) > sch.reqTimeThreshold)
}

//selectPeer from other peers select one that contains this height and is currently processing the fewest requests
func (sch *scheduler) selectPeer(pendingHeight uint64) string {
	peers := sch.getHeight(pendingHeight)
	if len(peers) == 0 {
		return ""
	}

	pendingReqInPeers := make(map[int][]string)
	for i := 0; i < len(peers); i++ {
		reqNum := sch.getPendingReqInPeer(peers[i])
		pendingReqInPeers[reqNum] = append(pendingReqInPeers[reqNum], peers[i])
	}
	min := math.MaxInt64
	for num := range pendingReqInPeers {
		if min > num {
			min = num
		}
	}
	peers = pendingReqInPeers[min]
	sort.Strings(peers)
	return peers[0]
}

//get all nodes containing this block height
func (sch *scheduler) getHeight(pendingHeight uint64) []string {
	peers := make([]string, 0, len(sch.peers)/2)
	for id, height := range sch.peers {
		if height >= pendingHeight {
			peers = append(peers, id)
		}
	}
	return peers
}

//getPendingReqInPeer count all blocks being processed by the 'peer'
func (sch *scheduler) getPendingReqInPeer(peer string) int {
	num := 0
	for _, id := range sch.pendingBlocks {
		if id == peer {
			num++
		}
	}
	return num
}

//handleSyncedBlockMsg check if there is any block that needs to be processed in the data received this time
//if so, hand over the data to the prosser for processing
func (sch *scheduler) handleSyncedBlockMsg(msg *SyncedBlockMsg) (queue.Item, error) {
	// 针对 `SyncMsg_BLOCK_SYNC_RESP` 消息处理函数，添加接收区块数量检查，超过 `bufferSize` 的额外信息会被暂时丢弃，
	//保证缓存的数据量可控
	// if len(receivedBlocks) > maxPendingBlocks, do not handle the msg
	if len(sch.receivedBlocks) > int(sch.maxPendingBlocks) {
		return nil, nil
	}
	// 如果已停止请求服务，将未停止服务前发送请求的到现在才收到的区块数据丢球
	if sch.stopSyncBlock {
		return nil, nil
	}
	blkBatch := syncPb.SyncBlockBatch{}
	if err := proto.Unmarshal(msg.msg, &blkBatch); err != nil {
		return nil, err
	}
	needToProcess := false
	sch.log.Debugf(
		"isFastSync: %v ,withRWSet: %v",
		localconf.ChainMakerConfig.NodeConfig.FastSyncConfig.Enable,
		blkBatch.WithRwset,
	)
	if blkBatch.GetBlockinfoBatch().Size() == 0 && blkBatch.GetBlockBatch().Size() == 0 {
		sch.log.Info("Get blocks is null")
		return nil, nil
	} else if blkBatch.GetBlockBatch().Size() != 0 {
		needToProcess = sch.updateSchedulerBySyncBlockBatch(
			msg.from, blkBatch.GetBlockBatch().GetBatches(), len(blkBatch.GetBlockBatch().GetBatches()))
	} else if blkBatch.GetBlockinfoBatch().Size() != 0 {
		needToProcess = sch.updateSchedulerBySyncBlockBatch(
			msg.from, blkBatch.GetBlockinfoBatch().GetBatch(), len(blkBatch.GetBlockinfoBatch().GetBatch()))
	}
	if needToProcess {
		return &ReceivedBlockInfos{
			SyncBlockBatch: &blkBatch,
			from:           msg.from,
		}, nil
	}
	return nil, nil
}

//according to the result of block verification, the following processing is performed
//1. validateFailed，verification failed，mark block state as "newBlock" waiting to be re-requested later
//at the same time, the node this block from needs to be removed from the locally cached peer data
//because it maybe a bad guy.
//2. addErr, failed to submit block data to local ledger，mark block state as "newBlock" waiting to be re-requested later
func (sch *scheduler) handleProcessedBlockResp(msg *ProcessedBlockResp) (queue.Item, error) {
	sch.log.Debugf("process block [height:%d] status[%d] from node"+
		" [%s], pendingHeight: %d", msg.height, msg.status, msg.from, sch.pendingRecvHeight)
	delete(sch.receivedBlocks, msg.height)
	//if the block was successfully processed
	//advance the block high value waiting for synchronization
	if msg.status == ok || msg.status == hasProcessed {
		delete(sch.blockStates, msg.height)
		if msg.height >= sch.pendingRecvHeight {
			sch.pendingRecvHeight = msg.height + 1
			sch.log.Debugf("increase pendingBlockHeight: %d", sch.pendingRecvHeight)
		}
	}
	if msg.status == validateFailed {
		sch.blockStates[msg.height] = newBlock
		delete(sch.peers, msg.from)
	}
	if msg.status == dbErr {
		return nil, fmt.Errorf("query db failed in processor")
	}
	if msg.status == addErr {
		sch.blockStates[msg.height] = newBlock
		delete(sch.peers, msg.from)
		return nil, fmt.Errorf("failed add block to chain")
	}
	return nil, nil
}

func (sch *scheduler) getServiceState() string {
	return fmt.Sprintf("pendingRecvHeight: %d, peers num: %d, blockStates num: %d, "+
		"pendingBlocks num: %d, receivedBlocks num: %d", sch.pendingRecvHeight, len(sch.peers), len(sch.blockStates),
		len(sch.pendingBlocks), len(sch.receivedBlocks))
}

func (sch *scheduler) isPeerArchivedTooHeight(localHeight, peerArchivedHeight uint64) bool {
	return peerArchivedHeight != 0 && localHeight <= peerArchivedHeight
}

//traverse the blocks that have been synchronized this time
//if the state of corresponding height is not "receivedBlock", indicates the data need to be processed in the next step
//set needToProcess to true moreover mark the state of corresponding height as "receivedBlock" in blockStates
//if the block is a config block without rwset, need to reacquire for rwset
func (sch *scheduler) updateSchedulerBySyncBlockBatch(msgFrom string, o interface{}, size int) bool {
	var (
		height      uint64
		hash        []byte
		block       *commonPb.Block
		hasRWset    bool
		resyncBlock []uint64
	)
	needToProcess := false

	for i := 0; i < size; i++ {
		switch ty := o.(type) {
		case []*commonPb.Block:
			block = ty[i]
			// height = ty[i].Header.BlockHeight
			// hash = ty[i].Header.BlockHash
			hasRWset = false
		case []*commonPb.BlockInfo:
			block = ty[i].Block
			// height = ty[i].Block.Header.BlockHeight
			// hash = ty[i].Block.Header.BlockHash
			hasRWset = ty[i].RwsetList != nil
		default:
			sch.log.Errorf("received unrecognized block type: [%t]", ty)
			continue
		}
		height = block.Header.BlockHeight
		hash = block.Header.BlockHash

		delete(sch.pendingBlocks, height)
		delete(sch.pendingTime, height)
		if state, exist := sch.blockStates[height]; exist {
			// 添加重复区块的检查，重复的区块不会被放入优先级队列中，
			//不做检查会将收到重复的区块返回数据放入优先级队列（该队列没有长度检查），这部分有内存泄漏的风险
			// if state == receivedBlock do not put into the msg queue, maintain needToProcess = false
			if state != receivedBlock {
				sch.log.Infof("received block [height:%d:%x] needToProcess: %v from "+
					"node [%s], withRwset:%t", height, hash, true, msgFrom, hasRWset)
				if (block.Header.BlockType == commonPb.BlockType_CONFIG_BLOCK ||
					block.Header.BlockType == commonPb.BlockType_CONFIG_BLOCK|commonPb.BlockType_HAS_COINBASE) &&
					!hasRWset { //如果是不带读写集的配置块，则重新请求带读写集的配置块
					resyncBlock = append(resyncBlock, height)
					continue
				}
				sch.blockStates[height] = receivedBlock
				sch.receivedBlocks[height] = msgFrom
				needToProcess = true
			}
		}
	}
	if len(resyncBlock) > 0 { //重新请求带读写集的配置块
		from := resyncBlock[0]
		peer := sch.selectPeer(resyncBlock[0])
		if len(peer) == 0 {
			sch.log.Debugf("no peers have block [%d] ", from)
			return needToProcess
		}
		batchs := resyncBlock[len(resyncBlock)-1] - resyncBlock[0] + 1
		if err := sch.sendSyncBlockRequest(msgFrom, resyncBlock[0], batchs, true); err != nil {
			sch.log.Warn("re-request config block[%d] with rwset from peer[%s] failed [%s]", resyncBlock[0], msgFrom, err)
			return needToProcess
		}
		sch.lastRequest = time.Now()
		for _, h := range resyncBlock {
			sch.blockStates[h] = pendingBlock
			sch.pendingTime[h] = sch.lastRequest
			sch.pendingBlocks[h] = msgFrom
		}
	}
	return needToProcess
}
