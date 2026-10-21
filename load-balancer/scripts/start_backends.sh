#!/usr/bin/env bash
# Spins up 3 backend servers on 8001, 8002, 8003 as separate processes.
# Ctrl-C stops all of them. Kill one by PID to watch the LB health check mark it DOWN.
set -euo pipefail
cd "$(dirname "$0")/../.." # repo root

mkdir -p bin
go build -o bin/lb-backend ./load-balancer/cmd/backend

pids=()
trap 'echo "stopping backends..."; kill "${pids[@]}" 2>/dev/null' EXIT INT TERM

for i in 1 2 3; do
  port=$((8000 + i))
  ./bin/lb-backend -id "$i" -port "$port" &
  pids+=($!)
  echo "started server-$i-$port (pid $!)"
done

wait
