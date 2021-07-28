/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"fmt"
	"strings"
	"sync"

	abftpb "chainmaker.org/chainmaker/pb-go/consensus/abft"
)

type bvalDelayedMsg struct {
	sender string
	bval   *abftpb.BValRequest
}

type auxDelayedMsg struct {
	sender string
	aux    *abftpb.AuxRequest
}

type receivedVals struct {
	bba *BBA
	typ string
	set map[string][]bool
}

func newReceivedVals(bba *BBA, typ string) *receivedVals {
	return &receivedVals{
		bba: bba,
		typ: typ,
		set: make(map[string][]bool),
	}
}

func (r *receivedVals) String() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "type: %s, set{", r.typ)
	for k, vals := range r.set {
		fmt.Fprintf(&builder, "%s: [", k)
		for _, v := range vals {
			fmt.Fprintf(&builder, "%v, ", v)
		}
		builder.WriteString("], ")
	}
	builder.WriteString("}")
	return builder.String()
}

func (r *receivedVals) addVal(sender string, val bool) error {
	if vals, ok := r.set[sender]; ok {
		for _, v := range vals {
			if v == val {
				r.bba.logger.Debugf("[%s](%d-%s-%d) BBA %s receivedVals add val: %v multiple times",
					r.bba.nodeID, r.bba.height, r.bba.id, r.bba.epoch, r.typ, val)
				return ErrDuplicatedRBCRequest
			}
		}
	}

	r.set[sender] = append(r.set[sender], val)
	return nil
}

func (r *receivedVals) countVals(val bool) int {
	n := 0

	for _, s := range r.set {
		for _, v := range s {
			if v == val {
				n++
			}
		}
	}
	return n
}

func (r *receivedVals) length() int {
	return len(r.set)
}

type BBA struct {
	*Config
	sync.Mutex
	epoch                         uint32
	binValues                     []bool
	sentBvals                     []bool
	receivedBvals                 *receivedVals
	receivedAux                   *receivedVals
	done                          bool
	estimated, outputted, decided bool
	estimation, output, decision  bool
	bvalBuffer                    []bvalDelayedMsg
	auxBuffer                     []auxDelayedMsg
	messages                      []*abftpb.ABFTMessageReq
}

func NewBBA(cfg *Config) *BBA {
	cfg.logger.Infof("NewBBA config: %s", cfg)
	bba := &BBA{
		Config:    cfg,
		epoch:     0,
		binValues: []bool{},
		sentBvals: []bool{},
	}
	bba.receivedBvals = newReceivedVals(bba, "Bvals")
	bba.receivedAux = newReceivedVals(bba, "Aux")

	return bba
}

func (bba *BBA) Messages() []*abftpb.ABFTMessageReq {
	bba.Lock()
	defer bba.Unlock()

	messages := bba.messages
	bba.messages = []*abftpb.ABFTMessageReq{}
	return messages
}

func (bba *BBA) AcceptInput() bool {
	bba.Lock()
	defer bba.Unlock()

	return bba.epoch == 0 && !bba.estimated
}

func (bba *BBA) Input(val bool) error {
	if !bba.AcceptInput() {
		return nil
	}

	bba.Lock()
	defer bba.Unlock()
	bba.logger.Debugf("[%s](%d-%s-%d) BBA input val: %v", bba.nodeID, bba.height, bba.id, bba.epoch, val)

	bba.estimated = true
	bba.estimation = val
	bba.sentBvals = append(bba.sentBvals, val)
	bba.appendBValRequests(val)
	return nil
}

func (bba *BBA) HandleMessage(sender string, msg *abftpb.BBARequest) error {
	bba.Lock()
	defer bba.Unlock()

	bba.logger.Debugf("[%s](%d-%s-%v) BBA HandleMessage from: %v", bba.nodeID, bba.height, bba.id, bba.epoch, sender)
	if bba.done {
		return nil
	}
	switch m := msg.Message.(type) {
	case *abftpb.BBARequest_Bval:
		return bba.handleBvalRequest(sender, m.Bval)
	case *abftpb.BBARequest_Aux:
		return bba.handleAuxRequest(sender, m.Aux)
	default:
		bba.logger.Errorf("[%s](%d-%s-%v) BBA receive invalid message: %+v, this should not happen",
			bba.nodeID, bba.height, bba.id, bba.epoch, msg)
	}

	return nil
}

func (bba *BBA) Output() (outputted bool, output bool) {
	bba.Lock()
	defer bba.Unlock()

	outputted = bba.outputted
	output = bba.output
	bba.logger.Debugf("[%s](%d-%s-%d) BBA outputted: %v, output: %v",
		bba.nodeID, bba.height, bba.id, bba.epoch, outputted, output)

	bba.outputted = false
	bba.output = false

	return outputted, output
}

func (bba *BBA) handleBvalRequest(sender string, bval *abftpb.BValRequest) error {
	if bval.Epoch < bba.epoch {
		bba.logger.Debugf("[%s](%d-%s-%d) BBA receive outdated Bval from: %v",
			bba.nodeID, bba.height, bba.id, bba.epoch, sender)
		return nil
	}

	if bval.Epoch > bba.epoch {
		bba.bvalBuffer = append(bba.bvalBuffer, bvalDelayedMsg{sender: sender, bval: bval})
		return nil
	}

	val := bval.Value

	if err := bba.receivedBvals.addVal(sender, val); err != nil {
		return err
	}
	bvalCount := bba.receivedBvals.countVals(val)

	bba.logger.Debugf("[%s](%d-%s-%d) BBA receive Bval value: %v, from: %v, bvalCount: %v",
		bba.nodeID, bba.height, bba.id, bba.epoch, bval.Value, sender, bvalCount)

	if bvalCount == bba.faultsNum+1 && !bba.hasSentBval(val) {
		bba.sentBvals = append(bba.sentBvals, val)
		bba.appendBValRequests(val)
	}

	if bvalCount == 2*bba.faultsNum+1 {
		for _, v := range bba.binValues {
			// Exits if the bba.binValues set contains val already
			if v == val {
				return nil
			}
		}
		bba.binValues = append(bba.binValues, val)
		bba.appendAuxRequests(val)
		bba.logger.Debugf("[%s](%d-%s-%d) BBA handleBvalRequest binValues: %v",
			bba.nodeID, bba.height, bba.id, bba.epoch, bba.binValues)
		return nil
	}

	return nil
}

func (bba *BBA) handleAuxRequest(sender string, aux *abftpb.AuxRequest) error {
	if aux.Epoch < bba.epoch {
		bba.logger.Debugf("[%s](%d-%s-%d) BBA receive outdated aux from: %v",
			bba.nodeID, bba.height, bba.id, bba.epoch, sender)
		return nil
	}

	if aux.Epoch > bba.epoch {
		bba.auxBuffer = append(bba.auxBuffer, auxDelayedMsg{sender: sender, aux: aux})
		return nil
	}

	bba.logger.Debugf("[%s](%d-%s-%d) BBA receive Aux value: %v from: %v",
		bba.nodeID, bba.height, bba.id, bba.epoch, aux.Value, sender)
	val := aux.Value
	if err := bba.receivedAux.addVal(sender, val); err != nil {
		return err
	}
	bba.tryOutputAgreement()
	return nil
}

func (bba *BBA) tryOutputAgreement() {
	if len(bba.binValues) == 0 {
		return
	}

	lenOutputs, vals := bba.countOutputs()
	if lenOutputs < bba.nodesNum-bba.faultsNum {
		return
	}

	coin := bba.epoch%2 == 0
	if bba.done || (bba.decided && bba.decision == coin) {
		bba.done = true
		return
	}

	bba.logger.Debugf("[%s](%d-%s-%d) BBA tryOutputAgreement vals: %v, coin: %v",
		bba.nodeID, bba.height, bba.id, bba.epoch, vals, coin)
	bba.increaseEpoch()
	if len(vals) == 1 {
		bba.estimation = vals[0]
		if !bba.decided && vals[0] == coin {
			bba.decided = true
			bba.decision = vals[0]
			bba.outputted = true
			bba.output = vals[0]
		}
	} else {
		bba.estimation = coin
	}

	bba.sentBvals = append(bba.sentBvals, bba.estimation)
	bba.appendBValRequests(bba.estimation)

	bvalBuffer := bba.bvalBuffer
	bba.bvalBuffer = nil
	for _, msg := range bvalBuffer {
		err := bba.handleBvalRequest(msg.sender, msg.bval)
		if err != nil {
			bba.logger.Errorf("[%s](%d-%s-%v) BBA handleBvalRequest error: %v",
				bba.nodeID, bba.height, bba.id, bba.epoch, err)
		}
	}

	auxBuffer := bba.auxBuffer
	bba.auxBuffer = nil
	for _, msg := range auxBuffer {
		err := bba.handleAuxRequest(msg.sender, msg.aux)
		if err != nil {
			bba.logger.Errorf("[%s](%d-%s-%v) BBA handleAuxRequest error: %v",
				bba.nodeID, bba.height, bba.id, bba.epoch, err)
		}
	}
}

func (bba *BBA) increaseEpoch() {
	bba.logger.Debugf("[%s](%d-%s-%d) BBA increaseEpoch", bba.nodeID, bba.height, bba.id, bba.epoch)
	bba.binValues = []bool{}
	bba.sentBvals = []bool{}
	bba.receivedBvals = newReceivedVals(bba, "Bvals")
	bba.receivedAux = newReceivedVals(bba, "Aux")
	bba.epoch++
}

func (bba *BBA) countOutputs() (int, []bool) {
	length := bba.receivedAux.length()

	bba.logger.Debugf("[%s](%d-%s-%d) BBA countOutputs receivedAux: %s, binValues: %v",
		bba.nodeID, bba.height, bba.id, bba.epoch, bba.receivedAux.String(), bba.binValues)
	return length, bba.binValues
}

func (bba *BBA) hasSentBval(val bool) bool {
	for _, ok := range bba.sentBvals {
		if ok == val {
			return true
		}
	}
	return false
}

func (bba *BBA) appendBValRequests(val bool) {
	bvalRequest := &abftpb.BValRequest{
		Epoch: bba.epoch,
		Value: val,
	}
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
		abftMessage := &abftpb.ABFTMessageReq{
			Height: bba.height,
			From:   bba.nodeID,
			To:     n,
			Id:     bba.id,
			Acs:    acsMessage,
		}
		bba.messages = append(bba.messages, abftMessage)

		bba.logger.Debugf("[%s](%d-%s-%d) BBA appendBValRequests value: %v to: %v",
			bba.nodeID, bba.height, bba.id, bba.epoch, val, n)
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
		abftMessage := &abftpb.ABFTMessageReq{
			Height: bba.height,
			From:   bba.nodeID,
			To:     n,
			Id:     bba.id,
			Acs:    acsMessage,
		}
		bba.messages = append(bba.messages, abftMessage)
		bba.logger.Debugf("[%s](%d-%s-%d) BBA appendAuxRequests value: %v to: %v",
			bba.nodeID, bba.height, bba.id, bba.epoch, val, n)
	}
}
