package raft

import (
	"testing"
	rs "github.com/coldog/raft/raftservice"
	"github.com/coldog/raft/store"
	"github.com/coldog/raft/rpb"
	"time"
	"fmt"
	"github.com/stretchr/testify/assert"
)

func testNode(id uint64) *clusterNode {
	service := &rs.RaftMockService{}
	service.Configure(&rs.Config{
		ID:        id,
		Advertise: "127.0.0.1:3000",
		Listen:    "127.0.0.1:3000",
	})

	go service.Start()

	service.SendAppendEntriesChan()

	logStore := store.NewInMem()
	testRaft := New(id, service, logStore)

	return &clusterNode{testRaft, service}
}

type clusterNode struct {
	raft *Raft
	srvc *rs.RaftMockService
}

var clusterIds []uint64
var cluster map[uint64]*clusterNode = map[uint64]*clusterNode{}

func testCluster(count int) {
	for i := 1; i < count + 1; i++ {
		clusterIds = append(clusterIds, uint64(i))
		node := testNode(uint64(i))
		node.raft.BootstrapExpect = count

		go readAppendEntries(node)
		go readVoteRequests(node)
		cluster[uint64(i)] = node
	}

	// add each node manually
	for id, node := range cluster {
		for innerId, inner := range cluster {
			if id == innerId {
				continue
			}

			node.srvc.Nodes[inner.raft.ID] = &rpb.Node{inner.raft.ID, "127.0.0.1:3000"}
		}
	}

	for _, node := range cluster {
		go node.raft.Start()
	}

	fmt.Printf("clusterIds: %+v\n", clusterIds)
}

func readAppendEntries(n *clusterNode)  {
	// read all of the send append entries for this node
	for msg := range n.srvc.SendAppendEntriesChan() {
		var ids []uint64
		if msg.Broadcast {
			ids = clusterIds
		} else {
			ids = []uint64{msg.ID}
		}

		for _, id := range ids {
			if id == n.raft.ID {
				continue
			}

			res := make(chan *rpb.Response, 2)

			// in a new thread, read back the message from the target node and send it to the node that requested it
			go func() {
				r := <-res
				fmt.Printf(">>> %d <- %d appendEntries response %+v\n", n.raft.ID, id, r)
				n.srvc.AppendEntriesResChan() <- r
			}()

			fmt.Printf(">>> %d -> %d appendEntries %+v\n", n.raft.ID, id, msg.Msg)

			// when we get the message, send it off to the target node
			cluster[id].srvc.AppendEntriesReqChan() <- &rs.AppendEntriesFuture{
				Msg: msg.Msg,
				Response: res,
			}
		}
	}
}

func readVoteRequests(n *clusterNode)  {
	// read all of the send append entries for this node
	for msg := range n.srvc.SendVoteRequestChan() {
		var ids []uint64
		if msg.Broadcast {
			ids = clusterIds
		} else {
			ids = []uint64{msg.ID}
		}

		for _, id := range ids {
			if id == n.raft.ID {
				continue
			}

			res := make(chan *rpb.Response, 2)

			// in a new thread, read back the message from the target node and send it to the node that requested it
			go func() {
				r := <-res
				fmt.Printf(">>> %d <- %d voteRequest response %+v\n", n.raft.ID, id, r)
				n.srvc.VoteRequestResChan() <- r
			}()

			fmt.Printf(">>> %d -> %d voteRequest %+v\n", n.raft.ID, id, msg.Msg)

			// when we get the message, send it off to the target node
			cluster[id].srvc.VoteRequestReqChan() <- &rs.VoteRequestFuture{
				Msg: msg.Msg,
				Response: res,
			}
		}
	}
}

func TestRaftCluster_BootstrapWithin5Seconds(t *testing.T) {
	testCluster(2)
	time.Sleep(5 * time.Second)

	assert.NotEqual(t, Candidate, cluster[1].raft.State)
	assert.NotEqual(t, Candidate, cluster[2].raft.State)

	if cluster[1].raft.State == Leader && cluster[2].raft.State == Leader {
		t.Fatal("Two leaders")
	}
}

func TestRaftCluster_AddEntry(t *testing.T) {
	testCluster(3)
	time.Sleep(5 * time.Second)

	var leader *Raft
	for _, node := range cluster {
		if node.raft.State == Leader {
			leader = node.raft
			break
		}
	}

	for i := 0; i < 2; i++ {
		leader.AddEntry([]byte("test"))
	}

	time.Sleep(2 * time.Second)
	for _, node := range cluster {
		assert.Equal(t, uint64(2), node.raft.commitIdx)
	}
}

func TestRaftCluster_VerifyLogs(t *testing.T) {
	testCluster(2)
	time.Sleep(5 * time.Second)

	var leader *Raft
	if cluster[1].raft.State == Leader {
		leader = cluster[1].raft
	} else {
		leader = cluster[2].raft
	}

	for i := 0; i < 20; i++ {
		leader.AddEntry([]byte("test"))
	}

	time.Sleep(1 * time.Second)
	assert.Equal(t, uint64(20), cluster[1].raft.commitIdx)
	assert.Equal(t, uint64(20), cluster[2].raft.commitIdx)
}

