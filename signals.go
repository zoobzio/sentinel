package sentinel

import "github.com/zoobzio/zlog"

// Sentinel signals for observability events.
// These signals allow users to route sentinel's internal events to appropriate sinks.
//
//nolint:revive // ALL_CAPS is idiomatic for signal constants
const (
	// METADATA_EXTRACTED is emitted when type metadata is successfully extracted.
	// Event type: ExtractionEvent
	METADATA_EXTRACTED = zlog.Signal("METADATA_EXTRACTED")

	// CACHE_HIT is emitted when metadata is found in cache.
	// Event type: CacheEvent
	CACHE_HIT = zlog.Signal("CACHE_HIT")

	// CACHE_MISS is emitted when metadata is not found in cache.
	// Event type: CacheEvent
	CACHE_MISS = zlog.Signal("CACHE_MISS")

	// POLICY_APPLIED is emitted when policies modify metadata.
	// Event type: PolicyEvent
	POLICY_APPLIED = zlog.Signal("POLICY_APPLIED")

	// POLICY_VIOLATION is emitted when policy validation fails.
	// Event type: ValidationEvent
	POLICY_VIOLATION = zlog.Signal("POLICY_VIOLATION")

	// TAG_REGISTERED is emitted when a custom tag is registered.
	// Event type: TagEvent
	TAG_REGISTERED = zlog.Signal("TAG_REGISTERED")
)
