package catalog

import (
	"reflect"
	"time"

	"github.com/zoobzio/zlog"
)

// PUBLIC API - Only two functions exposed

// Inspect returns comprehensive metadata for a type.
// Handles everything internally: cache check, reflection, storage.
// This is the ONLY way to get metadata - always works.
func Inspect[T any]() ModelMetadata {
	var zero T
	t := reflect.TypeOf(zero)

	// Catalog only supports struct types - use concrete types for primitives
	if t != nil && t.Kind() != reflect.Struct {
		if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
			// Allow pointers to structs
			t = t.Elem()
		} else {
			panic("catalog: Select only supports struct types - wrap primitive types in a struct (e.g. type MyStrings []string)")
		}
	}

	typeName := getTypeName(t)

	// Check cache first
	cacheMutex.RLock()
	if cached, exists := metadataCache[typeName]; exists {
		cacheMutex.RUnlock()
		logCacheHit(typeName)
		return cached
	}
	cacheMutex.RUnlock()

	// Log cache miss
	logCacheMiss(typeName)

	// Extract and cache metadata
	start := time.Now()
	metadata := extractMetadata(t, zero)
	logExtraction(typeName, start, len(metadata.Fields))

	cacheMutex.Lock()
	metadataCache[typeName] = metadata
	cacheMutex.Unlock()

	// Log cache storage
	zlog.Emit(MetadataCached, "Metadata cached",
		zlog.String("type", typeName),
		zlog.Int("field_count", len(metadata.Fields)),
	)

	return metadata
}

// Browse returns all type names that have been registered in the catalog.
// Useful for type discovery and debugging.
func Browse() []string {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	types := make([]string, 0, len(metadataCache))
	for typeName := range metadataCache {
		types = append(types, typeName)
	}
	return types
}

// Tag registers a struct tag to be extracted during metadata processing.
// This allows adapters to register their tags with catalog.
func Tag(tagName string) {
	tagMutex.Lock()
	defer tagMutex.Unlock()
	registeredTags[tagName] = true
	logTagRegistration(tagName)
}

// Internal helper functions (not exported)

// ensureMetadata is now incorporated directly into Inspect[T]()
// getByTypeName is removed - only Inspect[T]() should be used
// All other convenience functions removed - users extract from Inspect[T]()

// TypeIngestedEvent represents a type being ingested into the catalog.
// This is a generic event that preserves the type information
//
//nolint:govet // fieldalignment false positive with generics
type TypeIngestedEvent[T any] struct {
	Metadata  ModelMetadata
	TypeName  string
	ZeroValue T // Preserves the generic type
}

// TypeIngestedEventType is the event type for type ingestion.
type TypeIngestedEventType string

const (
	TypeIngested TypeIngestedEventType = "catalog.type_ingested"
)
