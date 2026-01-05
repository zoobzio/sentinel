//go:build testing

package sentinel

// Reset clears the cache and tag registry.
// This function is only available when building with -tags testing.
// It is intended for test isolation and should never be used in production.
func Reset() {
	instance.tagMutex.Lock()
	defer instance.tagMutex.Unlock()

	instance.cache = NewCache()
	instance.registeredTags = make(map[string]bool)
}
