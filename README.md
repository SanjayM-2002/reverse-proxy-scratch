# reverse-proxy-scratch

Building different kinds of reverse proxies from scratch in Go. One folder per project; run everything from the repo root.

Module: `github.com/SanjayM-2002/reverse-proxy-scratch` (single module, one package per folder).

| Project | What it is |
| --- | --- |
| [load-balancer](load-balancer/) | L4 TCP load balancer: round robin, weighted round robin, least connections |
| [db-router](db-router/) | DB reverse proxy: read/write splitting across replicas, sharding (modulo / consistent hashing) |

Each project follows the same layout:

```
<project>/
  cmd/        binaries (main packages)
  internal/   project logic, not importable by other projects
  config/     config files
  scripts/    helpers to run demos
```

Code shared by two or more projects goes in `pkg/` (create it when needed).
