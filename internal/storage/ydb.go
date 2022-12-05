package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topicoptions"
	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topicsugar"
	"github.com/ydb-platform/ydb-go-sdk/v3/topic/topictypes"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/ydb-platform/ydb-go-sdk/v3"
)

var (
	ydbCounter = uint64(0)
)

type ydbStorage struct {
	id uint64
	db *sql.DB
}

var (
	_ Storage  = &ydbStorage{}
	_ Listener = &ydbStorage{}
)

func NewYdb(ctx context.Context, dsn string) (*ydbStorage, error) {
	db, err := sql.Open("ydb", dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("cannot ping database: %w", err)
	}

	return initOnceYdb(ctx, &ydbStorage{
		id: atomic.AddUint64(&ydbCounter, 1),
		db: db,
	})
}

var (
	ydbStorageInitOnce sync.Once
)

func initOnceYdb(ctx context.Context, s *ydbStorage) (_ *ydbStorage, err error) {
	ydbStorageInitOnce.Do(func() {
		_, err = s.db.ExecContext(ydb.WithQueryMode(ctx, ydb.ScriptingQueryMode), `
			CREATE TABLE bus (id Utf8, freeSeats Int64, PRIMARY KEY(id));
	
			ALTER TABLE 
				bus
			ADD CHANGEFEED
				updates
			WITH (
				FORMAT = 'JSON',
				MODE = 'UPDATES'
			);
		`)
		if err != nil {
			err = fmt.Errorf("cannot create table: %w", err)
			return
		}
		_, err = s.db.ExecContext(ydb.WithQueryMode(ctx, ydb.ScriptingQueryMode), `
			UPSERT INTO bus (id, freeSeats) VALUES ("bus1", 40), ("bus2", 60);
		`)
		if err != nil {
			err = fmt.Errorf("cannot upsert data: %w", err)
			return
		}
	})
	return s, err
}

func (s *ydbStorage) BusIDs(ctx context.Context) (ids []string, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM bus")
	if err != nil {
		return nil, fmt.Errorf("cannot get bus ids: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("cannot scan query row: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *ydbStorage) FreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	rows := s.db.QueryRowContext(ctx, `
		DECLARE $id AS Utf8;
		SELECT freeSeats FROM bus WHERE id=$id;
	`, sql.Named("id", id))
	err = rows.Scan(&freeSeats)
	if err != nil {
		err = fmt.Errorf("cannot get bus free seats: %w", err)
	}
	return freeSeats, err
}

func (s *ydbStorage) DecrementFreeSeats(ctx context.Context, id string) (freeSeats int64, err error) {
	_, err = s.db.ExecContext(ctx, `
		DECLARE $id AS Utf8;		
		UPDATE bus SET freeSeats = freeSeats - 1 WHERE id=$id;
	`, sql.Named("id", id))
	if err != nil {
		return 0, fmt.Errorf("cannot decrement bus free seats: %w", err)
	}
	return s.FreeSeats(ctx, id)
}

func (s *ydbStorage) ListenFreeSeats(ctx context.Context, listener func(id string, freeSeats int64)) error {
	c, err := ydb.Unwrap(s.db)
	if err != nil {
		return fmt.Errorf("cannot unwrap *sql.DB to ydb.Connection: %w", err)
	}
	consumer := "consumer-" + strconv.Itoa(int(s.id))

	d, err := c.Topic().Describe(ctx, "bus/updates")
	if err != nil {
		return err
	}
	if func() bool {
		for _, feed := range d.Consumers {
			if feed.Name == consumer {
				return false
			}
		}
		return true
	}() {
		err = c.Topic().Alter(ctx, "bus/updates", topicoptions.AlterWithAddConsumers(topictypes.Consumer{
			Name: consumer,
		}))
		if err != nil {
			return fmt.Errorf("cannot add consumer: %w", err)
		}
	}

	go func() {
		reader, err := c.Topic().StartReader(consumer, topicoptions.ReadSelectors{
			{
				Path:     "bus/updates",
				ReadFrom: time.Now(),
			},
		},
		)
		if err != nil {
			log.Fatalf("failed to start reader: %+v", err)
		}

		log.Printf("Start cdc listen for server: %v", s.id)
		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Fatalf("failed to read message: %+v", err)
			}

			var cdcEvent struct {
				Key    []string
				Update struct {
					FreeSeats int64
				}
			}

			err = topicsugar.JSONUnmarshal(msg, &cdcEvent)
			if err != nil {
				log.Fatalf("failed to unmarshal message: %+v", err)
			}

			busID := cdcEvent.Key[0]
			listener(busID, cdcEvent.Update.FreeSeats)

			err = reader.Commit(ctx, msg)
			if err != nil {
				log.Printf("failed to commit message: %+v", err)
			}
		}
	}()

	return nil
}

func (s *ydbStorage) Close() error {
	return s.db.Close()
}

func (s *ydbStorage) Shutdown(context.Context) error {
	return s.db.Close()
}
