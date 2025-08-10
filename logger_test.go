package sentinel

import (
	"context"
	"testing"

	"github.com/zoobzio/pipz"
	"github.com/zoobzio/zlog"
)

func TestLogger(t *testing.T) {
	t.Run("TypedLoggersAreAccessible", func(t *testing.T) {
		if Logger.Extraction == nil {
			t.Error("Expected Extraction logger to be accessible")
		}
		if Logger.Cache == nil {
			t.Error("Expected Cache logger to be accessible")
		}
		if Logger.Policy == nil {
			t.Error("Expected Policy logger to be accessible")
		}
		if Logger.Admin == nil {
			t.Error("Expected Admin logger to be accessible")
		}
		if Logger.Tag == nil {
			t.Error("Expected Tag logger to be accessible")
		}
		if Logger.Validation == nil {
			t.Error("Expected Validation logger to be accessible")
		}
	})

	t.Run("CanRegisterSinkForExtractionEvents", func(t *testing.T) {
		// Setup
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}
		admin.Seal()

		// User registers a hook to capture extraction events
		var capturedEvent ExtractionEvent
		var eventFired bool

		// Users create pipz processors for hooks
		hook := pipz.Apply[zlog.Event[ExtractionEvent]]("test-hook", func(_ context.Context, event zlog.Event[ExtractionEvent]) (zlog.Event[ExtractionEvent], error) {
			capturedEvent = event.Data
			eventFired = true
			return event, nil
		})

		// Users register hooks directly with the typed logger
		Logger.Extraction.Hook("METADATA_EXTRACTED", hook)

		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		// This should trigger METADATA_EXTRACTED
		Inspect[TestStruct]()

		// Give sink time to process
		// Note: In real usage, sinks process asynchronously
		if !eventFired {
			t.Fatal("Expected extraction event to fire")
		}

		// Verify complete event data
		if capturedEvent.TypeName != "TestStruct" {
			t.Errorf("Expected TypeName 'TestStruct', got '%s'", capturedEvent.TypeName)
		}

		if capturedEvent.FieldCount != 2 {
			t.Errorf("Expected FieldCount 2, got %d", capturedEvent.FieldCount)
		}

		if capturedEvent.CacheHit != false {
			t.Error("Expected CacheHit to be false for first analysis")
		}

		if capturedEvent.Duration <= 0 {
			t.Error("Expected Duration to be positive")
		}

		// Verify enhanced fields
		if capturedEvent.Timestamp.IsZero() {
			t.Error("Expected Timestamp to be set")
		}

		// Verify complete metadata is included
		if len(capturedEvent.Metadata.Fields) != 2 {
			t.Errorf("Expected 2 fields in metadata, got %d", len(capturedEvent.Metadata.Fields))
		}

		// Verify struct tags are accessible
		var nameField, ageField *FieldMetadata
		for i, field := range capturedEvent.Metadata.Fields {
			if field.Name == "Name" {
				nameField = &capturedEvent.Metadata.Fields[i]
			} else if field.Name == "Age" {
				ageField = &capturedEvent.Metadata.Fields[i]
			}
		}

		if nameField == nil || ageField == nil {
			t.Fatal("Expected to find Name and Age fields in metadata")
		}

		// Check tags are accessible in the event data
		if nameField.Tags["json"] != "name" {
			t.Errorf("Expected Name field json tag 'name', got '%s'", nameField.Tags["json"])
		}

		if ageField.Tags["json"] != "age" {
			t.Errorf("Expected Age field json tag 'age', got '%s'", ageField.Tags["json"])
		}
	})

	t.Run("CanRegisterSinkForCacheEvents", func(t *testing.T) {
		// Setup
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}
		admin.Seal()

		var cacheEvents []CacheEvent

		// Create pipz processor for cache events
		hook := pipz.Apply[zlog.Event[CacheEvent]]("cache-hook", func(_ context.Context, event zlog.Event[CacheEvent]) (zlog.Event[CacheEvent], error) {
			cacheEvents = append(cacheEvents, event.Data)
			return event, nil
		})

		// Register hook for both cache events
		Logger.Cache.Hook("CACHE_HIT", hook)
		Logger.Cache.Hook("CACHE_MISS", hook)

		type CacheTestStruct struct {
			Data string
		}

		// First call: cache miss + cache set
		Inspect[CacheTestStruct]()

		// Second call: cache hit
		Inspect[CacheTestStruct]()

		// Should have captured cache events
		if len(cacheEvents) < 2 {
			t.Errorf("Expected at least 2 cache events, got %d", len(cacheEvents))
		}

		// Verify event types and enhanced data
		foundMiss := false
		foundHit := false

		for _, event := range cacheEvents {
			if event.TypeName == "CacheTestStruct" {
				if event.Operation == "miss" {
					foundMiss = true
				}
				if event.Operation == "hit" || event.Operation == "set" {
					foundHit = true
				}
				// Verify enhanced fields
				if event.Timestamp.IsZero() {
					t.Error("Expected CacheEvent Timestamp to be set")
				}
			}
		}

		if !foundMiss {
			t.Error("Expected to find cache miss event")
		}
		if !foundHit {
			t.Error("Expected to find cache hit or set event")
		}
	})

	t.Run("CanRegisterSinkForAdminEvents", func(t *testing.T) {
		// Setup
		resetAdminForTesting()

		var adminEvents []AdminEvent

		// Create pipz processor for admin events
		hook := pipz.Apply[zlog.Event[AdminEvent]]("admin-hook", func(_ context.Context, event zlog.Event[AdminEvent]) (zlog.Event[AdminEvent], error) {
			adminEvents = append(adminEvents, event.Data)
			return event, nil
		})

		Logger.Admin.Hook("ADMIN_ACTION", hook)

		// These should trigger admin events
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		policy := Policy{
			Name:     "AdminTestPolicy",
			Policies: []TypePolicy{{Match: "*", Classification: "public"}},
		}
		admin.AddPolicy(policy)
		admin.Seal()

		// Should have captured: policy_added, sealed
		if len(adminEvents) < 2 {
			t.Errorf("Expected at least 2 admin events, got %d", len(adminEvents))
		}

		// Verify event data
		foundPolicyAdded := false
		foundSealed := false

		for _, event := range adminEvents {
			if event.Action == "policy_added" {
				foundPolicyAdded = true
			} else if event.Action == "sealed" {
				foundSealed = true
			}

			// Verify enhanced fields
			if event.Timestamp.IsZero() {
				t.Error("Expected AdminEvent Timestamp to be set")
			}
			if event.PolicyCount <= 0 {
				t.Errorf("Expected positive PolicyCount, got %d", event.PolicyCount)
			}
		}

		if !foundPolicyAdded {
			t.Error("Expected to find policy_added admin event")
		}
		if !foundSealed {
			t.Error("Expected to find sealed admin event")
		}
	})
}
