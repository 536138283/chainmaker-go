/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"encoding/hex"
	"sync"

	"chainmaker.org/chainmaker-go/utils"
	"go.uber.org/zap"

	"github.com/gogo/protobuf/proto"
	"github.com/thoas/go-funk"

	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	abftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	netpb "chainmaker.org/chainmaker-go/pb/protogo/net"
	"chainmaker.org/chainmaker-go/protocol"

	"chainmaker.org/chainmaker-go/logger"
)

var clog *zap.SugaredLogger = zap.S()

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
	logger    *logger.CMLogger
	chainID   string
	Id        string
	msgbus    msgbus.MessageBus
	chainConf protocol.ChainConf
	singer    protocol.SigningMember
	height    int64
	// acsInstances map[int64]*ACS
	acs       *ACS
	msgBuffer []*abftpb.ABFTMessage
	closeC    chan struct{}
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
	consensus.msgBuffer = make([]*abftpb.ABFTMessage, 0)

	height, err := config.LedgerCache.CurrentHeight()
	if err != nil {
		return nil, err
	}
	consensus.height = height + 1

	return consensus, nil
}

// Start implements the Stop method of ConsensusEngine interface
// and starts the abft instance.
func (consensus *ConsensusABFTImpl) Start() error {
	consensus.logger.Infof("[%s] started", consensus.Id)
	consensus.msgbus.Register(msgbus.ProposedBlock, consensus)
	consensus.msgbus.Register(msgbus.VerifyResult, consensus)
	consensus.msgbus.Register(msgbus.BlockInfo, consensus)
	consensus.msgbus.Register(msgbus.RecvConsensusMsg, consensus)

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

	consensus.sendPackageSingal(consensus.height)

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
	consensus.Lock()
	defer consensus.Unlock()

	consensus.logger.Debugf("[%s](%v) OnMessage receive topic: %s", consensus.Id, consensus.height, message.Topic)

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

	buffer := make([]*abftpb.ABFTMessage, 0)
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
	consensus.handleMessage(abftMsg)
	consensus.processEvent()
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

func (consensus *ConsensusABFTImpl) handleMessage(msg *abftpb.ABFTMessage) {
	consensus.logger.Debugf("[%s](%d) handleMessage height: %v, from: %v, to: %v, Id: %v",
		consensus.Id, consensus.height, msg.Height, msg.From, msg.To, msg.Id)
	if msg.Height < consensus.height {
		return
	} else if msg.Height > consensus.height {
		consensus.msgBuffer = append(consensus.msgBuffer, msg)
		return
	}
	if msg.To != consensus.Id {
		// consensus.acs.HandleMessage(msg.From, msg.Id, msg.Acs)
		netMsg := &netpb.NetMsg{
			Payload: mustMarshal(msg),
			Type:    netpb.NetMsg_CONSENSUS_MSG,
			To:      msg.To,
		}
		consensus.publishToMsgbus(netMsg)
		return
	} else {
		err := consensus.acs.HandleMessage(msg.From, msg.Id, msg.Acs)
		if err != nil {
			consensus.logger.Errorf("[%s] handleMessage to: %s, error: %v", consensus.Id, msg.To, err)
		}
	}
}

func (consensus *ConsensusABFTImpl) sendPackageSingal(height int64) {
	consensus.logger.Debugf("[%s] sendPackageSingal height: %d", consensus.Id, height)
	signal := &abftpb.PackagedSignal{BlockHeight: height}
	consensus.msgbus.PublishSafe(msgbus.PackageSignal, signal)
}

func (consensus *ConsensusABFTImpl) publishToMsgbus(msg *netpb.NetMsg) {
	consensus.logger.Debugf("[%s] publishToMsgbus size: %d", consensus.Id, proto.Size(msg))
	consensus.msgbus.PublishSafe(msgbus.SendConsensusMsg, msg)
}

func GetNodeListFromConfig(chainConfig *config.ChainConfig) (validators []string, err error) {
	nodes := chainConfig.Consensus.Nodes
	for _, node := range nodes {
		for _, addr := range node.Address {
			uid, err := helper.GetNodeUidFromAddr(addr)
			if err != nil {
				return nil, err
			}
			validators = append(validators, uid)
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
