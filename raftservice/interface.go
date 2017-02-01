package raftservice

import (
	"github.com/coldog/raft/rpb"
	"errors"
)

var ErrNoNode = errors.New("no node")

type Config struct {
	ID int64
	Listen string
	Advertise string
	JoinOnBoot []string
}

type RaftService interface {
	SendAppendEntriesChan() chan *SendAppendEntries
	AppendEntriesReqChan() chan *AppendEntriesFuture
	AppendEntriesResChan() chan *rpb.Response
	SendVoteRequestChan() chan *SendVoteRequest
	VoteRequestReqChan() chan *VoteRequestFuture
	VoteRequestResChan() chan *rpb.Response

	Nodes() []*rpb.Node
	NodeCount() int

	Configure(*Config)
	Start() error
}

type SendAppendEntries struct {
	ID int64
	Broadcast bool
	Msg *rpb.AppendRequest
}

type SendVoteRequest struct {
	ID int64
	Broadcast bool
	Msg *rpb.VoteRequest
}

type AppendEntriesFuture struct {
	Msg *rpb.AppendRequest
	Response chan *rpb.Response
}

type VoteRequestFuture struct {
	Msg *rpb.VoteRequest
	Response chan *rpb.Response
}
