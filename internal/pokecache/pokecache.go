package pokecache

import (
	"sync"
	"time"
)

type CacheEntry struct {
	createdAt time.Time
	val       []byte
}

type Cache struct {
	mu                sync.Mutex
	items             map[string]CacheEntry
	defaultExpiration time.Duration
	cleanupInterval   time.Duration
}

func NewCache(interval time.Duration) *Cache {
	c := &Cache{
		items:           make(map[string]CacheEntry),
		cleanupInterval: interval,
	}
	go c.reapLoop()
	return c
}

func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	createdAt := time.Now()
	c.items[key] = CacheEntry{
		createdAt: createdAt,
		val:       val,
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	return item.val, true
}

func (c *Cache) reapLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	for range ticker.C {
		c.deleteExpired()
	}
}

func (c *Cache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	//now := time.Now().UnixNano()

	for key, item := range c.items {
		elapsed := time.Since(item.createdAt)
		if elapsed > c.cleanupInterval {
			delete(c.items, key)
		}
	}

}
