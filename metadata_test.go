package catalog

import (
	"reflect"
	"testing"
	"time"
)

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		{
			name:     "simple struct",
			input:    reflect.TypeOf(SimpleStruct{}),
			expected: "SimpleStruct",
		},
		{
			name:     "pointer to struct",
			input:    reflect.TypeOf(&SimpleStruct{}),
			expected: "SimpleStruct",
		},
		{
			name:     "nil type",
			input:    nil,
			expected: "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractAllTags(t *testing.T) {
	// Register custom tags
	Tag("custom")
	Tag("validate")

	type TestStruct struct {
		WithTags    string `json:"with_tags" custom:"value" validate:"required"`
		WithoutTags string
	}

	tests := []struct {
		name      string
		fieldName string
		expected  map[string]string
	}{
		{
			name:      "field_with_tags",
			fieldName: "WithTags",
			expected: map[string]string{
				"json":     "with_tags",
				"custom":   "value",
				"validate": "required",
			},
		},
		{
			name:      "field_without_tags",
			fieldName: "WithoutTags",
			expected:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := reflect.TypeOf(TestStruct{})
			field, found := typ.FieldByName(tt.fieldName)
			if !found {
				t.Fatalf("field %s not found", tt.fieldName)
			}

			result := extractAllTags(field)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d: %v", len(tt.expected), len(result), result)
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected tag %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("expected tag %s=%s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestExtractFieldMetadata(t *testing.T) {
	// Register tags we want to test
	Tag("validate")
	Tag("encrypt")
	Tag("db")
	Tag("desc")

	type CompleteStruct struct {
		FullField string `json:"full" db:"full_field" desc:"A complete field" validate:"required,email" encrypt:"pii"`
		Minimal   string `json:"minimal"`
		Private   string // No tags
	}

	typ := reflect.TypeOf(CompleteStruct{})
	fields := extractFieldMetadata(typ)

	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	// Find the full field
	var fullField *FieldMetadata
	for i, f := range fields {
		if f.Name == "FullField" {
			fullField = &fields[i]
			break
		}
	}

	if fullField == nil {
		t.Fatal("FullField not found")
	}

	// Check basic properties
	if fullField.Name != "FullField" {
		t.Errorf("expected Name 'FullField', got %s", fullField.Name)
	}

	if fullField.Type != "string" {
		t.Errorf("expected Type 'string', got %s", fullField.Type)
	}

	// Check tags
	expectedTags := map[string]string{
		"json":     "full",
		"db":       "full_field",
		"desc":     "A complete field",
		"validate": "required,email",
		"encrypt":  "pii",
	}

	for key, expectedValue := range expectedTags {
		if actualValue, exists := fullField.Tags[key]; !exists {
			t.Errorf("expected tag %s not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected tag %s=%s, got %s", key, expectedValue, actualValue)
		}
	}
}

func TestGetModelMetadata(t *testing.T) {
	t.Run("existing_metadata", func(t *testing.T) {
		// First populate cache
		_ = Inspect[SimpleStruct]()

		metadata, exists := GetModelMetadata("SimpleStruct")
		if !exists {
			t.Error("expected metadata to exist")
		}
		if metadata.TypeName != "SimpleStruct" {
			t.Errorf("expected TypeName 'SimpleStruct', got %s", metadata.TypeName)
		}
	})

	t.Run("non-existing_metadata", func(t *testing.T) {
		metadata, exists := GetModelMetadata("NonExistentType")
		if exists {
			t.Error("expected metadata to not exist")
		}
		if metadata.TypeName != "" {
			t.Error("expected empty metadata for non-existent type")
		}
	})
}

func TestFieldTypeString(t *testing.T) {
	type TypeTest struct {
		String    string
		Int       int
		Bool      bool
		Float     float64
		Slice     []string
		Map       map[string]int
		Struct    SimpleStruct
		Pointer   *string
		Interface interface{}
		Time      time.Time
	}

	metadata := Inspect[TypeTest]()

	expectedTypes := map[string]string{
		"String":    "string",
		"Int":       "int",
		"Bool":      "bool",
		"Float":     "float64",
		"Slice":     "[]string",
		"Map":       "map[string]int",
		"Struct":    "catalog.SimpleStruct",
		"Pointer":   "*string",
		"Interface": "interface {}",
		"Time":      "time.Time",
	}

	for _, field := range metadata.Fields {
		if expectedType, exists := expectedTypes[field.Name]; exists {
			if field.Type != expectedType {
				t.Errorf("field %s: expected type %s, got %s", field.Name, expectedType, field.Type)
			}
		}
	}
}

// SimpleStruct is defined in api_test.go
