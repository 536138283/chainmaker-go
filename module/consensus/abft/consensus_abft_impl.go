/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/utils"
	"go.uber.org/zap"

	"github.com/gogo/protobuf/proto"

	"chainmaker.org/chainmaker-go/common/helper"
	"chainmaker.org/chainmaker-go/common/msgbus"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	"chainmaker.org/chainmaker-go/pb/protogo/config"
	consensuspb "chainmaker.org/chainmaker-go/pb/protogo/consensus"
	abftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
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
	acs       *ACS
	closeC    chan struct{}
}

// ConsensusABFTImplConfig contains initialization config for ConsensusABFTImpl
type ConsensusABFTImplConfig struct {
	ChainID   string
	Id        string
	MsgBus    msgbus.MessageBus
	ChainConf protocol.ChainConf
	Singer    protocol.SigningMember
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

	nodeList, err := GetNodeListFromConfig(consensus.chainConf.ChainConfig())
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		logger: logger,
		height: 0,
		nodeID: consensus.Id,
		nodes:  nodeList,
	}
	cfg.fillWithDefault()
	consensus.acs = NewACS(*cfg)

	return consensus, nil
}

// Start implements the Stop method of ConsensusEngine interface
// and starts the abft instance.
func (consensus *ConsensusABFTImpl) Start() error {
	consensus.logger.Infof("[%s] started", consensus.Id)
	consensus.msgbus.Register(msgbus.ProposedBlock, consensus)
	consensus.msgbus.Register(msgbus.VerifyResult, consensus)

	go func() {
		time.Sleep(3 * time.Second)
		consensus.sendPackageSingal(1)
	}()

	go consensus.run()

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

			// Add hash and signature to block
			hash, sig, err := utils.SignBlock(consensus.chainConf.ChainConfig().Crypto.Hash, consensus.singer, block)
			if err != nil {
				consensus.logger.Errorf("[%s]sign block failed, %s", consensus.Id, err)
			}
			block.Header.BlockHash = hash[:]
			block.Header.Signature = sig

			data := mustMarshal(block)
			consensus.acs.InputRBC(data)
		}
	case msgbus.VerifyResult:
		if verifyResult, ok := message.Payload.(*consensuspb.VerifyResult); ok {
			consensus.logger.Debugf("[%s] verify result: %s", consensus.Id, verifyResult.Code)
			data := mustMarshal(verifyResult.VerifiedBlock)
			consensus.acs.InputBBA(data)
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

func (consensus *ConsensusABFTImpl) run() {
	for {
		select {
		case msg := <-consensus.acs.MessageCh():
			consensus.handleMessage(msg)
		case msg := <-consensus.acs.RbcOutputCh():
			consensus.logger.Debugf("[%s] verify", consensus.Id)
			block := &common.Block{}
			mustUnmarshal(msg, block)
			consensus.msgbus.PublishSafe(msgbus.VerifyBlock, block)
		}
	}
}

func (consensus *ConsensusABFTImpl) handleMessage(msg *abftpb.ABFTMessage) {
	if msg.To == consensus.Id {
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
			BlockHeight: 1,
			TxBatchHash: [][]byte{block.Header.BlockHash},
		})
		break
	}
}
