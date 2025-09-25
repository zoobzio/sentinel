package sentinel

import (
	"context"
	"reflect"
	"sync"

	"github.com/zoobzio/hookz"
	"github.com/zoobzio/metricz"
	"github.com/zoobzio/tracez"
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

	// Observability
	metrics         *metricz.Registry
	tracer          *tracez.Tracer
	cacheHooks      *hookz.Hooks[CacheEvent]
	extractionHooks *hookz.Hooks[ExtractionEvent]
	registryHooks   *hookz.Hooks[RegistryEvent]
}

// Config holds configuration for a Sentinel instance.
type Config struct {
	// StrictMode causes policy violations to return errors instead of warnings
	StrictMode bool
}

// Inspect returns comprehensive metadata for a type.
func Inspect[T any](ctx context.Context) ModelMetadata {
	// Start span if tracer configured
	var span *tracez.ActiveSpan
	if instance.tracer != nil {
		ctx, span = instance.tracer.StartSpan(ctx, InspectSpan)
		defer span.Finish()
		span.SetTag("type", reflect.TypeOf((*T)(nil)).Elem().String())
	}

	// Track total inspections
	if instance.metrics != nil {
		instance.metrics.Counter(ExtractionsTotal).Inc()
	}

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
	if instance.tracer != nil {
		_, cacheSpan := instance.tracer.StartSpan(ctx, CacheLookupSpan)
		defer cacheSpan.Finish()
	}

	if cached, exists := instance.cache.Get(typeName); exists {
		// Cache hit
		if instance.metrics != nil {
			instance.metrics.Counter(CacheHitsTotal).Inc()
			instance.metrics.Counter(ExtractionsFromCache).Inc()
		}

		// Emit cache hit event
		if instance.cacheHooks != nil {
			// Intentionally ignoring error: hook emission failures (queue full,
			// service closed) should not fail metadata inspection
			_ = instance.cacheHooks.Emit(ctx, "cache.hit", CacheEvent{ //nolint:errcheck
				TypeName:  typeName,
				Operation: "hit",
				Entries:   instance.cache.Size(),
			})
		}

		return cached
	}

	// Cache miss
	if instance.metrics != nil {
		instance.metrics.Counter(CacheMissesTotal).Inc()
	}

	// Emit cache miss event
	if instance.cacheHooks != nil {
		// Intentionally ignoring error: hook emission failures should not
		// fail metadata inspection
		_ = instance.cacheHooks.Emit(ctx, "cache.miss", CacheEvent{ //nolint:errcheck
			TypeName:  typeName,
			Operation: "miss",
			Entries:   instance.cache.Size(),
		})
	}

	// Extract metadata
	metadata := instance.extractMetadata(ctx, t, zero)

	// Store in cache
	if instance.tracer != nil {
		_, storeSpan := instance.tracer.StartSpan(ctx, CacheStoreSpan)
		defer storeSpan.Finish()
	}

	instance.cache.Set(typeName, metadata)

	// Track cache store
	if instance.metrics != nil {
		instance.metrics.Counter(CacheStoresTotal).Inc()
		instance.metrics.Gauge(CacheEntriesCount).Set(float64(instance.cache.Size()))
	}

	// Emit cache store event
	if instance.cacheHooks != nil {
		// Intentionally ignoring error: hook emission failures should not
		// fail metadata inspection
		_ = instance.cacheHooks.Emit(ctx, "cache.store", CacheEvent{ //nolint:errcheck
			TypeName:  typeName,
			Operation: "store",
			Entries:   instance.cache.Size(),
		})
	}

	return metadata
}

// Tag registers a struct tag to be extracted during metadata processing.
// This can be called regardless of seal status.
func Tag(ctx context.Context, tagName string) {
	instance.tagMutex.Lock()
	defer instance.tagMutex.Unlock()

	// Check if already registered
	isNew := !instance.registeredTags[tagName]
	instance.registeredTags[tagName] = true

	// Track metrics
	if instance.metrics != nil && isNew {
		instance.metrics.Counter(RegistryTagsRegistered).Inc()
		instance.metrics.Gauge(RegistryTagsTotal).Set(float64(len(instance.registeredTags)))
	}

	// Emit registry event for new tags
	if instance.registryHooks != nil && isNew {
		// Intentionally ignoring error: hook emission failures should not
		// fail tag registration
		_ = instance.registryHooks.Emit(ctx, "tag.registered", RegistryEvent{ //nolint:errcheck
			Operation: "tag_registered",
			TagName:   tagName,
			TotalTags: len(instance.registeredTags),
		})
	}
}

// Browse returns all type names that have been cached.
func Browse() []string {
	return instance.cache.Keys()
}

// Lookup returns cached metadata for a type name if it exists.
// This allows external packages to access metadata that has already been extracted.
func Lookup(typeName string) (ModelMetadata, bool) {
	return instance.cache.Get(typeName)
}

// Schema returns all cached metadata as a map.
// This is useful for generating documentation, exporting schemas, or analyzing
// the complete type graph of inspected types.
func Schema() map[string]ModelMetadata {
	schema := make(map[string]ModelMetadata)
	for _, typeName := range instance.cache.Keys() {
		if metadata, exists := instance.cache.Get(typeName); exists {
			schema[typeName] = metadata
		}
	}
	return schema
}
