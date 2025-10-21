package sentinel

import (
	"reflect"
)

// GetRelationships returns all relationships from a type to other types.
func GetRelationships[T any]() []TypeRelationship {
	metadata := Inspect[T]()
	return metadata.Relationships
}

// GetReferencedBy returns all types that reference the given type.
// This performs a reverse lookup across all cached metadata.
func GetReferencedBy[T any]() []TypeRelationship {
	var zero T
	t := reflect.TypeOf(zero)
	targetName := getTypeName(t)

	var references []TypeRelationship

	// Search through all cached types
	for _, typeName := range instance.cache.Keys() {
		if metadata, found := instance.cache.Get(typeName); found {
			// Check each relationship in this type
			for _, rel := range metadata.Relationships {
				if rel.To == targetName {
					references = append(references, rel)
				}
			}
		}
	}

	return references
}

// extractRelationships discovers relationships to other types within the same package domain.
// If visited is non-nil, it will recursively scan related types in the same module.
func (s *Sentinel) extractRelationships(t reflect.Type, visited map[string]bool) []TypeRelationship {
	var relationships []TypeRelationship

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return relationships
	}

	// Get the root package for domain filtering
	rootPackage := t.PkgPath()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		// Check if field type is a struct or related type
		rel := s.extractRelationship(field, rootPackage)
		if rel != nil {
			rel.From = t.Name()
			relationships = append(relationships, *rel)

			// If visited map is provided (Scan mode), recursively scan related types
			if visited != nil && s.isInModuleDomain(rel.ToPackage, rootPackage) {
				// Extract the underlying struct type from the field
				relType := s.getStructTypeFromField(field.Type)
				if relType != nil {
					s.extractMetadataInternal(relType, visited)
				}
			}
		}
	}

	return relationships
}

// extractRelationship checks if a field represents a relationship to another struct type.
func (s *Sentinel) extractRelationship(field reflect.StructField, rootPackage string) *TypeRelationship {
	ft := field.Type

	// Handle different field types
	switch ft.Kind() {
	case reflect.Struct:
		// Direct struct embedding
		if field.Anonymous {
			return s.createRelationshipIfInDomain(field, ft, RelationshipEmbedding, rootPackage)
		}
		// Regular struct field
		return s.createRelationshipIfInDomain(field, ft, RelationshipReference, rootPackage)

	case reflect.Ptr:
		// Pointer to struct
		elem := ft.Elem()
		if elem.Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, elem, RelationshipReference, rootPackage)
		}

	case reflect.Slice, reflect.Array:
		// Slice/array of structs
		elem := ft.Elem()
		// Handle []T and []*T
		if elem.Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, elem, RelationshipCollection, rootPackage)
		} else if elem.Kind() == reflect.Ptr && elem.Elem().Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, elem.Elem(), RelationshipCollection, rootPackage)
		}

	case reflect.Map:
		// Map with struct values
		val := ft.Elem()
		// Handle map[K]V and map[K]*V where V is struct
		if val.Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, val, RelationshipMap, rootPackage)
		} else if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, val.Elem(), RelationshipMap, rootPackage)
		}
	}

	return nil
}

// createRelationshipIfInDomain creates a TypeRelationship if the target type is in the same package domain.
func (s *Sentinel) createRelationshipIfInDomain(field reflect.StructField, targetType reflect.Type, kind string, rootPackage string) *TypeRelationship {
	targetPkg := targetType.PkgPath()

	// Skip types without package (built-in types)
	if targetPkg == "" {
		return nil
	}

	// Check if in same package domain
	if !s.isInPackageDomain(targetPkg, rootPackage) {
		return nil
	}

	return &TypeRelationship{
		To:        targetType.Name(),
		Field:     field.Name,
		Kind:      kind,
		ToPackage: targetPkg,
	}
}

// isInPackageDomain checks if a target package is within the same domain as the source.
// For Inspect: requires exact package match.
// For Scan: checks module root match (first 3 path segments).
func (*Sentinel) isInPackageDomain(targetPkg, sourcePkg string) bool {
	// Only include exact same package to avoid noise from external dependencies
	return targetPkg == sourcePkg
}

// isInModuleDomain checks if a target package shares the same module root as the source.
// Uses the first 3 path segments to determine module boundary.
func (*Sentinel) isInModuleDomain(targetPkg, sourcePkg string) bool {
	if targetPkg == "" || sourcePkg == "" {
		return false
	}
	return getModuleRoot(targetPkg) == getModuleRoot(sourcePkg)
}

// getStructTypeFromField extracts the underlying struct type from a field.
// Handles pointers, slices, arrays, and maps.
func (*Sentinel) getStructTypeFromField(ft reflect.Type) reflect.Type {
	switch ft.Kind() {
	case reflect.Struct:
		return ft
	case reflect.Ptr:
		if ft.Elem().Kind() == reflect.Struct {
			return ft.Elem()
		}
	case reflect.Slice, reflect.Array:
		elem := ft.Elem()
		if elem.Kind() == reflect.Struct {
			return elem
		}
		if elem.Kind() == reflect.Ptr && elem.Elem().Kind() == reflect.Struct {
			return elem.Elem()
		}
	case reflect.Map:
		val := ft.Elem()
		if val.Kind() == reflect.Struct {
			return val
		}
		if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct {
			return val.Elem()
		}
	}
	return nil
}
