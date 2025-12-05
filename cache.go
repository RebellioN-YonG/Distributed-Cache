package rebelcache

import (
	// "context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RebellioN-YonG/Distrbuted-Cache/store"
)

// Cache: encapsulates underlying cache store
type Cache struct {
	mtx         sync.RWMutex
	store       store.Store  // underlying store
	opts        CacheOptions // cache options
	hits        int64        // number of cache hits
	misses      int64        // number of cache misses
	initialized int32        // whether the cache has been initialized
	closed      int32        // whether the cache has been closed
}

// CacheOptions: options for cache
type CacheOptions struct {
	CacheType    store.CacheType                     // type of cache
	MaxBytes     int64                               // max bytes of cache
	BucketCnt    uint16                              // number of buckets
	CapPerBucket uint16                              // capacity of lru2's cache buckets
	Level2Cap    uint16                              // capacity of lru2's lv2 cache buckets
	CleanupTime  time.Duration                       // cleanup duration
	OnEvicted    func(key string, value store.Value) // eviction callback
}

// DefaultCacheOptions: return default cache config
func DefaultCacheOptions() CacheOptions {
	return CacheOptions{
		CacheType:    store.LRU2,
		MaxBytes:     8 * 1024 * 1024, // 8MB
		BucketCnt:    16,
		CapPerBucket: 512,
		Level2Cap:    256,
		CleanupTime:  time.Minute,
		OnEvicted:    nil,
	}
}

// NewCache: create a new cache example
func NewCache(opts CacheOptions) *Cache {
	return &Cache{
		opts: opts,
	}
}

// ensureInit
func (c *Cache) ensureInit() {
	// rapid check
	if atomic.LoadInt32(&c.initialized) == 1 {
		return
	}

	// double check
	c.mtx.Lock()
	defer c.mtx.Unlock()
	// if c.initialized == store.Options {

	// }
}
