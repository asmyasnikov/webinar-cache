package storage

import (
	"context"
	"time"

	"github.com/asmyasnikov/webinar-cache/internal/cache"
)

type cachedStorage struct {
	s     StorageListener
	cache *cache.Cache
}

var (
	_ Storage = &cachedStorage{}
)

type StorageListener interface {
	Storage
	Listener
}

func NewCached(ctx context.Context, s StorageListener, ttl time.Duration, listenChanges bool) (*cachedStorage, error) {
	cached := &cachedStorage{
		s:     s,
		cache: cache.New(ttl),
	}
	if listenChanges {
		err := s.ListenFreeSeats(ctx, func(id string, freeSeats int64) {
			cached.cache.Set(id, freeSeats)
		})
		if err != nil {
			return nil, err
		}
	}
	return cached, nil
}

func (s cachedStorage) BusIDs(ctx context.Context) (ids []string, err error) {
	return s.s.BusIDs(ctx)
}

func (s cachedStorage) FreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	if cachedValue, ok := s.cache.Get(id); ok {
		return cachedValue, nil
	}
	defer func() {
		if err == nil {
			s.cache.Set(id, freeSeats)
		}
	}()
	return s.s.FreeSeats(ctx, id)
}

func (s cachedStorage) DecrementFreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	defer func() {
		if err == nil {
			s.cache.Set(id, freeSeats)
		}
	}()
	return s.s.DecrementFreeSeats(ctx, id)
}

func (s cachedStorage) Close() error {
	return s.s.Close()
}

func (s cachedStorage) Shutdown(ctx context.Context) error {
	return s.s.Shutdown(ctx)
}
