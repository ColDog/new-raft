package raft

import (
	"errors"
	rs "github.com/coldog/raft/raftservice"
	"github.com/coldog/raft/rpb"
	"github.com/coldog/raft/store"
	"log"
	"math/rand"
	"time"
)

var (
	ErrLowerTerm          = errors.New("lower term")
	ErrVoteConditionsFail = errors.New("cannot grant vote")
	ErrTermNotMatch       = errors.New("terms do not match at the provided index")
	ErrNotLeader          = errors.New("not the leader")
)

const (
	FollowerTimeout      = 3 * time.Second
	LeaderTimeout        = 2 * time.Second
	CandidateTimeout     = 6 * time.Second
	MaxEntriesPerMessage = 10
)

var stateNames = map[State]string{
	Candidate: "CANDIDATE",
	Follower:  "FOLLOWER",
	Leader:    "LEADER",
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
		ID:              id,
		service:         srvc,
		logStore:        store,
		nextIdx:         map[uint64]uint64{},
		matchIdx:        map[uint64]uint64{},
		lastSentIdx:     map[uint64]uint64{},
		addEntries:      make(chan *addEntryFuture, 100),
		commitIdxEvents: newIdxEvents(),
		timeoutInterrupt:  make(chan struct{}, 10),
		timeoutReached:    make(chan struct{}),
		triggerLogEntries: make(chan struct{}, 50),
		changeToLeader:    make(chan struct{}, 2),
		exitLeader:        make(chan struct{}, 2),
		quit:              make(chan struct{}),
	}
	r.configure()
	return r
}

type Raft struct {
	ID              uint64
	BootstrapExpect int

	State State

	leaderID    uint64
	currentTerm uint64
	votedFor    uint64
	voteCount   int
	logStore    store.Store

	commitIdx       uint64
	lastAppliedIdx  uint64
	lastAppliedTerm uint64

	nextIdx     map[uint64]uint64
	matchIdx    map[uint64]uint64
	lastSentIdx map[uint64]uint64

	service           rs.RaftService
	sendAppendEntries chan *rs.SendAppendEntries
	appendEntriesReq  chan *rs.AppendEntriesFuture
	appendEntriesRes  chan *rpb.Response
	sendVoteReq       chan *rs.SendVoteRequest
	voteReq           chan *rs.VoteRequestFuture
	voteRes           chan *rpb.Response

	timeoutInterrupt  chan struct{}
	timeoutReached    chan struct{}

	triggerLogEntries chan struct{}
	changeToLeader    chan struct{}
	exitLeader        chan struct{}

	quit chan struct{}

	addEntries      chan *addEntryFuture
	commitIdxEvents *idxEvents
}

type addEntryFuture struct {
	command  []byte
	response chan *addEntryResponse
}

type addEntryResponse struct {
	idx uint64
	err error
}

func (r *Raft) AddEntry(cmd []byte) (uint64, error) {
	c := make(chan *addEntryResponse, 2)
	r.addEntries <- &addEntryFuture{cmd, c}
	res := <-c

	if res.err != nil {
		log.Printf("[WARN] raft: add entry fail %v", res.err)
		return 0, res.err
	}

	log.Printf("[DEBU] raft: added entry, idx: %d", res.idx)
	sub := make(chan uint64)
	unsubscribe := r.commitIdxEvents.subscribe(sub)
	defer unsubscribe()

	for idx := range sub {
		if idx >= res.idx {
			return res.idx, nil
		}
	}

	panic("Subscription closed early")
	return 0, nil
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
		Accepted:       false,
		Error:          err.Error(),
		SenderID:       r.ID,
		LeaderID:       r.leaderID,
		LastAppliedIdx: r.lastAppliedIdx,
	}
}

func (r *Raft) response() *rpb.Response {
	return &rpb.Response{
		Accepted:       true,
		SenderID:       r.ID,
		LeaderID:       r.leaderID,
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
		r.lastAppliedTerm = r.currentTerm
	}

	n := min(r.lastAppliedIdx, msg.Msg.LeaderCommitIdx)
	if n > r.commitIdx {
		r.commitIdx = n
		r.commitIdxEvents.publish(n)
	}

	msg.Response <- r.response()
}

func (r *Raft) respondToVoteRequest(msg *rs.VoteRequestFuture) {
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
	case req := <-r.addEntries:
		req.response <- &addEntryResponse{0, ErrNotLeader}
	case msg := <-r.appendEntriesReq:
		log.Printf("[DEBU] raft: %s receive appendEntries from %d: %+v", r.State.Name(), msg.Msg.SenderID, msg.Msg)
		r.respondToAppendEntriesAsFollower(msg)
	case msg := <-r.voteReq:
		log.Printf("[DEBU] raft: %s receive voteRequest for %d", r.State.Name(), msg.Msg.CandidateID)
		r.respondToVoteRequest(msg)
	case msg := <-r.appendEntriesRes:
		log.Printf("[DEBU] raft: %s receive appendEntries response from %d", r.State.Name(), msg.SenderID)
	case msg := <-r.voteRes:
		log.Printf("[DEBU] raft: %s receive vote response from %d", r.State.Name(), msg.SenderID)
	case <-r.timeoutReached:
		log.Printf("[DEBU] raft: %s follower timeout, converting to candidate", r.State.Name())
		r.toCandidate()
	case <-r.quit:
		return
	}
}

func (r *Raft) appendEntriesMsg() *rpb.AppendRequest {
	return &rpb.AppendRequest{
		SenderID:        r.ID,
		Term:            r.currentTerm,
		LeaderID:        r.leaderID,
		LeaderCommitIdx: r.commitIdx,
	}
}

func (r *Raft) sendHeartbeats() {
	msg := r.appendEntriesMsg()
	r.sendAppendEntries <- &rs.SendAppendEntries{
		Broadcast: true,
		Msg:       msg,
	}
}

func (r *Raft) sendLogEntries() {
	for _, n := range r.service.ListNodes() {
		if n.ID == r.ID {
			continue
		}
		msg := r.appendEntriesMsg()

		nextIdx := r.nextIdx[n.ID]
		if r.lastAppliedIdx <= nextIdx {
			log.Printf("[DEBU] raft: send heartbeat to: %d", n.ID)
			r.sendAppendEntries <- &rs.SendAppendEntries{n.ID, false, msg}
			continue
		}

		lastSentIdx, entries, err := r.logStore.GetFrom(nextIdx, MaxEntriesPerMessage)
		if err != nil {
			log.Printf("[ERRO] raft: could not get log entries: %v", err)
			r.sendAppendEntries <- &rs.SendAppendEntries{n.ID, false, msg}
			continue
		}

		r.lastSentIdx[n.ID] = lastSentIdx

		log.Printf("[DEBU] raft: send entries to: %d", n.ID)

		if r.matchIdx[n.ID] != 0 {
			prevLogEntry, err := r.logStore.Get(r.matchIdx[n.ID])
			if err != nil {
				log.Printf("[ERRO] raft: could not get log entries: %v", err)
				r.sendAppendEntries <- &rs.SendAppendEntries{n.ID, false, msg}
				continue
			}
			msg.PrevLogTerm = prevLogEntry.Term
			msg.PrevLogIdx = prevLogEntry.Idx
		}

		msg.Entries = entries
		r.sendAppendEntries <- &rs.SendAppendEntries{n.ID, false, msg}
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

// • If there exists an N such that N > commitIndex, a majority of matchIndex[i] ≥ N, and log[N].term == currentTerm:
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
		if matchCount < r.service.NodeCount()/2 {
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

	if n > r.commitIdx {
		r.commitIdx = n
		r.commitIdxEvents.publish(n)
	}
}

func (r *Raft) applyLogEntry(req *addEntryFuture) {
	nextIdx := r.lastAppliedIdx + 1
	e := &rpb.Entry{
		Command: req.command,
		Term:    r.currentTerm,
		Idx:     nextIdx,
	}

	_, err := r.logStore.Add(e)
	if err != nil {
		log.Printf("[ERRO] raft: error applying entry to store: %v", err)
		req.response <- &addEntryResponse{0, err}
		return
	}

	r.lastAppliedIdx = nextIdx
	r.lastAppliedTerm = r.currentTerm
	req.response <- &addEntryResponse{nextIdx, err}
}

func (r *Raft) runAsLeader() {
	select {
	case req := <-r.addEntries:
		r.applyLogEntry(req)
		r.triggerLogEntries <- struct{}{}
	case msg := <-r.appendEntriesReq:
		log.Printf("[DEBU] raft: %s receive appendEntries from %d", r.State.Name(), msg.Msg.SenderID)
		if msg.Msg.Term >= r.currentTerm {
			log.Printf("[INFO] raft: leader received append entries from greater term %+v", msg.Msg)
			r.State = Follower
			r.currentTerm = msg.Msg.Term
		}
		r.respondToAppendEntriesAsFollower(msg)
	case res := <-r.appendEntriesRes:
		log.Printf("[DEBU] raft: %s receive appendEntries response from %d", r.State.Name(), res.SenderID)
		r.handleAppendEntriesRes(res)
	case msg := <-r.voteReq:
		log.Printf("[DEBU] raft: %s receive voteRequest from %d", r.State.Name(), msg.Msg.CandidateID)
		r.respondToVoteRequest(msg)
	case res := <-r.voteRes:
		log.Printf("[DEBU] raft: %s receive vote response from %d", r.State.Name(), res.SenderID)
	case <-r.timeoutReached:
		r.triggerLogEntries <- struct{}{}
	case <-r.quit:
		return
	}
}

func (r *Raft) sendVoteRequests() {
	log.Printf("[DEBU] raft: requesting votes")
	msg := &rpb.VoteRequest{
		CandidateID: r.ID,
		Term:        r.currentTerm,
		LastLogIdx:  r.lastAppliedIdx,
		LastLogTerm: r.lastAppliedTerm,
	}
	r.sendVoteReq <- &rs.SendVoteRequest{
		Broadcast: true,
		Msg:       msg,
	}
}

func (r *Raft) toCandidate() {
	r.sendVoteRequests()
	r.currentTerm += 1
	r.State = Candidate
}

func (r *Raft) handleVoteResponse(msg *rpb.Response) {
	if msg.Accepted {
		r.voteCount += 1
	}
	if (r.voteCount + 1) > r.service.NodeCount()/2 {
		log.Printf("[DEBU] raft: converting to leading, received %d votes", r.voteCount)
		// majority agree
		r.State = Leader
		r.leaderID = r.ID
		r.votedFor = 0
		r.voteCount = 0
		r.sendHeartbeats()
	}
}

func (r *Raft) runAsCandidate() {
	select {
	case req := <-r.addEntries:
		req.response <- &addEntryResponse{0, ErrNotLeader}
	case msg := <-r.appendEntriesReq:
		log.Printf("[DEBU] raft: %s receive appendEntries from %d", r.State.Name(), msg.Msg.SenderID)
		r.currentTerm = msg.Msg.Term // todo: need to check term greater than here?
		r.State = Follower
		r.respondToAppendEntriesAsFollower(msg)
	case res := <-r.appendEntriesRes:
		log.Printf("[DEBU] raft: %s receive appendEntries response from %d", r.State.Name(), res.SenderID)
	case msg := <-r.voteReq:
		log.Printf("[DEBU] raft: %s receive voteRequest for %d", r.State.Name(), msg.Msg.CandidateID)
		r.respondToVoteRequest(msg)
	case msg := <-r.voteRes:
		log.Printf("[DEBU] raft: %s receive vote response for %d", r.State.Name(), msg.SenderID)
		r.handleVoteResponse(msg)
	case <-r.timeoutReached:
		r.toCandidate()
	case <-r.quit:
		return
	}
}

func (r *Raft) run() {
	if r.BootstrapExpect == 0 {
		r.BootstrapExpect = 2
	}

	for {
		nCount := r.service.NodeCount()
		if r.BootstrapExpect == nCount {
			log.Printf("[INF0] raft: met bootstrap count of %d", r.BootstrapExpect)
			break
		}

		log.Printf("[INF0] raft: node count is %d, waiting for %d", nCount, r.BootstrapExpect)
		time.Sleep(5 * time.Second)
	}

	log.Printf("[INFO] raft: starting...")

	go r.runTimer()
	go r.runLogEntrySender()

	r.runMain()
}

func (r *Raft) runMain() {
	for {
		prevState := r.State
		switch r.State {
		case Leader:
			r.runAsLeader()
		case Candidate:
			r.runAsCandidate()
		case Follower:
			r.runAsFollower()
		}

		r.timeoutInterrupt <- struct{}{}
		if r.State != prevState {
			log.Printf("[INFO] raft: changed to State %s", r.State.Name())

			if prevState == Leader {
				r.exitLeader <- struct{}{}
			}

			if r.State == Leader {
				r.changeToLeader <- struct{}{}
			}
		}
	}
}

func (r *Raft) runTimer() {
	for {
		var timeout time.Duration
		switch r.State {
		case Leader:
			timeout = LeaderTimeout
		case Candidate:
			timeout = CandidateTimeout
		case Follower:
			timeout = FollowerTimeout
		}

		select {
		case <-r.timeoutInterrupt:
			break
		case <-time.After(jitter(timeout)):
			r.timeoutReached <- struct{}{}
		case <-r.quit:
			return
		}
	}
}

func (r *Raft) runLogEntrySender() {
	for range r.changeToLeader {
		LEADER_LOOP:
		for {
			select {
			case <-r.triggerLogEntries:
				// send out the log entries
				r.sendLogEntries()
			case <-r.exitLeader:
				break LEADER_LOOP
			case <-r.quit:
				return
			}
		}
	}
}

func (r *Raft) Start() {
	r.configure()
	r.run()
}

func (r *Raft) Stop() {
	close(r.quit)
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
	return time.Duration((rand.Float64()*0.5)*float64(t) + float64(t/2))
}
