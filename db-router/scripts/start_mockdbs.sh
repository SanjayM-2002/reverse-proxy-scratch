#!/usr/bin/env bash
# Starts mock databases for one topology. Ctrl-C stops them all.
#   ./db-router/scripts/start_mockdbs.sh replica   -> primary-9001, replica-1-9002, replica-2-9003
#   ./db-router/scripts/start_mockdbs.sh sharded   -> shard-0-9101, shard-1-9102, shard-2-9103
set -euo pipefail
cd "$(dirname "$0")/../.." # repo root

mode="${1:-replica}"

case "$mode" in
  replica) dbs=("primary-9001:9001" "replica-1-9002:9002" "replica-2-9003:9003") ;;
  sharded) dbs=("shard-0-9101:9101" "shard-1-9102:9102" "shard-2-9103:9103") ;;
  *) echo "usage: $0 replica|sharded" >&2; exit 1 ;;
esac

mkdir -p bin
go build -o bin/mockdb ./db-router/cmd/mockdb

pids=()
trap 'echo "stopping mock dbs..."; kill "${pids[@]}" 2>/dev/null' EXIT INT TERM

for db in "${dbs[@]}"; do
  name="${db%%:*}"
  port="${db##*:}"
  ./bin/mockdb -name "$name" -port "$port" &
  pids+=($!)
  echo "started $name (pid $!)"
done

wait
