package store

import (
	"sync"
	"errors"
	"github.com/coldog/raft/rpb"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	Add(...*rpb.Entry) (uint64, error)
	Get(uint64) (*rpb.Entry, error)
	DeleteFrom(uint64) error
	GetFrom(uint64, int) ([]*rpb.Entry, error)
}

func NewInMem() Store {
	return &InMemStore{
		entries: make(map[uint64]*rpb.Entry),
		lock: &sync.RWMutex{},
	}
}

type InMemStore struct {
	entries map[uint64]*rpb.Entry
	lock *sync.RWMutex
}

func (s *InMemStore) Add(entries ...*rpb.Entry) (uint64, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var last uint64
	for _, e := range entries {
		s.entries[e.Idx] = e
		last = e.Idx
	}
	return last, nil
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

func (s *InMemStore) GetFrom(idx uint64, limit int) ([]*rpb.Entry, error) {
	if limit == 0 {
		limit = 10
	}
	s.lock.RLock()
	defer s.lock.RUnlock()

	entries := make([]*rpb.Entry, 0)
	for _, e := range s.entries {
		if e.Idx > idx {
			entries = append(entries, e)
		}

		if len(entries) >= limit {
			return entries, nil
		}
	}
	return entries, nil
}
