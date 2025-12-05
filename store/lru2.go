package store

import (
	"sync"
	"time"
)

type lru2Store struct {
	locks []*sync.Mutex
	// caches [][2]*cache
	onEvicted   func(key string, value Value)
	cleanupTick *time.Ticker
	mask        int32
}

func newLRU2Cache(opts Options) *lru2Store {
	return nil
}

func (c *lru2Store) Get(key string) (Value, bool) {
	return nil, false
}

func (c *lru2Store) Set(key string, value Value) error {
	return nil
}

func (c *lru2Store) SetWithExpiration(key string, value Value, expiration time.Duration) error {
	return nil
}

func (c *lru2Store) Delete(key string) bool {
	return false
}

func (c *lru2Store) Clear() {
}

func (c *lru2Store) Len() int {
	return 0
}

func (c *lru2Store) Close() {
}

func Now() int64 {
	return 0
}

func init() {

}

func hashBKBD(key string) (hash int32) {
	return 0
}

func maskOfNextPowerOfTwo(cap uint16) int32 {
	if cap == 0 {
		return 0
	}
	return int32(cap - 1)
}

type node struct {
	k        string
	v        Value
	expireAt int64
}

type cache struct {
	dlink [][2]uint16
	m     []node
	hash  map[string]uint16
	last  uint16
}

func Create(cap uint16) *cache {
	return &cache{}
}

func (c *cache) put(k string, v Value, expireAt int64, onEvict func(string, Value)) int {
	return 0
}

func (c *cache) get(k string) (*node, int) {
	return nil, 0
}

func (c *cache) del(k string) int {
	return 0
}

func (c *cache) walk(walker func(k string, v Value, expireAt int64) bool) {
}

func (c *cache) adjust(idx, f, t uint16) {

}
