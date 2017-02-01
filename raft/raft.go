package raft

import (
	"github.com/coldog/raft/store"
	rs "github.com/coldog/raft/raftservice"
	"github.com/coldog/raft/rpb"
	"time"
	"errors"
	"log"
)

var (
	ErrLowerTerm = errors.New("lower term")
	ErrVoteConditionsFail = errors.New("cannot grant vote")
	ErrTermNotMatch = errors.New("terms do not match at the provided index")
)

const (
	FollowerTimeout = 300 * time.Millisecond
	LeaderTimeout = 300 * time.Millisecond
	CandidateTimeout = 300 * time.Millisecond
	MaxEntriesPerMessage = 10
)

type State int

const (
	Candidate State = iota
	Follower
	Leader
)

func New(id uint64, srvc rs.RaftService, store store.Store) *Raft {
	return &Raft{
		ID: id,
		service: srvc,
		logStore: store,
	}
}

type Raft struct {
	ID uint64
	state State

	leaderID uint64
	currentTerm uint64
	votedFor uint64
	voteCount int
	logStore store.Store

	commitIdx uint64
	lastAppliedIdx uint64
	lastAppliedTerm uint64

	nextIdx map[uint64]uint64
	matchIdx map[uint64]uint64
	lastSentIdx map[uint64]uint64

	service rs.RaftService

	sendAppendEntries chan *rs.SendAppendEntries
	appendEntriesReq chan *rs.AppendEntriesFuture
	appendEntriesRes chan *rpb.Response

	sendVoteReq chan *rs.SendVoteRequest
	voteReq chan *rs.VoteRequestFuture
	voteRes chan *rpb.Response
}

func (r *Raft) configure() {
	r.sendAppendEntries = r.service.SendAppendEntriesChan()
	r.appendEntriesReq = r.service.AppendEntriesReqChan()
	r.appendEntriesRes = r.service.AppendEntriesResChan()
	r.sendVoteReq = r.service.SendVoteRequestChan()
	r.voteReq = r.service.VoteRequestReqChan()
	r.voteRes = r.service.VoteRequestResChan()
}

func (r *Raft) errResponse(err error) *rpb.Response {
	return &rpb.Response{
		Accepted: false,
		Error: err.Error(),
		SenderID: r.ID,
		LeaderID: r.leaderID,
		LastAppliedIdx: r.lastAppliedIdx,
	}
}

func (r *Raft) response() *rpb.Response {
	return &rpb.Response{
		Accepted: true,
		SenderID: r.ID,
		LeaderID: r.leaderID,
		LastAppliedIdx: r.lastAppliedIdx,
	}
}

func (r *Raft) respondToAppendEntriesAsFollower(msg *rs.AppendEntriesFuture) {
	r.votedFor = 0
	r.leaderID = msg.Msg.SenderID

	if msg.Msg.Term > r.currentTerm {
		r.currentTerm = msg.Msg.Term
	}

	if msg.Msg.Term < r.currentTerm {
		msg.Response <- r.errResponse(ErrLowerTerm)
		return
	}

	if msg.Msg.PrevLogIdx != 0 {
		prevEntry, err := r.logStore.Get(msg.Msg.PrevLogIdx)
		if err != nil {
			log.Printf("[WARN] raft: could not get previous log entry: %v", err)
			msg.Response <- r.errResponse(err)
			return
		}

		if prevEntry.Term != msg.Msg.PrevLogTerm {
			log.Printf("[WARN] raft: prev log entry (%d) does not match: %v", msg.Msg.PrevLogIdx, err)
			msg.Response <- r.errResponse(ErrTermNotMatch)

			// delete all future terms
			err = r.logStore.DeleteFrom(msg.Msg.PrevLogIdx)
			if err != nil {
				log.Printf("[WARN] raft: error removing entries: %v", err)
			}
			return
		}
	}

	if len(msg.Msg.Entries) > 0 {
		lastIdx, err := r.logStore.Add(msg.Msg.Entries...)
		if err != nil {
			log.Printf("[WARN] raft: error adding entries: %v", err)
			msg.Response <- r.errResponse(err)
			return
		}
		r.lastAppliedIdx = lastIdx
		r.commitIdx = min(r.lastAppliedIdx, msg.Msg.LeaderCommitIdx)
	}
	msg.Response <- r.response()
}

func (r *Raft) respondToVoteRequest(msg *rs.VoteRequestFuture) {
	if msg.Msg.Term < r.currentTerm {
		msg.Response <- r.errResponse(ErrLowerTerm)
		return
	}

	if (r.votedFor == 0 || r.votedFor == msg.Msg.CandidateID) && msg.Msg.LastLogIdx == r.lastAppliedIdx {
		msg.Response <- r.response()
	}
	msg.Response <- r.errResponse(ErrVoteConditionsFail)
}

func (r *Raft) runAsFollower() {
	select {
	case msg := <-r.appendEntriesReq:
		r.respondToAppendEntriesAsFollower(msg)
	case msg := <-r.voteReq:
		r.respondToVoteRequest(msg)
	case <-r.appendEntriesRes:
		log.Println("[WARN] raft: received unexpected response to append entries")
	case <-r.voteRes:
		log.Println("[WARN] raft: received unexpected response to vote")
	case <-time.After(FollowerTimeout):
		r.toCandidate()
	}
}

func (r *Raft) appendEntriesMsg(entries []*rpb.Entry) *rpb.AppendRequest {
	return &rpb.AppendRequest{
		SenderID: r.ID,
		Term: r.currentTerm,
		LeaderID: r.leaderID,
		PrevLogIdx: r.lastAppliedIdx,
		PrevLogTerm: r.lastAppliedTerm,
		LeaderCommitIdx: r.commitIdx,
		Entries: entries,
	}
}

func (r *Raft) sendHeartbeats() {
	msg := r.appendEntriesMsg(nil)
	r.sendAppendEntries <- &rs.SendAppendEntries{
		Broadcast: true,
		Msg: msg,
	}
}

func (r *Raft) sendLogEntries() {
	for _, n := range r.service.ListNodes() {
		nextIdx := r.nextIdx[n.ID]
		if r.lastAppliedIdx <= nextIdx {
			continue
		}

		entries, err := r.logStore.GetFrom(nextIdx, MaxEntriesPerMessage)
		if err != nil {
			log.Printf("[ERRO] raft: could not get log entries: %v", err)
			continue
		}

		r.lastSentIdx[n.ID] = entries[len(entries) - 1].Idx
		msg := r.appendEntriesMsg(entries)
		r.sendAppendEntries <- &rs.SendAppendEntries{
			ID: n.ID,
			Msg: msg,
		}
	}
}

func (r *Raft) handleAppendEntriesRes(msg *rpb.Response) {
	if msg.Accepted {
		r.nextIdx[msg.SenderID] = r.lastSentIdx[msg.SenderID]
		r.matchIdx[msg.SenderID] = r.lastSentIdx[msg.SenderID]
	} else {
		r.nextIdx[msg.SenderID]--
	}

	r.checkCommitIdx()
}

func (r *Raft) checkCommitIdx() {

}

func (r *Raft) runAsLeader() {
	select {
	case msg := <-r.appendEntriesReq:
		if msg.Msg.Term > r.currentTerm {
			r.state = Follower
			r.currentTerm = msg.Msg.Term
		}
		r.respondToAppendEntriesAsFollower(msg)
	case msg := <-r.appendEntriesRes:
		r.handleAppendEntriesRes(msg)
	case msg := <-r.voteReq:
		r.respondToVoteRequest(msg)
	case <-r.voteRes:
		log.Println("[WARN] raft: received unexpected response to vote")
	case <-time.After(LeaderTimeout):
		r.sendLogEntries()
	}
}

func (r *Raft) sendVoteRequests() {
	msg := &rpb.VoteRequest{
		CandidateID: r.ID,
		Term: r.currentTerm,
		LastLogIdx: r.lastAppliedIdx,
		LastLogTerm: r.lastAppliedTerm,
	}
	r.sendVoteReq <- &rs.SendVoteRequest{
		Broadcast: true,
		Msg: msg,
	}
}

func (r *Raft) toCandidate() {
	r.sendVoteRequests()
	r.currentTerm += 1
	r.state = Candidate
}

func (r *Raft) runAsCandidate() {
	select {
	case msg := <-r.appendEntriesReq:
		r.currentTerm = msg.Msg.Term // todo: need to check term greater than here?
		r.state = Follower
		r.respondToAppendEntriesAsFollower(msg)
	case <-r.appendEntriesRes:
		log.Println("[WARN] raft: received unexpected response to append entries")
	case msg := <-r.voteReq:
		r.respondToVoteRequest(msg)
	case msg := <-r.voteRes:
		if msg.Accepted {
			r.voteCount += 1
		}
		if (r.voteCount + 1) / 2 >= r.service.NodeCount() {
			// majority agree
			r.state = Leader
			r.sendHeartbeats()
			return
		}
	case <-time.After(CandidateTimeout):
		r.toCandidate()
	}
}

func (r *Raft) run() {
	for {
		switch r.state {
		case Leader:
			r.runAsLeader()
		case Candidate:
			r.runAsCandidate()
		case Follower:
			r.runAsFollower()
		}
	}
}

func (r *Raft) Start() {
	r.configure()
	r.run()
}

func min(args ...uint64) (m uint64) {
	for _, a := range args {
		if a < m || m == 0 {
			m = a
		}
	}
	return m
}
