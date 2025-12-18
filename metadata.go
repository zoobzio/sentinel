package sentinel

import (
	"reflect"
)

// Metadata contains comprehensive information about a user model.
type Metadata struct {
	TypeName      string             `json:"type_name"`
	PackageName   string             `json:"package_name"`
	Fields        []FieldMetadata    `json:"fields"`
	Relationships []TypeRelationship `json:"relationships,omitempty"`
}

// FieldMetadata captures field-level information and all struct tags.
type FieldMetadata struct {
	Tags map[string]string `json:"tags,omitempty"`
	Name string            `json:"name"`
	Type string            `json:"type"`
}

// getTypeName extracts the type name from a reflect.Type.
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

// TypeRelationship represents a relationship between two types.
type TypeRelationship struct {
	From      string `json:"from"`       // Source type name
	To        string `json:"to"`         // Target type name
	Field     string `json:"field"`      // Field creating the relationship
	Kind      string `json:"kind"`       // "reference", "collection", "embedding"
	ToPackage string `json:"to_package"` // Target type's package path
}

// RelationshipKind constants for different relationship types.
const (
	RelationshipReference  = "reference"  // Direct field reference (e.g., Profile *Profile)
	RelationshipCollection = "collection" // Slice/array of types (e.g., Orders []Order)
	RelationshipEmbedding  = "embedding"  // Anonymous field embedding
	RelationshipMap        = "map"        // Map with struct values
)
