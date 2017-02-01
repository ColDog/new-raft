package raft

import (
	"testing"
	rs "github.com/coldog/raft/raftservice"
	"github.com/coldog/raft/rpb"
	"github.com/stretchr/testify/assert"
	"github.com/coldog/raft/store"
)

var testRaft *Raft

func init()  {
	service := &rs.RaftMockService{}
	service.Configure(&rs.Config{})
	go service.Start()

	st := store.NewInMem()

	testRaft = New(1, service, st)
}

func TestRaft_RequestVotResponse(t *testing.T) {
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
