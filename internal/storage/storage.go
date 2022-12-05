package storage

import (
	"context"
	"errors"
)

var ErrNotEnthoughtFreeSeats = errors.New("not enough free seats")

type Storage interface {
	BusIDs(ctx context.Context) (ids []string, err error)
	FreeSeats(ctx context.Context, id string) (freeSeats int64, err error)
	DecrementFreeSeats(ctx context.Context, id string) (freeSeats int64, err error)

	Close() error
	Shutdown(ctx context.Context) error
}

type Listener interface {
	ListenFreeSeats(ctx context.Context, listener func(id string, freeSeats int64)) error
}
