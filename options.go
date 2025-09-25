package sentinel

import (
	"github.com/zoobzio/hookz"
	"github.com/zoobzio/metricz"
	"github.com/zoobzio/tracez"
)

// Option configures a Sentinel instance.
type Option func(*Sentinel)

// WithMetrics configures metrics collection.
func WithMetrics(registry *metricz.Registry) Option {
	return func(s *Sentinel) {
		s.metrics = registry
	}
}

// WithTracer configures span collection.
func WithTracer(tracer *tracez.Tracer) Option {
	return func(s *Sentinel) {
		s.tracer = tracer
	}
}

// WithCacheHooks enables cache event hooks.
func WithCacheHooks() Option {
	return func(s *Sentinel) {
		s.cacheHooks = hookz.New[CacheEvent]()
	}
}

// WithExtractionHooks enables extraction event hooks.
func WithExtractionHooks() Option {
	return func(s *Sentinel) {
		s.extractionHooks = hookz.New[ExtractionEvent]()
	}
}

// WithRegistryHooks enables registry event hooks.
func WithRegistryHooks() Option {
	return func(s *Sentinel) {
		s.registryHooks = hookz.New[RegistryEvent]()
	}
}

// WithAllHooks enables all event hooks.
func WithAllHooks() Option {
	return func(s *Sentinel) {
		s.cacheHooks = hookz.New[CacheEvent]()
		s.extractionHooks = hookz.New[ExtractionEvent]()
		s.registryHooks = hookz.New[RegistryEvent]()
	}
}

// Configure applies options to the global sentinel instance.
func Configure(opts ...Option) {
	for _, opt := range opts {
		opt(instance)
	}
}

// CacheHooks returns the cache hooks for registering handlers.
func CacheHooks() *hookz.Hooks[CacheEvent] {
	return instance.cacheHooks
}

// ExtractionHooks returns the extraction hooks for registering handlers.
func ExtractionHooks() *hookz.Hooks[ExtractionEvent] {
	return instance.extractionHooks
}

// RegistryHooks returns the registry hooks for registering handlers.
func RegistryHooks() *hookz.Hooks[RegistryEvent] {
	return instance.registryHooks
}
