package sentinel

import (
	"testing"

	"github.com/zoobzio/hookz"
	"github.com/zoobzio/metricz"
	"github.com/zoobzio/tracez"
)

func TestWithMetrics(t *testing.T) {
	// Create test registry
	registry := &metricz.Registry{}

	// Create new instance with metrics
	s := &Sentinel{
		cache: NewMemoryCache(),
	}

	WithMetrics(registry)(s)

	if s.metrics != registry {
		t.Error("expected metrics to be set")
	}
}

func TestWithTracer(t *testing.T) {
	tracer := &tracez.Tracer{}

	s := &Sentinel{
		cache: NewMemoryCache(),
	}

	WithTracer(tracer)(s)

	if s.tracer != tracer {
		t.Error("expected tracer to be set")
	}
}

func TestWithCacheHooks(t *testing.T) {
	s := &Sentinel{
		cache: NewMemoryCache(),
	}

	WithCacheHooks()(s)

	if s.cacheHooks == nil {
		t.Fatal("expected cache hooks to be set")
	}

	// Test that hooks are of correct type
	var _ *hookz.Hooks[CacheEvent] = s.cacheHooks
}

func TestWithExtractionHooks(t *testing.T) {
	s := &Sentinel{
		cache: NewMemoryCache(),
	}

	WithExtractionHooks()(s)

	if s.extractionHooks == nil {
		t.Fatal("expected extraction hooks to be set")
	}

	// Test that hooks are of correct type
	var _ *hookz.Hooks[ExtractionEvent] = s.extractionHooks
}

func TestWithRegistryHooks(t *testing.T) {
	s := &Sentinel{
		cache: NewMemoryCache(),
	}

	WithRegistryHooks()(s)

	if s.registryHooks == nil {
		t.Fatal("expected registry hooks to be set")
	}

	// Test that hooks are of correct type
	var _ *hookz.Hooks[RegistryEvent] = s.registryHooks
}

func TestWithAllHooks(t *testing.T) {
	s := &Sentinel{
		cache: NewMemoryCache(),
	}

	WithAllHooks()(s)

	if s.cacheHooks == nil {
		t.Error("expected cache hooks to be set")
	}

	if s.extractionHooks == nil {
		t.Error("expected extraction hooks to be set")
	}

	if s.registryHooks == nil {
		t.Error("expected registry hooks to be set")
	}
}

func TestConfigure(t *testing.T) {
	// Save original instance
	originalInstance := instance
	defer func() {
		instance = originalInstance
	}()

	// Create fresh instance
	instance = &Sentinel{
		cache:          NewMemoryCache(),
		registeredTags: make(map[string]bool),
	}

	// Test multiple options
	registry := &metricz.Registry{}
	tracer := &tracez.Tracer{}

	Configure(WithMetrics(registry), WithTracer(tracer))

	// Verify global instance was configured
	if instance.metrics != registry {
		t.Error("expected global instance metrics to be set")
	}

	if instance.tracer != tracer {
		t.Error("expected global instance tracer to be set")
	}
}

func TestHookGetters(t *testing.T) {
	// Save original instance
	originalInstance := instance
	defer func() {
		instance = originalInstance
	}()

	// Create instance with all hooks
	instance = &Sentinel{
		cache:           NewMemoryCache(),
		registeredTags:  make(map[string]bool),
		cacheHooks:      hookz.New[CacheEvent](),
		extractionHooks: hookz.New[ExtractionEvent](),
		registryHooks:   hookz.New[RegistryEvent](),
	}

	// Test CacheHooks getter
	if hooks := CacheHooks(); hooks == nil {
		t.Error("expected CacheHooks to return non-nil")
	} else if hooks != instance.cacheHooks {
		t.Error("CacheHooks should return instance.cacheHooks")
	}

	// Test ExtractionHooks getter
	if hooks := ExtractionHooks(); hooks == nil {
		t.Error("expected ExtractionHooks to return non-nil")
	} else if hooks != instance.extractionHooks {
		t.Error("ExtractionHooks should return instance.extractionHooks")
	}

	// Test RegistryHooks getter
	if hooks := RegistryHooks(); hooks == nil {
		t.Error("expected RegistryHooks to return non-nil")
	} else if hooks != instance.registryHooks {
		t.Error("RegistryHooks should return instance.registryHooks")
	}

	// Test with nil hooks
	instance = &Sentinel{
		cache:          NewMemoryCache(),
		registeredTags: make(map[string]bool),
	}

	if CacheHooks() != nil {
		t.Error("expected nil CacheHooks when not set")
	}

	if ExtractionHooks() != nil {
		t.Error("expected nil ExtractionHooks when not set")
	}

	if RegistryHooks() != nil {
		t.Error("expected nil RegistryHooks when not set")
	}
}
