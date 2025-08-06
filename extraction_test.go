package sentinel

import (
	"context"
	"reflect"
	"testing"
)

func TestExtractionContext(t *testing.T) {
	t.Run("struct fields", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}

		ec := &ExtractionContext{
			Type:     reflect.TypeOf(TestStruct{}),
			Instance: TestStruct{},
			Metadata: ModelMetadata{
				TypeName: "TestStruct",
			},
		}

		if ec.Type.Name() != "TestStruct" {
			t.Errorf("expected Type name 'TestStruct', got %s", ec.Type.Name())
		}
		if ec.Metadata.TypeName != "TestStruct" {
			t.Errorf("expected Metadata.TypeName 'TestStruct', got %s", ec.Metadata.TypeName)
		}
	})
}

func TestBasicReflection(t *testing.T) {
	// Create a test sentinel instance for testing internal methods
	s := &Sentinel{
		registeredTags: make(map[string]bool),
		policies:       []Policy{},
	}

	t.Run("simple struct", func(t *testing.T) {
		type SimpleStruct struct {
			Name string `json:"name" validate:"required"`
		}

		ec := &ExtractionContext{
			Type:     reflect.TypeOf(SimpleStruct{}),
			Instance: SimpleStruct{},
		}

		result, err := s.basicReflection(context.Background(), ec)
		if err != nil {
			t.Fatalf("basicReflection failed: %v", err)
		}

		if result.Metadata.TypeName != "SimpleStruct" {
			t.Errorf("expected TypeName 'SimpleStruct', got %s", result.Metadata.TypeName)
		}
		if len(result.Metadata.Fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(result.Metadata.Fields))
		}

		field := result.Metadata.Fields[0]
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

		ec := &ExtractionContext{
			Type:     reflect.TypeOf(ComplexStruct{}),
			Instance: ComplexStruct{},
		}

		result, err := s.basicReflection(context.Background(), ec)
		if err != nil {
			t.Fatalf("basicReflection failed: %v", err)
		}

		// Should only have 3 fields (unexported excluded)
		if len(result.Metadata.Fields) != 3 {
			t.Errorf("expected 3 fields, got %d", len(result.Metadata.Fields))
		}

		// Verify field names
		expectedNames := []string{"ID", "Name", "Active"}
		for i, expected := range expectedNames {
			if result.Metadata.Fields[i].Name != expected {
				t.Errorf("field %d: expected name %s, got %s", i, expected, result.Metadata.Fields[i].Name)
			}
		}
	})

	t.Run("empty struct", func(t *testing.T) {
		type EmptyStruct struct{}

		ec := &ExtractionContext{
			Type:     reflect.TypeOf(EmptyStruct{}),
			Instance: EmptyStruct{},
		}

		result, err := s.basicReflection(context.Background(), ec)
		if err != nil {
			t.Fatalf("basicReflection failed: %v", err)
		}

		if len(result.Metadata.Fields) != 0 {
			t.Errorf("expected 0 fields for empty struct, got %d", len(result.Metadata.Fields))
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

func TestValidateMetadata(t *testing.T) {
	s := &Sentinel{}

	t.Run("valid metadata", func(t *testing.T) {
		ec := &ExtractionContext{
			Metadata: ModelMetadata{
				TypeName: "TestStruct",
			},
		}

		result, err := s.validateMetadata(context.Background(), ec)
		if err != nil {
			t.Errorf("validateMetadata failed: %v", err)
		}
		if result != ec {
			t.Error("expected same ExtractionContext returned")
		}
	})

	t.Run("missing type name", func(t *testing.T) {
		ec := &ExtractionContext{
			Metadata: ModelMetadata{
				TypeName: "",
			},
		}

		_, err := s.validateMetadata(context.Background(), ec)
		if err == nil {
			t.Error("expected error for missing type name")
		}
		if err.Error() != "extracted metadata missing type name" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestBuildExtractionPipeline(t *testing.T) {
	s := &Sentinel{
		registeredTags: make(map[string]bool),
		policies:       []Policy{},
	}

	pipeline := s.buildExtractionPipeline()
	if pipeline == nil {
		t.Fatal("expected non-nil pipeline")
	}

	// Test that pipeline has expected stages
	// This is more of an integration test - we'd need to expose pipeline internals
	// to test more thoroughly
	t.Run("pipeline executes", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		ec := &ExtractionContext{
			Type:     reflect.TypeOf(TestStruct{}),
			Instance: TestStruct{},
		}

		result, err := pipeline.Process(context.Background(), ec)
		if err != nil {
			t.Fatalf("pipeline execution failed: %v", err)
		}

		if result.Metadata.TypeName != "TestStruct" {
			t.Errorf("expected TypeName 'TestStruct', got %s", result.Metadata.TypeName)
		}
		if len(result.Metadata.Fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(result.Metadata.Fields))
		}
	})
}
