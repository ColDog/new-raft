package main

import (
	"github.com/coldog/raft/raft"
	"github.com/coldog/raft/store"
	"github.com/coldog/raft/raftservice"

	"log"
	"flag"
	"strings"
)

var (
	listenF = flag.String("listen", ":3002", "Address to listen on")
	advertiseF = flag.String("advertise", "127.0.0.1:3002", "Address of this node")
	joinToF = flag.String("join-to", "", "Address of a node to join")
	idF = flag.Int64("id", 1, "Node ID")
)

func main() {
	flag.Parse()

	var id uint64 = uint64(*idF)
	var service raftservice.RaftService
	var logStore store.Store

	service = &raftservice.RaftGRPCService{}
	service.Configure(&raftservice.Config{
		ID: id,
		Advertise: *advertiseF,
		Listen: *listenF,
		JoinOnBoot: strings.Split(*joinToF, ","),
	})

	logStore = store.NewInMem()

	r := raft.New(id, service, logStore)
	go r.Start()

	log.Fatal(service.Start())
}
