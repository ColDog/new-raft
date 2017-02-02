package raftservice

import (
	"errors"
	"github.com/coldog/raft/rpb"
)

var ErrNoNode = errors.New("no node")

type Config struct {
	ID         uint64
	Listen     string
	Advertise  string
	JoinOnBoot []string
}

type RaftService interface {
	SendAppendEntriesChan() chan *SendAppendEntries
	AppendEntriesReqChan() chan *AppendEntriesFuture
	AppendEntriesResChan() chan *rpb.Response
	SendVoteRequestChan() chan *SendVoteRequest
	VoteRequestReqChan() chan *VoteRequestFuture
	VoteRequestResChan() chan *rpb.Response

	GetNode(uint64) *rpb.Node
	ListNodes() []*rpb.Node
	NodeCount() int

	Configure(*Config)
	Start() error
}

type SendAppendEntries struct {
	ID        uint64
	Broadcast bool
	Msg       *rpb.AppendRequest
}

type SendVoteRequest struct {
	ID        uint64
	Broadcast bool
	Msg       *rpb.VoteRequest
}

type AppendEntriesFuture struct {
	Msg      *rpb.AppendRequest
	Response chan *rpb.Response
}

type VoteRequestFuture struct {
	Msg      *rpb.VoteRequest
	Response chan *rpb.Response
}
