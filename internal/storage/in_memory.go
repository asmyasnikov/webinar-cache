package storage

import (
	"context"
	"fmt"
	"sync"
)

type inMemoryStorage struct {
	listenChanges bool
	data          map[string]int64
	dataMtx       sync.RWMutex
}

var (
	_ Storage  = &inMemoryStorage{}
	_ Listener = &inMemoryStorage{}
)

func NewInMemory() *inMemoryStorage {
	return &inMemoryStorage{
		data: map[string]int64{
			"0": 40,
			"1": 40,
		},
	}
}

func (s *inMemoryStorage) BusIDs(ctx context.Context) (ids []string, err error) {
	s.dataMtx.RLock()
	defer s.dataMtx.RUnlock()
	for k := range s.data {
		ids = append(ids, k)
	}
	return ids, nil
}

func (s *inMemoryStorage) FreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	s.dataMtx.RLock()
	defer s.dataMtx.RUnlock()
	if freeSeats, has := s.data[id]; has {
		return freeSeats, nil
	}
	return 0, fmt.Errorf("unknown id '%s'", id)
}

func (s *inMemoryStorage) DecrementFreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	s.dataMtx.Lock()
	defer s.dataMtx.Unlock()
	if freeSeats, has := s.data[id]; has {
		s.data[id]--
		return freeSeats - 1, nil
	}
	return 0, fmt.Errorf("unknown id '%s'", id)
}

func (s *inMemoryStorage) ListenFreeSeats(ctx context.Context, listener func(id string, freeSeats int64)) (err error) {
	// nop
	return nil
}

func (s *inMemoryStorage) Close() error {
	// nop
	return nil
}

func (s *inMemoryStorage) Shutdown(ctx context.Context) error {
	// nop
	return nil
}
