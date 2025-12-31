package sentinel

import (
	"reflect"
	"testing"
)

func TestMetadata(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		type User struct {
			ID string `json:"id"`
		}
		metadata := Metadata{
			ReflectType: reflect.TypeOf(User{}),
			TypeName:    "User",
			PackageName: "main",
			Fields: []FieldMetadata{
				{
					Name: "ID",
					Type: "string",
					Tags: map[string]string{"json": "id"},
				},
			},
		}

		if metadata.TypeName != "User" {
			t.Errorf("expected TypeName 'User', got %s", metadata.TypeName)
		}
		if metadata.PackageName != "main" {
			t.Errorf("expected PackageName 'main', got %s", metadata.PackageName)
		}
		if len(metadata.Fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(metadata.Fields))
		}
		if metadata.ReflectType == nil {
			t.Error("expected ReflectType to be set")
		} else if metadata.ReflectType.Kind() != reflect.Struct {
			t.Errorf("expected ReflectType kind Struct, got %v", metadata.ReflectType.Kind())
		}
	})

	t.Run("json tags", func(t *testing.T) {
		// Verify JSON struct tags are properly defined
		metadata := Metadata{}
		metaType := reflect.TypeOf(metadata)

		expectedTags := map[string]string{
			"TypeName":      "type_name",
			"PackageName":   "package_name",
			"Fields":        "fields",
			"Relationships": "relationships,omitempty",
		}

		for fieldName, expectedTag := range expectedTags {
			field, found := metaType.FieldByName(fieldName)
			if !found {
				t.Errorf("field %s not found", fieldName)
				continue
			}
			if tag := field.Tag.Get("json"); tag != expectedTag {
				t.Errorf("field %s: expected json tag %q, got %q", fieldName, expectedTag, tag)
			}
		}

		// Verify ReflectType is excluded from JSON
		reflectField, found := metaType.FieldByName("ReflectType")
		if !found {
			t.Error("ReflectType field not found")
		} else if tag := reflectField.Tag.Get("json"); tag != "-" {
			t.Errorf("ReflectType: expected json tag '-', got %q", tag)
		}
	})
}

func TestFieldMetadata(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		field := FieldMetadata{
			Index:       []int{0},
			Name:        "Email",
			Type:        "string",
			Kind:        KindScalar,
			ReflectType: reflect.TypeOf(""),
			Tags: map[string]string{
				"json":     "email",
				"validate": "required,email",
				"encrypt":  "pii",
			},
		}

		if field.Name != "Email" {
			t.Errorf("expected Name 'Email', got %s", field.Name)
		}
		if field.Type != "string" {
			t.Errorf("expected Type 'string', got %s", field.Type)
		}
		if field.Kind != KindScalar {
			t.Errorf("expected Kind 'scalar', got %s", field.Kind)
		}
		if len(field.Index) != 1 || field.Index[0] != 0 {
			t.Errorf("expected Index [0], got %v", field.Index)
		}
		if field.ReflectType.Kind() != reflect.String {
			t.Errorf("expected ReflectType kind String, got %v", field.ReflectType.Kind())
		}
		if len(field.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(field.Tags))
		}
		if field.Tags["json"] != "email" {
			t.Errorf("expected json tag 'email', got %s", field.Tags["json"])
		}
	})

	t.Run("json tags", func(t *testing.T) {
		// Verify JSON struct tags are properly defined
		field := FieldMetadata{}
		fieldType := reflect.TypeOf(field)

		expectedTags := map[string]string{
			"Index": "index",
			"Tags":  "tags,omitempty",
			"Name":  "name",
			"Type":  "type",
			"Kind":  "kind",
		}

		for fieldName, expectedTag := range expectedTags {
			f, found := fieldType.FieldByName(fieldName)
			if !found {
				t.Errorf("field %s not found", fieldName)
				continue
			}
			if tag := f.Tag.Get("json"); tag != expectedTag {
				t.Errorf("field %s: expected json tag %q, got %q", fieldName, expectedTag, tag)
			}
		}

		// Verify ReflectType is excluded from JSON
		reflectField, found := fieldType.FieldByName("ReflectType")
		if !found {
			t.Error("ReflectType field not found")
		} else if tag := reflectField.Tag.Get("json"); tag != "-" {
			t.Errorf("ReflectType: expected json tag '-', got %q", tag)
		}
	})

	t.Run("nil tags map", func(_ *testing.T) {
		_ = FieldMetadata{
			Index: []int{0},
			Name:  "ID",
			Type:  "int",
			Kind:  KindScalar,
			Tags:  nil,
		}

		// Should not panic.
		// When Tags is nil, this is expected and allowed behavior.
	})
}

func TestFieldKindConstants(t *testing.T) {
	t.Run("constant values", func(t *testing.T) {
		if KindScalar != "scalar" {
			t.Errorf("expected KindScalar 'scalar', got %s", KindScalar)
		}
		if KindPointer != "pointer" {
			t.Errorf("expected KindPointer 'pointer', got %s", KindPointer)
		}
		if KindSlice != "slice" {
			t.Errorf("expected KindSlice 'slice', got %s", KindSlice)
		}
		if KindStruct != "struct" {
			t.Errorf("expected KindStruct 'struct', got %s", KindStruct)
		}
		if KindMap != "map" {
			t.Errorf("expected KindMap 'map', got %s", KindMap)
		}
		if KindInterface != "interface" {
			t.Errorf("expected KindInterface 'interface', got %s", KindInterface)
		}
	})
}

func TestGetFieldKind(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected FieldKind
	}{
		{
			name:     "nil type",
			input:    nil,
			expected: KindInterface,
		},
		{
			name:     "string type",
			input:    reflect.TypeOf(""),
			expected: KindScalar,
		},
		{
			name:     "int type",
			input:    reflect.TypeOf(0),
			expected: KindScalar,
		},
		{
			name:     "float64 type",
			input:    reflect.TypeOf(0.0),
			expected: KindScalar,
		},
		{
			name:     "bool type",
			input:    reflect.TypeOf(true),
			expected: KindScalar,
		},
		{
			name:     "pointer type",
			input:    reflect.TypeOf((*string)(nil)),
			expected: KindPointer,
		},
		{
			name:     "pointer to struct",
			input:    reflect.TypeOf(&Metadata{}),
			expected: KindPointer,
		},
		{
			name:     "slice type",
			input:    reflect.TypeOf([]string{}),
			expected: KindSlice,
		},
		{
			name:     "array type",
			input:    reflect.TypeOf([5]int{}),
			expected: KindSlice,
		},
		{
			name:     "struct type",
			input:    reflect.TypeOf(Metadata{}),
			expected: KindStruct,
		},
		{
			name:     "map type",
			input:    reflect.TypeOf(map[string]int{}),
			expected: KindMap,
		},
		{
			name:     "interface type",
			input:    reflect.TypeOf((*error)(nil)).Elem(),
			expected: KindInterface,
		},
		{
			name:     "channel type",
			input:    reflect.TypeOf(make(chan int)),
			expected: KindScalar,
		},
		{
			name:     "func type",
			input:    reflect.TypeOf(func() {}),
			expected: KindScalar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldKind(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetFQDN(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		{
			name:     "nil type",
			input:    nil,
			expected: "nil",
		},
		{
			name:     "named struct type",
			input:    reflect.TypeOf(Metadata{}),
			expected: "github.com/zoobzio/sentinel.Metadata",
		},
		{
			name:     "pointer to struct",
			input:    reflect.TypeOf(&Metadata{}),
			expected: "github.com/zoobzio/sentinel.Metadata",
		},
		{
			name:     "built-in string type",
			input:    reflect.TypeOf(""),
			expected: "string",
		},
		{
			name:     "built-in int type",
			input:    reflect.TypeOf(0),
			expected: "int",
		},
		{
			name:     "pointer to built-in",
			input:    reflect.TypeOf((*string)(nil)),
			expected: "string",
		},
		{
			name:     "anonymous struct",
			input:    reflect.TypeOf(struct{ Name string }{}),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFQDN(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		{
			name:     "nil type",
			input:    nil,
			expected: "nil",
		},
		{
			name:     "string type",
			input:    reflect.TypeOf(""),
			expected: "string",
		},
		{
			name:     "int type",
			input:    reflect.TypeOf(0),
			expected: "int",
		},
		{
			name:     "struct type",
			input:    reflect.TypeOf(struct{ Name string }{}),
			expected: "",
		},
		{
			name:     "named struct type",
			input:    reflect.TypeOf(Metadata{}),
			expected: "Metadata",
		},
		{
			name:     "pointer to struct",
			input:    reflect.TypeOf(&Metadata{}),
			expected: "Metadata",
		},
		{
			name:     "pointer to string",
			input:    reflect.TypeOf((*string)(nil)),
			expected: "string",
		},
		{
			name:     "slice type",
			input:    reflect.TypeOf([]string{}),
			expected: "",
		},
		{
			name:     "map type",
			input:    reflect.TypeOf(map[string]int{}),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
