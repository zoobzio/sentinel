package sentinel

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventInterface(t *testing.T) {
	// Verify all event types implement the Event interface
	events := []Event{
		ExtractionEvent{},
		CacheEvent{},
		PolicyEvent{},
		ValidationEvent{},
		TagEvent{},
	}

	expectedTypes := []string{
		"extraction",
		"cache",
		"policy",
		"validation",
		"tag",
	}

	for i, event := range events {
		if event.EventType() != expectedTypes[i] {
			t.Errorf("event %T: expected type %q, got %q", event, expectedTypes[i], event.EventType())
		}
	}
}

func TestExtractionEvent(t *testing.T) {
	event := ExtractionEvent{
		TypeName:   "UserModel",
		FieldCount: 5,
		Duration:   100 * time.Millisecond,
		CacheHit:   false,
		Package:    "github.com/example/models",
	}

	t.Run("fields", func(t *testing.T) {
		if event.TypeName != "UserModel" {
			t.Errorf("expected TypeName 'UserModel', got %s", event.TypeName)
		}
		if event.FieldCount != 5 {
			t.Errorf("expected FieldCount 5, got %d", event.FieldCount)
		}
		if event.Duration != 100*time.Millisecond {
			t.Errorf("expected Duration 100ms, got %v", event.Duration)
		}
		if event.CacheHit {
			t.Error("expected CacheHit false")
		}
		if event.Package != "github.com/example/models" {
			t.Errorf("expected Package 'github.com/example/models', got %s", event.Package)
		}
	})

	t.Run("event type", func(t *testing.T) {
		if event.EventType() != "extraction" {
			t.Errorf("expected EventType 'extraction', got %s", event.EventType())
		}
	})

	t.Run("json marshaling", func(t *testing.T) {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}

		var decoded ExtractionEvent
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if decoded.TypeName != event.TypeName {
			t.Error("TypeName not preserved through JSON marshaling")
		}
		if decoded.Duration != event.Duration {
			t.Error("Duration not preserved through JSON marshaling")
		}
	})
}

func TestCacheEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    CacheEvent
		wantType string
		wantSize bool
	}{
		{
			name: "cache hit",
			event: CacheEvent{
				TypeName:  "UserModel",
				Operation: "hit",
			},
			wantType: "cache",
			wantSize: false,
		},
		{
			name: "cache miss",
			event: CacheEvent{
				TypeName:  "UserModel",
				Operation: "miss",
			},
			wantType: "cache",
			wantSize: false,
		},
		{
			name: "cache set with size",
			event: CacheEvent{
				TypeName:  "UserModel",
				Operation: "set",
				CacheSize: 42,
			},
			wantType: "cache",
			wantSize: true,
		},
		{
			name: "cache clear",
			event: CacheEvent{
				Operation: "clear",
				CacheSize: 0,
			},
			wantType: "cache",
			wantSize: false, // zero value is omitted due to omitempty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.EventType() != tt.wantType {
				t.Errorf("EventType() = %v, want %v", tt.event.EventType(), tt.wantType)
			}

			// Verify JSON marshaling
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			if tt.wantSize && !hasJSONField(data, "cache_size") {
				t.Error("expected cache_size in JSON output")
			}
			if !tt.wantSize && hasJSONField(data, "cache_size") {
				t.Error("unexpected cache_size in JSON output")
			}
		})
	}
}

func TestPolicyEvent(t *testing.T) {
	event := PolicyEvent{
		TypeName:       "UserRequest",
		PolicyName:     "security-policy",
		FieldsModified: 3,
		TagsApplied:    5,
		Warnings:       []string{"deprecated field 'Username'"},
		Errors:         []string{"missing required field 'ID'"},
	}

	t.Run("fields", func(t *testing.T) {
		if event.TypeName != "UserRequest" {
			t.Errorf("expected TypeName 'UserRequest', got %s", event.TypeName)
		}
		if event.PolicyName != "security-policy" {
			t.Errorf("expected PolicyName 'security-policy', got %s", event.PolicyName)
		}
		if event.FieldsModified != 3 {
			t.Errorf("expected FieldsModified 3, got %d", event.FieldsModified)
		}
		if event.TagsApplied != 5 {
			t.Errorf("expected TagsApplied 5, got %d", event.TagsApplied)
		}
		if len(event.Warnings) != 1 {
			t.Errorf("expected 1 warning, got %d", len(event.Warnings))
		}
		if len(event.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(event.Errors))
		}
	})

	t.Run("event type", func(t *testing.T) {
		if event.EventType() != "policy" {
			t.Errorf("expected EventType 'policy', got %s", event.EventType())
		}
	})

	t.Run("empty warnings/errors omitted", func(t *testing.T) {
		emptyEvent := PolicyEvent{
			TypeName:   "Test",
			PolicyName: "test",
		}

		data, err := json.Marshal(emptyEvent)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		if hasJSONField(data, "warnings") {
			t.Error("empty warnings should be omitted from JSON")
		}
		if hasJSONField(data, "errors") {
			t.Error("empty errors should be omitted from JSON")
		}
	})
}

func TestValidationEvent(t *testing.T) {
	event := ValidationEvent{
		TypeName:   "UserModel",
		FieldName:  "Email",
		PolicyName: "validation-policy",
		Errors:     []string{"invalid email format", "missing domain"},
		Fatal:      true,
	}

	t.Run("fields", func(t *testing.T) {
		if event.TypeName != "UserModel" {
			t.Errorf("expected TypeName 'UserModel', got %s", event.TypeName)
		}
		if event.FieldName != "Email" {
			t.Errorf("expected FieldName 'Email', got %s", event.FieldName)
		}
		if event.PolicyName != "validation-policy" {
			t.Errorf("expected PolicyName 'validation-policy', got %s", event.PolicyName)
		}
		if len(event.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(event.Errors))
		}
		if !event.Fatal {
			t.Error("expected Fatal true")
		}
	})

	t.Run("event type", func(t *testing.T) {
		if event.EventType() != "validation" {
			t.Errorf("expected EventType 'validation', got %s", event.EventType())
		}
	})

	t.Run("optional field name", func(t *testing.T) {
		typeEvent := ValidationEvent{
			TypeName:   "UserModel",
			PolicyName: "type-policy",
			Errors:     []string{"missing required fields"},
			Fatal:      false,
		}

		data, err := json.Marshal(typeEvent)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Field name should be omitted when empty
		if hasJSONField(data, "field_name") {
			t.Error("empty field_name should be omitted from JSON")
		}
	})
}

func TestTagEvent(t *testing.T) {
	t.Run("new tag", func(t *testing.T) {
		event := TagEvent{
			TagName:       "custom",
			AlreadyExists: false,
		}

		if event.TagName != "custom" {
			t.Errorf("expected TagName 'custom', got %s", event.TagName)
		}
		if event.AlreadyExists {
			t.Error("expected AlreadyExists false for new tag")
		}
		if event.EventType() != "tag" {
			t.Errorf("expected EventType 'tag', got %s", event.EventType())
		}
	})

	t.Run("existing tag", func(t *testing.T) {
		event := TagEvent{
			TagName:       "json",
			AlreadyExists: true,
		}

		if !event.AlreadyExists {
			t.Error("expected AlreadyExists true for existing tag")
		}
	})
}

func TestSentinelEventAlias(t *testing.T) {
	// Verify that SentinelEvent is an alias for Event.
	var _ SentinelEvent = ExtractionEvent{}
	var _ SentinelEvent = CacheEvent{}
	var _ SentinelEvent = PolicyEvent{}
	var _ SentinelEvent = ValidationEvent{}
	var _ SentinelEvent = TagEvent{}

	// Both types should be interchangeable.
	var e1 Event = ExtractionEvent{}
	//nolint:revive // Testing that type alias assignment works correctly
	var e2 SentinelEvent = e1 // Direct assignment should work since it's an alias

	if e2.EventType() != "extraction" {
		t.Error("SentinelEvent alias not working correctly")
	}
}

// Helper function to check if a JSON field exists.
func hasJSONField(data []byte, field string) bool {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	_, exists := m[field]
	return exists
}
