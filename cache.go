package sentinel

import (
	"sync"
)

// Cache stores extracted metadata permanently.
// Since types are immutable at runtime, entries never expire.
type Cache struct {
	store map[string]Metadata
	mu    sync.RWMutex
}

// NewCache creates a new cache.
func NewCache() *Cache {
	return &Cache{
		store: make(map[string]Metadata),
	}
}

// Get retrieves metadata from the cache.
func (c *Cache) Get(typeName string) (Metadata, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metadata, exists := c.store[typeName]
	return metadata, exists
}

// Set stores metadata in the cache.
func (c *Cache) Set(typeName string, metadata Metadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[typeName] = metadata
}

// Clear removes all entries from the cache.
// This should only be used in tests.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]Metadata)
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.store)
}

// Keys returns all cached type names.
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.store))
	for key := range c.store {
		keys = append(keys, key)
	}
	return keys
}

// All returns a copy of all cached metadata.
func (c *Cache) All() map[string]Metadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]Metadata, len(c.store))
	for k, v := range c.store {
		result[k] = v
	}
	return result
}
