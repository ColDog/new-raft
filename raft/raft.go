package raft

import (
	"github.com/coldog/raft/store"
	rs "github.com/coldog/raft/raftservice"
	"github.com/coldog/raft/rpb"
	"time"
	"errors"
	"log"
	"math/rand"
)

var (
	ErrLowerTerm = errors.New("lower term")
	ErrVoteConditionsFail = errors.New("cannot grant vote")
	ErrTermNotMatch = errors.New("terms do not match at the provided index")
)

const (
	FollowerTimeout = 7 * time.Second
	LeaderTimeout = 2 * time.Second
	CandidateTimeout = 5 * time.Second
	MaxEntriesPerMessage = 10
)

var stateNames = map[State]string{
	Candidate: "CANDIDATE",
	Follower: "FOLLOWER",
	Leader: "LEADER",
}

type State int

func (s State) Name() string {
	return stateNames[s]
}

const (
	Candidate State = iota
	Follower
	Leader
)

func New(id uint64, srvc rs.RaftService, store store.Store) *Raft {
	r := &Raft{
		ID: id,
		service: srvc,
		logStore: store,
		nextIdx: map[uint64]uint64{},
		matchIdx: map[uint64]uint64{},
		lastSentIdx: map[uint64]uint64{},
	}
	r.configure()
	return r
}

type Raft struct {
	ID uint64
	BootstrapExpect int

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
	log.Printf("[DEBU] raft: receive append entries: %+v", msg.Msg)

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
	log.Printf("[DEBU] received vote request %+v\n", msg.Msg)
	if msg.Msg.Term < r.currentTerm {
		msg.Response <- r.errResponse(ErrLowerTerm)
		return
	}

	if (r.votedFor == 0 || r.votedFor == msg.Msg.CandidateID) && msg.Msg.LastLogIdx == r.lastAppliedIdx {
		log.Printf("[DEBU] raft: voting for %d", msg.Msg.CandidateID)
		r.votedFor = msg.Msg.CandidateID
		msg.Response <- r.response()
	}
	msg.Response <- r.errResponse(ErrVoteConditionsFail)
}

func (r *Raft) runAsFollower() {
	r.votedFor = 0
	r.voteCount = 0
	select {
	case msg := <-r.appendEntriesReq:
		r.respondToAppendEntriesAsFollower(msg)
	case msg := <-r.voteReq:
		r.respondToVoteRequest(msg)
	case <-r.appendEntriesRes:
		log.Println("[WARN] raft: received unexpected response to append entries")
	case <-r.voteRes:
		log.Println("[WARN] raft: received unexpected response to vote")
	case <-time.After(jitter(FollowerTimeout)):
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
	log.Printf("[DEBU] raft: sending heartbeats")
	msg := r.appendEntriesMsg(nil)
	r.sendAppendEntries <- &rs.SendAppendEntries{
		Broadcast: true,
		Msg: msg,
	}
}

func (r *Raft) sendLogEntries() {
	for _, n := range r.service.ListNodes() {
		if n.ID == r.ID {
			continue
		}

		nextIdx := r.nextIdx[n.ID]
		if r.lastAppliedIdx <= nextIdx {
			continue
		}

		lastSentIdx, entries, err := r.logStore.GetFrom(nextIdx, MaxEntriesPerMessage)
		if err != nil {
			log.Printf("[ERRO] raft: could not get log entries: %v", err)
			continue
		}

		r.lastSentIdx[n.ID] = lastSentIdx

		log.Printf("[DEBU] raft: send entries to: %d", n.ID)
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
		if r.nextIdx[msg.SenderID] > 0 {
			r.nextIdx[msg.SenderID]--
		}
	}

	r.checkCommitIdx()
}

// • If there exists an N such that N > commitIndex, a majority
// of matchIndex[i] ≥ N, and log[N].term == currentTerm:
// set commitIndex = N (§5.3, §5.4).
func (r *Raft) checkCommitIdx() {
	n := r.commitIdx + 1
	for {
		// check majority of matchIndex
		matchCount := 0
		for _, mIdx := range r.matchIdx {
			if mIdx >= n {
				matchCount++
			}
		}
		if matchCount < r.service.NodeCount() / 2 {
			n--
			break
		}

		e, err := r.logStore.Get(n)
		if err != nil {
			n--
			break
		}

		if e.Term != r.currentTerm {
			n--
			break
		}

		n += 1
	}

	r.commitIdx = n
}

func (r *Raft) runAsLeader() {
	select {
	case msg := <-r.appendEntriesReq:
		if msg.Msg.Term >= r.currentTerm {
			log.Printf("[DEBU] raft: leader received append entries from greater term %+v", msg.Msg)
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
	case <-time.After(jitter(LeaderTimeout)):
		// r.sendLogEntries()
		r.sendHeartbeats()
	}
}

func (r *Raft) sendVoteRequests() {
	log.Printf("[DEBU] raft: requesting votes")
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

func (r *Raft) handleVoteResponse(msg *rpb.Response) {
	if msg.Accepted {
		r.voteCount += 1
	}
	if (r.voteCount + 1) > r.service.NodeCount() / 2 {
		log.Printf("[DEBU] raft: converting to leading, received %d votes", r.voteCount)
		// majority agree
		r.state = Leader
		r.votedFor = 0
		r.voteCount = 0
		r.sendHeartbeats()
	}
}

func (r *Raft) runAsCandidate() {
	select {
	case msg := <-r.appendEntriesReq:
		r.currentTerm = msg.Msg.Term // todo: need to check term greater than here?
		r.state = Follower
		r.respondToAppendEntriesAsFollower(msg)
	case <-r.appendEntriesRes:
		log.Printf("[WARN] raft: received unexpected response to append entries")
	case msg := <-r.voteReq:
		r.respondToVoteRequest(msg)
	case msg := <-r.voteRes:
		r.handleVoteResponse(msg)
	case <-time.After(jitter(CandidateTimeout)):
		r.toCandidate()
	}
}

func (r *Raft) run() {
	for {
		if r.BootstrapExpect == 0 && r.service.NodeCount() > 1 {
			break
		}

		if r.BootstrapExpect == r.service.NodeCount() {
			log.Printf("[INF0] raft: met bootstrap count of %d", r.BootstrapExpect)
			break
		}

		log.Printf("[INF0] raft: node count is 1, waiting for more nodes...")
		time.Sleep(5 * time.Second)
	}

	log.Printf("[INFO] raft: starting...")
	for {
		prevState := r.state
		switch r.state {
		case Leader:
			r.runAsLeader()
		case Candidate:
			r.runAsCandidate()
		case Follower:
			r.runAsFollower()
		}

		if r.state != prevState {
			log.Printf("[INFO] raft: changed to state %s", r.state.Name())
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

func jitter(t time.Duration) time.Duration {
	return time.Duration((rand.Float64() * 0.5) * float64(t) + float64(t / 2))
}
