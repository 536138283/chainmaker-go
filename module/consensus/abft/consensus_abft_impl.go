/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker-go/utils"
	"go.uber.org/zap"

	"github.com/gogo/protobuf/proto"

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
	acs    *ACS
	runC   chan struct{}
	closeC chan struct{}
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
	consensus.runC = make(chan struct{})

	go consensus.run(consensus.runC)
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
	consensus.logger.Debugf("[%s] OnMessage receive topic: %s", consensus.Id, message.Topic)

	switch message.Topic {
	case msgbus.ProposedBlock:
		if block, ok := message.Payload.(*common.Block); ok {
			if block.Header.BlockHeight != consensus.height {
				consensus.logger.Warnf("[%s](%v) receive wrong proposed block height: %v", consensus.Id, consensus.height, block.Header.BlockHeight)
				return
			}

			// Add hash and signature to block
			hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
			if err != nil {
				consensus.logger.Errorf("[%s]sign block failed, %s", consensus.Id, err)
			}
			block.Header.BlockHash = hash[:]
			block.Header.Signature = sig

			data := mustMarshal(block)
			// acs := consensus.getACS(block.Header.BlockHeight)
			if err = consensus.acs.InputRBC(data); err != nil {
				consensus.logger.Errorf("[%s] input RBC error: %v", consensus.Id, err)
			}
		}
	case msgbus.VerifyResult:
		if verifyResult, ok := message.Payload.(*consensuspb.VerifyResult); ok {
			if verifyResult.VerifiedBlock.Header.BlockHeight != consensus.height {
				consensus.logger.Warnf("[%s](%v) receive wrong verifyResult height: %v", consensus.Id, consensus.height, verifyResult.VerifiedBlock.Header.BlockHeight)
				return
			}

			consensus.logger.Debugf("[%s] verify result code: %s, msg: %s", consensus.Id, verifyResult.Code, verifyResult.Msg)
			data := mustMarshal(verifyResult.VerifiedBlock)
			// acs := consensus.getACS(verifyResult.VerifiedBlock.Header.BlockHeight)
			consensus.acs.InputBBA(data)
		}
	case msgbus.BlockInfo:
		if blockInfo, ok := message.Payload.(*common.BlockInfo); ok {
			if blockInfo == nil || blockInfo.Block == nil {
				consensus.logger.Errorf("receive message failed, error message BlockInfo = nil")
				return
			}
			close(consensus.runC)
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
			consensus.runC = make(chan struct{})

			go consensus.run(consensus.runC)
			consensus.sendPackageSingal(consensus.height)
		}
	case msgbus.RecvConsensusMsg:
		if msg, ok := message.Payload.(*netpb.NetMsg); ok {
			abftMsg := new(abftpb.ABFTMessage)
			mustUnmarshal(msg.Payload, abftMsg)
			consensus.handleMessage(abftMsg)
		} else {
			panic(fmt.Errorf("receive message failed, error message type"))
		}
	}
}

// OnQuit implements the OnQuit interface of msgbus.Subscriber
func (consensus *ConsensusABFTImpl) OnQuit() {
	consensus.logger.Debugf("[%s] OnQuit", consensus.Id)
}

func (consensus *ConsensusABFTImpl) sendPackageSingal(height int64) {
	consensus.logger.Debugf("[%s] sendPackageSingal height: %d", consensus.Id, height)
	signal := &abftpb.PackagedSignal{BlockHeight: height}
	consensus.msgbus.PublishSafe(msgbus.PackageSignal, signal)
}

// func (consensus *ConsensusABFTImpl) getACS(height int64) *ACS {
//   if acs, ok := consensus.acsInstances[height]; ok {
//     return acs
//   }

//   nodeList, _ := GetNodeListFromConfig(consensus.chainConf.ChainConfig())
//   cfg := &Config{
//     logger: consensus.logger,
//     height: height,
//     id:     consensus.Id,
//     nodeID: consensus.Id,
//     nodes:  nodeList,
//   }
//   cfg.fillWithDefaults()
//   acs := NewACS(*cfg)
//   consensus.acsInstances[height] = acs
//   return acs
// }

func (consensus *ConsensusABFTImpl) run(closeC chan struct{}) {
	for {
		// acs := consensus.getACS(consensus.height)
		select {
		case <-closeC:
			return
		case msg := <-consensus.acs.MessageCh():
			consensus.handleMessage(msg)
		case msg := <-consensus.acs.RbcOutputCh():
			block := &common.Block{}
			mustUnmarshal(msg, block)
			consensus.msgbus.PublishSafe(msgbus.VerifyBlock, block)
			consensus.logger.Debugf("[%s] verify msg.len: %v, block: %v", consensus.Id, len(msg), block)
		}
	}
}

func (consensus *ConsensusABFTImpl) handleMessage(msg *abftpb.ABFTMessage) {
	consensus.logger.Debugf("[%s] handleMessage from: %v, to: %v, Id: %v", consensus.Id, msg.From, msg.To, msg.Id)
	if msg.To != consensus.Id {
		consensus.acs.HandleMessage(msg.From, msg.Id, msg.Acs)
		netMsg := &netpb.NetMsg{
			Payload: mustMarshal(msg),
			Type:    netpb.NetMsg_CONSENSUS_MSG,
			To:      msg.To,
		}
		consensus.publishToMsgbus(netMsg)
		return

	} else {
		// acs := consensus.getACS(consensus.height)
		consensus.acs.HandleMessage(msg.From, msg.Id, msg.Acs)
	}

	output := consensus.acs.Output()
	if output == nil || len(output) == 0 {
		return
	}

	for _, data := range output {
		block := &common.Block{}
		mustUnmarshal(data, block)
		consensus.logger.Debugf("[%s] commit block: %v", consensus.Id, block)
		consensus.msgbus.PublishSafe(msgbus.CommitedTxBatchs, &abftpb.TxBatchAfterABA{
			BlockHeight: consensus.height,
			TxBatchHash: [][]byte{block.Header.BlockHash},
		})
		break
	}
}

func (consensus *ConsensusABFTImpl) publishToMsgbus(msg *netpb.NetMsg) {
	consensus.logger.Debugf("[%s] publishToMsgbus size: %d", consensus.Id, proto.Size(msg))
	consensus.msgbus.Publish(msgbus.SendConsensusMsg, msg)
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
