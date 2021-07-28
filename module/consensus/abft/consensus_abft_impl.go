/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"encoding/hex"
	"path"
	"sync"

	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/utils"
	"go.uber.org/zap"

	"github.com/gogo/protobuf/proto"
	"github.com/thoas/go-funk"
	"github.com/tidwall/wal"

	"chainmaker.org/chainmaker/common/msgbus"
	"chainmaker.org/chainmaker/pb-go/common"
	"chainmaker.org/chainmaker/pb-go/config"
	consensuspb "chainmaker.org/chainmaker/pb-go/consensus"
	abftpb "chainmaker.org/chainmaker/pb-go/consensus/abft"
	netpb "chainmaker.org/chainmaker/pb-go/net"
	"chainmaker.org/chainmaker/protocol"

	"chainmaker.org/chainmaker-go/logger"
)

var clog *zap.SugaredLogger = zap.S()
var (
	walDir = "abftwal"
)

// mustMarshal marshals protobuf message to byte slice or panic
func mustMarshal(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return data
}

// mustUnmarshal unmarshals from byte slice to protobuf message or panic
func mustUnmarshal(b []byte, msg proto.Message) {
	if err := proto.Unmarshal(b, msg); err != nil {
		panic(err)
	}
}

// ConsensusABFTImpl is the implementation of ABFT algorithm
// and it implements the ConsensusEngine interface.
type ConsensusABFTImpl struct {
	sync.RWMutex
	logger      *logger.CMLogger
	chainID     string
	Id          string
	msgbus      msgbus.MessageBus
	chainConf   protocol.ChainConf
	singer      protocol.SigningMember
	ledgerCache protocol.LedgerCache
	height      uint64
	msgSender   *msgSender
	waldir      string
	wal         *wal.Log
	// acsInstances map[int64]*ACS
	acs              *ACS
	msgBuffer        []*abftpb.ABFTMessageReq
	heightFirstIndex uint64
	closeC           chan struct{}
}

// ConsensusABFTImplConfig contains initialization config for ConsensusABFTImpl
type ConsensusABFTImplConfig struct {
	ChainID     string
	Id          string
	MsgBus      msgbus.MessageBus
	ChainConf   protocol.ChainConf
	Singer      protocol.SigningMember
	LedgerCache protocol.LedgerCache
}

// New creates a abft consensus instance
func New(config *ConsensusABFTImplConfig) (*ConsensusABFTImpl, error) {
	logger := logger.GetLoggerByChain(logger.MODULE_CONSENSUS, config.ChainID)
	logger.Infof("New ConsensusABFTImpl[%s]", config.Id)
	consensus := &ConsensusABFTImpl{}
	consensus.logger = logger
	consensus.chainID = config.ChainID
	consensus.Id = config.Id
	consensus.msgbus = config.MsgBus
	consensus.chainConf = config.ChainConf
	consensus.singer = config.Singer
	consensus.ledgerCache = config.LedgerCache
	consensus.msgBuffer = make([]*abftpb.ABFTMessageReq, 0)
	consensus.msgSender = newMsgSender(logger, consensus.Id)

	height, err := config.LedgerCache.CurrentHeight()
	if err != nil {
		return nil, err
	}
	consensus.height = height + 1

	consensus.waldir = path.Join(localconf.ChainMakerConfig.StorageConfig.StorePath, consensus.chainID, walDir)
	consensus.wal, err = wal.Open(consensus.waldir, nil)
	if err != nil {
		return nil, err
	}
	consensus.heightFirstIndex = 0

	return consensus, nil
}

// Start implements the Stop method of ConsensusEngine interface
// and starts the abft instance.
func (consensus *ConsensusABFTImpl) Start() error {
	consensus.logger.Infof("[%s] started", consensus.Id)

	// consensus.acsInstances = make(map[int64]*ACS)
	nodeList, _ := GetNodeListFromConfig(consensus.chainConf.ChainConfig())
	cfg := &Config{
		logger: consensus.logger,
		height: consensus.height,
		id:     consensus.Id,
		nodeID: consensus.Id,
		nodes:  nodeList,
	}
	cfg.fillWithDefaults()
	consensus.acs = NewACS(cfg)

	go consensus.run()
	err := consensus.replayWal()
	if err != nil {
		return err
	}

	consensus.sendPackageSingal(consensus.height)
	consensus.msgbus.Register(msgbus.ProposedBlock, consensus)
	consensus.msgbus.Register(msgbus.VerifyResult, consensus)
	consensus.msgbus.Register(msgbus.BlockInfo, consensus)
	consensus.msgbus.Register(msgbus.RecvConsensusMsg, consensus)
	return nil
}

// Stop implements the Stop method of ConsensusEngine interface
// and stops the abft instance.
func (consensus *ConsensusABFTImpl) Stop() error {
	consensus.Lock()
	defer consensus.Unlock()

	close(consensus.closeC)
	consensus.logger.Infof("[%s] stopped", consensus.Id)
	return nil
}

// OnMessage implements the OnMessage interface of msgbus.Subscriber
func (consensus *ConsensusABFTImpl) OnMessage(message *msgbus.Message) {
	consensus.onMessage(message, false)
}

func (consensus *ConsensusABFTImpl) onMessage(message *msgbus.Message, replayWalMode bool) {
	consensus.Lock()
	defer consensus.Unlock()

	consensus.logger.Debugf("[%s](%v) OnMessage receive topic: %s", consensus.Id, consensus.height, message.Topic)

	if !replayWalMode {
		consensus.saveWalEntry(message)
	}

	switch message.Topic {
	case msgbus.ProposedBlock:
		consensus.onProposedBlock(message)
	case msgbus.VerifyResult:
		consensus.onVerifyResult(message)
	case msgbus.BlockInfo:
		consensus.onBlockInfo(message)
	case msgbus.RecvConsensusMsg:
		consensus.onRecvConsensusMsg(message)
	}
}

func (consensus *ConsensusABFTImpl) run() {
	for {
		select {
		case <-consensus.closeC:
			return
		case msg := <-consensus.msgSender.msgCh:
			consensus.logger.Debugf("[%s](%d) send req seq: %v, height: %v, from: %v, Id: %v",
				consensus.Id, consensus.height, msg.Seq, msg.Height, msg.From, msg.To)
			abftMessage := &abftpb.ABFTMessage{
				Message: &abftpb.ABFTMessage_Req{Req: msg},
			}
			netMsg := &netpb.NetMsg{
				Payload: mustMarshal(abftMessage),
				Type:    netpb.NetMsg_CONSENSUS_MSG,
				To:      msg.To,
			}
			consensus.publishToMsgbus(netMsg)
		}
	}
}

// OnQuit implements the OnQuit interface of msgbus.Subscriber
func (consensus *ConsensusABFTImpl) OnQuit() {
	consensus.logger.Debugf("[%s] OnQuit", consensus.Id)
}

func (consensus *ConsensusABFTImpl) onProposedBlock(message *msgbus.Message) {
	block, ok := message.Payload.(*common.Block)
	if !ok {
		consensus.logger.Panicf("[%s](%v) receive wrong payload which should be Block", consensus.Id, consensus.height)
		return
	}

	if block.Header.BlockHeight != consensus.height {
		consensus.logger.Warnf("[%s](%v) receive wrong proposed block height: %v", consensus.Id, consensus.height, block.Header.BlockHeight)
		return
	}

	// Add hash and signature to block
	hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
	if err != nil {
		consensus.logger.Errorf("[%s]sign block failed, %s", consensus.Id, err)
		return
	}
	block.Header.BlockHash = hash[:]
	block.Header.Signature = sig

	consensus.logger.Debugf("[%s](%v) receive proposed block: (%v-%x-%x)",
		consensus.Id, consensus.height, block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash)

	data := mustMarshal(block)
	if err = consensus.acs.InputRBC(data); err != nil {
		consensus.logger.Errorf("[%s] input RBC error: %v", consensus.Id, err)
	}

	consensus.processEvent()
}

func (consensus *ConsensusABFTImpl) onVerifyResult(message *msgbus.Message) {
	verifyResult, ok := message.Payload.(*consensuspb.VerifyResult)
	if !ok {
		consensus.logger.Panicf("[%s](%v) receive wrong payload which should be VerifiedBlock", consensus.Id, consensus.height)
		return
	}

	if verifyResult.VerifiedBlock.Header.BlockHeight != consensus.height {
		consensus.logger.Warnf("[%s](%v) receive wrong verifyResult height: %v", consensus.Id, consensus.height, verifyResult.VerifiedBlock.Header.BlockHeight)
		return
	}

	consensus.logger.Debugf("[%s](%v) verify result code: %s, msg: %s, block: (%v-%x-%x)",
		consensus.Id, consensus.height, verifyResult.Code, verifyResult.Msg,
		verifyResult.VerifiedBlock.Header.BlockHeight, verifyResult.VerifiedBlock.Header.BlockHash, verifyResult.VerifiedBlock.Header.PreBlockHash)
	if verifyResult.Code != consensuspb.VerifyResult_SUCCESS {
		return
	}
	data := mustMarshal(verifyResult.VerifiedBlock)
	err := consensus.acs.InputBBA(data)
	if err != nil {
		consensus.logger.Errorf("acs input error: %v", err)
	}
	consensus.processEvent()
}

func (consensus *ConsensusABFTImpl) onBlockInfo(message *msgbus.Message) {
	blockInfo, ok := message.Payload.(*common.BlockInfo)
	if !ok {
		consensus.logger.Panicf("[%s](%v) receive wrong payload which should be BlockInfo", consensus.Id, consensus.height)
		return
	}
	if blockInfo == nil || blockInfo.Block == nil {
		consensus.logger.Errorf("receive message failed, error message BlockInfo = nil")
		return
	}

	lastIndex, err := consensus.wal.LastIndex()
	if err != nil {
		consensus.logger.Fatalf("[%s] failed to get lastIndex wal log %v",
			consensus.Id, consensus.height, err)
	}
	consensus.heightFirstIndex = lastIndex
	err = consensus.deleteWalEntry(consensus.height, lastIndex)
	if err != nil {
		consensus.logger.Fatalf("[%s] failed to delete wal log %v",
			consensus.Id, consensus.height, err)
	}

	consensus.msgSender.cleanHeight(consensus.height)

	consensus.height = blockInfo.Block.Header.BlockHeight + 1
	nodeList, _ := GetNodeListFromConfig(consensus.chainConf.ChainConfig())
	cfg := &Config{
		logger: consensus.logger,
		height: consensus.height,
		id:     consensus.Id,
		nodeID: consensus.Id,
		nodes:  nodeList,
	}
	cfg.fillWithDefaults()
	consensus.acs = NewACS(cfg)
	consensus.sendPackageSingal(consensus.height)

	buffer := make([]*abftpb.ABFTMessageReq, 0)
	for _, msg := range consensus.msgBuffer {
		if msg.Height == consensus.height {
			consensus.handleMessage(msg)
		} else if msg.Height > consensus.height {
			buffer = append(buffer, msg)
		}
	}
	consensus.msgBuffer = buffer
	consensus.processEvent()
}

func (consensus *ConsensusABFTImpl) onRecvConsensusMsg(message *msgbus.Message) {
	msg, ok := message.Payload.(*netpb.NetMsg)
	if !ok {
		consensus.logger.Panicf("[%s](%v) receive wrong payload which should be NetMsg", consensus.Id, consensus.height)
		return
	}

	abftMsg := new(abftpb.ABFTMessage)
	mustUnmarshal(msg.Payload, abftMsg)

	switch m := abftMsg.Message.(type) {
	case *abftpb.ABFTMessage_Req:
		consensus.handleMessage(m.Req)
		consensus.processEvent()
	case *abftpb.ABFTMessage_Rsp:
		consensus.msgSender.ack(m.Rsp)
	}
}

func (consensus *ConsensusABFTImpl) processEvent() {
	event := consensus.acs.Event()
	if event == nil {
		return
	}

	if event.rbcOutputs != nil {
		for _, output := range event.rbcOutputs {
			block := &common.Block{}
			mustUnmarshal(output, block)
			consensus.msgbus.PublishSafe(msgbus.VerifyBlock, block)
			consensus.logger.Debugf("[%s](%v) verify block: (%v-%x-%x)",
				consensus.Id, consensus.height, block.Header.BlockHeight, block.Header.BlockHash, block.Header.PreBlockHash)
		}
	}

	if event.messages != nil {
		for _, msg := range event.messages {
			consensus.handleMessage(msg)
		}
	}

	if event.outputs != nil && len(event.outputs) != 0 {
		var txBatchHash [][]byte
		for _, data := range event.outputs {
			block := &common.Block{}
			mustUnmarshal(data, block)
			txBatchHash = append(txBatchHash, block.Header.BlockHash)
		}

		consensus.logger.Debugf("[%s](%v) commit batchs: %v",
			consensus.Id, consensus.height,
			funk.Map(txBatchHash, func(data []byte) string { return hex.EncodeToString(data) }))
		consensus.msgbus.PublishSafe(msgbus.CommitedTxBatchs, &abftpb.TxBatchAfterABA{
			BlockHeight: consensus.height,
			TxBatchHash: txBatchHash,
		})
	}
}

func (consensus *ConsensusABFTImpl) handleMessage(msg *abftpb.ABFTMessageReq) {
	consensus.logger.Debugf("[%s](%d) handleMessage seq: %v, height: %v, from: %v, to: %v, Id: %v",
		consensus.Id, consensus.height, msg.Seq, msg.Height, msg.From, msg.To, msg.Id)
	if msg.Height < consensus.height {
		// Response with outdated error
		consensus.responseWithCode(msg, abftpb.ErrorCode_FailOfOutdatedHeight)
		return
	} else if msg.Height > consensus.height {
		//consensus.msgBuffer = append(consensus.msgBuffer, msg)
		return
	}
	if msg.To != consensus.Id {
		// netMsg := &netpb.NetMsg{
		//   Payload: mustMarshal(msg),
		//   Type:    netpb.NetMsg_CONSENSUS_MSG,
		//   To:      msg.To,
		// }
		// consensus.publishToMsgbus(netMsg)

		// Send messages to other nodes
		consensus.msgSender.addMsg(msg)
		return
	} else {
		err := consensus.acs.HandleMessage(msg.From, msg.Id, msg.Acs)
		if err != nil {
			consensus.logger.Errorf("[%s] handleMessage to: %s, error: %v", consensus.Id, msg.To, err)
			if msg.From != consensus.Id {
				consensus.responseWithCode(msg, abftpb.ErrorCode_FailOfUnkown)
			}
			return
		}
		if msg.From != consensus.Id {
			consensus.responseWithCode(msg, abftpb.ErrorCode_Success)
		}
	}
}

func (consensus *ConsensusABFTImpl) sendPackageSingal(height uint64) {
	consensus.logger.Debugf("[%s] sendPackageSingal height: %d", consensus.Id, height)
	signal := &abftpb.PackagedSignal{BlockHeight: height}
	consensus.msgbus.PublishSafe(msgbus.PackageSignal, signal)
}

func (consensus *ConsensusABFTImpl) publishToMsgbus(msg *netpb.NetMsg) {
	consensus.logger.Debugf("[%s] publishToMsgbus size: %d", consensus.Id, proto.Size(msg))
	consensus.msgbus.PublishSafe(msgbus.SendConsensusMsg, msg)
}

func (consensus *ConsensusABFTImpl) responseWithCode(msg *abftpb.ABFTMessageReq, code abftpb.ErrorCode) {
	consensus.logger.Debugf("[%s](%d) responseWithCode msg seq: %v, height: %v, from: %v, Id: %v, code: %s",
		consensus.Id, consensus.height, msg.Seq, msg.Height, msg.From, msg.Id, code)
	abftMessage := &abftpb.ABFTMessage{
		Message: &abftpb.ABFTMessage_Rsp{
			Rsp: &abftpb.ABFTMessageRsp{
				Seq:    msg.Seq,
				Height: msg.Height,
				From:   consensus.Id,
				To:     msg.From,
				Id:     msg.Id,
				Code:   code,
			},
		},
	}
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(abftMessage),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      msg.From,
	}
	consensus.publishToMsgbus(netMsg)
}

func GetNodeListFromConfig(chainConfig *config.ChainConfig) (validators []string, err error) {
	nodes := chainConfig.Consensus.Nodes
	for _, node := range nodes {
		for _, nid := range node.NodeId {
			validators = append(validators, nid)
		}
	}
	return validators, nil
}

// VerifyBlockSignatures verifies whether the signatures in block
// is qulified with the consensus algorithm. It should return nil
// error when verify successfully, and return corresponding error
// when failed.
func VerifyBlockSignatures(chainConf protocol.ChainConf, ac protocol.AccessControlProvider, block *common.Block) error {
	return nil
}

// saveWalEntry saves entry to Wal
func (consensus *ConsensusABFTImpl) saveWalEntry(message *msgbus.Message) {
	var data []byte

	switch message.Topic {
	case msgbus.ProposedBlock:
		data = mustMarshal(message.Payload.(*common.Block))
	case msgbus.VerifyResult:
		data = mustMarshal(message.Payload.(*consensuspb.VerifyResult))
	case msgbus.BlockInfo:
		data = mustMarshal(message.Payload.(*common.BlockInfo))
	case msgbus.RecvConsensusMsg:
		data = mustMarshal(message.Payload.(*netpb.NetMsg))
	default:
		consensus.logger.Fatalf("[%s](%d) save wal of unknown topic: %s",
			consensus.Id, consensus.height, message.Topic)
	}

	lastIndex, err := consensus.wal.LastIndex()
	if err != nil {
		consensus.logger.Fatalf("[%s](%d) save wal of topic: %s get last index error: ",
			consensus.Id, consensus.height, message.Topic, err)
	}

	walEntry := abftpb.WalEntry{
		Height:           consensus.height,
		HeightFirstIndex: consensus.heightFirstIndex,
		Topic:            int32(message.Topic),
		Data:             data,
	}

	log := mustMarshal(&walEntry)
	err = consensus.wal.Write(lastIndex+1, log)
	if err != nil {
		consensus.logger.Fatalf("[%s](%d) save wal of topic: %s write error: %v",
			consensus.Id, consensus.height, message.Topic, err)
	}
	consensus.logger.Debugf("[%s](%d) save wal of topic: %s data length: %v",
		consensus.Id, consensus.height, message.Topic, len(data))
}

// replayWal replays the wal when the node starting
func (consensus *ConsensusABFTImpl) replayWal() error {
	currentHeight, err := consensus.ledgerCache.CurrentHeight()
	if err != nil {
		return err
	}
	consensus.logger.Infof("[%s] replayWal currentHeight: %d", consensus.Id, currentHeight)

	lastIndex, err := consensus.wal.LastIndex()
	if err != nil {
		return err
	}
	consensus.logger.Infof("[%s] replayWal lastIndex of wal: %d", consensus.Id, lastIndex)

	data, err := consensus.wal.Read(lastIndex)
	if err == wal.ErrNotFound {
		consensus.logger.Infof("[%s] replayWal can't found log entry in wal", consensus.Id)
		return nil
	}
	if err != nil {
		return err
	}

	entry := &abftpb.WalEntry{}
	mustUnmarshal(data, entry)

	height := entry.Height
	if currentHeight < height-1 {
		consensus.logger.Fatalf("[%s] replay currentHeight: %v < height-1: %v, this should not happen",
			consensus.Id, currentHeight, height-1)
	}

	if currentHeight >= height {
		// consensus is slower than ledger
		return nil
	} else {
		// replay wal log
		for i := entry.HeightFirstIndex; i <= lastIndex; i++ {
			consensus.logger.Debugf("[%d] replay entry type: %s, Data.len: %d", consensus.Id, entry.Topic, len(entry.Data))
			var payload interface{}
			switch msgbus.Topic(entry.Topic) {
			case msgbus.ProposedBlock:
				block := new(common.Block)
				mustUnmarshal(entry.Data, block)
				payload = block
			case msgbus.VerifyResult:
				result := new(consensuspb.VerifyResult)
				mustUnmarshal(entry.Data, result)
				payload = result
			case msgbus.BlockInfo:
				info := new(common.BlockInfo)
				mustUnmarshal(entry.Data, info)
				payload = info
			case msgbus.RecvConsensusMsg:
				msg := new(netpb.NetMsg)
				mustUnmarshal(entry.Data, msg)
				payload = msg
			}

			msg := &msgbus.Message{
				Topic:   msgbus.Topic(entry.Topic),
				Payload: payload,
			}

			consensus.onMessage(msg, true)
		}
	}

	return nil
}

func (consensus *ConsensusABFTImpl) deleteWalEntry(num uint64, index uint64) error {
	// Block height is begin from zero,Delete the block data every 10 blocks.
	// If the block height is 10, there are 11 blocks in total and delete the consensus state data of the first 10 blocks
	i := num % 10
	if i != 0 {
		return nil
	}

	err := consensus.wal.TruncateFront(index)
	if err != nil {
		return err
	}

	consensus.logger.Infof("deleteWalEntry success! walLastIndex:%d consensus.height:%d", index, consensus.height)
	return nil
}
