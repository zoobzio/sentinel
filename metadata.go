package sentinel

import (
	"reflect"
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
