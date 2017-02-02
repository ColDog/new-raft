package raft

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
)

func raftInfo(r *Raft) map[string]interface{} {
	info := map[string]interface{}{}
	raftState := map[string]interface{}{}

	raftState["state"] = r.state.Name()

	raftState["nextIdx"] = r.nextIdx
	raftState["matchIdx"] = r.matchIdx
	raftState["lastSentIdx"] = r.lastSentIdx
	raftState["leaderId"] = r.leaderID
	raftState["commitIdx"] = r.commitIdx
	raftState["lastAppliedIdx"] = r.lastAppliedIdx
	raftState["lastAppliedTerm"] = r.lastAppliedTerm
	raftState["term"] = r.currentTerm

	_, entries, _ := r.logStore.GetFrom(0, 100)

	info["raft"] = raftState
	info["entries"] = entries
	info["nodes"] = r.service.ListNodes()
	info["goroutines"] = runtime.NumGoroutine()

	return info
}

func DebugServer(raft *Raft, listen string) {
	if listen == "" {
		listen = fmt.Sprintf(":%d", 8000+int(raft.ID))
	}

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(raftInfo(raft))
	})

	http.HandleFunc("/apply", func(w http.ResponseWriter, r *http.Request) {
		var cmd []byte

		if r.URL.Query().Get("cmd") != "" {
			cmd = []byte(r.URL.Query().Get("cmd"))
		} else {
			cmd = make([]byte, 1024)
			_, err := r.Body.Read(cmd)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
		}

		_, err := raft.AddEntry(cmd)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}

		w.Write([]byte("OK"))
	})

	go http.ListenAndServe(listen, nil)
}
