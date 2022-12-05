# Environment

## YDB run

```bash
docker run -it -h localhost --name ydb -p 2135:2135 -p 2136:2136 cr.yandex/yc/yandex-docker-local-ydb:latest
```

## redis run

```bash
docker run -d --name redis -p 6379:6379 redis
```

# Run service

## Flags

1. `--backend-count` - count of backends under the balancer. Default: `1`
2. `--storage-type` - type of storage. Default: `in-memory`
3. `--dsn` - data source name for database. Default: empty string
4. `--with-cache` - flag for use cache. Default: `false`
5. `--listen` - flag for subscribing on changes of data in database. Default: `false`
6. `--host` - host/ip for start listener. Default: `localhost`
7. `--port` - port to listen. Default: `3619`
8. `--ttl` - time to live of cache key. Default: `1m`

## Run command for different storages

### In-memory

```bash
go run ./cmd --backend-count=2
```

### redis

```bash
go run ./cmd --backend-count=2 --storage-type=redis --dsn="localhost:6379" --with-cache=true --listen=true
```

### YDB

```bash
go run ./cmd --backend-count=2 --storage-type=ydb --dsn="grpc://localhost:2136/local" --with-cache=true --listen=true
```