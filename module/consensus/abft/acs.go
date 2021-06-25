/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"sync"

	abftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
	"github.com/thoas/go-funk"
)

type Event struct {
	rbcOutputs [][]byte
	messages   []*abftpb.ABFTMessageReq
	outputs    [][]byte
}

type rbcOutput struct {
	id     string
	output []byte
}

type ACS struct {
	*Config
	sync.Mutex
	rbcInstances   map[string]*RBC
	bbaInstances   map[string]*BBA
	rbcResults     map[string][]byte
	bbaResults     map[string]bool
	rbcOutputCache map[string]*rbcOutput
	rbcOutputs     [][]byte
	outputs        [][]byte
	messages       []*abftpb.ABFTMessageReq
	decided        bool
}

func NewACS(cfg *Config) *ACS {
	cfg.logger.Infof("NewACS config: %s", cfg)
	acs := &ACS{
		Config:         cfg,
		rbcInstances:   make(map[string]*RBC),
		bbaInstances:   make(map[string]*BBA),
		rbcResults:     make(map[string][]byte),
		bbaResults:     make(map[string]bool),
		rbcOutputCache: make(map[string]*rbcOutput),
	}

	for _, id := range cfg.nodes {
		rbcCfg := cfg.clone()
		rbcCfg.id = id
		acs.rbcInstances[id] = NewRBC(rbcCfg)

		bbaCfg := cfg.clone()
		bbaCfg.id = id
		acs.bbaInstances[id] = NewBBA(bbaCfg)
	}
	return acs
}

func (acs *ACS) InputRBC(val []byte) error {
	acs.Lock()
	defer acs.Unlock()
	acs.logger.Debugf("[%s](%d-%s) ACS input RBC len: %v", acs.nodeID, acs.height, acs.id, len(val))
	rbc, ok := acs.rbcInstances[acs.nodeID]
	if !ok {
		return fmt.Errorf("[%s](%d) cannot find rbc instance: %s", acs.nodeID, acs.height, acs.nodeID)
	}

	if err := rbc.Input(val); err != nil {
		return err
	}

	acs.appendMessages(rbc.Messages())

	if output := rbc.Output(); output != nil {
		acs.handleRBCOutput(acs.nodeID, output)
		// acs.rbcResults[acs.nodeID] = output
		// acs.processBBA(acs.nodeID, func(bba *BBA) error {
		//   if bba.AcceptInput() {
		//     return bba.Input(true)
		//   }

		//   return nil
		// })
	}

	return nil
}

func (acs *ACS) InputBBA(output []byte) error {
	acs.Lock()
	defer acs.Unlock()

	hash := md5.Sum(output)
	hashStr := base64.StdEncoding.EncodeToString(hash[:])
	rbcOutput, ok := acs.rbcOutputCache[hashStr]
	if !ok {
		return fmt.Errorf("[%s](%d-%s) ACS receive invalid BBA: %v, this should not happen",
			acs.nodeID, acs.height, acs.nodeID, hashStr)
	}

	acs.logger.Debugf("[%s](%d) ACS InputBBA id: %v", acs.nodeID, acs.height, rbcOutput.id)
	acs.rbcResults[rbcOutput.id] = rbcOutput.output
	if acs.tryComplete() {
		return nil
	}
	return acs.processBBA(rbcOutput.id, func(bba *BBA) error {
		if bba.AcceptInput() {
			return bba.Input(true)
		}

		return nil
	})
}

func (acs *ACS) HandleMessage(sender string, id string, acsMessage *abftpb.ACSMessage) error {
	acs.Lock()
	defer acs.Unlock()

	switch m := acsMessage.Message.(type) {
	case *abftpb.ACSMessage_Rbc:
		return acs.handleRBC(sender, id, m.Rbc)
	case *abftpb.ACSMessage_Bba:
		return acs.handleBBA(sender, id, m.Bba)
	default:
		acs.logger.Errorf("[%s](%d-%s) ACS receive invalid message: %+v, this should not happen", acs.nodeID, acs.height, id, m)
	}
	return nil
}

func (acs *ACS) Event() *Event {
	acs.Lock()
	defer acs.Unlock()

	event := &Event{}
	if len(acs.rbcOutputs) != 0 {
		event.rbcOutputs = acs.rbcOutputs
		acs.rbcOutputs = nil
	}
	if len(acs.messages) != 0 {
		event.messages = acs.messages
		acs.messages = nil
	}
	if len(acs.outputs) != 0 {
		event.outputs = acs.outputs
		acs.outputs = nil
	}
	return event
}

func (acs *ACS) handleRBC(sender string, id string, rbcMessage *abftpb.RBCRequest) error {
	return acs.processRBC(id, func(rbc *RBC) error {
		return rbc.HandleMessage(sender, rbcMessage)
	})
}

func (acs *ACS) handleBBA(sender string, id string, bbaMessage *abftpb.BBARequest) error {
	return acs.processBBA(id, func(bba *BBA) error {
		return bba.HandleMessage(sender, bbaMessage)
	})
}

func (acs *ACS) appendMessages(msgs []*abftpb.ABFTMessageReq) {
	acs.messages = append(acs.messages, msgs...)
}

func (acs *ACS) processRBC(id string, f func(rbc *RBC) error) error {
	rbc, ok := acs.rbcInstances[id]
	if !ok {
		return fmt.Errorf("[%s](%d) cannot find RBC instance: %s", acs.nodeID, acs.height, id)
	}

	if err := f(rbc); err != nil {
		return err
	}

	acs.logger.Debugf("[%s](%d) ACS processRBC id: %v", acs.nodeID, acs.height, id)
	acs.appendMessages(rbc.Messages())

	if output := rbc.Output(); output != nil {
		acs.handleRBCOutput(id, output)
	}

	return nil
}

func (acs *ACS) handleRBCOutput(id string, output []byte) {
	data := &rbcOutput{
		id:     id,
		output: output,
	}
	hash := md5.Sum(output)
	hashStr := base64.StdEncoding.EncodeToString(hash[:])
	acs.rbcOutputCache[hashStr] = data
	acs.rbcOutputs = append(acs.rbcOutputs, output)
}

func (acs *ACS) processBBA(id string, f func(bba *BBA) error) error {
	bba, ok := acs.bbaInstances[id]
	if !ok {
		return fmt.Errorf("[%s](%d) cannot find ABA instance: %s", acs.nodeID, acs.height, id)
	}

	if bba.done {
		return nil
	}

	if err := f(bba); err != nil {
		return err
	}

	acs.logger.Debugf("[%s](%d) ACS processBBA id: %v", acs.nodeID, acs.height, id)
	acs.appendMessages(bba.Messages())

	if outputted, output := bba.Output(); outputted {
		if _, ok := acs.bbaResults[id]; ok {
			return fmt.Errorf("[%s](%d-%s) BBA outputs multiple result: %v", acs.nodeID, acs.height, id, acs.bbaResults[id])
		}

		acs.bbaResults[id] = output
		if output && acs.countFinishedBBA() == acs.nodesNum-acs.faultsNum {
			for id, bba := range acs.bbaInstances {
				if bba.AcceptInput() {
					if err := bba.Input(false); err != nil {
						return err
					}

					acs.appendMessages(bba.Messages())
					if outputted, output := bba.Output(); outputted {
						acs.bbaResults[id] = output
					}
				}
			}
		}
		acs.tryComplete()
	}

	return nil
}

func (acs *ACS) countFinishedBBA() int {
	n := 0
	for _, ok := range acs.bbaResults {
		if ok {
			n++
		}
	}
	return n
}

func (acs *ACS) tryComplete() bool {
	acs.logger.Debugf("[%s](%d) ACS tryComplete decided: %v, finishedBBACount: %v, bbaResults.len: %v",
		acs.nodeID, acs.height, acs.decided, acs.countFinishedBBA(), len(acs.bbaResults))
	if acs.decided ||
		acs.countFinishedBBA() < acs.nodesNum-acs.faultsNum ||
		len(acs.bbaResults) < acs.nodesNum {
		return false
	}

	bbaTrueDecision := []string{}
	for id, ok := range acs.bbaResults {
		if ok {
			bbaTrueDecision = append(bbaTrueDecision, id)
		}
	}

	outputs := make([][]byte, 0)
	for _, id := range bbaTrueDecision {
		val, ok := acs.rbcResults[id]
		if !ok {
			// Wait for RBC to complete
			acs.logger.Infof("[%s](%d) ACS tryComplete wait for rbcResults id: %v, rbcResults: %v",
				acs.nodeID, acs.height, id, funk.Keys(acs.rbcResults))
			return false
		}
		outputs = append(outputs, val)
	}

	acs.outputs = outputs
	acs.decided = true
	acs.logger.Debugf("[%s](%d) ACS complete output.len: %v", acs.nodeID, acs.height, len(acs.outputs))
	return true
}
