package sentinel

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractMetadata(t *testing.T) {
	// Register custom tags for testing
	Tag("custom")
	Tag("validate")

	s := &Sentinel{
		registeredTags: instance.registeredTags,
	}

	t.Run("simple struct", func(t *testing.T) {
		type SimpleStruct struct {
			Name string `json:"name" validate:"required"`
		}

		typ := reflect.TypeOf(SimpleStruct{})
		metadata := s.extractMetadata(typ)

		if metadata.TypeName != "SimpleStruct" {
			t.Errorf("expected TypeName 'SimpleStruct', got %s", metadata.TypeName)
		}
		if len(metadata.Fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(metadata.Fields))
		}

		field := metadata.Fields[0]
		if field.Name != "Name" {
			t.Errorf("expected field name 'Name', got %s", field.Name)
		}
		if field.Type != "string" {
			t.Errorf("expected field type 'string', got %s", field.Type)
		}
		if field.Tags["json"] != "name" {
			t.Errorf("expected json tag 'name', got %s", field.Tags["json"])
		}
		if field.Tags["validate"] != "required" {
			t.Errorf("expected validate tag 'required', got %s", field.Tags["validate"])
		}
	})

	t.Run("struct with multiple fields", func(t *testing.T) {
		type ComplexStruct struct {
			ID         int    `json:"id" db:"id"`
			Name       string `json:"name"`
			Active     bool   `json:"active"`
			unexported string //nolint:unused
		}

		typ := reflect.TypeOf(ComplexStruct{})
		metadata := s.extractMetadata(typ)

		// Should only have 3 fields (unexported excluded)
		if len(metadata.Fields) != 3 {
			t.Errorf("expected 3 fields, got %d", len(metadata.Fields))
		}

		// Verify field names
		expectedNames := []string{"ID", "Name", "Active"}
		for i, expected := range expectedNames {
			if metadata.Fields[i].Name != expected {
				t.Errorf("field %d: expected name %s, got %s", i, expected, metadata.Fields[i].Name)
			}
		}
	})

	t.Run("empty struct", func(t *testing.T) {
		type EmptyStruct struct{}

		typ := reflect.TypeOf(EmptyStruct{})
		metadata := s.extractMetadata(typ)

		if len(metadata.Fields) != 0 {
			t.Errorf("expected 0 fields for empty struct, got %d", len(metadata.Fields))
		}
	})

	t.Run("array type", func(t *testing.T) {
		type ArrayStruct struct {
			Items [5]string `json:"items"`
		}

		typ := reflect.TypeOf(ArrayStruct{})
		metadata := s.extractMetadata(typ)

		if metadata.TypeName != "ArrayStruct" {
			t.Errorf("expected TypeName 'ArrayStruct', got %s", metadata.TypeName)
		}

		if len(metadata.Fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(metadata.Fields))
		}

		field := metadata.Fields[0]
		if field.Type != "[5]string" {
			t.Errorf("expected type '[5]string', got %s", field.Type)
		}
	})

	t.Run("recursive struct", func(t *testing.T) {
		type Node struct {
			Value    string `json:"value"`
			Children []Node `json:"children"`
		}

		typ := reflect.TypeOf(Node{})
		metadata := s.extractMetadata(typ)

		if metadata.TypeName != "Node" {
			t.Errorf("expected TypeName 'Node', got %s", metadata.TypeName)
		}

		if len(metadata.Fields) != 2 {
			t.Fatalf("expected 2 fields, got %d", len(metadata.Fields))
		}

		// Check recursive type handling
		for _, field := range metadata.Fields {
			if field.Name == "Children" && !strings.Contains(field.Type, "Node") {
				t.Errorf("expected type to contain 'Node', got %s", field.Type)
			}
		}
	})

	t.Run("struct with complex nested types", func(t *testing.T) {
		type DeepStruct struct {
			MapOfSlices map[string][]int        `json:"map_of_slices"`
			SliceOfMaps []map[string]string     `json:"slice_of_maps"`
			MapOfMaps   map[string]map[int]bool `json:"map_of_maps"`
			ChanOfChans chan chan string        `json:"chan_of_chans"`
			FuncReturns func() (string, error)  `json:"func_returns"`
		}

		typ := reflect.TypeOf(DeepStruct{})
		metadata := s.extractMetadata(typ)

		if len(metadata.Fields) != 5 {
			t.Fatalf("expected 5 fields, got %d", len(metadata.Fields))
		}

		// Verify complex types are captured correctly
		typeMap := make(map[string]string)
		for _, f := range metadata.Fields {
			typeMap[f.Name] = f.Type
		}

		expectedTypes := map[string]string{
			"MapOfSlices": "map[string][]int",
			"SliceOfMaps": "[]map[string]string",
			"MapOfMaps":   "map[string]map[int]bool",
			"ChanOfChans": "chan chan string",
			"FuncReturns": "func() (string, error)",
		}

		for name, expectedType := range expectedTypes {
			if typeMap[name] != expectedType {
				t.Errorf("field %s: expected type %s, got %s", name, expectedType, typeMap[name])
			}
		}
	})
}

func TestExtractFieldMetadata(t *testing.T) {
	s := &Sentinel{
		registeredTags: make(map[string]bool),
	}

	t.Run("common tags", func(t *testing.T) {
		type TestStruct struct {
			Field string `json:"field" validate:"required" db:"field_name" scope:"admin" encrypt:"pii" redact:"***" desc:"Test field" example:"test"`
		}

		fields := s.extractFieldMetadata(reflect.TypeOf(TestStruct{}))
		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}

		tags := fields[0].Tags
		expectedTags := map[string]string{
			"json":     "field",
			"validate": "required",
			"db":       "field_name",
			"scope":    "admin",
			"encrypt":  "pii",
			"redact":   "***",
			"desc":     "Test field",
			"example":  "test",
		}

		for tag, expected := range expectedTags {
			if tags[tag] != expected {
				t.Errorf("tag %s: expected %q, got %q", tag, expected, tags[tag])
			}
		}
	})

	t.Run("registered custom tags", func(t *testing.T) {
		// Register custom tags
		s.registeredTags["custom1"] = true
		s.registeredTags["custom2"] = true

		type TestStruct struct {
			Field string `custom1:"value1" custom2:"value2" unregistered:"ignored"`
		}

		fields := s.extractFieldMetadata(reflect.TypeOf(TestStruct{}))
		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}

		tags := fields[0].Tags
		if tags["custom1"] != "value1" {
			t.Errorf("expected custom1 tag 'value1', got %s", tags["custom1"])
		}
		if tags["custom2"] != "value2" {
			t.Errorf("expected custom2 tag 'value2', got %s", tags["custom2"])
		}
		if _, exists := tags["unregistered"]; exists {
			t.Error("unregistered tag should not be extracted")
		}
	})

	t.Run("pointer type", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		// Test with pointer type
		fields := s.extractFieldMetadata(reflect.TypeOf(&TestStruct{}))
		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].Name != "Name" {
			t.Errorf("expected field name 'Name', got %s", fields[0].Name)
		}
	})

	t.Run("non-struct type", func(t *testing.T) {
		// Should return empty for non-struct types
		fields := s.extractFieldMetadata(reflect.TypeOf("string"))
		if len(fields) != 0 {
			t.Errorf("expected 0 fields for non-struct type, got %d", len(fields))
		}
	})
}
