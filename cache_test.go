package sentinel

import (
	"sync"
	"testing"
)

func TestMemoryCache(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		cache := NewMemoryCache()

		// Test empty cache
		if size := cache.Size(); size != 0 {
			t.Errorf("expected empty cache, got size %d", size)
		}

		// Test Get on empty cache
		_, exists := cache.Get("TestType")
		if exists {
			t.Error("expected Get to return false for non-existent type")
		}

		// Test Set and Get
		metadata := ModelMetadata{
			TypeName:    "TestType",
			PackageName: "test",
			Fields: []FieldMetadata{
				{Name: "Field1", Type: "string", Tags: map[string]string{"json": "field1"}},
			},
		}
		cache.Set("TestType", metadata)

		// Verify Get returns the data
		retrieved, exists := cache.Get("TestType")
		if !exists {
			t.Error("expected Get to return true after Set")
		}
		if retrieved.TypeName != metadata.TypeName {
			t.Errorf("expected TypeName %s, got %s", metadata.TypeName, retrieved.TypeName)
		}

		// Test Size
		if size := cache.Size(); size != 1 {
			t.Errorf("expected size 1, got %d", size)
		}
	})

	t.Run("Keys method", func(t *testing.T) {
		cache := NewMemoryCache()

		// Empty cache
		keys := cache.Keys()
		if len(keys) != 0 {
			t.Errorf("expected empty keys, got %v", keys)
		}

		// Add multiple entries
		cache.Set("Type1", ModelMetadata{TypeName: "Type1"})
		cache.Set("Type2", ModelMetadata{TypeName: "Type2"})
		cache.Set("Type3", ModelMetadata{TypeName: "Type3"})

		keys = cache.Keys()
		if len(keys) != 3 {
			t.Errorf("expected 3 keys, got %d", len(keys))
		}

		// Verify all keys are present
		keyMap := make(map[string]bool)
		for _, key := range keys {
			keyMap[key] = true
		}
		for _, expected := range []string{"Type1", "Type2", "Type3"} {
			if !keyMap[expected] {
				t.Errorf("expected key %s not found", expected)
			}
		}
	})

	t.Run("Clear method", func(t *testing.T) {
		cache := NewMemoryCache()

		// Add entries
		cache.Set("Type1", ModelMetadata{TypeName: "Type1"})
		cache.Set("Type2", ModelMetadata{TypeName: "Type2"})

		// Verify they exist
		if size := cache.Size(); size != 2 {
			t.Errorf("expected size 2 before clear, got %d", size)
		}

		// Clear cache
		cache.Clear()

		// Verify empty
		if size := cache.Size(); size != 0 {
			t.Errorf("expected size 0 after clear, got %d", size)
		}

		// Verify Get returns false
		_, exists := cache.Get("Type1")
		if exists {
			t.Error("expected Get to return false after Clear")
		}
	})

	t.Run("overwrite existing entry", func(t *testing.T) {
		cache := NewMemoryCache()

		// Set initial value
		metadata1 := ModelMetadata{
			TypeName: "TestType",
			Fields:   []FieldMetadata{{Name: "Field1"}},
		}
		cache.Set("TestType", metadata1)

		// Overwrite with new value
		metadata2 := ModelMetadata{
			TypeName: "TestType",
			Fields:   []FieldMetadata{{Name: "Field1"}, {Name: "Field2"}},
		}
		cache.Set("TestType", metadata2)

		// Verify new value is stored
		retrieved, _ := cache.Get("TestType")
		if len(retrieved.Fields) != 2 {
			t.Errorf("expected 2 fields after overwrite, got %d", len(retrieved.Fields))
		}

		// Size should still be 1
		if size := cache.Size(); size != 1 {
			t.Errorf("expected size 1 after overwrite, got %d", size)
		}
	})

	t.Run("concurrent access", func(_ *testing.T) {
		cache := NewMemoryCache()
		var wg sync.WaitGroup

		// Concurrent writes
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				typeName := string(rune('A' + n%26))
				cache.Set(typeName, ModelMetadata{TypeName: typeName})
			}(i)
		}

		// Concurrent reads
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				typeName := string(rune('A' + n%26))
				cache.Get(typeName)
			}(i)
		}

		// Concurrent operations
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = cache.Size()
		}()
		go func() {
			defer wg.Done()
			_ = cache.Keys()
		}()
		go func() {
			defer wg.Done()
			cache.Clear()
		}()

		wg.Wait()
		// If we get here without deadlock/panic, concurrent access is safe
	})
}

// TestCacheInterface verifies that MemoryCache implements the Cache interface.
func TestCacheInterface(_ *testing.T) {
	var _ Cache = (*MemoryCache)(nil)
	var _ Cache = NewMemoryCache()
}
