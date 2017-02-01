package store

import (
	"sync"
	"errors"
	"github.com/coldog/raft/rpb"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	Add(...*rpb.Entry) error
	Get(uint64) (*rpb.Entry, error)
	DeleteAllFrom(uint64) error
	GetAllFrom(uint64) ([]*rpb.Entry, error)
}

type InMemStore struct {
	entries map[uint64]*rpb.Entry
	lock *sync.RWMutex
}

func (s *InMemStore) Add(e *rpb.Entry) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.entries[e.Idx] = e
	return nil
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
