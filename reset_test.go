//go:build testing

package sentinel

import "testing"

func TestReset(t *testing.T) {
	t.Run("clears cache and tag registry", func(t *testing.T) {
		// Populate cache with some types
		Inspect[SimpleStruct]()
		Inspect[TestUser]()

		// Register a custom tag
		Tag("testreset")

		// Verify cache has entries
		if instance.cache.Size() == 0 {
			t.Fatal("expected cache to have entries before reset")
		}

		// Verify tag was registered
		instance.tagMutex.RLock()
		_, tagExists := instance.registeredTags["testreset"]
		instance.tagMutex.RUnlock()
		if !tagExists {
			t.Fatal("expected tag to be registered before reset")
		}

		// Call Reset
		Reset()

		// Verify cache is cleared
		if instance.cache.Size() != 0 {
			t.Errorf("expected cache to be empty after reset, got %d entries", instance.cache.Size())
		}

		// Verify tag registry is cleared
		instance.tagMutex.RLock()
		_, tagStillExists := instance.registeredTags["testreset"]
		instance.tagMutex.RUnlock()
		if tagStillExists {
			t.Error("expected tag registry to be cleared after reset")
		}

		// Verify Browse returns empty
		types := Browse()
		if len(types) != 0 {
			t.Errorf("expected Browse to return empty after reset, got %d types", len(types))
		}
	})
}
