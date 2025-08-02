package sentinel

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/zoobzio/pipz"
	"github.com/zoobzio/zlog"
)

// TestObservability demonstrates how to use sentinel's observability features.
func TestObservability(t *testing.T) {
	// Create a test sink to capture events
	var mu sync.Mutex
	var capturedEvents []SentinelEvent

	// Create a hook that captures sentinel events
	captureHook := pipz.Apply[zlog.Event[SentinelEvent]]("capture",
		func(_ context.Context, event zlog.Event[SentinelEvent]) (zlog.Event[SentinelEvent], error) {
			mu.Lock()
			capturedEvents = append(capturedEvents, event.Data)
			mu.Unlock()
			return event, nil
		})

	// Create sentinel with observability hooks
	s := New().
		WithHook(METADATA_EXTRACTED, captureHook).
		WithHook(CACHE_HIT, captureHook).
		WithHook(CACHE_MISS, captureHook).
		WithHook(TAG_REGISTERED, captureHook).
		Build()

	// Register a custom tag
	s.Tag("custom")

	// First inspection (cache miss)
	type TestStruct struct {
		Field1 string `json:"field1" custom:"value"`
		Field2 int    `json:"field2"`
	}

	_ = Inspect[TestStruct](s)

	// Second inspection (cache hit)
	_ = Inspect[TestStruct](s)

	// Check captured events
	mu.Lock()
	defer mu.Unlock()

	if len(capturedEvents) != 5 {
		t.Fatalf("expected 5 events, got %d", len(capturedEvents))
	}

	// Verify event sequence
	expectedTypes := []string{"tag", "cache", "extraction", "cache", "cache"}
	for i, event := range capturedEvents {
		if event.EventType() != expectedTypes[i] {
			t.Errorf("event %d: expected type %s, got %s", i, expectedTypes[i], event.EventType())
		}
	}

	// Verify specific events
	tagEvent, ok := capturedEvents[0].(TagEvent)
	if !ok {
		t.Fatal("expected TagEvent")
	}
	if tagEvent.TagName != "custom" {
		t.Errorf("expected tag name 'custom', got %s", tagEvent.TagName)
	}

	cacheMissEvent, ok := capturedEvents[1].(CacheEvent)
	if !ok {
		t.Fatal("expected CacheEvent")
	}
	if cacheMissEvent.Operation != "miss" {
		t.Errorf("expected cache miss, got %s", cacheMissEvent.Operation)
	}

	extractionEvent, ok := capturedEvents[2].(ExtractionEvent)
	if !ok {
		t.Fatal("expected ExtractionEvent")
	}
	if extractionEvent.FieldCount != 2 {
		t.Errorf("expected 2 fields, got %d", extractionEvent.FieldCount)
	}
	if extractionEvent.Duration == 0 {
		t.Error("expected non-zero extraction duration")
	}

	cacheHitEvent, ok := capturedEvents[4].(CacheEvent)
	if !ok {
		t.Fatal("expected CacheEvent")
	}
	if cacheHitEvent.Operation != "hit" {
		t.Errorf("expected cache hit, got %s", cacheHitEvent.Operation)
	}
}

// TestObservabilityWithWatch demonstrates forwarding events to global logger.
func TestObservabilityWithWatch(_ *testing.T) {
	// This test shows how WithWatch() would forward events to the global logger
	s := New().
		WithWatch(). // Forward all events to global zlog
		Build()

	// Any operations will now emit events both to the typed logger
	// and to the global zlog system
	type User struct {
		Name string `json:"name"`
	}

	_ = Inspect[User](s)

	// In a real application, these events would appear in your global logs
	// alongside other application events, with full context propagation
}

// TestPolicyObservability demonstrates policy-related events.
func TestPolicyObservability(t *testing.T) {
	var policyEvents []PolicyEvent
	var validationEvents []ValidationEvent

	// Capture policy events
	policyHook := pipz.Apply[zlog.Event[SentinelEvent]]("policy-capture",
		func(_ context.Context, event zlog.Event[SentinelEvent]) (zlog.Event[SentinelEvent], error) {
			if pe, ok := event.Data.(PolicyEvent); ok {
				policyEvents = append(policyEvents, pe)
			}
			if ve, ok := event.Data.(ValidationEvent); ok {
				validationEvents = append(validationEvents, ve)
			}
			return event, nil
		})

	// Create sentinel with strict policy
	s := New().
		WithHook(POLICY_APPLIED, policyHook).
		WithHook(POLICY_VIOLATION, policyHook).
		WithPolicy(Policy{
			Name: "test-policy",
			Policies: []TypePolicy{
				{
					Match: "*Request",
					Ensure: map[string]string{
						"ID": "string",
					},
				},
			},
		}).
		WithStrictMode().
		Build()

	// Type that violates policy
	type BadRequest struct {
		Name string // Missing required ID field
	}

	// This should trigger policy violation event
	defer func() {
		if r := recover(); r != nil {
			// Expected panic from strict mode
			if len(validationEvents) != 1 {
				t.Error("expected validation event")
			}
			if !validationEvents[0].Fatal {
				t.Error("expected fatal validation")
			}
		}
	}()

	Inspect[BadRequest](s)
}

// ExampleLogger shows how to use sentinel's observability features.
func ExampleLogger() {
	// Create a metrics hook
	metricsHook := pipz.Apply[zlog.Event[SentinelEvent]]("metrics",
		func(_ context.Context, event zlog.Event[SentinelEvent]) (zlog.Event[SentinelEvent], error) {
			switch e := event.Data.(type) {
			case ExtractionEvent:
				fmt.Printf("Extraction took %v for type %s\n", e.Duration, e.TypeName)
			case CacheEvent:
				fmt.Printf("Cache %s for type %s\n", e.Operation, e.TypeName)
			}
			return event, nil
		})

	// Build sentinel with monitoring
	s := New().
		WithHook(METADATA_EXTRACTED, metricsHook).
		WithHook(CACHE_HIT, metricsHook).
		WithHook(CACHE_MISS, metricsHook).
		WithWatch(). // Also forward to global logger
		Build()

	// Use sentinel - all operations are now observable
	_ = s
}
