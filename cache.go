package sentinel

import (
	"sync"
)

// Cache defines the interface for metadata storage.
// This allows sentinel to work with different caching strategies.
type Cache interface {
	// Get retrieves metadata for a type name
	Get(typeName string) (ModelMetadata, bool)

	// Set stores metadata for a type name
	Set(typeName string, metadata ModelMetadata)

	// Clear removes all cached metadata
	Clear()

	// Size returns the number of cached entries
	Size() int

	// Keys returns all cached type names
	Keys() []string
}

// MemoryCache is the default in-memory cache implementation.
type MemoryCache struct {
	store map[string]ModelMetadata
	mu    sync.RWMutex
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		store: make(map[string]ModelMetadata),
	}
}

// Get retrieves metadata from the cache.
func (c *MemoryCache) Get(typeName string) (ModelMetadata, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metadata, exists := c.store[typeName]
	return metadata, exists
}

// Set stores metadata in the cache.
func (c *MemoryCache) Set(typeName string, metadata ModelMetadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[typeName] = metadata
}

// Clear removes all entries from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]ModelMetadata)
}

// Size returns the number of cached entries.
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.store)
}

// Keys returns all cached type names.
func (c *MemoryCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.store))
	for key := range c.store {
		keys = append(keys, key)
	}
	return keys
}
