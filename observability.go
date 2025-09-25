package sentinel

import (
	"time"

	"github.com/zoobzio/metricz"
	"github.com/zoobzio/tracez"
)

// Metric keys for cache operations.
const (
	CacheHitsTotal    = metricz.Key("sentinel.cache.hits.total")
	CacheMissesTotal  = metricz.Key("sentinel.cache.misses.total")
	CacheStoresTotal  = metricz.Key("sentinel.cache.stores.total")
	CacheClearsTotal  = metricz.Key("sentinel.cache.clears.total")
	CacheEntriesCount = metricz.Key("sentinel.cache.entries.count")
	CacheSizeBytes    = metricz.Key("sentinel.cache.size.bytes")
)

// Metric keys for extraction operations.
const (
	ExtractionsTotal             = metricz.Key("sentinel.extractions.total")
	ExtractionsFromCache         = metricz.Key("sentinel.extractions.from_cache.total")
	ExtractionDurationMs         = metricz.Key("sentinel.extraction.duration.ms")
	ExtractionFieldsCount        = metricz.Key("sentinel.extraction.fields.count")
	ExtractionTagsCount          = metricz.Key("sentinel.extraction.tags.count")
	ExtractionRelationshipsCount = metricz.Key("sentinel.extraction.relationships.count")
)

// Metric keys for registry operations.
const (
	RegistryTypesDiscovered = metricz.Key("sentinel.registry.types.discovered")
	RegistryTagsRegistered  = metricz.Key("sentinel.registry.tags.registered")
	RegistryTypesTotal      = metricz.Key("sentinel.registry.types.total")
	RegistryTagsTotal       = metricz.Key("sentinel.registry.tags.total")
)

// Span keys for tracing.
const (
	InspectSpan              = tracez.Key("sentinel.inspect")
	CacheLookupSpan          = tracez.Key("sentinel.cache.lookup")
	ExtractMetadataSpan      = tracez.Key("sentinel.extract.metadata")
	ExtractFieldsSpan        = tracez.Key("sentinel.extract.fields")
	ExtractTagsSpan          = tracez.Key("sentinel.extract.tags")
	ExtractRelationshipsSpan = tracez.Key("sentinel.extract.relationships")
	CacheStoreSpan           = tracez.Key("sentinel.cache.store")
)

// CacheEvent represents cache operation events.
type CacheEvent struct {
	TypeName  string        // Type being cached
	Operation string        // "hit", "miss", "store", "clear"
	Size      int           // Size of the cached entry in bytes (estimated)
	Entries   int           // Total cache entries after operation
	Duration  time.Duration // Time taken for the operation
}

// ExtractionEvent represents metadata extraction events.
//
//nolint:govet // Field order optimized for clarity over alignment
type ExtractionEvent struct {
	TypeName      string             // Type being extracted
	PackageName   string             // Package of the type
	FieldCount    int                // Number of fields extracted
	TagCount      int                // Number of unique tags found
	RelationCount int                // Number of relationships discovered
	Relationships []TypeRelationship // The actual relationships found
	Duration      time.Duration      // Total extraction time
	FromCache     bool               // Whether this came from cache
}

// RegistryEvent represents type/tag registration events.
type RegistryEvent struct {
	Operation  string // "type_discovered", "tag_registered"
	TypeName   string // For type discovery events
	TagName    string // For tag registration events
	TotalTypes int    // Total registered types after operation
	TotalTags  int    // Total registered tags after operation
}
