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
		profileMeta := Inspect[Profile]()
		orderMeta := Inspect[Order]()
		settingsMeta := Inspect[Settings]()

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
			if profile.From != metadata.FQDN {
				t.Errorf("Expected From='%s', got '%s'", metadata.FQDN, profile.From)
			}
			if profile.To != profileMeta.FQDN {
				t.Errorf("Expected To='%s', got '%s'", profileMeta.FQDN, profile.To)
			}
			if profile.Kind != RelationshipReference {
				t.Errorf("Expected Kind='reference', got '%s'", profile.Kind)
			}
		}

		// Check Orders collection
		if orders, ok := relMap["Orders"]; !ok {
			t.Error("Expected Orders relationship")
		} else {
			if orders.To != orderMeta.FQDN {
				t.Errorf("Expected To='%s', got '%s'", orderMeta.FQDN, orders.To)
			}
			if orders.Kind != RelationshipCollection {
				t.Errorf("Expected Kind='collection', got '%s'", orders.Kind)
			}
		}

		// Check embedded Settings
		if settings, ok := relMap["Settings"]; !ok {
			t.Error("Expected Settings embedding relationship")
		} else {
			if settings.To != settingsMeta.FQDN {
				t.Errorf("Expected To='%s', got '%s'", settingsMeta.FQDN, settings.To)
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
		// Inspect Profile and Address to get FQDNs
		profileMeta := Inspect[Profile]()
		addressMeta := Inspect[Address]()

		// Should have relationship to Address
		var hasAddress bool
		for _, rel := range profileMeta.Relationships {
			if rel.To == addressMeta.FQDN {
				hasAddress = true
				break
			}
		}

		if !hasAddress {
			t.Errorf("Expected Profile to have relationship to Address (%s)", addressMeta.FQDN)
		}
	})

	t.Run("MapRelationships", func(t *testing.T) {
		// Inspect Settings and Data to get FQDNs
		settingsMeta := Inspect[Settings]()
		dataMeta := Inspect[Data]()

		var hasMetadata bool
		for _, rel := range settingsMeta.Relationships {
			if rel.Field == "Metadata" && rel.To == dataMeta.FQDN {
				hasMetadata = true
				if rel.Kind != RelationshipMap {
					t.Errorf("Expected map relationship, got '%s'", rel.Kind)
				}
				break
			}
		}

		if !hasMetadata {
			t.Errorf("Expected Settings to have map relationship to Data (%s)", dataMeta.FQDN)
		}
	})

	t.Run("PackageBoundary", func(t *testing.T) {
		// Types from external packages should not be included
		type LocalType struct {
			DB ExternalDB // This should not create a relationship
		}

		metadata := Inspect[LocalType]()
		externalMeta := Inspect[ExternalDB]()

		// ExternalDB is actually in the same package (test file), so it WILL be included
		// This test is actually incorrect - let's fix it by checking what we got
		if len(metadata.Relationships) != 1 {
			t.Errorf("Expected 1 relationship (ExternalDB is in same package), got %d", len(metadata.Relationships))
		} else if metadata.Relationships[0].To != externalMeta.FQDN {
			t.Errorf("Expected relationship to ExternalDB (%s), got %s", externalMeta.FQDN, metadata.Relationships[0].To)
		}
	})
}

func TestRelationshipAPIs(t *testing.T) {
	// Reset and inspect our test types
	instance.cache.Clear()
	userMeta := Inspect[User]()
	profileMeta := Inspect[Profile]()
	orderMeta := Inspect[Order]()
	Inspect[OrderItem]()

	t.Run("GetRelationships", func(t *testing.T) {
		relationships := GetRelationships[User]()

		if len(relationships) == 0 {
			t.Error("Expected User to have relationships")
		}

		// Check specific relationships exist (using FQDNs)
		var hasProfile, hasOrders bool
		for _, rel := range relationships {
			if rel.To == profileMeta.FQDN {
				hasProfile = true
			}
			if rel.To == orderMeta.FQDN {
				hasOrders = true
			}
		}

		if !hasProfile {
			t.Errorf("Expected User to have relationship to Profile (%s)", profileMeta.FQDN)
		}
		if !hasOrders {
			t.Errorf("Expected User to have relationship to Order (%s)", orderMeta.FQDN)
		}
	})

	t.Run("GetReferencedBy", func(t *testing.T) {
		// Find what references Order
		references := GetReferencedBy[Order]()

		// User should reference Order through Orders field (using FQDN)
		var foundUser bool
		for _, ref := range references {
			if ref.From == userMeta.FQDN && ref.Field == "Orders" {
				foundUser = true
				break
			}
		}

		if !foundUser {
			t.Errorf("Expected Order to be referenced by User.Orders (from %s)", userMeta.FQDN)
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
		itemMeta := Inspect[Item]()

		// Should detect relationship through slice of pointers
		if len(metadata.Relationships) != 1 {
			t.Fatalf("Expected 1 relationship, got %d", len(metadata.Relationships))
		}

		rel := metadata.Relationships[0]
		if rel.To != itemMeta.FQDN {
			t.Errorf("Expected relationship to Item (%s), got %s", itemMeta.FQDN, rel.To)
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
		serviceMeta := Inspect[Service]()

		// Should detect map relationship
		if len(metadata.Relationships) != 1 {
			t.Fatalf("Expected 1 relationship, got %d", len(metadata.Relationships))
		}

		rel := metadata.Relationships[0]
		if rel.To != serviceMeta.FQDN {
			t.Errorf("Expected relationship to Service (%s), got %s", serviceMeta.FQDN, rel.To)
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
		baseMeta := Inspect[Base]()

		// Should detect embedding relationship
		var found bool
		for _, rel := range metadata.Relationships {
			if rel.To == baseMeta.FQDN && rel.Kind == RelationshipEmbedding {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected embedding relationship to Base (%s)", baseMeta.FQDN)
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

		// Get FQDNs for related types
		profileMeta := Inspect[Profile]()
		orderMeta := Inspect[Order]()
		settingsMeta := Inspect[Settings]()
		addressMeta := Inspect[Address]()
		orderItemMeta := Inspect[OrderItem]()
		dataMeta := Inspect[Data]()

		// Verify related types were also cached
		types := Browse()
		typeMap := make(map[string]bool)
		for _, name := range types {
			typeMap[name] = true
		}

		// User should be cached
		if !typeMap[metadata.FQDN] {
			t.Errorf("Expected User (%s) to be cached", metadata.FQDN)
		}

		// Profile should be cached (referenced by User)
		if !typeMap[profileMeta.FQDN] {
			t.Errorf("Expected Profile (%s) to be cached", profileMeta.FQDN)
		}

		// Order should be cached (collection in User)
		if !typeMap[orderMeta.FQDN] {
			t.Errorf("Expected Order (%s) to be cached", orderMeta.FQDN)
		}

		// Settings should be cached (embedded in User)
		if !typeMap[settingsMeta.FQDN] {
			t.Errorf("Expected Settings (%s) to be cached", settingsMeta.FQDN)
		}

		// Address should be cached (referenced by Profile)
		if !typeMap[addressMeta.FQDN] {
			t.Errorf("Expected Address (%s) to be cached (transitive)", addressMeta.FQDN)
		}

		// OrderItem should be cached (referenced by Order)
		if !typeMap[orderItemMeta.FQDN] {
			t.Errorf("Expected OrderItem (%s) to be cached (transitive)", orderItemMeta.FQDN)
		}

		// Data should be cached (map value in Settings)
		if !typeMap[dataMeta.FQDN] {
			t.Errorf("Expected Data (%s) to be cached", dataMeta.FQDN)
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
		userMeta := Scan[User]()
		profileMeta := Inspect[Profile]()

		// All sentinel test types should be included
		types := Browse()
		typeMap := make(map[string]bool)
		for _, name := range types {
			typeMap[name] = true
		}

		// Verify our types are included (using FQDNs)
		if !typeMap[userMeta.FQDN] {
			t.Errorf("Expected User (%s) to be cached", userMeta.FQDN)
		}
		if !typeMap[profileMeta.FQDN] {
			t.Errorf("Expected Profile (%s) to be cached", profileMeta.FQDN)
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
		profileMeta := Inspect[Profile]()

		// User and Profile are in same package
		found := false
		for _, rel := range metadata.Relationships {
			if rel.To == profileMeta.FQDN {
				found = true
				if rel.ToPackage != metadata.PackageName {
					t.Errorf("expected same package, got From=%s To=%s", metadata.PackageName, rel.ToPackage)
				}
			}
		}

		if !found {
			t.Errorf("expected relationship to Profile (%s) in same package", profileMeta.FQDN)
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
		valueMeta := Inspect[Value]()

		rel := s.extractRelationship(field, typ.PkgPath())

		if rel == nil {
			t.Fatal("expected relationship for map with pointer values")
		}
		if rel.Kind != RelationshipMap {
			t.Errorf("expected Kind='map', got '%s'", rel.Kind)
		}
		if rel.To != valueMeta.FQDN {
			t.Errorf("expected To='%s', got '%s'", valueMeta.FQDN, rel.To)
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
		innerType := reflect.TypeOf(Inner{})
		innerFQDN := getFQDN(innerType)
		visited := make(map[string]bool)

		// Extract relationships in Scan mode (with visited map)
		relationships := s.extractRelationships(typ, visited)

		// Should find the relationship to Inner
		if len(relationships) != 1 {
			t.Fatalf("expected 1 relationship, got %d", len(relationships))
		}

		// Inner should have been extracted recursively (using FQDN)
		if !visited[innerFQDN] {
			t.Errorf("expected Inner (%s) to be visited during Scan mode", innerFQDN)
		}

		// Inner should be cached (using FQDN)
		if _, exists := instance.cache.Get(innerFQDN); !exists {
			t.Errorf("expected Inner (%s) to be cached during Scan mode", innerFQDN)
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
		innerType := reflect.TypeOf(InnerB{})
		innerFQDN := getFQDN(innerType)

		// Extract relationships in Inspect mode (nil visited map)
		relationships := s.extractRelationships(typ, nil)

		// Should find the relationship to InnerB
		if len(relationships) != 1 {
			t.Fatalf("expected 1 relationship, got %d", len(relationships))
		}

		// InnerB should NOT be cached in Inspect mode (using FQDN)
		if _, exists := instance.cache.Get(innerFQDN); exists {
			t.Errorf("InnerB (%s) should not be cached in Inspect mode", innerFQDN)
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
		localType := reflect.TypeOf(LocalType{})
		localFQDN := getFQDN(localType)
		visited := make(map[string]bool)

		// Extract relationships - LocalType is in same module so should recurse
		relationships := s.extractRelationships(typ, visited)

		if len(relationships) != 1 {
			t.Fatalf("expected 1 relationship, got %d", len(relationships))
		}

		// LocalType should be cached since it's in same module (using FQDN)
		if _, exists := instance.cache.Get(localFQDN); !exists {
			t.Errorf("LocalType (%s) should be cached in same module domain", localFQDN)
		}
	})
}
