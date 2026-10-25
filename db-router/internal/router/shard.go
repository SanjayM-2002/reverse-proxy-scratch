package router

// ShardRouter sends each key to the one shard that owns it.
// Reads and writes for the same key always land on the same shard, because that's where the data lives.
type ShardRouter struct {
	shards []Target
	picker ShardPicker // decides key -> shard index (modulo or consistent hashing)
}

func NewShardRouter(shards []Target, picker ShardPicker) *ShardRouter {
	return &ShardRouter{shards: shards, picker: picker}
}

func (r *ShardRouter) Route(cmd Command) Target {
	return r.shards[r.picker.Pick(cmd.Key)]
}
