package raftservice

import (
	"sync"
	"github.com/coldog/raft/rpb"
	"math/rand"
)

type RaftMockService struct {
	id uint64
	addr string
	nodes map[uint64]*rpb.Node
	lock  *sync.RWMutex
	listen string
	joinOnBoot []string

	sendAppendEntries chan *SendAppendEntries
	appendEntriesReq chan *AppendEntriesFuture
	appendEntriesRes chan *rpb.Response

	sendVoteReq chan *SendVoteRequest
	voteReq chan *VoteRequestFuture
	voteRes chan *rpb.Response
}

func (s *RaftMockService) Start() error {
	return nil
}

func (s *RaftMockService) NodeCount() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.nodes)
}

func (s *RaftMockService) GetNode(id uint64) *rpb.Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.nodes[id]
}

func (s *RaftMockService) ListNodes() []*rpb.Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	nodes := make([]*rpb.Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (s *RaftMockService) Configure(c *Config) {
	if c.ID == 0 {
		s.id = uint64(rand.Intn(10000))
	} else {
		s.id = c.ID
	}

	s.addr = c.Advertise
	s.listen = c.Listen
	s.lock = &sync.RWMutex{}
	s.joinOnBoot = c.JoinOnBoot
	s.nodes = map[uint64]*rpb.Node{}
	s.nodes[s.id] = &rpb.Node{s.id, c.Advertise}
	s.sendAppendEntries = make(chan *SendAppendEntries, chanBuffer)
	s.appendEntriesReq = make(chan *AppendEntriesFuture, chanBuffer)
	s.appendEntriesRes = make(chan *rpb.Response, chanBuffer)
	s.sendVoteReq = make(chan *SendVoteRequest, chanBuffer)
	s.voteReq = make(chan *VoteRequestFuture, chanBuffer)
	s.voteRes = make(chan *rpb.Response, chanBuffer)
}

func (s *RaftMockService) SendAppendEntriesChan() chan *SendAppendEntries {
	return s.sendAppendEntries
}

func (s *RaftMockService) AppendEntriesReqChan() chan *AppendEntriesFuture {
	return s.appendEntriesReq
}

func (s *RaftMockService) AppendEntriesResChan() chan *rpb.Response {
	return s.appendEntriesRes
}

func (s *RaftMockService) SendVoteRequestChan() chan *SendVoteRequest {
	return s.sendVoteReq
}

func (s *RaftMockService) VoteRequestReqChan() chan *VoteRequestFuture {
	return s.voteReq
}

func (s *RaftMockService) VoteRequestResChan() chan *rpb.Response {
	return s.voteRes
}
