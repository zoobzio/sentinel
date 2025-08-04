package sentinel

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/zoobzio/pipz"
	"github.com/zoobzio/zlog"
)

// Global singleton instance.
var instance *Sentinel

// Initialize the global sentinel instance.
func init() {
	instance = &Sentinel{
		cache:          NewMemoryCache(),
		registeredTags: make(map[string]bool),
		policies:       []Policy{},
		config:         Config{StrictMode: false},
		logger:         zlog.NewLogger[SentinelEvent](),
	}
	instance.pipeline = instance.buildExtractionPipeline()
}

// Sentinel is the main type intelligence orchestrator.
// It provides metadata extraction, caching, and policy enforcement.
//
//nolint:govet // Field order is intentional for clarity
type Sentinel struct {
	// Extraction pipeline
	pipeline *pipz.Sequence[*ExtractionContext]

	// Cache for metadata storage
	cache Cache

	// Tag registry for custom tags
	registeredTags map[string]bool

	// Policies to apply during extraction
	policies []Policy

	// Tag registry mutex
	tagMutex sync.RWMutex

	// Configuration
	config Config

	// Observability logger
	logger *zlog.Logger[SentinelEvent]
}

// Config holds configuration for a Sentinel instance.
type Config struct {
	// Cache implementation (defaults to MemoryCache if nil)
	Cache Cache

	// StrictMode causes policy violations to return errors instead of warnings
	StrictMode bool
}

// Inspect returns comprehensive metadata for a type.
func Inspect[T any]() ModelMetadata {
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
		instance.logger.Emit(CACHE_HIT, "Cache hit for type", CacheEvent{
			TypeName:  typeName,
			Operation: "hit",
		})
		return cached
	}

	// Log cache miss
	instance.logger.Emit(CACHE_MISS, "Cache miss for type", CacheEvent{
		TypeName:  typeName,
		Operation: "miss",
	})

	// Create extraction context
	ec := &ExtractionContext{
		Type:     t,
		Instance: zero,
	}

	// Run through extraction pipeline
	start := time.Now()
	result, err := instance.pipeline.Process(context.Background(), ec)
	if err != nil {
		// In strict mode, return empty metadata with error info
		// In non-strict mode, we could potentially return partial metadata
		if instance.config.StrictMode {
			panic(fmt.Sprintf("sentinel: extraction failed: %v", err))
		}
	}

	metadata := result.Metadata

	// Emit extraction event
	instance.logger.Emit(METADATA_EXTRACTED, "Metadata extracted", ExtractionEvent{
		TypeName:   typeName,
		FieldCount: len(metadata.Fields),
		Duration:   time.Since(start),
		CacheHit:   false,
		Package:    t.PkgPath(),
	})

	instance.cache.Set(typeName, metadata)

	// Emit cache set event
	instance.logger.Emit(CACHE_HIT, "Metadata cached", CacheEvent{
		TypeName:  typeName,
		Operation: "set",
		CacheSize: instance.cache.Size(),
	})

	return metadata
}

// Tag registers a struct tag to be extracted during metadata processing.
func Tag(tagName string) {
	instance.tagMutex.Lock()
	defer instance.tagMutex.Unlock()

	alreadyExists := instance.registeredTags[tagName]
	instance.registeredTags[tagName] = true

	instance.logger.Emit(TAG_REGISTERED, "Tag registered", TagEvent{
		TagName:       tagName,
		AlreadyExists: alreadyExists,
	})
}

// Browse returns all type names that have been cached.
func Browse() []string {
	return instance.cache.Keys()
}

// AddPolicy adds one or more policies to be applied during extraction.
func AddPolicy(policies ...Policy) {
	instance.policies = append(instance.policies, policies...)
	// Rebuild the pipeline to include new policies
	instance.pipeline = instance.buildExtractionPipeline()
}

// SetPolicies replaces all policies with the provided set.
// Useful for refreshing policies from a distributed source.
func SetPolicies(policies []Policy) {
	instance.policies = policies
	// Rebuild the pipeline with new policies
	instance.pipeline = instance.buildExtractionPipeline()
}

// GetPolicies returns the currently configured policies.
func GetPolicies() []Policy {
	return instance.policies
}
