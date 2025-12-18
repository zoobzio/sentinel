// Package sentinel provides struct metadata extraction and relationship discovery for Go.
package sentinel

import (
	"errors"
	"reflect"
	"runtime/debug"
	"sync"
)

// ErrNotStruct is returned when a non-struct type is passed to Try* functions.
var ErrNotStruct = errors.New("sentinel: only struct types are supported")

// Global singleton instance.
var instance *Sentinel

// Initialize the global sentinel instance.
func init() {
	// Use PermanentCache since types are immutable at runtime
	instance = &Sentinel{
		cache:          NewPermanentCache(),
		registeredTags: make(map[string]bool),
		modulePath:     detectModulePath(),
	}
}

// detectModulePath returns the module path from build info, or empty string if unavailable.
func detectModulePath() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Path
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

	// Module path from build info (e.g., "github.com/user/repo")
	modulePath string
}

// Inspect returns comprehensive metadata for a type.
// Panics if T is not a struct type.
func Inspect[T any]() Metadata {
	metadata, err := TryInspect[T]()
	if err != nil {
		panic(err)
	}
	return metadata
}

// TryInspect returns comprehensive metadata for a type.
// Returns ErrNotStruct if T is not a struct type.
func TryInspect[T any]() (Metadata, error) {
	var zero T
	t := reflect.TypeOf(zero)

	// Sentinel only supports struct types
	if t != nil && t.Kind() != reflect.Struct {
		if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
			t = t.Elem()
		} else {
			return Metadata{}, ErrNotStruct
		}
	}

	typeName := getTypeName(t)

	// Check cache first
	if cached, exists := instance.cache.Get(typeName); exists {
		return cached, nil
	}

	// Extract metadata
	metadata := instance.extractMetadata(t)

	// Store in cache
	instance.cache.Set(typeName, metadata)

	return metadata, nil
}

// Scan performs recursive inspection of a type and all related types within the same module.
// Unlike Inspect which only processes a single type, Scan will follow relationships and
// automatically inspect any related types that share the same module root.
// Panics if T is not a struct type.
func Scan[T any]() Metadata {
	metadata, err := TryScan[T]()
	if err != nil {
		panic(err)
	}
	return metadata
}

// TryScan performs recursive inspection of a type and all related types within the same module.
// Unlike TryInspect which only processes a single type, TryScan will follow relationships and
// automatically inspect any related types that share the same module root.
// Returns ErrNotStruct if T is not a struct type.
func TryScan[T any]() (Metadata, error) {
	var zero T
	t := reflect.TypeOf(zero)

	// Sentinel only supports struct types
	if t != nil && t.Kind() != reflect.Struct {
		if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
			t = t.Elem()
		} else {
			return Metadata{}, ErrNotStruct
		}
	}

	// Use a visited map to prevent infinite loops from circular references
	visited := make(map[string]bool)
	instance.scanWithVisited(t, visited)

	// Return the metadata for the root type
	metadata, _ := instance.cache.Get(getTypeName(t))
	return metadata, nil
}

// Tag registers a struct tag to be extracted during metadata processing.
// This can be called regardless of seal status.
func Tag(tagName string) {
	instance.tagMutex.Lock()
	defer instance.tagMutex.Unlock()

	instance.registeredTags[tagName] = true
}

// Browse returns all type names that have been cached.
func Browse() []string {
	return instance.cache.Keys()
}

// Lookup returns cached metadata for a type name if it exists.
// This allows external packages to access metadata that has already been extracted.
func Lookup(typeName string) (Metadata, bool) {
	return instance.cache.Get(typeName)
}

// Schema returns all cached metadata as a map.
// This is useful for generating documentation, exporting schemas, or analyzing
// the complete type graph of inspected types.
func Schema() map[string]Metadata {
	schema := make(map[string]Metadata)
	for _, typeName := range instance.cache.Keys() {
		if metadata, exists := instance.cache.Get(typeName); exists {
			schema[typeName] = metadata
		}
	}
	return schema
}

// Reset clears the cache and tag registry.
// This is primarily useful for test isolation.
func Reset() {
	instance.tagMutex.Lock()
	defer instance.tagMutex.Unlock()

	instance.cache = NewPermanentCache()
	instance.registeredTags = make(map[string]bool)
}
