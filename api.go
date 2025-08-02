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

// Builder provides a fluent API for constructing a Sentinel instance.
type Builder struct {
	sentinel *Sentinel
}

// New creates a new Sentinel builder.
func New() *Builder {
	s := &Sentinel{
		registeredTags: make(map[string]bool),
		policies:       []Policy{},
		config:         Config{},
		logger:         zlog.NewLogger[SentinelEvent](),
	}

	return &Builder{sentinel: s}
}

// WithCache sets the cache implementation.
func (b *Builder) WithCache(cache Cache) *Builder {
	b.sentinel.cache = cache
	return b
}

// WithPolicy adds one or more policies to be applied during extraction.
func (b *Builder) WithPolicy(policies ...Policy) *Builder {
	b.sentinel.policies = append(b.sentinel.policies, policies...)
	return b
}

// WithStrictMode enables strict policy enforcement (errors instead of warnings).
func (b *Builder) WithStrictMode() *Builder {
	b.sentinel.config.StrictMode = true
	return b
}

// WithHook adds a hook to process specific sentinel events.
// This allows users to listen to internal operations like cache hits, policy applications, etc.
func (b *Builder) WithHook(signal zlog.Signal, hook pipz.Chainable[zlog.Event[SentinelEvent]]) *Builder {
	b.sentinel.logger.Hook(signal, hook)
	return b
}

// WithWatch enables forwarding of sentinel events to the global zlog logger.
// This integrates sentinel's observability with the application's logging system.
func (b *Builder) WithWatch() *Builder {
	b.sentinel.logger.Watch()
	return b
}

// Build creates the configured Sentinel instance.
func (b *Builder) Build() *Sentinel {
	// Set default cache if none provided
	if b.sentinel.cache == nil {
		b.sentinel.cache = NewMemoryCache()
	}

	// Build the extraction pipeline
	b.sentinel.pipeline = b.sentinel.buildExtractionPipeline()

	return b.sentinel
}

// Inspect returns comprehensive metadata for a type.
func Inspect[T any](s *Sentinel) ModelMetadata {
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
	if cached, exists := s.cache.Get(typeName); exists {
		s.logger.Emit(CACHE_HIT, "Cache hit for type", CacheEvent{
			TypeName:  typeName,
			Operation: "hit",
		})
		return cached
	}

	// Log cache miss
	s.logger.Emit(CACHE_MISS, "Cache miss for type", CacheEvent{
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
	result, err := s.pipeline.Process(context.Background(), ec)
	if err != nil {
		// In strict mode, return empty metadata with error info
		// In non-strict mode, we could potentially return partial metadata
		if s.config.StrictMode {
			panic(fmt.Sprintf("sentinel: extraction failed: %v", err))
		}
	}

	metadata := result.Metadata

	// Emit extraction event
	s.logger.Emit(METADATA_EXTRACTED, "Metadata extracted", ExtractionEvent{
		TypeName:   typeName,
		FieldCount: len(metadata.Fields),
		Duration:   time.Since(start),
		CacheHit:   false,
		Package:    t.PkgPath(),
	})

	s.cache.Set(typeName, metadata)

	// Emit cache set event
	s.logger.Emit(CACHE_HIT, "Metadata cached", CacheEvent{
		TypeName:  typeName,
		Operation: "set",
		CacheSize: s.cache.Size(),
	})

	return metadata
}

// Tag registers a struct tag to be extracted during metadata processing.
func (s *Sentinel) Tag(tagName string) {
	s.tagMutex.Lock()
	defer s.tagMutex.Unlock()

	alreadyExists := s.registeredTags[tagName]
	s.registeredTags[tagName] = true

	s.logger.Emit(TAG_REGISTERED, "Tag registered", TagEvent{
		TagName:       tagName,
		AlreadyExists: alreadyExists,
	})
}

// Browse returns all type names that have been cached.
func (s *Sentinel) Browse() []string {
	return s.cache.Keys()
}
