package router

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
)

// ShardPicker maps a key to a shard index (0..N-1).
type ShardPicker interface {
	Pick(key string) int
}

// hashKey maps a string to a point in 0..2^32-1.
// Uses the first 4 bytes of SHA-256: similar keys ("user:1", "user:2") land far apart on the ring.
// (A fast hash like FNV puts them right next to each other, so they'd all hit the same shard.)
func hashKey(key string) uint32 {
	sum := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(sum[:4])
}

// ---------------------------------------------------------------------------
// Modulo: shard = hash(key) % N
// Simple, but changing N remaps most keys.
// ---------------------------------------------------------------------------

type ModuloPicker struct {
	numShards int
}

func NewModuloPicker(numShards int) *ModuloPicker {
	return &ModuloPicker{numShards: numShards}
}

func (m *ModuloPicker) Pick(key string) int {
	return int(hashKey(key) % uint32(m.numShards))
}

// ---------------------------------------------------------------------------
// Consistent hashing: shards and keys are placed on a ring of hash values (0..2^32-1).
// A key belongs to the first shard point found clockwise from the key's hash.
// Adding/removing a shard only moves the keys next to its points on the ring.
//
// Each shard is placed on the ring many times ("virtual nodes") so keys spread evenly.
// ---------------------------------------------------------------------------

const virtualNodesPerShard = 100

type ConsistentHashPicker struct {
	points  []uint32       // sorted hash values on the ring
	ownerOf map[uint32]int // ring point -> shard index
}

func NewConsistentHashPicker(shards []Target) *ConsistentHashPicker {
	c := &ConsistentHashPicker{ownerOf: make(map[uint32]int)}

	for shardIndex, shard := range shards {
		for v := range virtualNodesPerShard {
			point := hashKey(fmt.Sprintf("%s#%d", shard.Name, v)) // e.g. "shard-0-9101#17"
			c.points = append(c.points, point)
			c.ownerOf[point] = shardIndex
		}
	}

	sort.Slice(c.points, func(i, j int) bool { return c.points[i] < c.points[j] })
	return c
}

func (c *ConsistentHashPicker) Pick(key string) int {
	h := hashKey(key)

	// first ring point >= h (binary search)
	i := sort.Search(len(c.points), func(i int) bool { return c.points[i] >= h })

	// past the last point -> wrap around to the start of the ring
	if i == len(c.points) {
		i = 0
	}

	return c.ownerOf[c.points[i]]
}
