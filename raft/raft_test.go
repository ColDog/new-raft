package raft

import (
	"testing"
	rs "github.com/coldog/raft/raftservice"
	"github.com/coldog/raft/rpb"
	"github.com/stretchr/testify/assert"
	"github.com/coldog/raft/store"
	"time"
	"fmt"
)

var testRaft *Raft
var service *rs.RaftMockService
var logStore *store.InMemStore

func setupTestRaft()  {
	service = &rs.RaftMockService{}
	service.Configure(&rs.Config{
		ID: 1,
		Advertise: "127.0.0.1:3000",
		Listen: "127.0.0.1:3000",
	})
	go service.Start()

	logStore = store.NewInMem()

	testRaft = New(1, service, logStore)
}

func TestRaft_RequestVoteResponse(t *testing.T) {
	setupTestRaft()

	c := make(chan *rpb.Response, 2)

	vr := &rpb.VoteRequest{
		CandidateID: testRaft.ID,
		Term: testRaft.currentTerm,
		LastLogIdx: testRaft.lastAppliedIdx,
		LastLogTerm: testRaft.lastAppliedTerm,
	}

	msg := &rs.VoteRequestFuture{
		Msg: vr,
		Response: c,
	}

	testRaft.respondToVoteRequest(msg)

	res := <- c
	assert.True(t, res.Accepted)
}

func TestRaft_AppendEntriesResponseNoEntry(t *testing.T) {
	setupTestRaft()

	c := make(chan *rpb.Response, 2)

	ar := &rpb.AppendRequest{
		SenderID: testRaft.ID,
		Term: testRaft.currentTerm,
		LeaderID: testRaft.leaderID,
		PrevLogIdx: testRaft.lastAppliedIdx,
		PrevLogTerm: testRaft.lastAppliedTerm,
		LeaderCommitIdx: testRaft.commitIdx,
		Entries: nil,
	}

	msg := &rs.AppendEntriesFuture{
		Msg: ar,
		Response: c,
	}

	testRaft.respondToAppendEntriesAsFollower(msg)

	// cannot find previous entry
	res := <- c
	assert.True(t, res.Accepted)
}

func TestRaft_AppendEntriesResponseNewEntries(t *testing.T) {
	setupTestRaft()

	c := make(chan *rpb.Response, 2)

	ar := &rpb.AppendRequest{
		SenderID: 1,
		Term: 1,
		LeaderID: 1,
		PrevLogIdx: 0,
		PrevLogTerm: 0,
		LeaderCommitIdx: 2,
		Entries: []*rpb.Entry{
			{Idx: 1, Command: []byte("testing"), Term: 1},
			{Idx: 2, Command: []byte("testing"), Term: 1},
		},
	}

	msg := &rs.AppendEntriesFuture{
		Msg: ar,
		Response: c,
	}

	testRaft.respondToAppendEntriesAsFollower(msg)

	// cannot find previous entry
	res := <- c
	assert.True(t, res.Accepted)

	assert.Equal(t, uint64(1), testRaft.currentTerm)
	assert.Equal(t, uint64(2), testRaft.commitIdx)
	assert.Equal(t, uint64(2), testRaft.lastAppliedIdx)
}

func TestRaft_AppendEntriesResponseWithoutLogs(t *testing.T) {
	setupTestRaft()

	c := make(chan *rpb.Response, 2)

	ar := &rpb.AppendRequest{
		SenderID: 1,
		Term: 1,
		LeaderID: 1,
		PrevLogIdx: 20,
		PrevLogTerm: 20,
		LeaderCommitIdx: 2,
	}

	msg := &rs.AppendEntriesFuture{
		Msg: ar,
		Response: c,
	}

	testRaft.respondToAppendEntriesAsFollower(msg)

	// cannot find previous entry
	res := <- c
	assert.False(t, res.Accepted)
}

func TestRaft_LeaderHeartbeats(t *testing.T) {
	setupTestRaft()

	service.Nodes[2] = &rpb.Node{2, "127.0.0.1:3004"}
	service.Nodes[3] = &rpb.Node{3, "127.0.0.1:3005"}

	c := service.SendAppendEntriesChan()

	testRaft.sendHeartbeats()
	time.Sleep(1 * time.Second)

	msg := <- c
	assert.True(t, msg.Broadcast)
}

func TestRaft_LeaderSendEntries(t *testing.T) {
	setupTestRaft()

	service.Nodes[2] = &rpb.Node{2, "127.0.0.1:3004"}
	service.Nodes[3] = &rpb.Node{3, "127.0.0.1:3005"}

	testRaft.commitIdx = 1
	testRaft.lastAppliedIdx = 1
	testRaft.lastAppliedTerm = 2
	testRaft.currentTerm = 2
	testRaft.logStore.Add(
		&rpb.Entry{Idx: 1, Command: []byte("testing"), Term: 2},
		&rpb.Entry{Idx: 2, Command: []byte("testing"), Term: 2},
		&rpb.Entry{Idx: 3, Command: []byte("testing"), Term: 2},
	)

	c := service.SendAppendEntriesChan()

	testRaft.sendLogEntries()

	msg1 := <- c
	res := &rpb.Response{
		SenderID: msg1.ID,
		Accepted: true,
	}
	testRaft.handleAppendEntriesRes(res)

	assert.Equal(t, uint64(3), testRaft.matchIdx[msg1.ID])
	assert.Equal(t, uint64(3), testRaft.matchIdx[msg1.ID])
	assert.Equal(t, uint64(3), testRaft.lastSentIdx[msg1.ID])

	fmt.Printf("%+v\n", testRaft.lastSentIdx)
	fmt.Printf("%+v\n", service.Nodes)
	fmt.Printf("%+v\n", logStore)
}

func TestRaft_RunAsFollowerElectionTimeout(t *testing.T) {
	setupTestRaft()

	testRaft.runAsFollower()

	c := service.SendVoteRequestChan()
	msg := <- c

	assert.Equal(t, uint64(1), msg.Msg.CandidateID)
}

func TestRaft_Election(t *testing.T) {
	setupTestRaft()

	service.Nodes[2] = &rpb.Node{2, "127.0.0.1:3004"}
	service.Nodes[3] = &rpb.Node{2, "127.0.0.1:3004"}

	testRaft.runAsFollower()

	c := service.SendVoteRequestChan()
	msg := <- c

	assert.Equal(t, uint64(1), msg.Msg.CandidateID)

	testRaft.handleVoteResponse(&rpb.Response{SenderID: 2, Accepted: true})

	assert.Equal(t, Leader, testRaft.state)
}
