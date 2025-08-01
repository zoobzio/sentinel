package catalog

import (
	"time"

	"github.com/zoobzio/zlog"
)

// Signal represents a zlog signal for observability.
type Signal = zlog.Signal

// Sentinel observability signals for monitoring metadata operations.
const (
	// Core metadata operations.
	MetadataExtracted = zlog.Signal("SENTINEL_METADATA_EXTRACTED")
	MetadataCached    = zlog.Signal("SENTINEL_METADATA_CACHED")
	CacheHit          = zlog.Signal("SENTINEL_CACHE_HIT")
	CacheMiss         = zlog.Signal("SENTINEL_CACHE_MISS")

	// Type registration.
	TagRegistered = zlog.Signal("SENTINEL_TAG_REGISTERED")

	// Performance monitoring.
	SlowExtraction = zlog.Signal("SENTINEL_SLOW_EXTRACTION")
)

// Initialize default observability routing.
func init() {
	// Enable standard logging for sentinel operations
	// Users can set up their own routing in their applications
}

// logExtraction logs metadata extraction with performance metrics.
func logExtraction(typeName string, start time.Time, fieldCount int) {
	duration := time.Since(start)

	zlog.Emit(MetadataExtracted, "Metadata extracted",
		zlog.String("type", typeName),
		zlog.Duration("duration", duration),
		zlog.Int("field_count", fieldCount),
		zlog.Int64("duration_ms", duration.Milliseconds()),
	)

	// Flag slow extractions for investigation
	if duration > 100*time.Millisecond {
		zlog.Emit(SlowExtraction, "Slow metadata extraction detected",
			zlog.String("type", typeName),
			zlog.Duration("duration", duration),
			zlog.Int("field_count", fieldCount),
		)
	}
}

// logCacheHit logs successful cache retrieval.
func logCacheHit(typeName string) {
	zlog.Emit(CacheHit, "Metadata cache hit",
		zlog.String("type", typeName),
	)
}

// logCacheMiss logs cache miss requiring extraction.
func logCacheMiss(typeName string) {
	zlog.Emit(CacheMiss, "Metadata cache miss",
		zlog.String("type", typeName),
	)
}

// logTagRegistration logs tag registration events.
func logTagRegistration(tagName string) {
	zlog.Emit(TagRegistered, "Tag registered",
		zlog.String("tag", tagName),
	)
}
