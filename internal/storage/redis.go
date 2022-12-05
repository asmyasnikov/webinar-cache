package storage

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

type redisStorage struct {
	listenChanges bool
	db            *redis.Client
}

var (
	_ Storage  = &redisStorage{}
	_ Listener = &redisStorage{}
)

func NewRedis(ctx context.Context, dsn string) (*redisStorage, error) {
	db := redis.NewClient(&redis.Options{
		Addr: dsn,
	})
	if err := db.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	for i := 0; i < 2; i++ {
		if _, err := db.Set(ctx, id(i), "40", 0).Result(); err != nil {
			return nil, err
		}
	}
	return &redisStorage{
		db: db,
	}, nil
}

func (s *redisStorage) BusIDs(ctx context.Context) (ids []string, err error) {
	return s.db.Keys(ctx, "bus.*").Result()
}

func (s *redisStorage) FreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	return s.db.Get(ctx, id).Int64()
}

func (s *redisStorage) DecrementFreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	defer func() {
		if s.listenChanges { // unsafe
			s.db.Publish(ctx, idToChannel(id), strconv.FormatInt(freeSeats, 10))
		}
	}()
	return s.db.Decr(ctx, id).Result()
}

const (
	feedPostfix = ".feed"
	busPrefix   = "bus."
)

func idToChannel(id string) (channel string) {
	return id + feedPostfix
}

func channelToId(channel string) (id string) {
	return strings.TrimRight(channel, feedPostfix)
}

func id(i int) string {
	return busPrefix + strconv.Itoa(i)
}

func (s *redisStorage) ListenFreeSeats(ctx context.Context, listener func(id string, freeSeats int64)) (err error) {
	defer func() {
		if err == nil {
			s.listenChanges = true // unsafe
		}
	}()
	pubsub := s.db.PSubscribe(ctx, idToChannel("bus.*"))
	if err := pubsub.Ping(ctx); err != nil {
		return err
	}
	go func() {
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				log.Fatal(err)
			}
			freeSeats, err := strconv.ParseInt(msg.Payload, 0, 64)
			if err != nil {
				log.Fatal(err)
			}
			listener(channelToId(msg.Channel), freeSeats)
		}
	}()
	return nil
}

func (s *redisStorage) Close() error {
	return s.db.Close()
}

func (s *redisStorage) Shutdown(ctx context.Context) error {
	return s.db.Shutdown(ctx).Err()
}
