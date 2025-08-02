package sentinel

import "github.com/zoobzio/zlog"

// Sentinel signals for observability events.
// These signals allow users to route sentinel's internal events to appropriate sinks.
//
//nolint:revive // ALL_CAPS is idiomatic for signal constants
const (
	// Event type: ExtractionEvent.
	METADATA_EXTRACTED = zlog.Signal("METADATA_EXTRACTED")

	// Event type: CacheEvent.
	CACHE_HIT = zlog.Signal("CACHE_HIT")

	// Event type: CacheEvent.
	CACHE_MISS = zlog.Signal("CACHE_MISS")

	// Event type: PolicyEvent.
	POLICY_APPLIED = zlog.Signal("POLICY_APPLIED")

	// Event type: ValidationEvent.
	POLICY_VIOLATION = zlog.Signal("POLICY_VIOLATION")

	// Event type: TagEvent.
	TAG_REGISTERED = zlog.Signal("TAG_REGISTERED")
)
