package sentinel

import (
	"time"
)

// Event is the base interface for all sentinel observability events.
// Each event type provides specific data about sentinel's internal operations.
type Event interface {
	EventType() string
}

// SentinelEvent is an alias for Event.
// Using alias to avoid breaking existing code that uses SentinelEvent.
//
//nolint:revive // Keeping alias for backward compatibility
type SentinelEvent = Event

// ExtractionEvent is emitted when metadata is extracted from a type.
//
//nolint:govet // Field order optimized for readability
type ExtractionEvent struct {
	TypeName   string        `json:"type_name"`
	FieldCount int           `json:"field_count"`
	Duration   time.Duration `json:"duration_ms"`
	CacheHit   bool          `json:"cache_hit"`
	Package    string        `json:"package,omitempty"`
}

func (ExtractionEvent) EventType() string { return "extraction" }

// CacheEvent is emitted for cache operations.
type CacheEvent struct {
	TypeName  string `json:"type_name"`
	Operation string `json:"operation"` // "hit", "miss", "set", "clear"
	CacheSize int    `json:"cache_size,omitempty"`
}

func (CacheEvent) EventType() string { return "cache" }

// PolicyEvent is emitted when policies are applied to metadata.
//
//nolint:govet // Field order optimized for readability
type PolicyEvent struct {
	TypeName       string   `json:"type_name"`
	PolicyName     string   `json:"policy_name"`
	FieldsModified int      `json:"fields_modified"`
	TagsApplied    int      `json:"tags_applied"`
	Warnings       []string `json:"warnings,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

func (PolicyEvent) EventType() string { return "policy" }

// ValidationEvent is emitted when validation errors occur during policy enforcement.
type ValidationEvent struct {
	TypeName   string   `json:"type_name"`
	FieldName  string   `json:"field_name,omitempty"`
	PolicyName string   `json:"policy_name"`
	Errors     []string `json:"errors"`
	Fatal      bool     `json:"fatal"`
}

func (ValidationEvent) EventType() string { return "validation" }

// TagEvent is emitted when tags are registered.
type TagEvent struct {
	TagName       string `json:"tag_name"`
	AlreadyExists bool   `json:"already_exists"`
}

func (TagEvent) EventType() string { return "tag" }
