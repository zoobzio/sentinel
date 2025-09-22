package sentinel

import (
	"context"
	"reflect"
	"sync"
)

// Global singleton instance.
var instance *Sentinel

// Initialize the global sentinel instance.
func init() {
	// Use PermanentCache since types are immutable at runtime
	instance = &Sentinel{
		cache:          NewPermanentCache(),
		registeredTags: make(map[string]bool),
		config:         Config{StrictMode: false},
	}
}

// Sentinel is the main type intelligence orchestrator.
// It provides metadata extraction and caching.
//
//nolint:govet // Field order is intentional for clarity
type Sentinel struct {
	// Cache for metadata storage
	cache Cache

	// Tag registry for custom tags
	registeredTags map[string]bool

	// Tag registry mutex
	tagMutex sync.RWMutex

	// Configuration
	config Config
}

// Config holds configuration for a Sentinel instance.
type Config struct {
	// StrictMode causes policy violations to return errors instead of warnings
	StrictMode bool
}

// Inspect returns comprehensive metadata for a type.
func Inspect[T any](ctx context.Context) ModelMetadata {

	var zero T
	t := reflect.TypeOf(zero)

	// Sentinel only supports struct types
	if t != nil && t.Kind() != reflect.Struct {
		if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
			t = t.Elem()
		} else {
			panic("sentinel: Inspect only supports struct types")
		}
	}

	typeName := getTypeName(t)

	// Check cache first
	if cached, exists := instance.cache.Get(typeName); exists {
		return cached
	}

	// Extract metadata
	metadata := instance.extractMetadata(ctx, t, zero)
	instance.cache.Set(typeName, metadata)
	return metadata
}

// Tag registers a struct tag to be extracted during metadata processing.
// This can be called regardless of seal status.
func Tag(_ context.Context, tagName string) {
	instance.tagMutex.Lock()
	defer instance.tagMutex.Unlock()

	instance.registeredTags[tagName] = true
}

// Browse returns all type names that have been cached.
func Browse() []string {
	return instance.cache.Keys()
}

// GetCachedMetadata returns cached metadata for a type name if it exists.
// This allows external packages to access metadata that has already been extracted.
func GetCachedMetadata(typeName string) (ModelMetadata, bool) {
	return instance.cache.Get(typeName)
}
