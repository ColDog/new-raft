package raft

import (
	"net/http"
	"runtime"
	"encoding/json"
	"fmt"
)

func raftInfo(r *Raft) map[string]interface{} {
	info := map[string]interface{}{}
	raftState := map[string]interface{}{}

	raftState["state"] = r.state.Name()

	raftState["nextIdx"] = r.nextIdx
	raftState["matchIdx"] = r.matchIdx
	raftState["lastSentIdx"] = r.lastSentIdx
	raftState["leaderId"] = r.leaderID

	info["nodes"] = r.service.ListNodes()
	info["goroutines"] = runtime.NumGoroutine()
	info["raft"] = raftState

	return info
}

func StatsServer(raft *Raft, listen string) {
	if listen == "" {
		listen = fmt.Sprintf(":%d", 8000 + int(raft.ID))
	}

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(raftInfo(raft))
	})
	go http.ListenAndServe(listen, nil)
}
