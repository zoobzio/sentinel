package sentinel

import (
	"reflect"
)

// extractMetadata performs the complete metadata extraction for a type.
func (s *Sentinel) extractMetadata(t reflect.Type) ModelMetadata {
	// Initialize metadata with basic reflection
	metadata := ModelMetadata{
		TypeName:    getTypeName(t),
		PackageName: t.PkgPath(),
	}

	// Extract fields
	metadata.Fields = s.extractFieldMetadata(t)

	// Extract relationships
	metadata.Relationships = s.extractRelationships(t)

	return metadata
}

// extractFieldMetadata extracts field information with registered tags.
func (s *Sentinel) extractFieldMetadata(t reflect.Type) []FieldMetadata {
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

		// Extract all tags
		tags := make(map[string]string)

		// Include registered tags
		s.tagMutex.RLock()
		for tagName := range s.registeredTags {
			if tagValue := field.Tag.Get(tagName); tagValue != "" {
				tags[tagName] = tagValue
			}
		}
		s.tagMutex.RUnlock()

		// Always include common tags
		commonTags := []string{"json", "validate", "db", "scope", "encrypt", "redact", "desc", "example"}
		for _, tagName := range commonTags {
			if tagValue := field.Tag.Get(tagName); tagValue != "" {
				tags[tagName] = tagValue
			}
		}

		fieldMeta := FieldMetadata{
			Name: field.Name,
			Type: field.Type.String(),
			Tags: tags,
		}

		fields = append(fields, fieldMeta)
	}

	return fields
}
