# db-router

A minimal reverse proxy for databases. The only job: **find the right target for each command.**

Clients send one command per line:

```
READ user:42
WRITE user:42
```

Mock databases store nothing. They log the command and reply with their name, so you can see where it went.

## Two modes

| Mode | Rule |
| --- | --- |
| `replica` | `WRITE` → primary. `READ` → round robin across replicas. |
| `sharded` | `hash(key)` → one shard. Same key always hits the same shard. Hashing: `modulo` or `consistent`. |

Unlike the load balancer (one backend per TCP connection), the router picks a target **per command**,
so one client connection can hit several databases.

## Code map

```
internal/router/
  command.go   "READ user:42" -> Command{Op, Key}
  router.go    Router interface: Route(cmd) Target
  replica.go   ReplicaRouter
  shard.go     ShardRouter
  hashing.go   ModuloPicker, ConsistentHashPicker
  proxy.go     read line -> Route -> forward -> reply
```

## Run (from repo root)

```bash
./db-router/scripts/start_mockdbs.sh replica                       # terminal 1
go run ./db-router/cmd/router -config db-router/config/replica.json # terminal 2
nc localhost 7070                                                   # terminal 3, then type commands
```

For sharding, use `sharded` and `db-router/config/sharded.json`. Set `"hashing": "modulo"` in the config to compare.

The router listens on `:7070` (macOS uses `:7000` for AirPlay).
