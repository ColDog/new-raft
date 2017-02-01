package main

import (
	"github.com/coldog/raft/raftservice"
	"log"
	"flag"
	"math/rand"
	"time"
	"strings"
)

var (
	listenF = flag.String("listen", ":3002", "Address to listen on")
	advertiseF = flag.String("advertise", "127.0.0.1:3002", "Address of this node")
	joinToF = flag.String("join-to", "", "Address of a node to join")
)

func main() {
	rand.Seed(time.Now().Unix())
	flag.Parse()

	var service raftservice.RaftService

	service = &raftservice.RaftGRPCService{}
	service.Configure(&raftservice.Config{
		Advertise: *advertiseF,
		Listen: *listenF,
		JoinOnBoot: strings.Split(*joinToF, ","),
	})

	log.Fatal(service.Start())
}
