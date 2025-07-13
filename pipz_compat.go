package catalog

import (
	"fmt"
	"sync"
)

// ServiceContract provides backward compatibility with the old pipz key-based API
type ServiceContract[K comparable, I, O any] struct {
	processors map[K]func(I) (O, error)
	mutex      sync.RWMutex
}

// NewServiceContract creates a new backward-compatible service contract
func NewServiceContract[K comparable, I, O any]() *ServiceContract[K, I, O] {
	return &ServiceContract[K, I, O]{
		processors: make(map[K]func(I) (O, error)),
	}
}

// Register registers a processor function for a given key
func (sc *ServiceContract[K, I, O]) Register(key K, processor func(I) (O, error)) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.processors[key] = processor
}

// Process executes the processor associated with the given key
func (sc *ServiceContract[K, I, O]) Process(key K, input I) (O, error) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	
	if processor, exists := sc.processors[key]; exists {
		return processor(input)
	}
	
	var zero O
	return zero, fmt.Errorf("processor not found for key: %v", key)
}

// HasProcessor checks if a processor is registered for the given key
func (sc *ServiceContract[K, I, O]) HasProcessor(key K) bool {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	_, exists := sc.processors[key]
	return exists
}

// ListKeys returns all registered keys
func (sc *ServiceContract[K, I, O]) ListKeys() []K {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	
	keys := make([]K, 0, len(sc.processors))
	for key := range sc.processors {
		keys = append(keys, key)
	}
	return keys
}

// Processor represents a backward-compatible processor function
type Processor[I, O any] func(I) (O, error)

// GetContract creates a new service contract (backward compatibility function)
func GetContract[K comparable, I, O any]() *ServiceContract[K, I, O] {
	return NewServiceContract[K, I, O]()
}