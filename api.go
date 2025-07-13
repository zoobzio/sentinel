package catalog

import (
	"reflect"
	"sync"
)

// PUBLIC API - Only two functions exposed

// Select returns comprehensive metadata for a type
// Handles everything internally: cache check, reflection, storage
// This is the ONLY way to get metadata - always works
func Select[T any]() ModelMetadata {
	var zero T
	t := reflect.TypeOf(zero)
	
	// Catalog only supports struct types - use concrete types for primitives
	if t != nil && t.Kind() != reflect.Struct {
		if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
			// Allow pointers to structs
			t = t.Elem()
		} else {
			panic("catalog: Select only supports struct types - wrap primitive types in a struct (e.g. type MyStrings []string)")
		}
	}
	
	typeName := getTypeName(t)
	
	// Check cache first
	cacheMutex.RLock()
	if cached, exists := metadataCache[typeName]; exists {
		cacheMutex.RUnlock()
		return cached
	}
	cacheMutex.RUnlock()
	
	// Extract and cache metadata
	metadata := extractMetadata(t, zero)
	
	cacheMutex.Lock()
	metadataCache[typeName] = metadata
	cacheMutex.Unlock()
	
	// Check type conventions (e.g., security registration)
	checkTypeConventions[T](metadata)
	
	return metadata
}

// Browse returns all type names that have been registered in the catalog
// Useful for type discovery and debugging
func Browse() []string {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	
	var types []string
	for typeName := range metadataCache {
		types = append(types, typeName)
	}
	return types
}

// RegisterType ensures a type is known to catalog for contract signatures
// This is used by packages to register their system types during init
func RegisterType[T any]() {
	var zero T
	t := reflect.TypeOf(zero)
	if t != nil {
		typeName := getTypeName(t)
		// Just cache the type name for later use
		cacheMutex.Lock()
		if _, exists := typeNameCache[typeName]; !exists {
			typeNameCache[typeName] = t
		}
		cacheMutex.Unlock()
	}
}

// RegisterTag registers a struct tag to be extracted during metadata processing
// This allows adapters to register their tags with catalog
func RegisterTag(tagName string) {
	tagMutex.Lock()
	defer tagMutex.Unlock()
	registeredTags[tagName] = true
}

// GetTypeName returns the string type name for a generic type
// This is the only type name extraction function
func GetTypeName[T any]() string {
	var zero T
	return getTypeName(reflect.TypeOf(zero))
}

// RegisterTransformer stores a transformer for type T
// This enables type-specific behavior storage (not just metadata)
func RegisterTransformer[T any](transformer StructTransformer[T]) {
	typeName := GetTypeName[T]()
	
	transformerMutex.Lock()
	transformerRegistry[typeName] = transformer
	transformerMutex.Unlock()
}

// GetTransformer retrieves a transformer for type T
// Returns the transformer and whether it exists
func GetTransformer[T any]() (any, bool) {
	typeName := GetTypeName[T]()
	
	transformerMutex.RLock()
	transformer, exists := transformerRegistry[typeName]
	transformerMutex.RUnlock()
	
	return transformer, exists
}


// GetFieldManipulators returns field manipulators for type T
// These provide type-safe field access without reflection in hot paths
func GetFieldManipulators[T any]() map[string]*FieldManipulator[T] {
	typeName := GetTypeName[T]()
	
	// Check if manipulators already exist
	manipulatorMutex.RLock()
	if existing, exists := manipulatorRegistry[typeName]; exists {
		manipulatorMutex.RUnlock()
		return existing.(map[string]*FieldManipulator[T])
	}
	manipulatorMutex.RUnlock()
	
	// Build manipulators (one-time reflection)
	manipulators := buildFieldManipulators[T]()
	
	// Cache for future use
	manipulatorMutex.Lock()
	manipulatorRegistry[typeName] = manipulators
	manipulatorMutex.Unlock()
	
	return manipulators
}

// CONVENTIONS SYSTEM

// HasDefaults defines the convention for types that can provide secure defaults
type HasDefaults[T any] interface {
	Defaults() T
}

// HasScope defines the convention for types that can provide object-level scoping
type HasScope[T any] interface {
	Scope() string
}

// Convention registry
var (
	conventionMutex    sync.RWMutex
	conventionRegistry = make(map[string]any)
)

// MASKING SYSTEM

// MaskFunction defines how to mask/redact a string value
type MaskFunction func(string) string

// Global mask registry for validate tags
var (
	maskRegistry = make(map[string]MaskFunction)
	maskMutex    sync.RWMutex
)

// VALIDATION SYSTEM

// FieldValidator validates a field value and returns an error if invalid
type FieldValidator func(fieldValue any) error

// Global validator registry for validate tags
var (
	fieldValidatorRegistry = make(map[string]FieldValidator)
	fieldValidatorMutex    sync.RWMutex
)

// RegisterMaskFunction registers a masking function for a validate tag
// This allows content-aware redaction based on field type
func RegisterMaskFunction(validateTag string, maskFunc MaskFunction) {
	maskMutex.Lock()
	defer maskMutex.Unlock()
	maskRegistry[validateTag] = maskFunc
}

// GetMaskFunction retrieves a mask function for a validate tag
func GetMaskFunction(validateTag string) (MaskFunction, bool) {
	maskMutex.RLock()
	defer maskMutex.RUnlock()
	fn, exists := maskRegistry[validateTag]
	return fn, exists
}

// RegisterFieldValidator registers a validation function for a validate tag
// This allows custom validation logic for specific field types
func RegisterFieldValidator(name string, validator FieldValidator) {
	fieldValidatorMutex.Lock()
	defer fieldValidatorMutex.Unlock()
	fieldValidatorRegistry[name] = validator
}

// GetFieldValidator retrieves a validator function for a validate tag
func GetFieldValidator(name string) (FieldValidator, bool) {
	fieldValidatorMutex.RLock()
	defer fieldValidatorMutex.RUnlock()
	validator, exists := fieldValidatorRegistry[name]
	return validator, exists
}

// TRANSFORMER TYPES

// StructTransformer is a function that transforms a struct for security
type StructTransformer[T any] func(source T, dest *T) error

// SECURITY TYPES - See security.go for:
// - SerializationContext: Context for security decisions during serialization
// - SecurityBehavior[T]: Type alias documenting security behavior signature
// - Prepare[T]: Apply security behaviors before serialization
// - HasPermission: Check permission scopes

// RegisterDefaults registers a defaults convention for type T
func RegisterDefaults[T HasDefaults[T]](handler func(T) T) {
	RegisterConvention("defaults", handler)
}

// RegisterScope registers a scope convention for type T
func RegisterScope[T HasScope[T]](handler func(T) string) {
	RegisterConvention("scope", handler)
}

// GetDefaults retrieves a defaults convention for type T
func GetDefaults[T any]() (func(T) T, bool) {
	return GetConvention[T, T]("defaults")
}

// GetScope retrieves a scope convention for type T
func GetScope[T any]() (func(T) string, bool) {
	return GetConvention[T, string]("scope")
}

// RegisterConvention stores a convention handler (internal)
func RegisterConvention[T any, R any](name string, handler func(T) R) {
	typeName := GetTypeName[T]()
	key := typeName + ":" + name
	
	conventionMutex.Lock()
	conventionRegistry[key] = handler
	conventionMutex.Unlock()
}

// GetConvention retrieves a convention handler (internal)
func GetConvention[T any, R any](name string) (func(T) R, bool) {
	typeName := GetTypeName[T]()
	key := typeName + ":" + name
	
	conventionMutex.RLock()
	handler, exists := conventionRegistry[key]
	conventionMutex.RUnlock()
	
	if exists {
		return handler.(func(T) R), true
	}
	return nil, false
}

// Internal helper functions (not exported)

// ensureMetadata is now incorporated directly into Select[T]()
// getByTypeName is removed - only Select[T]() should be used
// All other convenience functions removed - users extract from Select[T]()

// TypeIngestedEvent represents a type being ingested into the catalog
// This is a generic event that preserves the type information
type TypeIngestedEvent[T any] struct {
	TypeName string
	Metadata ModelMetadata
	ZeroValue T // Preserves the generic type
}

// TypeIngestedEventType is the event type for type ingestion
type TypeIngestedEventType string

const (
	TypeIngested TypeIngestedEventType = "catalog.type_ingested"
)