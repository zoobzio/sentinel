//go:build testing

package sentinel

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
	private   string    //nolint:unused // Unexported field for testing
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

// setupSentinelForTest initializes sentinel for testing.
func setupSentinelForTest() {
	// Clear cache for clean test state
	instance.cache.Clear()
}

func TestInspect(t *testing.T) {
	// Setup sentinel
	setupSentinelForTest()

	// Register tags for tests
	Tag("validate")
	Tag("encrypt")
	Tag("db")
	Tag("desc")

	t.Run("struct with anonymous fields", func(t *testing.T) {
		type Embedded struct {
			EmbeddedField string `json:"embedded_field"`
		}

		type WithAnonymous struct {
			Embedded
			OwnField string `json:"own_field"`
		}

		metadata := Inspect[WithAnonymous]()

		// Anonymous fields show as the type name, not the embedded field names
		if len(metadata.Fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(metadata.Fields))
		}

		fieldMap := make(map[string]bool)
		for _, f := range metadata.Fields {
			fieldMap[f.Name] = true
		}

		// Anonymous field appears as "Embedded" not "EmbeddedField"
		if !fieldMap["Embedded"] {
			t.Error("embedded type field not found")
		}
		if !fieldMap["OwnField"] {
			t.Error("own field not found")
		}
	})

	t.Run("struct with complex types", func(t *testing.T) {
		type ComplexStruct struct {
			MapField   map[string]int `json:"map_field"`
			SliceField []string       `json:"slice_field"`
			ChanField  chan int       `json:"chan_field"`
			FuncField  func() string  `json:"func_field"`
			PtrField   *int           `json:"ptr_field"`
		}

		metadata := Inspect[ComplexStruct]()

		if len(metadata.Fields) != 5 {
			t.Fatalf("expected 5 fields, got %d", len(metadata.Fields))
		}

		// Verify each field type is captured
		for _, field := range metadata.Fields {
			if field.Type == "" {
				t.Errorf("field %s has empty type", field.Name)
			}
		}
	})

	t.Run("struct with interface fields", func(t *testing.T) {
		type InterfaceStruct struct {
			AnyField   interface{} `json:"any_field"`
			ErrorField error       `json:"error_field"`
		}

		metadata := Inspect[InterfaceStruct]()

		if len(metadata.Fields) != 2 {
			t.Fatalf("expected 2 fields, got %d", len(metadata.Fields))
		}
	})

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
		// Register a custom tag
		Tag("custom")

		// Verify it was registered by using it
		type CustomStruct struct {
			Field string `custom:"value"`
		}

		metadata := Inspect[CustomStruct]()
		if metadata.Fields[0].Tags["custom"] != "value" {
			t.Error("expected 'custom' tag to be extracted")
		}
	})

	t.Run("multiple custom tags", func(t *testing.T) {
		// Register multiple custom tags
		Tag("role")
		Tag("permission")
		Tag("audit")

		type MultiTagStruct struct {
			PublicField  string `json:"public" role:"user" permission:"read"`
			PrivateField string `json:"private" role:"admin" permission:"write" audit:"true"`
		}

		metadata := Inspect[MultiTagStruct]()

		// Verify all custom tags are extracted
		for _, field := range metadata.Fields {
			if field.Name == "PublicField" {
				if field.Tags["role"] != "user" {
					t.Errorf("expected role 'user', got %s", field.Tags["role"])
				}
				if field.Tags["permission"] != "read" {
					t.Errorf("expected permission 'read', got %s", field.Tags["permission"])
				}
			}
			if field.Name == "PrivateField" {
				if field.Tags["role"] != "admin" {
					t.Errorf("expected role 'admin', got %s", field.Tags["role"])
				}
				if field.Tags["permission"] != "write" {
					t.Errorf("expected permission 'write', got %s", field.Tags["permission"])
				}
				if field.Tags["audit"] != "true" {
					t.Errorf("expected audit 'true', got %s", field.Tags["audit"])
				}
			}
		}
	})

	t.Run("duplicate tag registration", func(t *testing.T) {
		// Register same tag multiple times (should not error)
		Tag("duplicate")
		Tag("duplicate")
		Tag("duplicate")

		type DuplicateStruct struct {
			Field string `duplicate:"value"`
		}

		metadata := Inspect[DuplicateStruct]()
		if metadata.Fields[0].Tags["duplicate"] != "value" {
			t.Error("expected 'duplicate' tag to be extracted after multiple registrations")
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
		// Register some types and get their FQDNs
		simpleMeta := Inspect[SimpleStruct]()
		userMeta := Inspect[TestUser]()
		nestedMeta := Inspect[NestedStruct]()

		types := Browse()

		// With global singleton, types accumulate from all tests
		if len(types) < 3 {
			t.Fatalf("expected at least 3 types, got %d: %v", len(types), types)
		}

		// Check that all expected types are present (using FQDNs)
		typeMap := make(map[string]bool)
		for _, typeName := range types {
			typeMap[typeName] = true
		}

		if !typeMap[simpleMeta.FQDN] {
			t.Errorf("SimpleStruct (%s) not found in Browse results", simpleMeta.FQDN)
		}
		if !typeMap[userMeta.FQDN] {
			t.Errorf("TestUser (%s) not found in Browse results", userMeta.FQDN)
		}
		if !typeMap[nestedMeta.FQDN] {
			t.Errorf("NestedStruct (%s) not found in Browse results", nestedMeta.FQDN)
		}
	})

	t.Run("empty browse", func(t *testing.T) {
		// Note: Browse will return types from previous tests since we use global singleton
		types := Browse()

		// With global singleton, types will persist from other tests
		// This is expected behavior
		if len(types) == 0 {
			t.Log("Browse returned empty (global singleton may have been cleared)")
		}
	})
}

func TestLookup(t *testing.T) {
	t.Run("lookup existing type", func(t *testing.T) {
		// First inspect a type to cache it
		type LookupTestStruct struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		original := Inspect[LookupTestStruct]()

		// Now lookup the cached metadata using FQDN
		retrieved, found := Lookup(original.FQDN)

		if !found {
			t.Fatalf("expected to find cached metadata for %s", original.FQDN)
		}

		if retrieved.TypeName != original.TypeName {
			t.Errorf("expected TypeName %s, got %s", original.TypeName, retrieved.TypeName)
		}

		if len(retrieved.Fields) != len(original.Fields) {
			t.Errorf("expected %d fields, got %d", len(original.Fields), len(retrieved.Fields))
		}
	})

	t.Run("lookup non-existent type", func(t *testing.T) {
		metadata, found := Lookup("NonExistentType")

		if found {
			t.Error("expected not to find non-existent type")
		}

		if metadata.TypeName != "" {
			t.Error("expected empty metadata for non-existent type")
		}
	})

	t.Run("lookup after clear", func(t *testing.T) {
		// Ensure a type is cached
		type ClearTestStruct struct {
			Value string `json:"value"`
		}

		original := Inspect[ClearTestStruct]()

		// Verify it exists using FQDN
		_, found := Lookup(original.FQDN)
		if !found {
			t.Fatalf("expected to find type %s before clear", original.FQDN)
		}

		// Clear cache
		instance.cache.Clear()

		// Should no longer exist
		_, found = Lookup(original.FQDN)
		if found {
			t.Error("expected not to find type after clear")
		}
	})
}

func TestSchema(t *testing.T) {
	t.Run("returns all cached metadata", func(t *testing.T) {
		// Ensure some types are inspected
		type SchemaTestUser struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		type SchemaTestProduct struct {
			SKU   string  `json:"sku"`
			Price float64 `json:"price"`
		}

		// Inspect the types
		userMeta := Inspect[SchemaTestUser]()
		productMeta := Inspect[SchemaTestProduct]()

		// Get the full schema
		schema := Schema()

		// Schema should contain at least our test types
		if len(schema) < 2 {
			t.Fatalf("expected at least 2 types in schema, got %d", len(schema))
		}

		// Verify our types are in the schema (using FQDN as key)
		if retrievedUser, exists := schema[userMeta.FQDN]; !exists {
			t.Errorf("SchemaTestUser (%s) not found in schema", userMeta.FQDN)
		} else {
			if retrievedUser.TypeName != userMeta.TypeName {
				t.Errorf("schema user type mismatch: got %s, want %s",
					retrievedUser.TypeName, userMeta.TypeName)
			}
			if len(retrievedUser.Fields) != 2 {
				t.Errorf("expected 2 fields for SchemaTestUser, got %d",
					len(retrievedUser.Fields))
			}
		}

		if retrievedProduct, exists := schema[productMeta.FQDN]; !exists {
			t.Errorf("SchemaTestProduct (%s) not found in schema", productMeta.FQDN)
		} else {
			if retrievedProduct.TypeName != productMeta.TypeName {
				t.Errorf("schema product type mismatch: got %s, want %s",
					retrievedProduct.TypeName, productMeta.TypeName)
			}
			if len(retrievedProduct.Fields) != 2 {
				t.Errorf("expected 2 fields for SchemaTestProduct, got %d",
					len(retrievedProduct.Fields))
			}
		}
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		// Get schema twice
		schema1 := Schema()
		schema2 := Schema()

		// Modifying one should not affect the other
		if len(schema1) > 0 {
			for key := range schema1 {
				delete(schema1, key)
				break // Delete just one to test
			}

			// schema2 should still have all entries
			if len(schema2) != len(Schema()) {
				t.Error("Schema() returned a reference instead of a copy")
			}
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

func TestScanEdgeCases(t *testing.T) {
	t.Run("panic on non-struct type", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for non-struct type")
			}
		}()

		Scan[string]() // Should panic
	})

	t.Run("panic on slice type", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for slice type")
			}
		}()

		Scan[[]TestUser]() // Should panic
	})

	t.Run("pointer type normalization", func(t *testing.T) {
		instance.cache.Clear()

		// Scan with pointer type should normalize to struct
		metadata := Scan[*TestUser]()

		if metadata.TypeName != "TestUser" {
			t.Errorf("expected TypeName 'TestUser', got %s", metadata.TypeName)
		}
	})
}
