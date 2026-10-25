package router

import "sync/atomic"

// ReplicaRouter does read/write splitting:
//   - WRITE always goes to the primary (the only DB that accepts writes)
//   - READ is spread across replicas with round robin
type ReplicaRouter struct {
	primary  Target
	replicas []Target
	next     atomic.Uint64 // round robin counter
}

func NewReplicaRouter(primary Target, replicas []Target) *ReplicaRouter {
	return &ReplicaRouter{primary: primary, replicas: replicas}
}

func (r *ReplicaRouter) Route(cmd Command) Target {
	if cmd.Op == Write || len(r.replicas) == 0 {
		return r.primary
	}

	i := r.next.Add(1) - 1
	return r.replicas[i%uint64(len(r.replicas))]
}
