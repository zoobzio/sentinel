package sentinel

import (
	"reflect"
)

// ModelMetadata contains comprehensive information about a user model.
type ModelMetadata struct {
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

// getModuleRoot extracts the module root from a package path.
// It uses the first 3 path segments (e.g., "github.com/user/repo").
// This allows us to distinguish user modules from external libraries.
func getModuleRoot(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}

	// Find position of the 3rd slash (end of 3rd segment)
	slashCount := 0

	for i := 0; i < len(pkgPath); i++ {
		if pkgPath[i] == '/' {
			slashCount++
			if slashCount == 3 {
				// Found 3rd slash, return everything before it
				return pkgPath[:i]
			}
		}
	}

	// Less than 3 slashes (≤3 segments), return the whole path
	return pkgPath
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
