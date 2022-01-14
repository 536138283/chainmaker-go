/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/logger"

	"chainmaker.org/chainmaker-go/common/msgbus"
	tbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/tbft"
	"github.com/gogo/protobuf/proto"
)

const (
	defaultSleepTime  = 500 * time.Millisecond
	disconnectMaxTime = 10 * time.Minute
)

// gossipService if for gossipService consensus state between validators
type gossipService struct {
	sync.Mutex
	logger   *logger.CMLogger
	id       string
	msgbus   msgbus.MessageBus
	tbftImpl *ConsensusTBFTImpl

	peerStates   map[string]*PeerStateService
	recvStateC   chan *tbftpb.TBFTMsg
	recvFetchQCC chan *tbftpb.TBFTMsg
	eventC       chan struct{}
	closeC       chan struct{}
}

func newGossipService(logger *logger.CMLogger, tbftImpl *ConsensusTBFTImpl) *gossipService {
	g := &gossipService{
		logger:       logger,
		id:           tbftImpl.Id,
		msgbus:       tbftImpl.msgbus,
		tbftImpl:     tbftImpl,
		peerStates:   make(map[string]*PeerStateService),
		recvStateC:   make(chan *tbftpb.TBFTMsg, defaultChanCap),
		recvFetchQCC: make(chan *tbftpb.TBFTMsg, defaultChanCap),
		eventC:       make(chan struct{}, defaultChanCap),
		closeC:       make(chan struct{}),
	}

	for _, id := range g.tbftImpl.validatorSet.Validators {
		if id == g.tbftImpl.Id {
			continue
		}

		g.peerStates[id] = NewPeerStateService(logger, id, tbftImpl)
	}
	return g
}

func (g *gossipService) start() {
	go g.gossipStateRoutine()
	go g.recvStateRoutine()

	g.Lock()
	defer g.Unlock()
	for _, pss := range g.peerStates {
		pss.start()
	}
}

func (g *gossipService) stop() {
	g.Lock()
	defer g.Unlock()

	g.logger.Infof("[%s] stop gossip service", g.id)

	for _, v := range g.peerStates {
		v.stop()
	}
	close(g.closeC)
}

func (g *gossipService) addValidators(validators []string) error {
	if len(validators) == 0 {
		return nil
	}

	g.Lock()
	defer g.Unlock()

	g.logger.Infof("[%s] gossipService, add validators: %v", g.id, validators)
	for _, id := range validators {
		if id == g.id {
			continue
		}
		pss := NewPeerStateService(g.logger, id, g.tbftImpl)
		g.peerStates[id] = pss
		pss.start()
	}
	return nil
}

func (g *gossipService) removeValidators(validators []string) error {
	if len(validators) == 0 {
		return nil
	}

	g.Lock()
	defer g.Unlock()

	g.logger.Infof("[%s] gossipService, remove validators: %v", g.id, validators)
	for _, id := range validators {
		if pss, ok := g.peerStates[id]; ok {
			pss.stop()
			delete(g.peerStates, id)
		}
	}
	return nil
}

// select a random node to get qc
func (g *gossipService) selectPeers() string {
	var peers []string
	for _, p := range g.peerStates {
		// equal height and more than 1 rounds
		if p.Height == g.tbftImpl.Height && p.Round-1 > g.tbftImpl.Round {
			peers = append(peers, p.Id)
		}
	}
	if len(peers) == 0 {
		return ""
	}

	// select a random node
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(peers))))
	if err != nil {
		n = big.NewInt(0)
	}
	return peers[n.Int64()]
}

func (g *gossipService) triggerEvent() {
	g.logger.Infof("eventC len: %d", len(g.eventC))
	g.eventC <- struct{}{}
}

func (g *gossipService) gossipStateRoutine() {
	g.logger.Infof("start gossipStateRoutine, gossipService[%s]", g.id)
	defer g.logger.Infof("exit gossipStateRoutine, gossipService[%s]", g.id)

	loop := true
	for loop {
		select {
		case <-g.eventC:
			g.logger.Debugf("gossip because event")
			go g.gossipState()
			go g.fetchRoundQC()
		case <-time.After(defaultSleepTime):
			g.logger.Debugf("gossip because timeout")
			go g.gossipState()
			go g.fetchRoundQC()
		case <-g.closeC:
			loop = false
		}
	}
}

func (g *gossipService) fetchRoundQC() {
	g.Lock()
	defer g.Unlock()

	id := g.selectPeers()
	if id == "" {
		return
	}
	peer, ok := g.peerStates[id]
	if !ok {
		return
	}

	go peer.gossipFetchRoundQC()
}

func (g *gossipService) gossipState() {
	state := g.tbftImpl.ToGossipStateProto()
	// state := g.tbftImpl.ToProto()

	g.logger.Debugf("[%s](%d/%d/%s) gossip", state.Id, state.Height, state.Round, state.Step)

	g.Lock()
	defer g.Unlock()
	for _, p := range g.peerStates {
		go p.gossipState(state)
	}
}

func (g *gossipService) onRecvState(msg *tbftpb.TBFTMsg) {
	g.recvStateC <- msg
}

func (g *gossipService) onRecvFetchQC(fetchQC *tbftpb.TBFTMsg) {
	g.recvFetchQCC <- fetchQC
}

func (g *gossipService) recvStateRoutine() {
	g.logger.Infof("start recvStateRoutine, gossipService[%s]", g.id)
	defer g.logger.Infof("exit recvStateRoutine, gossipService[%s]", g.id)

	loop := true
	for loop {
		select {
		case msg := <-g.recvStateC:
			go g.procRecvState(msg)
		case fetchQC := <-g.recvFetchQCC:
			go g.procRecvFetchQC(fetchQC)
		case <-g.closeC:
			loop = false
		}
	}
}

func (g *gossipService) procRecvState(msg *tbftpb.TBFTMsg) {
	state := new(tbftpb.GossipState)
	if err := proto.Unmarshal(msg.Msg, state); err != nil {
		g.logger.Errorf("[%s] receive state unmarshal failed, %v", g.id, err)
		return
	}

	g.logger.Infof("[%s] receive state %s(%d/%d/%s)", g.id, state.Id, state.Height, state.Round, state.Step)

	g.Lock()
	peer, ok := g.peerStates[state.Id]
	g.Unlock()
	if !ok {
		return
	}
	peer.GetStateC() <- state
}

func (g *gossipService) procRecvFetchQC(msg *tbftpb.TBFTMsg) {
	fetcthQC := new(tbftpb.FetchRoundQC)
	if err := proto.Unmarshal(msg.Msg, fetcthQC); err != nil {
		g.logger.Errorf("[%s] receive fetcthQC unmarshal failed, %v", g.id, err)
		return
	}
	g.logger.Infof("[%s] receive fetcthQC %s(%d/%d)", g.id, fetcthQC.Id, fetcthQC.Height, fetcthQC.Round)

	g.Lock()
	peer, ok := g.peerStates[fetcthQC.Id]
	g.Unlock()
	if !ok {
		return
	}
	peer.GetFetchQCC() <- fetcthQC
}
