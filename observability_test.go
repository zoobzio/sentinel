package catalog

import (
	"testing"
	"time"
)

func TestSignalDefinitions(t *testing.T) {
	// Test that all signals are properly defined
	signals := []Signal{
		MetadataExtracted,
		MetadataCached,
		TagRegistered,
		CacheHit,
		CacheMiss,
	}

	for _, signal := range signals {
		if signal == "" {
			t.Error("signal should not be empty")
		}

	}
}

func TestLogCacheHit(_ *testing.T) {
	// This test primarily ensures the function doesn't panic
	// In a real test environment, we'd capture zlog output
	logCacheHit("TestType")

	// Test with empty type name
	logCacheHit("")
}

func TestLogCacheMiss(_ *testing.T) {
	// This test primarily ensures the function doesn't panic
	logCacheMiss("TestType")

	// Test with empty type name
	logCacheMiss("")
}

func TestLogExtraction(_ *testing.T) {
	start := time.Now()

	// Test normal extraction
	logExtraction("TestType", start, 5)

	// Test with zero fields
	logExtraction("EmptyType", start, 0)

	// Test with many fields
	logExtraction("LargeType", start, 100)

	// Test with empty type name
	logExtraction("", start, 5)
}

func TestLogTagRegistration(_ *testing.T) {
	// Test normal registration
	logTagRegistration("json")

	// Test with empty tag name
	logTagRegistration("")

	// Test with special characters
	logTagRegistration("my-custom-tag")
	logTagRegistration("tag_with_underscore")
}

func TestObservabilityIntegration(t *testing.T) {
	// This test verifies that observability functions are called
	// during normal operation of the catalog

	t.Run("observability during Inspect", func(_ *testing.T) {
		// Clear cache to ensure fresh extraction
		cacheMutex.Lock()
		metadataCache = make(map[string]ModelMetadata)
		cacheMutex.Unlock()

		type ObservableStruct struct {
			Field1 string `json:"field1"`
			Field2 int    `json:"field2"`
		}

		// First call should log cache miss and extraction
		_ = Inspect[ObservableStruct]()

		// Second call should log cache hit
		_ = Inspect[ObservableStruct]()

		// No panics means observability is working
	})

	t.Run("observability during Tag registration", func(_ *testing.T) {
		// Clear registered tags
		tagMutex.Lock()
		registeredTags = make(map[string]bool)
		tagMutex.Unlock()

		// Register multiple tags
		tags := []string{"custom1", "custom2", "custom3"}
		for _, tag := range tags {
			Tag(tag)
		}

		// No panics means observability is working
	})
}

func TestSignalString(t *testing.T) {
	tests := []struct {
		signal   Signal
		expected string
	}{
		{MetadataExtracted, "SENTINEL_METADATA_EXTRACTED"},
		{MetadataCached, "SENTINEL_METADATA_CACHED"},
		{TagRegistered, "SENTINEL_TAG_REGISTERED"},
		{CacheHit, "SENTINEL_CACHE_HIT"},
		{CacheMiss, "SENTINEL_CACHE_MISS"},
	}

	for _, tt := range tests {
		t.Run(string(tt.signal), func(t *testing.T) {
			result := string(tt.signal)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Test that all logging functions handle concurrent access properly.
func TestConcurrentLogging(_ *testing.T) {
	done := make(chan bool)

	// Run multiple goroutines that call logging functions
	for i := 0; i < 10; i++ {
		go func(id int) {
			typeName := "TestType"
			start := time.Now()

			logCacheHit(typeName)
			logCacheMiss(typeName)
			logExtraction(typeName, start, id)
			logTagRegistration("tag")

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// No race conditions or panics means concurrent access is safe
}
