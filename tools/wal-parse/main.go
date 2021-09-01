package main

import (
	"chainmaker.org/chainmaker-go/common/wal"
	"chainmaker.org/chainmaker-go/pb/protogo/common"
	tbftpb "chainmaker.org/chainmaker-go/pb/protogo/consensus/tbft"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"time"
)

// timeoutInfo is used for consensus state transition because of
// timeout
type timeoutInfo struct {
	time.Duration
	Height int64
	Round  int32
	Step   tbftpb.Step
}

func (ti timeoutInfo) string() string {

	return fmt.Sprintf("{time.Duration:%s,Height:%d,Round:%d,Step:%s}", ti.String(), ti.Height, ti.Round, ti.Step)

}

// mustUnmarshal unmarshals from byte slice to protobuf message or panic
func mustUnmarshal(b []byte, msg proto.Message) {
	if err := proto.Unmarshal(b, msg); err != nil {
		panic(err)
	}
}

func newTimeoutInfoFromProto(ti *tbftpb.TimeoutInfo) timeoutInfo {
	return timeoutInfo{
		Duration: time.Duration(ti.Duration),
		Height:   ti.Height,
		Round:    ti.Round,
		Step:     ti.Step,
	}
}

// Proposal represent a proposal to be vote for consensus
type Proposal struct {
	Voter       string
	Height      int64
	Round       int32
	PolRound    int32
	Block       *common.Block
	Endorsement *common.EndorsementEntry
}

func (p Proposal) String() string {

	return fmt.Sprintf("{Voter:%s,Height:%d,Round:%d,PolRound:%d}", p.Voter, p.Height, p.Round, p.PolRound)

}

// Vote represents a vote to proposal
type Vote struct {
	Type        tbftpb.VoteType
	Voter       string
	Height      int64
	Round       int32
	Hash        []byte
	Endorsement *common.EndorsementEntry
}

func (vote Vote) String() string {

	return fmt.Sprintf("{Type:%s,Voter:%s,Height:%d,Round:%d,Hash:%x}", vote.Type.String(), vote.Voter, vote.Height, vote.Round, vote.Hash)

}

// NewProposalFromProto create a new Proposal instance from pb
func NewProposalFromProto(p *tbftpb.Proposal) *Proposal {
	if p == nil {
		return nil
	}
	proposal := NewProposal(
		p.Voter,
		p.Height,
		p.Round,
		p.PolRound,
		p.Block,
	)
	proposal.Endorsement = p.Endorsement

	return proposal
}

// NewProposal create a new Proposal instance
func NewProposal(voter string, height int64, round int32, polRound int32, block *common.Block) *Proposal {
	return &Proposal{
		Voter:    voter,
		Height:   height,
		Round:    round,
		PolRound: polRound,
		Block:    block,
	}
}

// NewVoteFromProto create a new Vote instance from pb
func NewVoteFromProto(v *tbftpb.Vote) *Vote {
	vote := NewVote(
		v.Type,
		v.Voter,
		v.Height,
		v.Round,
		v.Hash,
	)
	vote.Endorsement = v.Endorsement

	return vote
}

// NewVote create a new Vote instance
func NewVote(typ tbftpb.VoteType, voter string, height int64, round int32, hash []byte) *Vote {
	return &Vote{
		Type:   typ,
		Voter:  voter,
		Height: height,
		Round:  round,
		Hash:   hash,
	}
}

func main() {

	waldir := "/Users/sweet/go/src/chainmaker-go/build/release/chainmaker-V1.0.0-wx-org.chainmaker.org/data/wx-org.chainmaker.org/ledgerData1/chain1/tbftwal"

	wal1, err := wal.Open(waldir, nil)
	if err != nil {
		fmt.Println(err)
	}

	lastIndex, err := wal1.LastIndex()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("wal lastIndex:%d \n", lastIndex)

	for i := uint64(1); i <= lastIndex; i++ {
		data, err := wal1.Read(i)
		if err != nil {
			fmt.Println(err)
		}
		entry := &tbftpb.WalEntry{}
		mustUnmarshal(data, entry)

		switch entry.Type {
		case tbftpb.WalEntryType_TimeoutEntry:
			timeoutInfoProto := new(tbftpb.TimeoutInfo)
			mustUnmarshal(entry.Data, timeoutInfoProto)

			timeoutInfo := newTimeoutInfoFromProto(timeoutInfoProto)
			fmt.Printf("walIndex:%d, height:%d, heightFirstIndex:%d, timeoutInfo:%+v\n", i, timeoutInfo.Height, entry.HeightFirstIndex, timeoutInfo.string())

		case tbftpb.WalEntryType_ProposalEntry:
			proposalProto := new(tbftpb.Proposal)
			mustUnmarshal(entry.Data, proposalProto)
			proposal := NewProposalFromProto(proposalProto)

			fmt.Printf("walIndex:%d, height:%d, heightFirstIndex:%d, proposal:%+v\n", i, proposal.Height, entry.HeightFirstIndex, proposal)

		case tbftpb.WalEntryType_VoteEntry:
			voteProto := new(tbftpb.Vote)
			mustUnmarshal(entry.Data, voteProto)
			vote := NewVoteFromProto(voteProto)

			fmt.Printf("walIndex:%d, height:%d, heightFirstIndex:%d, vote:%+v\n", i, vote.Height, entry.HeightFirstIndex, vote)
		}
	}
}
