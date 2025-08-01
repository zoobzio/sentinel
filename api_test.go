package catalog

import (
	"testing"
	"time"
)

// Test struct with various tags.
type TestUser struct {
	ID        string    `json:"id" db:"user_id"`
	Name      string    `json:"name" validate:"required" desc:"User's full name"`
	Email     string    `json:"email" validate:"required,email" encrypt:"pii"`
	Age       int       `json:"age" validate:"min=18,max=120"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Internal  string    // No tags
	private   string    //nolint:structcheck,unused // Unexported field for testing
}

// Simple struct for basic tests.
type SimpleStruct struct {
	Value string `json:"value"`
}

// Struct with nested types.
type NestedStruct struct {
	User    TestUser `json:"user"`
	Enabled bool     `json:"enabled"`
}

func TestInspect(t *testing.T) {
	// Register tags that we want to extract in our tests
	Tag("validate")
	Tag("encrypt")
	Tag("db")
	Tag("desc")

	t.Run("basic struct inspection", func(t *testing.T) {
		metadata := Inspect[SimpleStruct]()

		if metadata.TypeName != "SimpleStruct" {
			t.Errorf("expected TypeName 'SimpleStruct', got %s", metadata.TypeName)
		}

		if len(metadata.Fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(metadata.Fields))
		}

		field := metadata.Fields[0]
		if field.Name != "Value" {
			t.Errorf("expected field name 'Value', got %s", field.Name)
		}
		if field.Tags["json"] != "value" {
			t.Errorf("expected JSON tag 'value', got %s", field.Tags["json"])
		}
	})

	t.Run("complex struct with multiple tags", func(t *testing.T) {
		metadata := Inspect[TestUser]()

		if metadata.TypeName != "TestUser" {
			t.Errorf("expected TypeName 'TestUser', got %s", metadata.TypeName)
		}

		// Should have 6 exported fields (private field excluded)
		if len(metadata.Fields) != 6 {
			t.Fatalf("expected 6 fields, got %d", len(metadata.Fields))
		}

		// Check specific field metadata
		var emailField *FieldMetadata
		for i, f := range metadata.Fields {
			if f.Name == "Email" {
				emailField = &metadata.Fields[i]
				break
			}
		}

		if emailField == nil {
			t.Fatal("Email field not found")
		}

		if emailField.Tags["json"] != "email" {
			t.Errorf("expected JSON tag 'email', got %s", emailField.Tags["json"])
		}

		if emailField.Tags["validate"] != "required,email" {
			t.Errorf("expected validate tag 'required,email', got %s", emailField.Tags["validate"])
		}

		if emailField.Tags["encrypt"] != "pii" {
			t.Errorf("expected encrypt tag 'pii', got %s", emailField.Tags["encrypt"])
		}
	})

	t.Run("caching behavior", func(t *testing.T) {
		// Clear cache first
		cacheMutex.Lock()
		metadataCache = make(map[string]ModelMetadata)
		cacheMutex.Unlock()

		// First call should cache
		metadata1 := Inspect[TestUser]()

		// Second call should return cached value
		metadata2 := Inspect[TestUser]()

		// Verify caching worked by comparing field count (should be identical)
		if len(metadata2.Fields) != len(metadata1.Fields) {
			t.Error("expected cached metadata to have same structure")
		}

		// Verify same metadata structure
		if metadata1.TypeName != metadata2.TypeName {
			t.Error("cached metadata should have same type name")
		}
	})

	t.Run("pointer type normalization", func(t *testing.T) {
		// Both should return same metadata
		valueMeta := Inspect[TestUser]()
		pointerMeta := Inspect[*TestUser]()

		if valueMeta.TypeName != pointerMeta.TypeName {
			t.Errorf("expected same TypeName, got %s vs %s", valueMeta.TypeName, pointerMeta.TypeName)
		}

		if len(valueMeta.Fields) != len(pointerMeta.Fields) {
			t.Error("expected same number of fields for value and pointer types")
		}
	})

	t.Run("panic on non-struct types", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for non-struct type")
			}
		}()

		Inspect[string]() // Should panic
	})

	t.Run("panic on slice types", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for slice type")
			}
		}()

		Inspect[[]TestUser]() // Should panic
	})
}

func TestTag(t *testing.T) {
	t.Run("register custom tag", func(t *testing.T) {
		// Clear registered tags
		tagMutex.Lock()
		registeredTags = make(map[string]bool)
		tagMutex.Unlock()

		// Register a custom tag
		Tag("custom")

		// Verify it was registered
		tagMutex.RLock()
		registered := registeredTags["custom"]
		tagMutex.RUnlock()

		if !registered {
			t.Error("expected 'custom' tag to be registered")
		}
	})

	t.Run("custom tag extraction", func(t *testing.T) {
		// Register custom tag
		Tag("scope")

		type ScopedStruct struct {
			AdminField string `json:"admin_field" scope:"admin"`
			UserField  string `json:"user_field" scope:"user"`
		}

		metadata := Inspect[ScopedStruct]()

		// Check that scope tags were extracted
		for _, field := range metadata.Fields {
			scopeTag, exists := field.Tags["scope"]
			if !exists {
				t.Errorf("expected scope tag for field %s", field.Name)
				continue
			}

			if field.Name == "AdminField" && scopeTag != "admin" {
				t.Errorf("expected scope 'admin', got %s", scopeTag)
			}
			if field.Name == "UserField" && scopeTag != "user" {
				t.Errorf("expected scope 'user', got %s", scopeTag)
			}
		}
	})
}

func TestBrowse(t *testing.T) {
	t.Run("browse registered types", func(t *testing.T) {
		// Clear cache
		cacheMutex.Lock()
		metadataCache = make(map[string]ModelMetadata)
		cacheMutex.Unlock()

		// Register some types
		Inspect[SimpleStruct]()
		Inspect[TestUser]()
		Inspect[NestedStruct]()

		types := Browse()

		if len(types) != 3 {
			t.Fatalf("expected 3 types, got %d", len(types))
		}

		// Check that all expected types are present
		typeMap := make(map[string]bool)
		for _, typeName := range types {
			typeMap[typeName] = true
		}

		if !typeMap["SimpleStruct"] {
			t.Error("SimpleStruct not found in Browse results")
		}
		if !typeMap["TestUser"] {
			t.Error("TestUser not found in Browse results")
		}
		if !typeMap["NestedStruct"] {
			t.Error("NestedStruct not found in Browse results")
		}
	})

	t.Run("empty browse", func(t *testing.T) {
		// Clear cache
		cacheMutex.Lock()
		metadataCache = make(map[string]ModelMetadata)
		cacheMutex.Unlock()

		types := Browse()

		if len(types) != 0 {
			t.Errorf("expected empty browse result, got %d types", len(types))
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("struct with no fields", func(t *testing.T) {
		type EmptyStruct struct{}

		metadata := Inspect[EmptyStruct]()

		if len(metadata.Fields) != 0 {
			t.Errorf("expected 0 fields, got %d", len(metadata.Fields))
		}
	})

	t.Run("struct with only unexported fields", func(t *testing.T) {
		type PrivateStruct struct {
			private1 string //nolint:unused // intentionally unused for testing
			private2 int    //nolint:unused // intentionally unused for testing
		}

		metadata := Inspect[PrivateStruct]()

		if len(metadata.Fields) != 0 {
			t.Errorf("expected 0 exported fields, got %d", len(metadata.Fields))
		}
	})

	t.Run("nil pointer type", func(t *testing.T) {
		metadata := Inspect[*TestUser]()

		// Should still work and return metadata
		if metadata.TypeName != "TestUser" {
			t.Errorf("expected TypeName 'TestUser', got %s", metadata.TypeName)
		}
	})
}
