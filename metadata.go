package sentinel

import (
	"reflect"
)

// FieldKind represents the category of a field's type.
type FieldKind string

// FieldKind constants for type categorization.
const (
	KindScalar    FieldKind = "scalar"    // Basic types: string, int, float, bool, etc.
	KindPointer   FieldKind = "pointer"   // Pointer to any type
	KindSlice     FieldKind = "slice"     // Slice or array
	KindStruct    FieldKind = "struct"    // Struct type
	KindMap       FieldKind = "map"       // Map type
	KindInterface FieldKind = "interface" // Interface type
)

// Metadata contains comprehensive information about a user model.
type Metadata struct {
	ReflectType   reflect.Type       `json:"-"`
	FQDN          string             `json:"fqdn"`         // Fully qualified type name (e.g., "github.com/app/models.User")
	TypeName      string             `json:"type_name"`    // Simple type name (e.g., "User")
	PackageName   string             `json:"package_name"` // Package path (e.g., "github.com/app/models")
	Fields        []FieldMetadata    `json:"fields"`
	Relationships []TypeRelationship `json:"relationships,omitempty"`
}

// FieldMetadata captures field-level information and all struct tags.
type FieldMetadata struct {
	ReflectType reflect.Type      `json:"-"`
	Tags        map[string]string `json:"tags,omitempty"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Kind        FieldKind         `json:"kind"`
	Index       []int             `json:"index"`
}

// getFQDN returns the fully qualified type name (package path + type name).
func getFQDN(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if pkgPath := t.PkgPath(); pkgPath != "" {
		return pkgPath + "." + t.Name()
	}
	return t.Name()
}

// getTypeName extracts the simple type name from a reflect.Type.
func getTypeName(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// getFieldKind determines the FieldKind category from a reflect.Type.
func getFieldKind(t reflect.Type) FieldKind {
	if t == nil {
		return KindInterface
	}

	switch t.Kind() {
	case reflect.Ptr:
		return KindPointer
	case reflect.Slice, reflect.Array:
		return KindSlice
	case reflect.Struct:
		return KindStruct
	case reflect.Map:
		return KindMap
	case reflect.Interface:
		return KindInterface
	default:
		// All other kinds are scalars: bool, int*, uint*, float*, complex*, string, etc.
		return KindScalar
	}
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
