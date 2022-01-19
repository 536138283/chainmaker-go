/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tbft

import (
	"fmt"
	"strings"
	"sync"
	"time"

	netpb "chainmaker.org/chainmaker-go/pb/protogo/net"

	"chainmaker.org/chainmaker-go/logger"

	"chainmaker.org/chainmaker-go/common/msgbus"
	tbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/tbft"
	"github.com/gogo/protobuf/proto"
)

// PeerStateService represents the consensus state of peer node
type PeerStateService struct {
	sync.Mutex
	logger *logger.CMLogger
	Id     string
	Height int64
	Round  int32
	Step   tbftpb.Step

	Proposal         []byte // proposal
	VerifingProposal []byte
	LockedRound      int32
	LockedProposal   *Proposal // locked proposal
	ValidRound       int32
	ValidProposal    *Proposal // valid proposal
	RoundVoteSet     *roundVoteSet

	*PeerSendState
	stateC   chan *tbftpb.GossipState
	fetchQC  chan *tbftpb.FetchRoundQC
	tbftImpl *ConsensusTBFTImpl
	msgbus   msgbus.MessageBus
	closeC   chan struct{}
}

type PeerSendState struct {
	logger       *logger.CMLogger
	fibs         [100]int64
	Height       int64
	Round        int64
	beatTime     int64
	TriggerTime  int64 // The timestamp of sending proposals at the same height
	TriggerCount int64 // The count of sending proposals at the same height
}

// NewPeerSendState create a PeerSendState instance
func NewPeerSendState(logger *logger.CMLogger) *PeerSendState {
	pss := &PeerSendState{
		logger: logger,
		Height: -1, // The height starts at 0
		Round:  -1, // The height starts at 0
	}
	return pss
}

func (pss *PeerSendState) isSendTime(height, round int64) bool {

	triggerTimeDate := time.Unix(pss.TriggerTime/1000000000, 0).Format("2006-01-02 15:04:05")
	pss.logger.Debugf("PeerSendState params ([%d/%d],[%d/%s]) isSendTime to (%d/%d)",
		pss.Height, pss.Round, pss.TriggerCount, triggerTimeDate, height, round)

	//12点
	nowTime := time.Now().UnixNano()
	nowTimeDate := time.Unix(nowTime/1000000000, 0).Format("2006-01-02 15:04:05")
	if pss.beatTime == 0 {
		//12点
		pss.beatTime = nowTime
	}
	// determine the duration of node disconnection
	//12：10  > 12
	if pss.beatTime+int64(disconnectMaxTime) < nowTime {
		beatTimeDate := time.Unix(pss.beatTime/1000000000, 0).Format("2006-01-02 15:04:05")
		pss.logger.Debugf("PeerSendState ([%d/%d], [%s,%s]) ,no need to send msg to a disconnected node",
			pss.Height, pss.Round, beatTimeDate, nowTimeDate)
		return false
	}
	if pss.Height != height || pss.Round != round {
		pss.Height = height
		pss.Round = round
		//12点
		pss.TriggerTime = nowTime
		pss.TriggerCount = 1
		triggerTimeDate = time.Unix(pss.TriggerTime/1000000000, 0).Format("2006-01-02 15:04:05")
		pss.logger.Debugf("([%d/%d],[%d/%s]) isSendTime true to (%d/%d)",
			pss.Height, pss.Round, pss.TriggerCount, triggerTimeDate, height, round)
		return true
	}

	//500ms
	interval := pss.fibonacci(pss.TriggerCount) * int64(defaultSleepTime)
	pss.logger.Debugf("PeerSendState fibonacci ([%d/%d,],[%d/%s]) => (%d/%d),%d ms,%s",
		pss.Height, pss.Round, pss.TriggerCount, triggerTimeDate, height, round, interval/1000000, nowTimeDate)
	//12点+500ms  > 12点
	if pss.TriggerTime+interval < nowTime {
		pss.Height = height
		pss.Round = round
		pss.TriggerTime = nowTime
		pss.TriggerCount++
		triggerTimeDate = time.Unix(pss.TriggerTime/1000000000, 0).Format("2006-01-02 15:04:05")
		pss.logger.Debugf("([%d/%d,],[%d/%s]) isSendTime true to (%d/%d)",
			pss.Height, pss.Round, pss.TriggerCount, triggerTimeDate, height, round)
		return true
	}

	pss.logger.Debugf("([%d/%d],[%d/%s]) isSendTime false to (%d/%d)",
		pss.Height, pss.Round, pss.TriggerCount, triggerTimeDate, height, round)
	return false
}

func (pss *PeerSendState) fibonacci(n int64) (res int64) {
	// reduce unnecessary frequent calculations
	// cache: check if fibonacci(n) is already known in array
	if pss.fibs[n] != 0 {
		res = pss.fibs[n]
		return
	}
	if n <= 1 {
		res = 1
	} else {
		var pre, cur int64 = 1, 1
		var sum, i int64 = 0, 2
		for ; i <= n; i++ {
			sum = pre + cur
			pre = cur
			cur = sum
		}
		res = sum
	}
	pss.fibs[n] = res
	return
}

// NewPeerStateService create a PeerStateService instance
func NewPeerStateService(logger *logger.CMLogger, id string, tbftImpl *ConsensusTBFTImpl) *PeerStateService {
	pcs := &PeerStateService{
		logger:        logger,
		Id:            id,
		tbftImpl:      tbftImpl,
		PeerSendState: NewPeerSendState(logger),
		msgbus:        tbftImpl.msgbus,
	}
	pcs.stateC = make(chan *tbftpb.GossipState, defaultChanCap)
	pcs.fetchQC = make(chan *tbftpb.FetchRoundQC, defaultChanCap)
	pcs.closeC = make(chan struct{})
	return pcs
}

func (pcs *PeerStateService) updateWithProto(pcsProto *tbftpb.GossipState) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "[%s] update with proto to (%d/%d/%s)",
		pcs.Id, pcsProto.Height, pcsProto.Round, pcsProto.Step)

	if pcsProto.RoundVoteSet != nil &&
		pcsProto.RoundVoteSet.Prevotes != nil &&
		pcsProto.RoundVoteSet.Prevotes.Votes != nil {
		fmt.Fprintf(&builder, " prevote: [")
		for k := range pcsProto.RoundVoteSet.Prevotes.Votes {
			fmt.Fprintf(&builder, "%s, ", k)
		}
		fmt.Fprintf(&builder, "]")
	}

	if pcsProto.RoundVoteSet != nil &&
		pcsProto.RoundVoteSet.Precommits != nil &&
		pcsProto.RoundVoteSet.Precommits.Votes != nil {
		fmt.Fprintf(&builder, " precommit: [")
		for k := range pcsProto.RoundVoteSet.Precommits.Votes {
			fmt.Fprintf(&builder, "%s, ", k)
		}
		fmt.Fprintf(&builder, "]")
	}

	pcs.logger.Debugf(builder.String())

	pcs.Lock()
	defer pcs.Unlock()

	pcs.Height = pcsProto.Height
	pcs.Round = pcsProto.Round
	pcs.Step = pcsProto.Step
	pcs.Proposal = pcsProto.Proposal
	pcs.VerifingProposal = pcsProto.VerifingProposal
	pcs.beatTime = time.Now().UnixNano()
	validatorSet := pcs.tbftImpl.getValidatorSet()
	pcs.RoundVoteSet = newRoundVoteSetFromProto(pcs.logger, pcsProto.RoundVoteSet, validatorSet)
	// fetch votes from this node state
	if pcs.Height == pcs.tbftImpl.Height && pcs.Round == pcs.tbftImpl.Round &&
		pcs.RoundVoteSet != nil {
		pcs.logger.Debugf("[%s] updateVoteWithProto: [%d/%d]", pcs.Id, pcs.Height, pcs.Round)
		pcs.updateVoteWithProto(pcs.RoundVoteSet)
	}
	pcs.logger.Debugf("[%s] RoundVoteSet: %s", pcs.Id, pcs.RoundVoteSet)
}

// get the votes for tbft Engine based on the peer node state
func (pcs *PeerStateService) updateVoteWithProto(voteSet *roundVoteSet) {
	Validators := pcs.tbftImpl.getValidatorSet().Validators
	pcs.tbftImpl.RLock()
	defer pcs.tbftImpl.RUnlock()
	for _, voter := range Validators {
		pcs.logger.Debugf("%s updateVoteWithProto : %v,%v", voter, voteSet.Prevotes, voteSet.Precommits)
		// prevote Vote
		vote := voteSet.Prevotes.Votes[voter]
		if vote != nil && pcs.tbftImpl.Step < tbftpb.Step_Precommit &&
			pcs.tbftImpl.heightRoundVoteSet.isRequired(pcs.Round, vote) {
			pcs.logger.Debugf("updateVoteWithProto prevote : %s", voter)
			tbftMsg := createPrevoteMsg(vote)
			pcs.tbftImpl.internalMsgC <- tbftMsg
		}
		// precommit Vote
		vote = voteSet.Precommits.Votes[voter]
		if vote != nil && pcs.tbftImpl.Step < tbftpb.Step_Commit &&
			pcs.tbftImpl.heightRoundVoteSet.isRequired(pcs.Round, vote) {
			pcs.logger.Debugf("updateVoteWithProto precommit : %s", voter)
			tbftMsg := createPrevoteMsg(vote)
			pcs.tbftImpl.internalMsgC <- tbftMsg
		}
	}
}

func (pcs *PeerStateService) start() {
	go pcs.procStateChange()
}

// GetFetchQCC return the fetchQC channel
func (pcs *PeerStateService) GetFetchQCC() chan<- *tbftpb.FetchRoundQC {
	return pcs.fetchQC
}

func (pcs *PeerStateService) stop() {
	pcs.logger.Infof("[%s] stop PeerStateService", pcs.Id)
	close(pcs.closeC)
}

// GetStateC return the stateC channel
func (pcs *PeerStateService) GetStateC() chan<- *tbftpb.GossipState {
	return pcs.stateC
}

func (pcs *PeerStateService) procStateChange() {
	pcs.logger.Infof("PeerStateService[%s] start procStateChange", pcs.Id)
	defer pcs.logger.Infof("PeerStateService[%s] exit procStateChange", pcs.Id)

	loop := true
	for loop {
		select {
		case stateProto := <-pcs.stateC:
			pcs.updateWithProto(stateProto)

			pcs.sendStateChange()
		case fetchQCProto := <-pcs.fetchQC:
			pcs.sendRoundQC(fetchQCProto)
		case <-pcs.closeC:
			loop = false
		}
	}
}

// fetch the RoundQC
func (pcs *PeerStateService) gossipFetchRoundQC() {
	fetchRoundQC := &tbftpb.FetchRoundQC{
		Id:     pcs.tbftImpl.Id,
		Height: pcs.tbftImpl.Height,
		Round:  pcs.tbftImpl.Round,
	}

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_fetch_roundqc,
		Msg:  mustMarshal(fetchRoundQC),
	}

	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(tbftMsg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.logger.Infof("%s fetch round qc (%d/%d) to %s", pcs.tbftImpl.Id, pcs.Height, pcs.Round, pcs.Id)
	pcs.publishToMsgbus(netMsg)

}

func (pcs *PeerStateService) gossipState(state *tbftpb.GossipState) {
	pcs.Lock()
	defer pcs.Unlock()

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_state,
		Msg:  mustMarshal(state),
	}

	pcs.logger.Debugf("Proposal: %d, verifingProposal: %d, HeightRoundVoteSet: %d",
		len(state.Proposal),
		len(state.VerifingProposal),
		proto.Size(state.RoundVoteSet),
	)
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(tbftMsg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.logger.Debugf("%s gossip (%d/%d/%s) to %s", state.Id, state.Height, state.Round, state.Step, pcs.Id)
	pcs.publishToMsgbus(netMsg)

	go pcs.sendStateChange()
}

// send qc to the requesting node
func (pcs *PeerStateService) sendRoundQC(fetchQCProto *tbftpb.FetchRoundQC) {
	pcs.Lock()
	defer pcs.Unlock()

	if fetchQCProto.Height != pcs.tbftImpl.Height || fetchQCProto.Round >= pcs.tbftImpl.Round-1 {
		pcs.logger.Infof("[%s](%d/%d/%s) receive invalid fetch qc request from [%s](%d/%d)",
			pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
			pcs.Id, pcs.Height, pcs.Round)
		return
	}

	// tbftImpl.RLock()
	// need to protect the tbftImpl.heightRoundVoteSet
	pcs.tbftImpl.RLock()
	defer pcs.tbftImpl.RUnlock()

	var precommits *VoteSet
	// get the highest round of QC
	for round := pcs.tbftImpl.Round - 1; round > fetchQCProto.Round; round-- {
		roundVoteSet := pcs.tbftImpl.heightRoundVoteSet.getRoundVoteSet(round)
		if roundVoteSet == nil || roundVoteSet.Precommits == nil {
			continue
		}
		hash, ok := roundVoteSet.Precommits.twoThirdsMajority()
		// we need a QC with nil hash
		if ok && isNilHash(hash) {
			precommits = roundVoteSet.Precommits
			break
		}
	}
	if precommits == nil {
		pcs.logger.Infof("[%s](%d/%d/%s) do not have qc request send to [%s](%d/%d)",
			pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
			pcs.Id, pcs.Height, pcs.Round)
		return
	}

	roundQC := &tbftpb.RoundQC{
		Id:         pcs.tbftImpl.Id,
		Height:     pcs.tbftImpl.Height,
		Round:      pcs.tbftImpl.Round,
		Precommits: precommits.ToProto(),
	}

	tbftMsg := &tbftpb.TBFTMsg{
		Type: tbftpb.TBFTMsgType_send_roundqc,
		Msg:  mustMarshal(roundQC),
	}
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(tbftMsg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}

	pcs.logger.Infof("%s send round qc (%d/%d) to %s", pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.Id)
	pcs.publishToMsgbus(netMsg)
}

func (pcs *PeerStateService) sendStateChange() {
	pcs.Lock()
	defer pcs.Unlock()

	pcs.tbftImpl.RLock()
	defer pcs.tbftImpl.RUnlock()

	pcs.logger.Debugf("[%s](%d/%d/%s) sendStateChange to [%s](%d/%d/%s)",
		pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
		pcs.Id, pcs.Height, pcs.Round, pcs.Step,
	)
	if pcs.tbftImpl.Height < pcs.Height {
		return
	} else if pcs.tbftImpl.Height == pcs.Height {
		pcs.logger.Debugf("[%s](%d) sendStateOfRound to [%s](%d/%d/%s)",
			pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.Id, pcs.Height, pcs.Round, pcs.Step)
		pcs.sendStateOfRound()
	} else {
		pcs.logger.Debugf("[%s](%d) sendStateOfHeight to [%s](%d/%d/%s)",
			pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.Id, pcs.Height, pcs.Round, pcs.Step)
		go pcs.sendStateOfHeight(pcs.Height)
	}
}

func (pcs *PeerStateService) sendStateOfRound() {
	pcs.sendProposalOfRound(pcs.Height, pcs.Round)
	pcs.sendPrevoteOfRound(pcs.Round)
	pcs.sendPrecommitOfRound(pcs.Round)
}

func (pcs *PeerStateService) sendProposalOfRound(height int64, round int32) {
	// Send proposal (only proposer can send proposal)
	if pcs.tbftImpl.isProposer(height, round) &&
		pcs.tbftImpl.Proposal != nil &&
		pcs.VerifingProposal == nil &&
		pcs.Step >= tbftpb.Step_Propose {
		// appropriate send time
		if !pcs.PeerSendState.isSendTime(pcs.Height, int64(pcs.Round)) {
			pcs.logger.Infof("[%s](%d/%d/%s) sendStateChange to [%s](%d/%d/%s) is not send time",
				pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
				pcs.Id, pcs.Height, pcs.Round, pcs.Step,
			)
			return
		}
		pcs.sendProposal(pcs.tbftImpl.Proposal)
	}
}

func (pcs *PeerStateService) sendPrevoteOfRound(round int32) {
	pcs.logger.Debugf("[%s] RoundVoteSet: %s", pcs.Id, pcs.RoundVoteSet)
	// Send prevote
	prevoteVs := pcs.tbftImpl.heightRoundVoteSet.prevotes(round)
	if prevoteVs != nil {
		vote, ok := prevoteVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Prevotes != nil {

			var builder strings.Builder
			fmt.Fprintf(&builder, " prevote: [")
			for k := range pcs.RoundVoteSet.Prevotes.Votes {
				fmt.Fprintf(&builder, "%s, ", k)
			}
			fmt.Fprintf(&builder, "]")
			pcs.logger.Debugf(builder.String())

			if _, pOk := pcs.RoundVoteSet.Prevotes.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrevote(vote)
			}
		}
	}
}

func (pcs *PeerStateService) sendPrecommitOfRound(round int32) {
	pcs.logger.Debugf("[%s] RoundVoteSet: %s", pcs.Id, pcs.RoundVoteSet)
	// Send precommit
	precommitVs := pcs.tbftImpl.heightRoundVoteSet.precommits(round)
	if precommitVs != nil {
		vote, ok := precommitVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Precommits != nil {

			var builder strings.Builder
			fmt.Fprintf(&builder, " precommit: [")
			for k := range pcs.RoundVoteSet.Precommits.Votes {
				fmt.Fprintf(&builder, "%s, ", k)
			}
			fmt.Fprintf(&builder, "]")
			pcs.logger.Debugf(builder.String())

			if _, pOk := pcs.RoundVoteSet.Precommits.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrecommit(vote)
			}
		}
	}
}

func (pcs *PeerStateService) publishToMsgbus(msg *netpb.NetMsg) {
	pcs.logger.Debugf("[%s] publishToMsgbus size: %d", pcs.tbftImpl.Id, proto.Size(msg))
	pcs.msgbus.Publish(msgbus.SendConsensusMsg, msg)
}

func (pcs *PeerStateService) sendProposal(proposal *Proposal) {
	pcs.logger.Infof("[%s](%d/%d/%s) sendProposal [%s](%d/%d/%x) to %v",
		pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
		proposal.Voter, proposal.Height, proposal.Round, proposal.Block.Header.BlockHash, pcs.Id)

	// Send proposal
	msg := createProposalMsg(proposal)
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(msg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.publishToMsgbus(netMsg)

	pcs.logger.Debugf("send proposal(%d/%x) to %s(%d/%d/%s)",
		proposal.Block.Header.BlockHeight, proposal.Block.Header.BlockHash,
		pcs.Id, pcs.Height, pcs.Round, pcs.Step)
}

func (pcs *PeerStateService) sendPrevote(prevote *Vote) {
	// Send prevote
	msg := createPrevoteMsg(prevote)
	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(msg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.publishToMsgbus(netMsg)

	pcs.logger.Debugf("send prevote(%d/%d/%s/%x) to %s",
		pcs.Height, pcs.Round, pcs.Step, prevote.Hash, pcs.Id)
}

func (pcs *PeerStateService) sendPrecommit(precommit *Vote) {
	// Send precommit
	msg := createPrecommitMsg(precommit)

	netMsg := &netpb.NetMsg{
		Payload: mustMarshal(msg),
		Type:    netpb.NetMsg_CONSENSUS_MSG,
		To:      pcs.Id,
	}
	pcs.publishToMsgbus(netMsg)
	pcs.logger.Debugf("send precommit(%d/%d/%s/%x) to %s",
		pcs.Height, pcs.Round, pcs.Step, precommit.Hash, pcs.Id)

}

func (pcs *PeerStateService) sendStateOfHeight(height int64) {
	state := pcs.tbftImpl.consensusStateCache.getConsensusState(pcs.Height)
	if state == nil {
		return
	}
	pcs.sendProposalInState(state)
	pcs.sendPrevoteInState(state)
	pcs.sendPrecommitInState(state)
}

func (pcs *PeerStateService) sendProposalInState(state *ConsensusState) {
	// Send Proposal (only proposer can send proposal)
	if pcs.tbftImpl.isProposer(state.Height, state.Round) &&
		state.Proposal != nil &&
		pcs.VerifingProposal == nil &&
		pcs.Step >= tbftpb.Step_Propose {
		// appropriate send time
		if !pcs.PeerSendState.isSendTime(pcs.Height, int64(pcs.Round)) {
			pcs.logger.Infof("[%s](%d/%d/%s) sendStateChange to [%s](%d/%d/%s) is not send time",
				pcs.tbftImpl.Id, pcs.tbftImpl.Height, pcs.tbftImpl.Round, pcs.tbftImpl.Step,
				pcs.Id, pcs.Height, pcs.Round, pcs.Step,
			)
			return
		}
		pcs.sendProposal(state.Proposal)
	}
}

func (pcs *PeerStateService) sendPrevoteInState(state *ConsensusState) {
	// Send Prevote
	prevoteVs := state.heightRoundVoteSet.prevotes(pcs.Round)
	if prevoteVs != nil {
		vote, ok := prevoteVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Prevotes != nil {
			if _, pOk := pcs.RoundVoteSet.Prevotes.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrevote(vote)
			}
		}
	}
}

func (pcs *PeerStateService) sendPrecommitInState(state *ConsensusState) {
	// Send precommit
	precommitVs := state.heightRoundVoteSet.precommits(pcs.Round)
	if precommitVs != nil {
		vote, ok := precommitVs.Votes[pcs.tbftImpl.Id]
		if ok && pcs.RoundVoteSet != nil && pcs.RoundVoteSet.Precommits != nil {
			if _, pOk := pcs.RoundVoteSet.Precommits.Votes[pcs.tbftImpl.Id]; !pOk {
				pcs.sendPrecommit(vote)
			}
		}
	}
}
