package store

import (
	"container/list"
	"sync"
	"time"
)

// lruCache implements an LRU cache using a doubly linked list.
// It is safe for concurrent access by multiple goroutines.
type lruCache struct {
	mtx             sync.RWMutex                  // read-write mutex to protect the cache
	lru             *list.List                    // doubly linked list to maintain LRU order
	items           map[string]*list.Element      // map of keys to list elements for O(1) access
	expires         map[string]time.Time          // map of keys to their expiration times
	maxBytes        int64                         // maximum bytes the cache can hold
	usedBytes       int64                         // currently used bytes in the cache
	onEvicted       func(key string, value Value) // callback function when an item is evicted
	cleanupInterval time.Duration                 // interval for running cleanup operations
	cleanupTicker   *time.Ticker                  // ticker for periodic cleanup
	closeCh         chan struct{}                 // channel to signal cleanup goroutine to stop
}

// lruEntry represents a single entry in the LRU cache.
type lruEntry struct {
	key   string // the key of the cache entry
	value Value  // the value of the cache entry
}

// newLRUCache creates a new LRU cache with the given options.
//
// Parameters:
//   - opts: Options containing cache configuration such as max bytes, cleanup interval, and eviction callback
//
// Returns:
//   - *lruCache: A pointer to the newly created LRU cache
func newLRUCache(opts Options) *lruCache {
	cleanup := opts.CleanupInterval
	if cleanup <= 0 {
		cleanup = time.Minute
	}
	c := &lruCache{
		lru:             list.New(),
		items:           make(map[string]*list.Element),
		expires:         make(map[string]time.Time),
		maxBytes:        opts.MaxBytes,
		onEvicted:       opts.OnEvicted,
		cleanupInterval: cleanup,
		closeCh:         make(chan struct{}),
	}
	c.cleanupTicker = time.NewTicker(c.cleanupInterval)
	go c.cleanupLoop()
	return c
}

// Get retrieves the value associated with the given key from the cache.
//
// Parameters:
//   - key: The key to look up in the cache
//
// Returns:
//   - Value: The value associated with the key, or nil if not found or expired
//   - bool: True if the key was found and not expired, false otherwise
func (c *lruCache) Get(key string) (Value, bool) {
	c.mtx.RLock()
	elem, ok := c.items[key]
	if !ok {
		c.mtx.RUnlock()
		return nil, false
	}
	// check expiration
	if expire, isExpired := c.expires[key]; isExpired && time.Now().After(expire) {
		c.mtx.RUnlock()
		// asynchronously delete expired item
		go c.Delete(key)
		return nil, false
	}
	// entry to get the value and release r-lock
	entry := elem.Value.(*lruEntry)
	value := entry.value
	c.mtx.RUnlock()

	// update position in LRU with w-lock
	c.mtx.Lock()
	// check if item still exists
	if _, ok := c.items[key]; ok {
		c.lru.MoveToFront(elem)
	}
	c.mtx.Unlock()
	return value, true
}

// Set stores a key-value pair in the cache with no expiration.
//
// Parameters:
//   - key: The key to store
//   - value: The value to store
//
// Returns:
//   - error: Any error encountered during the operation
func (c *lruCache) Set(key string, value Value) error {
	return c.SetWithExpiration(key, value, 0)
}

// SetWithExpiration stores a key-value pair in the cache with an optional expiration duration.
//
// Parameters:
//   - key: The key to store
//   - value: The value to store
//   - expiration: The duration after which the item expires (0 for no expiration)
//
// Returns:
//   - error: Any error encountered during the operation
func (c *lruCache) SetWithExpiration(key string, value Value, expiration time.Duration) error {
	if value == nil {
		c.Delete(key)
		return nil
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// get expiration
	var expire time.Time
	if expiration > 0 {
		expire = time.Now().Add(expiration)
		c.expires[key] = expire
	} else {
		delete(c.expires, key)
	}

	if elem, ok := c.items[key]; ok {
		// update value if key exists
		entry := elem.Value.(*lruEntry)
		c.usedBytes += int64(value.Len() - entry.value.Len())
		entry.value = value
		c.lru.MoveToBack(elem)
		return nil
	}
	// add new key
	entry := &lruEntry{key: key, value: value}
	elem := c.lru.PushBack(entry)
	c.items[key] = elem
	c.usedBytes += int64(len(key) + value.Len())

	// evict if necessary
	c.evict()
	return nil
}

// Delete removes the item with the given key from the cache.
//
// Parameters:
//   - key: The key of the item to delete
//
// Returns:
//   - bool: True if the item was found and deleted, false otherwise
func (c *lruCache) Delete(key string) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
		return true
	}
	return false
}

// Clear removes all items from the cache.
func (c *lruCache) Clear() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	// if callback is set, traversal all items and call it
	if c.onEvicted != nil {
		for _, elem := range c.items {
			entry := elem.Value.(*lruEntry)
			c.onEvicted(entry.key, entry.value)
		}
	}
	// clear all items
	c.lru.Init()
	c.items = make(map[string]*list.Element)
	c.expires = make(map[string]time.Time)
	c.usedBytes = 0
}

// Len returns the number of items currently in the cache.
//
// Returns:
//   - int: The number of items in the cache
func (c *lruCache) Len() int {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.lru.Len()
}

// removeElement removes the specified element from the cache.
// Note: lock must be held before calling this function.
//
// Parameters:
//   - elem: The list element to remove
func (c *lruCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*lruEntry)
	c.lru.Remove(elem)
	delete(c.items, entry.key)
	delete(c.expires, entry.key)
	c.usedBytes -= int64(len(entry.key) + entry.value.Len())

	if c.onEvicted != nil {
		c.onEvicted(entry.key, entry.value)
	}
}

// evict removes expired items and/or least recently used items if the cache exceeds its limits.
// Note: lock must be held before calling this function.
func (c *lruCache) evict() {
	// evict expired items first
	now := time.Now()
	for key, expire := range c.expires {
		if now.After(expire) {
			c.removeElement(c.items[key])
		}
	}

	// evict items until within maxBytes
	for c.maxBytes > 0 && c.usedBytes > c.maxBytes {
		// get the least recently used element(head of the list) and remove it
		elem := c.lru.Front()
		if elem != nil {
			c.removeElement(elem)
		}
	}
}

// cleanupLoop runs periodically to clean up expired items.
func (c *lruCache) cleanupLoop() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.mtx.Lock()
			c.evict()
			c.mtx.Unlock()
		case <-c.closeCh:
			return
		}
	}
}

// Close stops the cleanup goroutine and closes the cache.
func (c *lruCache) Close() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
		close(c.closeCh)
	}
}

// GetWithExpiration retrieves the value and expiration duration for the given key.
//
// Parameters:
//   - key: The key to look up
//
// Returns:
//   - Value: The value associated with the key
//   - time.Duration: The remaining time until expiration, or 0 if no expiration
//   - bool: True if the key was found and not expired, false otherwise
func (c *lruCache) GetWithExpiration(key string) (Value, time.Duration, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	elem, ok := c.items[key]
	if !ok {
		return nil, 0, false
	}

	// check expiration
	now := time.Now()
	if expire, isExpired := c.expires[key]; isExpired {
		if now.After(expire) {
			// delete expired item
			return nil, 0, false
		}
		// get remaining expiration duratinon
		remaining := expire.Sub(now)
		c.lru.MoveToBack(elem)
		return elem.Value.(*lruEntry).value, remaining, true
	}
	// if not expiration
	c.lru.MoveToBack(elem)
	return elem.Value.(*lruEntry).value, 0, true
}

// GetExpiration returns the expiration time for the given key.
//
// Parameters:
//   - key: The key to look up
//
// Returns:
//   - time.Time: The expiration time of the key
//   - bool: True if the key has an expiration time, false otherwise
func (c *lruCache) GetExpiration(key string) (time.Time, bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	expire, ok := c.expires[key]
	return expire, ok
}

// UpdateExpiration updates the expiration time for the given key.
//
// Parameters:
//   - key: The key whose expiration time should be updated
//   - expiration: The new expiration duration from now
//
// Returns:
//   - bool: True if the key was found and expiration was updated, false otherwise
func (c *lruCache) UpdateExpiration(key string, expiration time.Duration) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if _, ok := c.items[key]; !ok {
		return false
	}

	if expiration > 0 {
		c.expires[key] = time.Now().Add(expiration)
	} else {
		delete(c.expires, key)
	}
	return true
}

// UsedBytes returns the number of bytes currently used by the cache.
//
// Returns:
//   - int64: The number of bytes currently used
func (c *lruCache) UsedBytes() int64 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.usedBytes
}

// MaxBytes returns the maximum number of bytes the cache can store.
// It represents the upper limit of the cache capacity.
//
// Returns:
//   - int64: The maximum bytes limit of the cache, 0 or negative value means no limit.
func (c *lruCache) MaxBytes() int64 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.maxBytes
}

// SetMaxBytes sets the maximum bytes limit for the cache.
// If the new limit is smaller than the currently used bytes,
// it will evict items until the cache size is within the limit.
// If the new limit is 0 or negative, no eviction will occur.
//
// Parameters:
//   - max: the maximum bytes limit for the cache
func (c *lruCache) SetMaxBytes(max int64) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.maxBytes = max
	if max > 0 {
		c.evict()
	}
}
