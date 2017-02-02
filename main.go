package main

import (
	"github.com/coldog/raft/raft"
	"github.com/coldog/raft/raftservice"
	"github.com/coldog/raft/store"
	"github.com/hashicorp/logutils"

	"flag"
	"log"
	"os"
	"strings"
)

var (
	listenF    = flag.String("listen", ":3001", "Address to listen on")
	advertiseF = flag.String("advertise", "127.0.0.1:3001", "Address of this node")
	joinToF    = flag.String("join-to", "", "Address of a node to join")
	bootstrapF = flag.Int("bootstrap", 2, "Boostrap expect")
	idF        = flag.Int64("id", 1, "Node ID")
)

func main() {
	flag.Parse()

	var id uint64 = uint64(*idF)
	var service raftservice.RaftService
	var logStore store.Store

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBU", "WARN", "ERRO"},
		MinLevel: logutils.LogLevel("DEBU"),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	service = &raftservice.RaftGRPCService{}
	service.Configure(&raftservice.Config{
		ID:         id,
		Advertise:  *advertiseF,
		Listen:     *listenF,
		JoinOnBoot: strings.Split(*joinToF, ","),
	})

	logStore = store.NewInMem()

	r := raft.New(id, service, logStore)
	r.BootstrapExpect = *bootstrapF

	go r.Start()
	go raft.DebugServer(r, "")

	log.Fatal(service.Start())
}
