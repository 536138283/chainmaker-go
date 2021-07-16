/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package abft

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"

	"chainmaker.org/chainmaker/pb-go/consensus/abft"
	abftpb "chainmaker.org/chainmaker/pb-go/consensus/abft"
	"github.com/NebulousLabs/merkletree"
	"github.com/klauspost/reedsolomon"
)

// RBC represents an instance of "Reliable Broadcast".
type RBC struct {
	*Config
	sync.Mutex
	enc                                reedsolomon.Encoder
	receivedEchos                      map[string]*abft.EchoRequest
	receivedReadys                     map[string][]byte
	messages                           []*abftpb.ABFTMessageReq
	echoSent, readySent, outputDecoded bool
	output                             []byte

	// channels for state transfer
	closeCh chan struct{}
}

// NewRBC returns an instance of RBC for reliable broadcast.
func NewRBC(cfg *Config) *RBC {
	cfg.logger.Infof("NewRBC config: %s", cfg)
	enc, err := reedsolomon.New(cfg.faultsNum+1, cfg.nodesNum-cfg.faultsNum-1)
	if err != nil {
		cfg.logger.Panicf("[%s] new reedsolomon error: %v", cfg.nodeID, err)
	}

	rbc := &RBC{
		Config:         cfg,
		enc:            enc,
		receivedEchos:  make(map[string]*abft.EchoRequest),
		receivedReadys: make(map[string][]byte),
		closeCh:        make(chan struct{}),
		messages:       []*abftpb.ABFTMessageReq{},
	}
	return rbc
}

// Input inputs data to the rbc instance.
func (rbc *RBC) Input(data []byte) error {
	rbc.logger.Debugf("[%s](%d-%s) RBC input data.len: %v", rbc.nodeID, rbc.height, rbc.id, len(data))
	rbc.Lock()
	defer rbc.Unlock()
	shards, err := rbc.makeShards(data)
	if err != nil {
		return err
	}

	for i := 0; i < len(shards); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(shards); j++ {
			tree.Push(shards[j])
		}

		root, proof, proofIndex, leaves := tree.Prove()
		proofRequest := &abftpb.ProofRequest{
			RootHash: root,
			Proof:    proof,
			Index:    proofIndex,
			Leaves:   leaves,
			Length:   int32(len(data)),
		}
		rbc.appendProofRequests(rbc.nodes[i], proofRequest)
	}

	return nil
}

func (rbc *RBC) HandleMessage(sender string, msg *abftpb.RBCRequest) error {
	rbc.Lock()
	defer rbc.Unlock()

	switch m := msg.Message.(type) {
	case *abft.RBCRequest_ProofRequest:
		return rbc.handleProofRequest(sender, m.ProofRequest)
	case *abft.RBCRequest_EchoRequest:
		return rbc.handleEchoRequest(sender, m.EchoRequest)
	case *abft.RBCRequest_ReadyRequest:
		return rbc.handleReadyRequest(sender, m.ReadyRequest)
	default:
		rbc.logger.Errorf("[%s](%d) receive invalid message: %+v, this should not happen", rbc.nodeID, rbc.height, msg)
	}
	return nil
}

func (rbc *RBC) Messages() []*abftpb.ABFTMessageReq {
	rbc.Lock()
	defer rbc.Unlock()

	msgs := rbc.messages
	rbc.messages = []*abftpb.ABFTMessageReq{}
	return msgs
}

func (rbc *RBC) Output() []byte {
	rbc.Lock()
	defer rbc.Unlock()

	if rbc.output != nil {
		output := rbc.output
		rbc.output = nil
		return output
	}

	return nil
}

func (rbc *RBC) stop() {
	close(rbc.closeCh)
}

// makeShards splits the data to shards with reedsolomon encoder.
func (rbc *RBC) makeShards(data []byte) ([][]byte, error) {
	shards, err := rbc.enc.Split(data)
	if err != nil {
		return nil, err
	}

	if err := rbc.enc.Encode(shards); err != nil {
		return nil, err
	}

	return shards, nil
}

func (rbc *RBC) verifyProofRequest(proof *abftpb.ProofRequest) bool {
	result := merkletree.VerifyProof(
		sha256.New(),
		proof.RootHash,
		proof.Proof,
		proof.Index,
		proof.Leaves,
	)

	return result
}

func (rbc *RBC) handleProofRequest(sender string, msg *abftpb.ProofRequest) error {
	if sender != rbc.id {
		return fmt.Errorf("[%s](%d-%s) RBC receive proof request from error node: %s",
			rbc.nodeID, rbc.height, rbc.id, sender)
	}

	if rbc.echoSent {
		return fmt.Errorf("[%s](%d-%s) RBC receive proof: %x from: %v multiple times",
			rbc.nodeID, rbc.height, rbc.id, msg.RootHash, sender)
	}

	if !rbc.verifyProofRequest(msg) {
		return fmt.Errorf("[%s](%d-%s) RBC receive invalid proof request from %s",
			rbc.nodeID, rbc.height, rbc.id, sender)
	}

	rbc.logger.Debugf("[%s](%d-%s) RBC receive proof: %x from: %v", rbc.nodeID, rbc.height, rbc.id, msg.RootHash, sender)
	rbc.echoSent = true
	echo := &abftpb.EchoRequest{ProofRequest: msg}
	rbc.appendEchoRequests(echo)

	return nil
}

func (rbc *RBC) handleEchoRequest(sender string, msg *abftpb.EchoRequest) error {
	if _, ok := rbc.receivedEchos[sender]; ok {
		return fmt.Errorf("[%s](%d) receive multiple echos from: %s", rbc.nodeID, rbc.height, sender)
	}

	if !rbc.verifyProofRequest(msg.ProofRequest) {
		return fmt.Errorf("[%s] receive invalid proof request from %s", rbc.nodeID, sender)
	}

	rbc.receivedEchos[sender] = msg
	rbc.logger.Debugf("[%s](%d-%s) RBC receive echo: %x from: %v", rbc.nodeID, rbc.height, rbc.id, msg.ProofRequest.RootHash, sender)
	if rbc.readySent || rbc.countEchos(msg.ProofRequest.RootHash) < rbc.nodesNum-rbc.faultsNum {
		return rbc.tryDecodeValue(msg.ProofRequest.RootHash)
	}

	rbc.readySent = true
	ready := &abftpb.ReadyRequest{RootHash: msg.ProofRequest.RootHash}
	rbc.appendReadyRequests(ready)
	return nil
}

func (rbc *RBC) handleReadyRequest(sender string, msg *abftpb.ReadyRequest) error {
	if _, ok := rbc.receivedReadys[sender]; ok {
		return fmt.Errorf("[%s](%d) receive multiple readys from: %s", rbc.nodeID, rbc.height, sender)
	}
	rbc.logger.Debugf("[%s](%d-%s) RBC receive ready: %x from: %v", rbc.nodeID, rbc.height, rbc.id, msg.RootHash, sender)

	rbc.receivedReadys[sender] = msg.RootHash
	if rbc.countReady(msg.RootHash) == rbc.faultsNum+1 && !rbc.readySent {
		rbc.readySent = true
		ready := &abftpb.ReadyRequest{RootHash: msg.RootHash}
		rbc.appendReadyRequests(ready)
	}
	return rbc.tryDecodeValue(msg.RootHash)
}

func (rbc *RBC) countEchos(hash []byte) int {
	n := 0

	for _, e := range rbc.receivedEchos {
		if bytes.Equal(hash, e.ProofRequest.RootHash) {
			n++
		}
	}
	return n
}

func (rbc *RBC) countReady(hash []byte) int {
	n := 0

	for _, r := range rbc.receivedReadys {
		if bytes.Equal(hash, r) {
			n++
		}
	}
	return n
}

func (rbc *RBC) tryDecodeValue(hash []byte) error {
	rbc.logger.Debugf("[%s](%d-%s) RBC tryDecodeValue outputDecoded: %v, countEchos: %v, countReady: %v",
		rbc.nodeID, rbc.height, rbc.id, rbc.outputDecoded, rbc.countEchos(hash), rbc.countReady(hash))
	if rbc.outputDecoded ||
		rbc.countEchos(hash) < rbc.faultsNum ||
		rbc.countReady(hash) <= 2*rbc.faultsNum {
		return nil
	}

	rbc.outputDecoded = true
	var prfs proofs
	for _, echo := range rbc.receivedEchos {
		prfs = append(prfs, echo.ProofRequest)
	}
	sort.Sort(prfs)

	shards := make([][]byte, rbc.nodesNum)
	for _, p := range prfs {
		shards[p.Index] = p.Proof[0]
	}
	if err := rbc.enc.Reconstruct(shards); err != nil {
		return err
	}

	for _, data := range shards[:rbc.faultsNum+1] {
		rbc.output = append(rbc.output, data...)
	}
	rbc.output = rbc.output[:prfs[0].Length]
	rbc.logger.Debugf("[%s](%d-%s) RBC output data.len: %v", rbc.nodeID, rbc.height, rbc.id, len(rbc.output))

	return nil
}

func (rbc *RBC) appendProofRequests(to string, proof *abftpb.ProofRequest) {
	rbcRequest := &abftpb.RBCRequest{
		Message: &abftpb.RBCRequest_ProofRequest{
			ProofRequest: proof,
		},
	}
	acsMessage := &abftpb.ACSMessage{
		Message: &abftpb.ACSMessage_Rbc{
			Rbc: rbcRequest,
		},
	}

	abftMessage := &abftpb.ABFTMessageReq{
		Height: rbc.height,
		From:   rbc.nodeID,
		To:     to,
		Id:     rbc.id,
		Acs:    acsMessage,
	}
	rbc.messages = append(rbc.messages, abftMessage)
	rbc.logger.Debugf("[%s](%d-%s) RBC append proof(%x-%v-%v) to: %v",
		rbc.nodeID, rbc.height, rbc.id, proof.RootHash, proof.Index, proof.Leaves, to)
}

func (rbc *RBC) appendEchoRequests(echo *abftpb.EchoRequest) {
	rbcRequest := &abftpb.RBCRequest{
		Message: &abftpb.RBCRequest_EchoRequest{
			EchoRequest: echo,
		},
	}
	acsMessage := &abftpb.ACSMessage{
		Message: &abftpb.ACSMessage_Rbc{
			Rbc: rbcRequest,
		},
	}

	for _, n := range rbc.nodes {
		abftMessage := &abftpb.ABFTMessageReq{
			Height: rbc.height,
			From:   rbc.nodeID,
			To:     n,
			Id:     rbc.id,
			Acs:    acsMessage,
		}
		rbc.messages = append(rbc.messages, abftMessage)
		rbc.logger.Debugf("[%s](%d-%s) RBC append echo id: %v, to: %v",
			rbc.nodeID, rbc.height, rbc.id, rbc.id, n)
	}
}

func (rbc *RBC) appendReadyRequests(ready *abftpb.ReadyRequest) {
	rbcRequest := &abftpb.RBCRequest{
		Message: &abftpb.RBCRequest_ReadyRequest{
			ReadyRequest: ready,
		},
	}
	acsMessage := &abftpb.ACSMessage{
		Message: &abftpb.ACSMessage_Rbc{
			Rbc: rbcRequest,
		},
	}

	for _, n := range rbc.nodes {
		abftMessage := &abftpb.ABFTMessageReq{
			Height: rbc.height,
			From:   rbc.nodeID,
			To:     n,
			Id:     rbc.id,
			Acs:    acsMessage,
		}
		rbc.messages = append(rbc.messages, abftMessage)
		rbc.logger.Debugf("[%s](%d-%s) RBC append ready id: %v, to: %v",
			rbc.nodeID, rbc.height, rbc.id, rbc.id, n)
	}
}

type proofs []*abftpb.ProofRequest

func (p proofs) Len() int           { return len(p) }
func (p proofs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p proofs) Less(i, j int) bool { return p[i].Index < p[j].Index }
