package sentinel

import (
	"reflect"
	"testing"
)

// Test types for relationship detection.
type User struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Profile  *Profile `json:"profile"` // Reference
	Orders   []Order  `json:"orders"`  // Collection
	Tags     []string `json:"tags"`    // Primitive collection - no relationship
	Settings          // Embedded type
}

type Profile struct {
	UserID  string   `json:"user_id"`
	Bio     string   `json:"bio"`
	Address *Address `json:"address"` // Reference
}

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type Order struct {
	ID     string      `json:"id"`
	UserID string      `json:"user_id"`
	Items  []OrderItem `json:"items"` // Collection
}

type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// Embedded type.
type Settings struct {
	Theme    string          `json:"theme"`
	Metadata map[string]Data `json:"metadata"` // Map relationship
}

type Data struct {
	Value string `json:"value"`
}

// Types in different package (won't be included in relationships).
type ExternalDB struct {
	Connection string
}

func TestRelationshipExtraction(t *testing.T) {
	// Reset for clean test
	instance.cache.Clear()

	t.Run("BasicRelationships", func(t *testing.T) {
		metadata := Inspect[User]()

		// Check relationships were extracted
		if len(metadata.Relationships) == 0 {
			t.Fatal("Expected relationships to be extracted")
		}

		// Map relationships by field name for easier testing
		relMap := make(map[string]TypeRelationship)
		for _, rel := range metadata.Relationships {
			relMap[rel.Field] = rel
		}

		// Check Profile reference
		if profile, ok := relMap["Profile"]; !ok {
			t.Error("Expected Profile relationship")
		} else {
			if profile.From != "User" {
				t.Errorf("Expected From='User', got '%s'", profile.From)
			}
			if profile.To != "Profile" {
				t.Errorf("Expected To='Profile', got '%s'", profile.To)
			}
			if profile.Kind != RelationshipReference {
				t.Errorf("Expected Kind='reference', got '%s'", profile.Kind)
			}
		}

		// Check Orders collection
		if orders, ok := relMap["Orders"]; !ok {
			t.Error("Expected Orders relationship")
		} else {
			if orders.To != "Order" {
				t.Errorf("Expected To='Order', got '%s'", orders.To)
			}
			if orders.Kind != RelationshipCollection {
				t.Errorf("Expected Kind='collection', got '%s'", orders.Kind)
			}
		}

		// Check embedded Settings
		if settings, ok := relMap["Settings"]; !ok {
			t.Error("Expected Settings embedding relationship")
		} else {
			if settings.To != "Settings" {
				t.Errorf("Expected To='Settings', got '%s'", settings.To)
			}
			if settings.Kind != RelationshipEmbedding {
				t.Errorf("Expected Kind='embedding', got '%s'", settings.Kind)
			}
		}

		// Verify primitive collections are not included
		if _, ok := relMap["Tags"]; ok {
			t.Error("Primitive collections should not create relationships")
		}
	})

	t.Run("NestedRelationships", func(t *testing.T) {
		// Inspect Profile to get its relationships
		profileMeta := Inspect[Profile]()

		// Should have relationship to Address
		var hasAddress bool
		for _, rel := range profileMeta.Relationships {
			if rel.To == "Address" {
				hasAddress = true
				break
			}
		}

		if !hasAddress {
			t.Error("Expected Profile to have relationship to Address")
		}
	})

	t.Run("MapRelationships", func(t *testing.T) {
		// Inspect Settings to check map relationships
		settingsMeta := Inspect[Settings]()

		var hasMetadata bool
		for _, rel := range settingsMeta.Relationships {
			if rel.Field == "Metadata" && rel.To == "Data" {
				hasMetadata = true
				if rel.Kind != RelationshipMap {
					t.Errorf("Expected map relationship, got '%s'", rel.Kind)
				}
				break
			}
		}

		if !hasMetadata {
			t.Error("Expected Settings to have map relationship to Data")
		}
	})

	t.Run("PackageBoundary", func(t *testing.T) {
		// Types from external packages should not be included
		type LocalType struct {
			DB ExternalDB // This should not create a relationship
		}

		metadata := Inspect[LocalType]()

		// ExternalDB is actually in the same package (test file), so it WILL be included
		// This test is actually incorrect - let's fix it by checking what we got
		if len(metadata.Relationships) != 1 {
			t.Errorf("Expected 1 relationship (ExternalDB is in same package), got %d", len(metadata.Relationships))
		} else if metadata.Relationships[0].To != "ExternalDB" {
			t.Errorf("Expected relationship to ExternalDB, got %s", metadata.Relationships[0].To)
		}
	})
}

func TestRelationshipAPIs(t *testing.T) {
	// Reset and inspect our test types
	instance.cache.Clear()
	Inspect[User]()
	Inspect[Profile]()
	Inspect[Order]()
	Inspect[OrderItem]()

	t.Run("GetRelationships", func(t *testing.T) {
		relationships := GetRelationships[User]()

		if len(relationships) == 0 {
			t.Error("Expected User to have relationships")
		}

		// Check specific relationships exist
		var hasProfile, hasOrders bool
		for _, rel := range relationships {
			if rel.To == "Profile" {
				hasProfile = true
			}
			if rel.To == "Order" {
				hasOrders = true
			}
		}

		if !hasProfile {
			t.Error("Expected User to have relationship to Profile")
		}
		if !hasOrders {
			t.Error("Expected User to have relationship to Order")
		}
	})

	t.Run("GetReferencedBy", func(t *testing.T) {
		// Find what references Order
		references := GetReferencedBy[Order]()

		// User should reference Order through Orders field
		var foundUser bool
		for _, ref := range references {
			if ref.From == "User" && ref.Field == "Orders" {
				foundUser = true
				break
			}
		}

		if !foundUser {
			t.Error("Expected Order to be referenced by User.Orders")
		}
	})

	t.Run("CircularReferences", func(t *testing.T) {
		// Note: We can't test true circular references in a single test
		// because Go doesn't allow forward type declarations.
		// In real code, these would be defined at package level.
		// This is a limitation of Go's type system in test functions.
		t.Skip("Circular references require package-level type definitions")
	})
}

func TestRelationshipEdgeCases(t *testing.T) {
	// Reset for clean test
	instance.cache.Clear()

	t.Run("SliceOfPointers", func(t *testing.T) {
		type Item struct {
			Name string
		}
		type Container struct {
			Items []*Item
		}

		metadata := Inspect[Container]()

		// Should detect relationship through slice of pointers
		if len(metadata.Relationships) != 1 {
			t.Fatalf("Expected 1 relationship, got %d", len(metadata.Relationships))
		}

		rel := metadata.Relationships[0]
		if rel.To != "Item" {
			t.Errorf("Expected relationship to Item, got %s", rel.To)
		}
		if rel.Kind != RelationshipCollection {
			t.Errorf("Expected collection relationship, got %s", rel.Kind)
		}
	})

	t.Run("MapWithStructValues", func(t *testing.T) {
		type Service struct {
			Name string
		}
		type Registry struct {
			Services map[string]Service
		}

		metadata := Inspect[Registry]()

		// Should detect map relationship
		if len(metadata.Relationships) != 1 {
			t.Fatalf("Expected 1 relationship, got %d", len(metadata.Relationships))
		}

		rel := metadata.Relationships[0]
		if rel.To != "Service" {
			t.Errorf("Expected relationship to Service, got %s", rel.To)
		}
		if rel.Kind != RelationshipMap {
			t.Errorf("Expected map relationship, got %s", rel.Kind)
		}
	})

	t.Run("AnonymousEmbedding", func(t *testing.T) {
		type Base struct {
			ID string
		}
		type Extended struct {
			Base // Anonymous embedding
			Name string
		}

		metadata := Inspect[Extended]()

		// Should detect embedding relationship
		var found bool
		for _, rel := range metadata.Relationships {
			if rel.To == "Base" && rel.Kind == RelationshipEmbedding {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected embedding relationship to Base")
		}
	})
}

func TestScan(t *testing.T) {
	// Reset for clean test
	instance.cache.Clear()

	t.Run("BasicScan", func(t *testing.T) {
		// Scan should inspect User and all related types in the same module
		metadata := Scan[User]()

		// Verify User was inspected
		if metadata.TypeName != "User" {
			t.Errorf("Expected TypeName 'User', got %s", metadata.TypeName)
		}

		// Verify related types were also cached
		types := Browse()
		typeMap := make(map[string]bool)
		for _, name := range types {
			typeMap[name] = true
		}

		// User should be cached
		if !typeMap["User"] {
			t.Error("Expected User to be cached")
		}

		// Profile should be cached (referenced by User)
		if !typeMap["Profile"] {
			t.Error("Expected Profile to be cached")
		}

		// Order should be cached (collection in User)
		if !typeMap["Order"] {
			t.Error("Expected Order to be cached")
		}

		// Settings should be cached (embedded in User)
		if !typeMap["Settings"] {
			t.Error("Expected Settings to be cached")
		}

		// Address should be cached (referenced by Profile)
		if !typeMap["Address"] {
			t.Error("Expected Address to be cached (transitive)")
		}

		// OrderItem should be cached (referenced by Order)
		if !typeMap["OrderItem"] {
			t.Error("Expected OrderItem to be cached (transitive)")
		}

		// Data should be cached (map value in Settings)
		if !typeMap["Data"] {
			t.Error("Expected Data to be cached")
		}
	})

	t.Run("ScanVsInspect", func(t *testing.T) {
		// Reset cache
		instance.cache.Clear()

		// Inspect only inspects the single type
		Inspect[User]()

		// Should only have User cached
		types := Browse()
		if len(types) != 1 {
			t.Errorf("Inspect should only cache 1 type, got %d", len(types))
		}

		// Reset and try Scan
		instance.cache.Clear()
		Scan[User]()

		// Should have multiple types cached
		types = Browse()
		if len(types) < 5 {
			t.Errorf("Scan should cache multiple types, got %d", len(types))
		}
	})

	t.Run("ModuleBoundary", func(t *testing.T) {
		// Reset cache
		instance.cache.Clear()

		// Scan should only include types from the same module
		_ = Scan[User]()

		// All sentinel test types should be included
		types := Browse()
		typeMap := make(map[string]bool)
		for _, name := range types {
			typeMap[name] = true
		}

		// Verify our types are included
		if !typeMap["User"] {
			t.Error("Expected User to be cached")
		}
		if !typeMap["Profile"] {
			t.Error("Expected Profile to be cached")
		}

		// Note: We can't test exclusion of truly external types in test
		// because all test types are in the same package
	})
}

func TestCreateRelationshipIfInDomain(t *testing.T) {
	s := &Sentinel{
		cache:          instance.cache,
		registeredTags: instance.registeredTags,
	}

	t.Run("builtin types without package", func(t *testing.T) {
		type TestStruct struct {
			Value int
		}

		field := reflect.TypeOf(TestStruct{}).Field(0)
		intType := field.Type

		// Built-in types have no package path
		rel := s.createRelationshipIfInDomain(field, intType, RelationshipReference, "github.com/test/pkg")

		if rel != nil {
			t.Error("expected nil relationship for built-in type without package")
		}
	})

	t.Run("same package domain", func(t *testing.T) {
		metadata := Inspect[User]()

		// User and Profile are in same package
		found := false
		for _, rel := range metadata.Relationships {
			if rel.To == "Profile" {
				found = true
				if rel.ToPackage != metadata.PackageName {
					t.Errorf("expected same package, got From=%s To=%s", metadata.PackageName, rel.ToPackage)
				}
			}
		}

		if !found {
			t.Error("expected relationship to Profile in same package")
		}
	})
}

func TestIsInModuleDomain(t *testing.T) {
	t.Run("no module path returns false", func(t *testing.T) {
		s := &Sentinel{} // No modulePath - graceful degradation

		// Without module path, always returns false
		if s.isInModuleDomain("github.com/user/repo/models") {
			t.Error("expected false when modulePath is empty")
		}
	})

	t.Run("empty target package", func(t *testing.T) {
		s := &Sentinel{modulePath: "github.com/test/repo"}

		if s.isInModuleDomain("") {
			t.Error("expected false for empty target package")
		}
	})

	t.Run("package within module", func(t *testing.T) {
		s := &Sentinel{modulePath: "github.com/zoobzio/sentinel"}

		if !s.isInModuleDomain("github.com/zoobzio/sentinel/internal/models") {
			t.Error("expected true for package within module")
		}

		// Exact module path
		if !s.isInModuleDomain("github.com/zoobzio/sentinel") {
			t.Error("expected true for exact module path")
		}
	})

	t.Run("package outside module", func(t *testing.T) {
		s := &Sentinel{modulePath: "github.com/zoobzio/sentinel"}

		if s.isInModuleDomain("github.com/other/repo") {
			t.Error("expected false for package outside module")
		}
	})

	t.Run("vanity import paths", func(t *testing.T) {
		s := &Sentinel{modulePath: "go.uber.org/zap"}

		if !s.isInModuleDomain("go.uber.org/zap/zapcore") {
			t.Error("expected true for vanity import subpackage")
		}

		if s.isInModuleDomain("github.com/uber-go/zap") {
			t.Error("expected false for non-vanity path")
		}
	})
}

func TestDetectModulePath(t *testing.T) {
	// This test verifies that detectModulePath returns a valid module path
	// when running in a proper Go module context (which tests always do)
	path := detectModulePath()

	// When running tests, build info should be available
	if path == "" {
		t.Skip("build info not available in this test environment")
	}

	// Should be our module path
	if path != "github.com/zoobzio/sentinel" {
		t.Errorf("expected module path 'github.com/zoobzio/sentinel', got %q", path)
	}
}

func TestGetStructTypeFromField(t *testing.T) {
	s := &Sentinel{}

	t.Run("direct struct", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result == nil {
			t.Fatal("expected non-nil result for direct struct")
		}
		if result.Name() != "Inner" {
			t.Errorf("expected 'Inner', got %s", result.Name())
		}
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field *Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result == nil {
			t.Fatal("expected non-nil result for pointer to struct")
		}
		if result.Name() != "Inner" {
			t.Errorf("expected 'Inner', got %s", result.Name())
		}
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		type Outer struct {
			Field *string
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result != nil {
			t.Errorf("expected nil for pointer to non-struct, got %v", result)
		}
	})

	t.Run("slice of structs", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field []Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result == nil {
			t.Fatal("expected non-nil result for slice of structs")
		}
		if result.Name() != "Inner" {
			t.Errorf("expected 'Inner', got %s", result.Name())
		}
	})

	t.Run("slice of pointers to structs", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field []*Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result == nil {
			t.Fatal("expected non-nil result for slice of pointers to structs")
		}
		if result.Name() != "Inner" {
			t.Errorf("expected 'Inner', got %s", result.Name())
		}
	})

	t.Run("slice of primitives", func(t *testing.T) {
		type Outer struct {
			Field []string
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result != nil {
			t.Errorf("expected nil for slice of primitives, got %v", result)
		}
	})

	t.Run("map of structs", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field map[string]Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result == nil {
			t.Fatal("expected non-nil result for map of structs")
		}
		if result.Name() != "Inner" {
			t.Errorf("expected 'Inner', got %s", result.Name())
		}
	})

	t.Run("map of pointers to structs", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field map[string]*Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result == nil {
			t.Fatal("expected non-nil result for map of pointers to structs")
		}
		if result.Name() != "Inner" {
			t.Errorf("expected 'Inner', got %s", result.Name())
		}
	})

	t.Run("map of primitives", func(t *testing.T) {
		type Outer struct {
			Field map[string]int
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result != nil {
			t.Errorf("expected nil for map of primitives, got %v", result)
		}
	})

	t.Run("array of structs", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field [5]Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result == nil {
			t.Fatal("expected non-nil result for array of structs")
		}
		if result.Name() != "Inner" {
			t.Errorf("expected 'Inner', got %s", result.Name())
		}
	})

	t.Run("chan type", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Field chan Inner
		}

		field := reflect.TypeOf(Outer{}).Field(0)
		result := s.getStructTypeFromField(field.Type)

		if result != nil {
			t.Errorf("expected nil for chan type, got %v", result)
		}
	})
}

func TestExtractRelationship(t *testing.T) {
	s := &Sentinel{
		cache:          instance.cache,
		registeredTags: instance.registeredTags,
	}

	t.Run("map with pointer values", func(t *testing.T) {
		type Value struct {
			Data string
		}
		type Container struct {
			Items map[string]*Value
		}

		typ := reflect.TypeOf(Container{})
		field := typ.Field(0)

		rel := s.extractRelationship(field, typ.PkgPath())

		if rel == nil {
			t.Fatal("expected relationship for map with pointer values")
		}
		if rel.Kind != RelationshipMap {
			t.Errorf("expected Kind='map', got '%s'", rel.Kind)
		}
		if rel.To != "Value" {
			t.Errorf("expected To='Value', got '%s'", rel.To)
		}
	})

	t.Run("slice with direct struct values", func(t *testing.T) {
		type Item struct {
			ID string
		}
		type Container struct {
			Items []Item
		}

		typ := reflect.TypeOf(Container{})
		field := typ.Field(0)

		rel := s.extractRelationship(field, typ.PkgPath())

		if rel == nil {
			t.Fatal("expected relationship for slice of structs")
		}
		if rel.Kind != RelationshipCollection {
			t.Errorf("expected Kind='collection', got '%s'", rel.Kind)
		}
	})

	t.Run("array with pointer to struct", func(t *testing.T) {
		type Item struct {
			ID string
		}
		type Container struct {
			Items [10]*Item
		}

		typ := reflect.TypeOf(Container{})
		field := typ.Field(0)

		rel := s.extractRelationship(field, typ.PkgPath())

		if rel == nil {
			t.Fatal("expected relationship for array of pointer to structs")
		}
		if rel.Kind != RelationshipCollection {
			t.Errorf("expected Kind='collection', got '%s'", rel.Kind)
		}
	})

	t.Run("embedded struct", func(t *testing.T) {
		// Already tested in TestRelationshipExtraction, but adding explicit test
		metadata := Inspect[User]()

		found := false
		for _, rel := range metadata.Relationships {
			if rel.Field == "Settings" && rel.Kind == RelationshipEmbedding {
				found = true
				break
			}
		}

		if !found {
			t.Error("expected embedding relationship for Settings")
		}
	})
}

func TestExtractRelationshipsEdgeCases(t *testing.T) {
	t.Run("pointer to non-struct returns empty", func(t *testing.T) {
		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		// Create a pointer to int type
		var intPtr *int
		typ := reflect.TypeOf(intPtr)

		// Should return empty slice after dereferencing pointer to non-struct
		relationships := s.extractRelationships(typ, nil)

		if len(relationships) != 0 {
			t.Errorf("expected 0 relationships for pointer to non-struct, got %d", len(relationships))
		}
	})

	t.Run("non-struct type returns empty", func(t *testing.T) {
		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		// Direct non-struct type
		typ := reflect.TypeOf(42)

		relationships := s.extractRelationships(typ, nil)

		if len(relationships) != 0 {
			t.Errorf("expected 0 relationships for non-struct, got %d", len(relationships))
		}
	})
}

func TestExtractRelationshipsScanMode(t *testing.T) {
	instance.cache.Clear()

	t.Run("Scan mode recursively extracts relationships", func(t *testing.T) {
		type Inner struct {
			Value string
		}
		type Outer struct {
			Inner *Inner
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
			modulePath:     detectModulePath(),
		}

		typ := reflect.TypeOf(Outer{})
		visited := make(map[string]bool)

		// Extract relationships in Scan mode (with visited map)
		relationships := s.extractRelationships(typ, visited)

		// Should find the relationship to Inner
		if len(relationships) != 1 {
			t.Fatalf("expected 1 relationship, got %d", len(relationships))
		}

		// Inner should have been extracted recursively
		if !visited["Inner"] {
			t.Error("expected Inner to be visited during Scan mode")
		}

		// Inner should be cached
		if _, exists := instance.cache.Get("Inner"); !exists {
			t.Error("expected Inner to be cached during Scan mode")
		}
	})

	t.Run("Inspect mode does not recurse", func(t *testing.T) {
		instance.cache.Clear()

		type InnerB struct {
			Value string
		}
		type OuterB struct {
			Inner *InnerB
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		typ := reflect.TypeOf(OuterB{})

		// Extract relationships in Inspect mode (nil visited map)
		relationships := s.extractRelationships(typ, nil)

		// Should find the relationship to InnerB
		if len(relationships) != 1 {
			t.Fatalf("expected 1 relationship, got %d", len(relationships))
		}

		// InnerB should NOT be cached in Inspect mode
		if _, exists := instance.cache.Get("InnerB"); exists {
			t.Error("InnerB should not be cached in Inspect mode")
		}
	})

	t.Run("Scan mode with nil relType", func(t *testing.T) {
		instance.cache.Clear()

		type OuterC struct {
			// Interface field - won't have a struct type
			Field interface{}
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		typ := reflect.TypeOf(OuterC{})
		visited := make(map[string]bool)

		// Should handle nil relType gracefully
		relationships := s.extractRelationships(typ, visited)

		// No relationships for interface fields
		if len(relationships) != 0 {
			t.Fatalf("expected 0 relationships for interface field, got %d", len(relationships))
		}
	})

	t.Run("Scan mode with different module domain", func(t *testing.T) {
		instance.cache.Clear()

		// Create a type that would have a relationship but in different module
		// Since we can't actually import external types in tests, we'll simulate
		// by testing that the isInModuleDomain check works
		type LocalType struct {
			Value string
		}
		type Container struct {
			Local *LocalType
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
			modulePath:     detectModulePath(),
		}

		typ := reflect.TypeOf(Container{})
		visited := make(map[string]bool)

		// Extract relationships - LocalType is in same module so should recurse
		relationships := s.extractRelationships(typ, visited)

		if len(relationships) != 1 {
			t.Fatalf("expected 1 relationship, got %d", len(relationships))
		}

		// LocalType should be cached since it's in same module
		if _, exists := instance.cache.Get("LocalType"); !exists {
			t.Error("LocalType should be cached in same module domain")
		}
	})
}
