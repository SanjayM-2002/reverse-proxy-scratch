# load-balancer

L4 (plain TCP) load balancer. Picks a backend per TCP connection and copies bytes both ways.

- **Server registry:** loaded from `config/servers.json` at boot (stand-in for a DB) into an in-memory map keyed by `ip:port`.
- **Strategies:** `round_robin` (`rr`), `weighted_round_robin` (`wrr`, nginx smooth WRR), `least_connections` (`lc`).
- **Session map:** each client→LB connection is tracked with its LB→backend connection in `Proxy.sessions`.
- **Health checks:** TCP dial every `health_check_interval`; a failed dial also marks the backend down.

## Run (from repo root)

```bash
./load-balancer/scripts/start_backends.sh   # terminal 1: server-1-8001, server-2-8002, server-3-8003
go run ./load-balancer/cmd/lb                # terminal 2: LB on :8080
```

```bash
for i in $(seq 9); do curl -s "localhost:8080/?strategy=wrr"; done
curl -s "localhost:8080/slow?strategy=lc&sleep=3s" &   # hold a connection open
go run ./load-balancer/cmd/lb -strategy lc             # change the default strategy
```

Install the LB binary straight from GitHub:

```bash
go install github.com/SanjayM-2002/reverse-proxy-scratch/load-balancer/cmd/lb@latest
```

The `?strategy=` param is a demo hack: the LB peeks at the HTTP request line. A real L4 LB never reads the payload.
Because routing is per connection, keep-alive clients stick to one backend.
