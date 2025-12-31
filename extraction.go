package sentinel

import (
	"reflect"
)

// extractMetadata performs the complete metadata extraction for a type.
// This is used by Inspect() for single-type inspection (no recursive scanning).
func (s *Sentinel) extractMetadata(t reflect.Type) Metadata {
	return s.extractMetadataInternal(t, nil)
}

// extractMetadataInternal performs metadata extraction with optional recursive scanning.
// If visited is non-nil, it will recursively scan related types in the same module.
func (s *Sentinel) extractMetadataInternal(t reflect.Type, visited map[string]bool) Metadata {
	if t == nil {
		return Metadata{}
	}

	// Normalize pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return Metadata{}
	}

	fqdn := getFQDN(t)
	typeName := getTypeName(t)

	// Check if already visited (cycle detection)
	if visited != nil && visited[fqdn] {
		// Already visited, return cached metadata
		if cached, exists := s.cache.Get(fqdn); exists {
			return cached
		}
		return Metadata{}
	}

	// Mark as visited before processing
	if visited != nil {
		visited[fqdn] = true
	}

	// Check cache first (if cache exists)
	if s.cache != nil {
		if cached, exists := s.cache.Get(fqdn); exists {
			// Even if cached, we still need to scan relationships if in Scan mode
			if visited != nil {
				// Re-extract relationships to trigger recursive scanning
				_ = s.extractRelationships(t, visited)
			}
			return cached
		}
	}

	// Initialize metadata with basic reflection
	metadata := Metadata{
		ReflectType: t,
		FQDN:        fqdn,
		TypeName:    typeName,
		PackageName: t.PkgPath(),
	}

	// Extract fields
	metadata.Fields = s.extractFieldMetadata(t)

	// Extract relationships (will recursively scan if visited is non-nil)
	metadata.Relationships = s.extractRelationships(t, visited)

	// Store in cache (if cache exists)
	if s.cache != nil {
		s.cache.Set(fqdn, metadata)
	}

	return metadata
}

// scanWithVisited recursively inspects a type and all related types within the same module.
// The visited map prevents infinite loops from circular references.
func (s *Sentinel) scanWithVisited(t reflect.Type, visited map[string]bool) {
	// All the work is now done by extractMetadataInternal
	s.extractMetadataInternal(t, visited)
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
			Index:       field.Index,
			Name:        field.Name,
			Type:        field.Type.String(),
			Kind:        getFieldKind(field.Type),
			ReflectType: field.Type,
			Tags:        tags,
		}

		fields = append(fields, fieldMeta)
	}

	return fields
}
