package raftservice

import (
	"golang.org/x/net/context"
	"github.com/coldog/raft/rpb"
	"google.golang.org/grpc"

	"sync"
	"log"
	"time"
	"net"
	"os"
	"os/signal"
	"math/rand"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const chanBuffer = 100

type RaftGRPCNode struct {
	*rpb.Node
	conn *grpc.ClientConn
	client rpb.RaftClient
}

func (n *RaftGRPCNode) connect() error {
	conn, err := grpc.Dial(n.Addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	n.client = rpb.NewRaftClient(conn)
	n.conn = conn
	return nil
}

type RaftGRPCService struct {
	ID uint64
	Addr string
	Nodes map[uint64]*RaftGRPCNode
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

func (s *RaftGRPCService) Configure(c *Config) {
	if c.ID == 0 {
		s.ID = uint64(rand.Intn(10000))
	} else {
		s.ID = c.ID
	}

	s.Addr = c.Advertise
	s.listen = c.Listen
	s.lock = &sync.RWMutex{}
	s.joinOnBoot = c.JoinOnBoot
	s.Nodes = map[uint64]*RaftGRPCNode{}
	s.Nodes[s.ID] = &RaftGRPCNode{Node: &rpb.Node{s.ID, c.Advertise}}
	s.sendAppendEntries = make(chan *SendAppendEntries, chanBuffer)
	s.appendEntriesReq = make(chan *AppendEntriesFuture, chanBuffer)
	s.appendEntriesRes = make(chan *rpb.Response, chanBuffer)
	s.sendVoteReq = make(chan *SendVoteRequest, chanBuffer)
	s.voteReq = make(chan *VoteRequestFuture, chanBuffer)
	s.voteRes = make(chan *rpb.Response, chanBuffer)
}

func (s *RaftGRPCService) Start() error {
	lis, err := net.Listen("tcp", s.listen)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	rpb.RegisterRaftServer(server, s)

	s.catchInterrupt()
	go s.run()
	go s.runJoinOnBoot()

	log.Println("[INFO] service: serving on", s.listen, "with id", s.ID)
	return server.Serve(lis)
}

func (s *RaftGRPCService) SendAppendEntriesChan() chan *SendAppendEntries {
	return s.sendAppendEntries
}

func (s *RaftGRPCService) AppendEntriesReqChan() chan *AppendEntriesFuture {
	return s.appendEntriesReq
}

func (s *RaftGRPCService) AppendEntriesResChan() chan *rpb.Response {
	return s.appendEntriesRes
}

func (s *RaftGRPCService) SendVoteRequestChan() chan *SendVoteRequest {
	return s.sendVoteReq
}

func (s *RaftGRPCService) VoteRequestReqChan() chan *VoteRequestFuture {
	return s.voteReq
}

func (s *RaftGRPCService) VoteRequestResChan() chan *rpb.Response {
	return s.voteRes
}

func (s *RaftGRPCService) GetNode(id uint64) *rpb.Node {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if node, ok := s.Nodes[id]; ok {
		return node.Node
	}
	return nil
}

func (s *RaftGRPCService) ListNodes() []*rpb.Node {
	return s.nodeList()
}

func (s *RaftGRPCService) getNode(id uint64) (*RaftGRPCNode, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	node, ok := s.Nodes[id]
	if !ok {
		return nil, ErrNoNode
	}

	if node.client == nil {
		err := node.connect()
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

func (s *RaftGRPCService) AppendEntries(ctx context.Context, req *rpb.AppendRequest) (*rpb.Response, error) {
	resCh := make(chan *rpb.Response)
	s.appendEntriesReq <- &AppendEntriesFuture{req, resCh}
	res := <- resCh
	return res, nil
}

func (s *RaftGRPCService) RequestVote(ctx context.Context, req *rpb.VoteRequest) (*rpb.Response, error) {
	resCh := make(chan *rpb.Response)
	s.voteReq <- &VoteRequestFuture{req, resCh}
	res := <- resCh
	return res, nil
}

func (s *RaftGRPCService) JoinTo(addr string) error {
	log.Printf("[DEBU] service: join to %s", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := rpb.NewRaftClient(conn)
	nodes, err := client.Join(context.Background(), &rpb.Node{
		ID: s.ID,
		Addr: addr,
	})
	log.Printf("[DEBU] service: join to received response %s: %+v", addr, nodes)

	if err != nil {
		return err
	}
	conn.Close()

	s.lock.Lock()
	defer s.lock.Unlock()
	s.syncNodes(nodes.Nodes)

	msg := &rpb.Node{ID: s.ID, Addr: s.Addr}

	for _, node := range s.Nodes {
		if node.ID == s.ID {
			continue
		}

		if node.client != nil {
			continue
		}

		err := node.connect()
		if err != nil {
			log.Printf("[ERRO] service: error while joining node: %s %v", node.Addr, err)
			continue
		}

		nodes, err := node.client.Join(context.Background(), msg)
		if err != nil {
			log.Printf("[ERRO] service: error while joining node: %s %v", node.Addr, err)
			continue
		}

		s.syncNodes(nodes.Nodes)
	}

	return nil
}

func (s *RaftGRPCService) Join(ctx context.Context, req *rpb.Node) (*rpb.Nodes, error) {
	log.Printf("[DEBU] service: handling join from: %s", req.Addr)

	s.syncNode(req)
	res := &rpb.Nodes{
		SenderID: s.ID,
		Nodes: s.nodeList(),
	}
	return res, nil
}

func (s *RaftGRPCService) LeaveCluster() {
	s.lock.RLock()
	defer s.lock.RUnlock()

	msg := &rpb.Node{
		ID: s.ID,
		Addr: s.Addr,
	}

	for _, node := range s.Nodes {
		if node.client != nil {
			_, err := node.client.Leave(context.Background(), msg)
			if err != nil {
				log.Printf("[ERRO] service: failed to leave: %v", err)
			}
		}
	}
}

func (s *RaftGRPCService) Leave(ctx context.Context, req *rpb.Node) (*rpb.Ack, error) {
	log.Printf("[DEBU] service: handling leave from: %s", req.Addr)

	s.removeNode(req.ID)
	return &rpb.Ack{s.ID}, nil
}

func (s *RaftGRPCService) ClusterState(ctx context.Context, req *rpb.Nodes) (*rpb.Nodes, error) {
	s.syncNodesL(req.Nodes)
	res := &rpb.Nodes{
		SenderID: s.ID,
		Nodes: s.nodeList(),
	}
	return res, nil
}

func (s *RaftGRPCService) deliverAppendEntries(msg *SendAppendEntries) error {
	node, err := s.getNode(msg.ID)
	if err != nil {
		return err
	}

	res, err := node.client.AppendEntries(context.Background(), msg.Msg)
	if err != nil {
		return err
	}

	s.appendEntriesRes <- res
	return nil
}

func (s *RaftGRPCService) deliverVoteRequest(msg *SendVoteRequest) error {
	node, err := s.getNode(msg.ID)
	if err != nil {
		return err
	}

	res, err := node.client.RequestVote(context.Background(), msg.Msg)
	if err != nil {
		return err
	}

	s.voteRes <- res
	return nil
}

func (s *RaftGRPCService) randNodeID() uint64 {
	var nodeId uint64
	s.lock.RLock()
	defer s.lock.RUnlock()
	for id, _ := range s.Nodes {
		if id != s.ID {
			nodeId = id
			break
		}
	}
	return nodeId
}

func (s *RaftGRPCService) pingClusterState() error {
	node, err := s.getNode(s.randNodeID())
	if err != nil {
		return err
	}

	log.Printf("[DEBU] service: pinging cluster state to %d as %d", node.ID, s.ID)

	res, err := node.client.ClusterState(context.Background(), &rpb.Nodes{Nodes: s.nodeList()})
	if err != nil {
		return err
	}

	s.syncNodesL(res.Nodes)
	return nil
}

func (s *RaftGRPCService) run() {
	for {
		select {
		case msg := <- s.sendAppendEntries:
			err := s.deliverAppendEntries(msg)
			if err != nil {
				log.Printf("[ERRO] service: could not send msg: %d, %v", msg.ID, err)
			}
		case msg := <-s.sendVoteReq:
			err := s.deliverVoteRequest(msg)
			if err != nil {
				log.Printf("[ERRO] service: could not send msg: %d, %v", msg.ID, err)
			}
		case <-time.After(10 * time.Second):
			err := s.pingClusterState()
			if err != nil {
				log.Printf("[ERRO] service: could not ping state: %v", err)
			}
		}
	}
}

func (s *RaftGRPCService) syncNode(n *rpb.Node) {
	if n.ID == s.ID { return }

	s.lock.RLock()
	_, ok := s.Nodes[n.ID]
	s.lock.RUnlock()

	if !ok {
		node := &RaftGRPCNode{Node: n}
		node.connect()
		s.lock.Lock()
		s.Nodes[n.ID] = node
		s.lock.Unlock()
	}
}

func (s *RaftGRPCService) removeNode(id uint64) {
	if id == s.ID { return }

	s.lock.RLock()
	_, ok := s.Nodes[id]
	s.lock.RUnlock()

	if ok {
		s.lock.Lock()
		delete(s.Nodes, id)
		s.lock.Unlock()
	}
}

func (s *RaftGRPCService) syncNodesL(nodes []*rpb.Node) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.syncNodes(nodes)
}

func (s *RaftGRPCService) syncNodes(nodes []*rpb.Node) {
	for _, n := range nodes {
		if n.ID == s.ID { continue }

		if _, ok := s.Nodes[n.ID]; !ok {
			node := &RaftGRPCNode{Node: n}
			node.connect()
			s.Nodes[n.ID] = node
		}
	}
}

func (s *RaftGRPCService) nodeList() []*rpb.Node {
	s.lock.RLock()
	defer s.lock.RUnlock()

	nodes := make([]*rpb.Node, 0, len(s.Nodes))
	for _, n := range s.Nodes {
		nodes = append(nodes, n.Node)
	}
	return nodes
}

func (s *RaftGRPCService) catchInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	go func(){
		<-c
		go func() {
			time.Sleep(10 * time.Second)
			os.Exit(1)
		}()

		s.LeaveCluster()
		os.Exit(0)
	}()
}

func (s *RaftGRPCService) runJoinOnBoot() {
	time.Sleep(1 * time.Second)
	for _, addr := range s.joinOnBoot {
		if addr == "" {
			continue
		}

		log.Printf("[INFO] service: join to %s", addr)
		err := s.JoinTo(addr)
		if err != nil {
			log.Printf("[INFO] service: join to %s failed %v", addr, err)
		}
	}
}
