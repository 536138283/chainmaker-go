/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"sync"

	abftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/abft"
)

type BBA struct {
	*Config
	sync.Mutex
	epoch         uint32
	binValues     []bool
	sentBvals     []bool
	receivedBvals map[string]bool
	receivedAux   map[string]bool
	done          bool
	// inputted, ouputted          bool
	// estimated, output           bool
	output, estimated, decision interface{}
	messages                    []*abftpb.ABFTMessage
}

func NewBBA(cfg *Config) *BBA {
	cfg.logger.Infof("NewBBA config: %s", cfg)
	bba := &BBA{
		Config:        cfg,
		epoch:         0,
		binValues:     []bool{},
		sentBvals:     []bool{},
		receivedBvals: make(map[string]bool),
		receivedAux:   make(map[string]bool),
	}

	return bba
}

func (bba *BBA) Messages() []*abftpb.ABFTMessage {
	bba.Lock()
	defer bba.Unlock()

	messages := bba.messages
	bba.messages = []*abftpb.ABFTMessage{}
	return messages
}

func (bba *BBA) AcceptInput() bool {
	bba.Lock()
	defer bba.Unlock()

	return bba.epoch == 0 && bba.estimated == nil
	// return bba.epoch == 0 && !bba.inputted
}

func (bba *BBA) Input(val bool) error {
	if !bba.AcceptInput() {
		return nil
	}

	bba.Lock()
	defer bba.Unlock()
	bba.logger.Debugf("[%s](%d-%s) BBA input val: %v", bba.nodeID, bba.height, bba.id, val)
	// bba.inputted = true
	bba.estimated = val
	bba.sentBvals = append(bba.sentBvals, val)
	bba.appendBValRequests(val)
	return nil
}

func (bba *BBA) appendBValRequests(val bool) {
	bvalRequest := &abftpb.BValRequest{
		Epoch: bba.epoch,
		Value: val,
	}
	bba.logger.Debugf("[%s](%d-%s) BBA appendBValRequests: {epoch: %v, value: %v}",
		bba.nodeID, bba.height, bba.id, bvalRequest.Epoch, bvalRequest.Value)
	bbaRequest := &abftpb.BBARequest{
		Message: &abftpb.BBARequest_Bval{
			Bval: bvalRequest,
		},
	}
	acsMessage := &abftpb.ACSMessage{
		Message: &abftpb.ACSMessage_Bba{
			Bba: bbaRequest,
		},
	}

	for _, n := range bba.nodes {
		if n != bba.nodeID {
			abftMessage := &abftpb.ABFTMessage{
				Height: bba.height,
				From:   bba.nodeID,
				To:     n,
				Id:     bba.id,
				Acs:    acsMessage,
			}
			bba.messages = append(bba.messages, abftMessage)
		}
	}
}

func (bba *BBA) appendAuxRequests(val bool) {
	auxRequest := &abftpb.AuxRequest{
		Epoch: bba.epoch,
		Value: val,
	}
	bbaRequest := &abftpb.BBARequest{
		Message: &abftpb.BBARequest_Aux{
			Aux: auxRequest,
		},
	}
	acsMessage := &abftpb.ACSMessage{
		Message: &abftpb.ACSMessage_Bba{
			Bba: bbaRequest,
		},
	}

	for _, n := range bba.nodes {
		abftMessage := &abftpb.ABFTMessage{
			Height: bba.height,
			From:   bba.nodeID,
			To:     n,
			Id:     bba.id,
			Acs:    acsMessage,
		}
		bba.messages = append(bba.messages, abftMessage)
	}
}

func (bba *BBA) HandleMessage(sender string, msg *abftpb.BBARequest) error {
	bba.Lock()
	defer bba.Unlock()

	bba.logger.Debugf("[%s](%d-%s) BBA HandleMessage from: %v", bba.nodeID, bba.height, bba.id, sender)
	if bba.done {
		return nil
	}
	switch m := msg.Message.(type) {
	case *abftpb.BBARequest_Bval:
		return bba.handleBvalRequest(sender, m.Bval)
	case *abftpb.BBARequest_Aux:
		return bba.handleAuxRequest(sender, m.Aux)
	default:
		bba.logger.Errorf("[%s](%d) BBA receive invalid message: %+v, this should not happen", bba.nodeID, bba.height, msg)
	}

	return nil
}

func (bba *BBA) Output() interface{} {
	bba.Lock()
	defer bba.Unlock()

	if bba.output != nil {
		output := bba.output
		bba.output = nil
		bba.logger.Debugf("[%s](%d-%s) BBA output: %v", bba.nodeID, bba.height, bba.id, output)
		return output
	}

	return nil
}

func (bba *BBA) handleBvalRequest(sender string, bval *abftpb.BValRequest) error {
	if bval.Epoch < bba.epoch {
		bba.logger.Debugf("[%s](%d) BBA receive outdated Bval from: %v", bba.nodeID, bba.height, sender)
		return nil
	}

	val := bval.Value
	bba.receivedBvals[sender] = val
	bvalCount := bba.countBvals(val)

	bba.logger.Debugf("[%s](%d-%s) BBA receive Bval: {epoch: %v, value: %v}, from: %v, bvalCount: %v",
		bba.nodeID, bba.height, bba.id, bval.Epoch, bval.Value, sender, bvalCount)
	if bvalCount == 2*bba.faultsNum+1 {
		bba.binValues = append(bba.binValues, val)
		if len(bba.binValues) == 1 {
			bba.appendAuxRequests(val)
		}
		bba.logger.Debugf("[%s](%d-%s) BBA handleBvalRequest binValues: %v",
			bba.nodeID, bba.height, bba.id, bba.binValues)
		return nil
	}

	if bvalCount == bba.faultsNum+1 && !bba.hasSentBval(val) {
		bba.sentBvals = append(bba.sentBvals, val)
		bba.appendBValRequests(val)
	}
	return nil
}

func (bba *BBA) handleAuxRequest(sender string, aux *abftpb.AuxRequest) error {
	if aux.Epoch < bba.epoch {
		bba.logger.Debugf("[%s](%d) BBA receive outdated aux from: %v", bba.nodeID, bba.height, sender)
		return nil
	}

	bba.logger.Debugf("[%s](%d-%s) BBA receive Aux: {epoch: %v, value: %v} from: %v",
		bba.nodeID, bba.height, bba.id, aux.Epoch, aux.Value, sender)
	val := aux.Value
	bba.receivedAux[sender] = val
	bba.tryOutputAgreement()
	return nil
}

func (bba *BBA) tryOutputAgreement() {
	if len(bba.binValues) == 0 {
		return
	}

	lenOutputs, values := bba.countOutputs()
	if lenOutputs < bba.nodesNum-bba.faultsNum {
		return
	}

	coin := bba.epoch%2 == 0
	if bba.done || (bba.decision != nil && bba.decision.(bool) == coin) {
		// if bba.done || (!bba.decided && bba.decision == coin) {
		bba.done = true
		return
	}

	bba.logger.Debugf("[%s](%d-%s) BBA len(values): %v, coin: %v, values: %v",
		bba.nodeID, bba.height, bba.id, len(values), coin, values)
	bba.increaseEpoch()
	if len(values) == 1 {
		// bba.estimated = values[0]
		// if !bba.decision && values[0] == coin {
		//   bba.output = values[0]
		//   bba.decision = values[0]
		// }

		bba.estimated = values[0]
		if bba.decision == nil && values[0] == coin {
			// if !bba.decision && values[0] == coin {
			bba.output = values[0]
			bba.decision = values[0]
		}
	} else {
		bba.estimated = coin
	}

	// bba.sentBvals = append(bba.sentBvals, bba.estimated)
	// bba.appendBValRequests(bba.estimated)

	estimated := bba.estimated.(bool)
	bba.sentBvals = append(bba.sentBvals, estimated)
	bba.appendBValRequests(estimated)
}

func (bba *BBA) increaseEpoch() {
	bba.binValues = []bool{}
	bba.sentBvals = []bool{}
	bba.receivedBvals = make(map[string]bool)
	bba.receivedAux = make(map[string]bool)
	bba.epoch++
}

func (bba *BBA) countBvals(val bool) int {
	n := 0

	for _, v := range bba.receivedBvals {
		if v == val {
			n++
		}
	}
	return n
}

func (bba *BBA) countOutputs() (int, []bool) {
	m := map[bool]string{}
	for s, val := range bba.receivedAux {
		m[val] = s
	}
	vals := []bool{}
	for _, val := range bba.binValues {
		if _, ok := m[val]; ok {
			vals = append(vals, val)
		}
	}

	bba.logger.Debugf("[%s](%d-%s) BBA countOutputs receivedAux: %v, binValues: %v",
		bba.nodeID, bba.height, bba.id, bba.receivedAux, bba.binValues)
	return len(bba.receivedAux), vals
}

func (bba *BBA) hasSentBval(val bool) bool {
	for _, ok := range bba.sentBvals {
		if ok == val {
			return true
		}
	}
	return false
}
