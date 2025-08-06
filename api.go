package sentinel

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zoobzio/pipz"
	"github.com/zoobzio/zlog"
)

// Loggers provides typed event loggers for observability.
// Users can register sinks directly with these loggers using zlog's native API.
type Loggers struct {
	Extraction *zlog.Logger[ExtractionEvent]
	Cache      *zlog.Logger[CacheEvent]
	Policy     *zlog.Logger[PolicyEvent]
	Admin      *zlog.Logger[AdminEvent]
	Tag        *zlog.Logger[TagEvent]
	Validation *zlog.Logger[ValidationEvent]
}

// Logger provides access to Sentinel's typed event loggers.
// Users can register hooks directly: sentinel.Logger.Extraction.Hook(signal, hook).
var Logger Loggers

// Global singleton instance.
var instance *Sentinel

// Initialize the global sentinel instance.
func init() {
	// Initialize typed loggers
	Logger = Loggers{
		Extraction: zlog.NewLogger[ExtractionEvent](),
		Cache:      zlog.NewLogger[CacheEvent](),
		Policy:     zlog.NewLogger[PolicyEvent](),
		Admin:      zlog.NewLogger[AdminEvent](),
		Tag:        zlog.NewLogger[TagEvent](),
		Validation: zlog.NewLogger[ValidationEvent](),
	}

	// Use PermanentCache since types are immutable at runtime
	instance = &Sentinel{
		cache:          NewPermanentCache(),
		registeredTags: make(map[string]bool),
		policies:       []Policy{},
		config:         Config{StrictMode: false},
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

	// Configuration sealed flag - inspection only allowed after sealing
	configSealed atomic.Bool

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
func Inspect[T any]() ModelMetadata {
	// Ensure configuration is sealed before allowing inspection
	if !instance.configSealed.Load() {
		panic("sentinel: cannot inspect types before configuration is sealed - call admin.Seal() first")
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
	if cached, exists := instance.cache.Get(typeName); exists {
		Logger.Cache.Emit("CACHE_HIT", "Cache hit for type", CacheEvent{
			Timestamp: time.Now(),
			TypeName:  typeName,
			Operation: "hit",
			Reason:    "cached",
			CacheSize: instance.cache.Size(),
		})
		return cached
	}

	// Log cache miss
	Logger.Cache.Emit("CACHE_MISS", "Cache miss for type", CacheEvent{
		Timestamp: time.Now(),
		TypeName:  typeName,
		Operation: "miss",
		Reason:    "first_time",
		CacheSize: instance.cache.Size(),
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
	Logger.Extraction.Emit("METADATA_EXTRACTED", "Metadata extracted", ExtractionEvent{
		TypeName:   typeName,
		FieldCount: len(metadata.Fields),
		Duration:   time.Since(start),
		CacheHit:   false,
		Package:    t.PkgPath(),
		Metadata:   metadata,
		Timestamp:  time.Now(),
	})

	instance.cache.Set(typeName, metadata)

	// Emit cache set event
	Logger.Cache.Emit("CACHE_SET", "Metadata cached", CacheEvent{
		Timestamp: time.Now(),
		TypeName:  typeName,
		Operation: "set",
		Reason:    "stored",
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

	// Calculate usage count by checking all cached metadata
	usageCount := 0
	for _, typeName := range instance.cache.Keys() {
		if metadata, found := instance.cache.Get(typeName); found {
			for _, field := range metadata.Fields {
				if _, hasTag := field.Tags[tagName]; hasTag {
					usageCount++
					break // Count each type only once
				}
			}
		}
	}

	Logger.Tag.Emit("TAG_REGISTERED", "Tag registered", TagEvent{
		Timestamp:     time.Now(),
		TagName:       tagName,
		UsageCount:    usageCount,
		AlreadyExists: alreadyExists,
	})
}

// Browse returns all type names that have been cached.
func Browse() []string {
	return instance.cache.Keys()
}

// Loggers are accessible via the global Logger variable.
// Users can register sinks directly with typed loggers using zlog's native API.
//
// Example:
//
//	hook := pipz.Apply[zlog.Event[ExtractionEvent]]("analytics", func(ctx context.Context, event zlog.Event[ExtractionEvent]) (zlog.Event[ExtractionEvent], error) {
//	    fmt.Printf("Analyzed %s with %d fields\n", event.Data.TypeName, event.Data.FieldCount)
//	    return event, nil
//	})
//	sentinel.Logger.Extraction.Hook("METADATA_EXTRACTED", hook)

// GetCachedMetadata returns cached metadata for a type name if it exists.
// This allows external packages to access metadata that has already been extracted.
func GetCachedMetadata(typeName string) (ModelMetadata, bool) {
	return instance.cache.Get(typeName)
}

// Policy modification functions have been moved to Admin.
// Use sentinel.NewAdmin() to get write access to policies.

// GetPolicies returns a copy of the currently configured policies.
// This is read-only access. Use Admin to modify policies.
func GetPolicies() []Policy {
	// Return a copy to prevent mutation
	policies := make([]Policy, len(instance.policies))
	copy(policies, instance.policies)
	return policies
}

// HasConvention checks if a type implements a specific convention.
func HasConvention[T any](name string) bool {
	metadata := Inspect[T]()
	for _, conv := range metadata.Conventions {
		if conv == name {
			return true
		}
	}
	return false
}

// GetConventions returns all conventions implemented by a type.
func GetConventions[T any]() []string {
	metadata := Inspect[T]()
	return metadata.Conventions
}

// GetClassification returns the classification level for a type.
// Returns empty string if no classification is set.
func GetClassification[T any]() string {
	metadata := Inspect[T]()
	return metadata.Classification
}

// HasClassification checks if a type has any classification set.
func HasClassification[T any]() bool {
	return GetClassification[T]() != ""
}

// GetRelationships returns all relationships from a type to other types.
func GetRelationships[T any]() []TypeRelationship {
	metadata := Inspect[T]()
	return metadata.Relationships
}

// GetReferencedBy returns all types that reference the given type.
// This performs a reverse lookup across all cached metadata.
func GetReferencedBy[T any]() []TypeRelationship {
	var zero T
	t := reflect.TypeOf(zero)
	targetName := getTypeName(t)

	var references []TypeRelationship

	// Search through all cached types
	for _, typeName := range instance.cache.Keys() {
		if metadata, found := instance.cache.Get(typeName); found {
			// Check each relationship in this type
			for _, rel := range metadata.Relationships {
				if rel.To == targetName {
					references = append(references, rel)
				}
			}
		}
	}

	return references
}
