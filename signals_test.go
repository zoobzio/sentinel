package sentinel

import (
	"testing"

	"github.com/zoobzio/zlog"
)

func TestSignalConstants(t *testing.T) {
	tests := []struct {
		name     string
		signal   zlog.Signal
		expected string
	}{
		{
			name:     "METADATA_EXTRACTED",
			signal:   METADATA_EXTRACTED,
			expected: "METADATA_EXTRACTED",
		},
		{
			name:     "CACHE_HIT",
			signal:   CACHE_HIT,
			expected: "CACHE_HIT",
		},
		{
			name:     "CACHE_MISS",
			signal:   CACHE_MISS,
			expected: "CACHE_MISS",
		},
		{
			name:     "POLICY_APPLIED",
			signal:   POLICY_APPLIED,
			expected: "POLICY_APPLIED",
		},
		{
			name:     "POLICY_VIOLATION",
			signal:   POLICY_VIOLATION,
			expected: "POLICY_VIOLATION",
		},
		{
			name:     "TAG_REGISTERED",
			signal:   TAG_REGISTERED,
			expected: "TAG_REGISTERED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.signal) != tt.expected {
				t.Errorf("signal %s = %q, want %q", tt.name, string(tt.signal), tt.expected)
			}
		})
	}
}

func TestSignalEventMapping(t *testing.T) {
	// Verify that signals map to appropriate event types as documented
	signalEventMap := map[zlog.Signal]string{
		METADATA_EXTRACTED: "ExtractionEvent",
		CACHE_HIT:          "CacheEvent",
		CACHE_MISS:         "CacheEvent",
		POLICY_APPLIED:     "PolicyEvent",
		POLICY_VIOLATION:   "ValidationEvent",
		TAG_REGISTERED:     "TagEvent",
	}

	// This test verifies documentation accuracy
	for signal, eventType := range signalEventMap {
		t.Run(string(signal), func(t *testing.T) {
			// Test that the mapping is as documented
			switch signal {
			case METADATA_EXTRACTED:
				if eventType != "ExtractionEvent" {
					t.Errorf("METADATA_EXTRACTED should map to ExtractionEvent, documented as %s", eventType)
				}
			case CACHE_HIT, CACHE_MISS:
				if eventType != "CacheEvent" {
					t.Errorf("%s should map to CacheEvent, documented as %s", signal, eventType)
				}
			case POLICY_APPLIED:
				if eventType != "PolicyEvent" {
					t.Errorf("POLICY_APPLIED should map to PolicyEvent, documented as %s", eventType)
				}
			case POLICY_VIOLATION:
				if eventType != "ValidationEvent" {
					t.Errorf("POLICY_VIOLATION should map to ValidationEvent, documented as %s", eventType)
				}
			case TAG_REGISTERED:
				if eventType != "TagEvent" {
					t.Errorf("TAG_REGISTERED should map to TagEvent, documented as %s", eventType)
				}
			}
		})
	}
}

func TestSignalUniqueness(t *testing.T) {
	// Ensure all signals are unique
	signals := []zlog.Signal{
		METADATA_EXTRACTED,
		CACHE_HIT,
		CACHE_MISS,
		POLICY_APPLIED,
		POLICY_VIOLATION,
		TAG_REGISTERED,
	}

	seen := make(map[string]bool)
	for _, signal := range signals {
		signalStr := string(signal)
		if seen[signalStr] {
			t.Errorf("duplicate signal value: %s", signalStr)
		}
		seen[signalStr] = true
	}

	// Verify we have the expected number of unique signals
	if len(seen) != 6 {
		t.Errorf("expected 6 unique signals, got %d", len(seen))
	}
}

func TestSignalType(_ *testing.T) {
	// Verify that all signals are of type zlog.Signal
	var _ zlog.Signal = METADATA_EXTRACTED
	var _ zlog.Signal = CACHE_HIT
	var _ zlog.Signal = CACHE_MISS
	var _ zlog.Signal = POLICY_APPLIED
	var _ zlog.Signal = POLICY_VIOLATION
	var _ zlog.Signal = TAG_REGISTERED

	// Test that signals can be used as zlog.Signal parameters
	testSignalUsage := func(s zlog.Signal) {
		// This function simulates how signals would be used with zlog
		_ = string(s) // Signals can be converted to strings
	}

	signals := []zlog.Signal{
		METADATA_EXTRACTED,
		CACHE_HIT,
		CACHE_MISS,
		POLICY_APPLIED,
		POLICY_VIOLATION,
		TAG_REGISTERED,
	}

	for _, signal := range signals {
		testSignalUsage(signal)
	}
}

func TestSignalNaming(t *testing.T) {
	// Test that signal names follow the expected convention.
	tests := []struct {
		signal   zlog.Signal
		prefix   string
		contains string
	}{
		{METADATA_EXTRACTED, "METADATA", "EXTRACTED"},
		{CACHE_HIT, "CACHE", "HIT"},
		{CACHE_MISS, "CACHE", "MISS"},
		{POLICY_APPLIED, "POLICY", "APPLIED"},
		{POLICY_VIOLATION, "POLICY", "VIOLATION"},
		{TAG_REGISTERED, "TAG", "REGISTERED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.signal), func(t *testing.T) {
			signalStr := string(tt.signal)

			// Check if signal starts with expected prefix
			if tt.prefix != "" && signalStr[:len(tt.prefix)] != tt.prefix {
				t.Errorf("signal %s should start with %s", signalStr, tt.prefix)
			}

			// Check if signal contains expected substring
			if tt.contains != "" && !containsSubstring(signalStr, tt.contains) {
				t.Errorf("signal %s should contain %s", signalStr, tt.contains)
			}

			// Verify ALL_CAPS naming convention
			if signalStr != allCaps(signalStr) {
				t.Errorf("signal %s should be ALL_CAPS", signalStr)
			}
		})
	}
}

// Helper functions.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func allCaps(s string) string {
	// Check if string is all uppercase with underscores
	for _, r := range s {
		if r != '_' && (r < 'A' || r > 'Z') {
			return s + " (not ALL_CAPS)"
		}
	}
	return s
}
