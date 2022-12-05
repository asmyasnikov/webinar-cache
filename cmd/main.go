package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/asmyasnikov/webinar-cache/internal/balancer"
	"github.com/asmyasnikov/webinar-cache/internal/server"
	"github.com/asmyasnikov/webinar-cache/internal/storage"
)

const defaultConnectionString = "grpc://localhost:2136/local"

var (
	host         = flag.String("listen-host", "localhost", "host/ip for start listener")
	port         = flag.Int("port", 3619, "port to listen")
	backendCount = flag.Int("backend-count", 1, "count of backend servers")
	storageType  = flag.String("storage-type", "", "type of storage")
	dsn          = flag.String("dsn", "", "data source name")
	ttl          = flag.Duration("ttl", time.Minute, "data source name")
	listen       = flag.Bool("listen", false, "listen changes flag")
	withCache    = flag.Bool("with-cache", false, "use cache for free seats")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	servers := make([]http.Handler, *backendCount)
	for i := 0; i < *backendCount; i++ {
		var (
			db  storage.StorageListener
			err error
		)

		switch *storageType {
		case "redis":
			db, err = storage.NewRedis(ctx, *dsn)
		case "ydb":
			db, err = storage.NewYdb(ctx, *dsn)
		default:
			db = storage.NewInMemory()
		}
		if err != nil {
			panic(err)
		}
		if *withCache {
			cached, err := storage.NewCached(ctx, db, *ttl, *listen)
			if err != nil {
				panic(err)
			}
			servers[i] = server.New(cached)
		} else {
			servers[i] = server.New(db)
		}
	}
	log.Printf("servers count: %v", len(servers))
	handler := balancer.New(servers...)

	addr := *host + ":" + strconv.Itoa(*port)
	log.Printf("Start listen http://%s\n", addr)

	err := http.ListenAndServe(addr, handler)
	if errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("failed to listen and serve: %+v", err)
	}
}
