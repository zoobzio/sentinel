package sentinel

// Test helpers for backward compatibility
// These use a default global sentinel instance for testing

var defaultSentinel = New().Build()

// Tag registers a tag on the default sentinel (for testing).
func Tag(tagName string) {
	defaultSentinel.Tag(tagName)
}

// InspectDefault uses the default sentinel for testing.
func InspectDefault[T any]() ModelMetadata {
	return Inspect[T](defaultSentinel)
}

// Browse returns all cached types from the default sentinel.
func Browse() []string {
	return defaultSentinel.Browse()
}
