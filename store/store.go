package store

import (
	"errors"
	"github.com/coldog/raft/rpb"
	"sync"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	Add(...*rpb.Entry) (uint64, error)
	Get(uint64) (*rpb.Entry, error)
	DeleteFrom(uint64) error
	GetFrom(uint64, int) (uint64, []*rpb.Entry, error)
}

func NewInMem() *InMemStore {
	return &InMemStore{
		entries: make(map[uint64]*rpb.Entry),
		lock:    &sync.RWMutex{},
	}
}

type InMemStore struct {
	entries map[uint64]*rpb.Entry
	lock    *sync.RWMutex
}

func (s *InMemStore) Add(entries ...*rpb.Entry) (uint64, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var max uint64
	for _, e := range entries {
		s.entries[e.Idx] = e
		if e.Idx > max {
			max = e.Idx
		}
	}
	return max, nil
}

func (s *InMemStore) Get(idx uint64) (*rpb.Entry, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	e, ok := s.entries[idx]
	if !ok {
		return nil, ErrNotFound
	}
	return e, nil
}

func (s *InMemStore) DeleteFrom(idx uint64) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, e := range s.entries {
		if e.Idx > idx {
			delete(s.entries, e.Idx)
		}
	}
	return nil
}

func (s *InMemStore) GetFrom(idx uint64, limit int) (max uint64, entries []*rpb.Entry, err error) {
	if limit == 0 {
		limit = 10
	}
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, e := range s.entries {
		if e.Idx > max {
			max = e.Idx
		}
		if e.Idx > idx {
			entries = append(entries, e)
		}

		if len(entries) >= limit {
			return
		}
	}
	return
}
