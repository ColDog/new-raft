package raft

import (
	"sync"
	"time"
	"log"
)

func newIdxEvents() *idxEvents {
	return &idxEvents{
		subs: make(map[uint]chan uint64),
		lock: &sync.Mutex{},
		id: 1,
	}
}

type idxEvents struct {
	subs map[uint]chan uint64
	lock *sync.Mutex
	id uint
}

func (p *idxEvents) subscribe(sub chan uint64) func() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.id++
	p.subs[p.id] = sub
	return func() { p.unsubscribe(p.id) }
}

func (p *idxEvents) unsubscribe(id uint) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.subs[id]; ok {
		delete(p.subs, id)
	}
}

func (p *idxEvents) publish(val uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for _, ch := range p.subs {
		select {
		case ch <- val:
		case <-time.After(300 * time.Millisecond):
			log.Printf("[ERRO] raft: could not publish to idxEvent channel")
		}
	}
}