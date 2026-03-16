package cache

import (
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// Cache is a simple in-memory TTL cache with prefix-based invalidation.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// New creates a new Cache with the given TTL for entries.
func New(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get returns the cached value for key if it exists and has not expired.
// Expired entries are lazily deleted.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		// Re-check under write lock to avoid deleting a freshly-Set entry
		if e, ok := c.entries[key]; ok && time.Now().After(e.expiresAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false
	}
	return entry.value, true
}

// Set stores a value with the cache's default TTL.
func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	c.entries[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Invalidate removes all entries whose keys start with any of the given prefixes.
func (c *Cache) Invalidate(prefixes ...string) {
	c.mu.Lock()
	for key := range c.entries {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				delete(c.entries, key)
				break
			}
		}
	}
	c.mu.Unlock()
}

// Clear removes all entries.
func (c *Cache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]cacheEntry)
	c.mu.Unlock()
}
