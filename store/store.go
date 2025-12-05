package store

import "time"

type Value interface {
	Len() int
}

type Store interface {
	Get(key string) (Value, bool)
	Set(key string, value Value) error
	SetWithExpiration(key string, value Value, expiration time.Duration) error
	Delete(key string) error
	Clear()
	Len() int
	Close()
}

type CacheType string

const (
	LRU  CacheType = "LRU"
	LRU2 CacheType = "LRU2"
)

// Options: general options for lru and lru2
type Options struct {
	MaxBytes        int64                         // max bytes of lru cache
	BucketCnt       uint16                        // number of lru2 buckets
	CapPerBucket    uint16                        // capacity of lru2's bucket
	Level2Cap       uint16                        // capacity of lru2's lv2 cache
	CleanupInterval time.Duration                 // cleanup Duration
	OnEvicted       func(key string, value Value) // eviction callback func
}

func NewOptions() Options {
	return Options{
		MaxBytes:        8 * 1024, // 8KB
		BucketCnt:       16,
		CapPerBucket:    512,
		Level2Cap:       256,
		CleanupInterval: time.Minute,
		OnEvicted:       nil,
	}
}

// NewStore: create a new store example
func NewStore(cacheType CacheType, opts Options) Store {
	switch cacheType {
	case LRU:
		// return newLRUCache(opts)
		return nil
	case LRU2:
		// return newLRU2Cache(opts)
		return nil

	default:
		// return newLRUCache(opts)
		return nil
	}

}
