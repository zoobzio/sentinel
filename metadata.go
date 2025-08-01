package catalog

import (
	"reflect"
	"sync"
)

// ModelMetadata contains comprehensive information about a user model.
type ModelMetadata struct {
	TypeName    string          `json:"type_name"`
	PackageName string          `json:"package_name"`
	Fields      []FieldMetadata `json:"fields"`
}

// FieldMetadata captures field-level information and all struct tags.
type FieldMetadata struct {
	Tags map[string]string `json:"tags,omitempty"`
	Name string            `json:"name"`
	Type string            `json:"type"`
}

// Global metadata cache - reflect once, use everywhere.
var (
	metadataCache = make(map[string]ModelMetadata)
	cacheMutex    sync.RWMutex

	// Tag registry - tracks which tags to extract.
	registeredTags = make(map[string]bool)
	tagMutex       sync.RWMutex
)

// GetModelMetadata retrieves cached metadata by type name.
func GetModelMetadata(typeName string) (ModelMetadata, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	metadata, exists := metadataCache[typeName]
	return metadata, exists
}

// ExtractAndCacheMetadata performs comprehensive reflection on a type and caches the result.
func ExtractAndCacheMetadata[T any](example T) ModelMetadata {
	t := reflect.TypeOf(example)
	typeName := getTypeName(t)

	// Check cache first
	cacheMutex.RLock()
	if cached, exists := metadataCache[typeName]; exists {
		cacheMutex.RUnlock()
		return cached
	}
	cacheMutex.RUnlock()

	// Extract comprehensive metadata
	metadata := extractMetadata(t, example)

	// Cache the result
	cacheMutex.Lock()
	metadataCache[typeName] = metadata
	cacheMutex.Unlock()

	return metadata
}

// extractMetadata performs the actual reflection and metadata extraction.
func extractMetadata(t reflect.Type, _ any) ModelMetadata {
	metadata := ModelMetadata{
		TypeName:    getTypeName(t),
		PackageName: t.PkgPath(),
		Fields:      extractFieldMetadata(t),
	}

	return metadata
}

// extractFieldMetadata extracts comprehensive field information using pipz contracts.
func extractFieldMetadata(t reflect.Type) []FieldMetadata {
	var fields []FieldMetadata

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fields
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		// Base field metadata
		fieldMeta := FieldMetadata{
			Name: field.Name,
			Type: field.Type.String(),
			Tags: extractAllTags(field),
		}

		fields = append(fields, fieldMeta)
	}

	return fields
}

// Helper functions for tag extraction

func getTypeName(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return "nil"
	}
	return t.Name()
}

func extractAllTags(field reflect.StructField) map[string]string {
	tags := make(map[string]string)

	// Always extract json tag for serialization
	if value, ok := field.Tag.Lookup("json"); ok {
		tags["json"] = value
	}

	// Extract all registered tags
	tagMutex.RLock()
	defer tagMutex.RUnlock()

	for tagName := range registeredTags {
		if value, ok := field.Tag.Lookup(tagName); ok {
			tags[tagName] = value
		}
	}

	return tags
}
