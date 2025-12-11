package integration

import (
	"sync"
	"testing"
	"time"

	"github.com/zoobzio/sentinel"
)

// Test types for relationship and scan testing.
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

func TestScanRecursiveDiscovery(t *testing.T) {
	t.Run("discovers all transitive types", func(t *testing.T) {
		metadata := sentinel.Scan[User]()

		if metadata.TypeName != "User" {
			t.Errorf("expected TypeName 'User', got %s", metadata.TypeName)
		}

		types := sentinel.Browse()
		typeMap := make(map[string]bool)
		for _, name := range types {
			typeMap[name] = true
		}

		// Direct relationships
		if !typeMap["User"] {
			t.Error("expected User to be cached")
		}
		if !typeMap["Profile"] {
			t.Error("expected Profile to be cached")
		}
		if !typeMap["Order"] {
			t.Error("expected Order to be cached")
		}
		if !typeMap["Settings"] {
			t.Error("expected Settings to be cached (embedded)")
		}

		// Transitive relationships
		if !typeMap["Address"] {
			t.Error("expected Address to be cached (via Profile)")
		}
		if !typeMap["OrderItem"] {
			t.Error("expected OrderItem to be cached (via Order)")
		}
		if !typeMap["Data"] {
			t.Error("expected Data to be cached (via Settings map)")
		}
	})
}

// Types for testing Inspect vs Scan behavior.
type Parent struct {
	Child *Child `json:"child"`
}

type Child struct {
	Value string `json:"value"`
}

func TestScanVsInspect(t *testing.T) {
	t.Run("inspect does not cache related types", func(t *testing.T) {
		// Get baseline of cached types
		baselineTypes := make(map[string]bool)
		for _, name := range sentinel.Browse() {
			baselineTypes[name] = true
		}

		sentinel.Inspect[Parent]()

		// Parent should be cached
		_, ok := sentinel.Lookup("Parent")
		if !ok {
			t.Error("Parent should be cached after Inspect")
		}

		// Child should NOT be cached by Inspect (only Scan recurses)
		_, ok = sentinel.Lookup("Child")
		if ok && !baselineTypes["Child"] {
			t.Error("Child should not be cached by Inspect alone")
		}
	})

	t.Run("scan caches related types", func(t *testing.T) {
		sentinel.Scan[Parent]()

		// Both should be cached
		_, ok := sentinel.Lookup("Parent")
		if !ok {
			t.Error("Parent should be cached after Scan")
		}

		_, ok = sentinel.Lookup("Child")
		if !ok {
			t.Error("Child should be cached after Scan")
		}
	})
}

func TestRelationshipKinds(t *testing.T) {
	metadata := sentinel.Scan[User]()

	relMap := make(map[string]sentinel.TypeRelationship)
	for _, rel := range metadata.Relationships {
		relMap[rel.Field] = rel
	}

	t.Run("reference relationship", func(t *testing.T) {
		rel, ok := relMap["Profile"]
		if !ok {
			t.Fatal("expected Profile relationship")
		}
		if rel.Kind != "reference" {
			t.Errorf("expected kind 'reference', got %s", rel.Kind)
		}
		if rel.To != "Profile" {
			t.Errorf("expected To 'Profile', got %s", rel.To)
		}
	})

	t.Run("collection relationship", func(t *testing.T) {
		rel, ok := relMap["Orders"]
		if !ok {
			t.Fatal("expected Orders relationship")
		}
		if rel.Kind != "collection" {
			t.Errorf("expected kind 'collection', got %s", rel.Kind)
		}
		if rel.To != "Order" {
			t.Errorf("expected To 'Order', got %s", rel.To)
		}
	})

	t.Run("embedding relationship", func(t *testing.T) {
		rel, ok := relMap["Settings"]
		if !ok {
			t.Fatal("expected Settings relationship")
		}
		if rel.Kind != "embedding" {
			t.Errorf("expected kind 'embedding', got %s", rel.Kind)
		}
	})

	t.Run("map relationship", func(t *testing.T) {
		// Settings has Metadata map[string]Data
		settingsMeta, ok := sentinel.Lookup("Settings")
		if !ok {
			t.Fatal("expected Settings to be cached")
		}

		var foundMapRel bool
		for _, rel := range settingsMeta.Relationships {
			if rel.Field == "Metadata" && rel.Kind == "map" && rel.To == "Data" {
				foundMapRel = true
				break
			}
		}
		if !foundMapRel {
			t.Error("expected map relationship from Settings.Metadata to Data")
		}
	})
}

func TestBrowseAfterScan(t *testing.T) {
	sentinel.Scan[User]()

	types := sentinel.Browse()

	if len(types) == 0 {
		t.Fatal("expected non-empty type list after scan")
	}

	// Should have multiple types cached
	if len(types) < 5 {
		t.Errorf("expected at least 5 types cached, got %d", len(types))
	}
}

func TestGetReferencedBy(t *testing.T) {
	sentinel.Scan[User]()

	refs := sentinel.GetReferencedBy[Profile]()

	var foundUser bool
	for _, ref := range refs {
		if ref.From == "User" {
			foundUser = true
			break
		}
	}

	if !foundUser {
		t.Error("expected User to reference Profile")
	}
}

func TestSchemaExport(t *testing.T) {
	sentinel.Scan[User]()

	schema := sentinel.Schema()

	if len(schema) == 0 {
		t.Fatal("expected non-empty schema")
	}

	user, ok := schema["User"]
	if !ok {
		t.Fatal("expected User in schema")
	}

	if len(user.Fields) == 0 {
		t.Error("expected User to have fields")
	}

	if len(user.Relationships) == 0 {
		t.Error("expected User to have relationships")
	}
}

// Deeply nested types for testing transitive scanning.
type Level1 struct {
	Next *Level2 `json:"next"`
}

type Level2 struct {
	Next *Level3 `json:"next"`
}

type Level3 struct {
	Next *Level4 `json:"next"`
}

type Level4 struct {
	Next *Level5 `json:"next"`
}

type Level5 struct {
	Value string `json:"value"`
}

func TestDeeplyNestedTypes(t *testing.T) {
	sentinel.Scan[Level1]()

	// All levels should be cached
	for _, name := range []string{"Level1", "Level2", "Level3", "Level4", "Level5"} {
		_, ok := sentinel.Lookup(name)
		if !ok {
			t.Errorf("expected %s to be cached via transitive scan", name)
		}
	}
}

func TestConcurrentScanning(t *testing.T) {
	t.Run("concurrent Inspect calls are safe", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = sentinel.Inspect[User]()
			}()
		}
		wg.Wait()

		// Should still have valid metadata
		meta, ok := sentinel.Lookup("User")
		if !ok {
			t.Fatal("User should be cached")
		}
		if meta.TypeName != "User" {
			t.Errorf("expected TypeName 'User', got %s", meta.TypeName)
		}
	})

	t.Run("concurrent Scan calls are safe", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = sentinel.Scan[User]()
			}()
		}
		wg.Wait()

		// All related types should be cached
		for _, name := range []string{"User", "Profile", "Order", "Settings"} {
			_, ok := sentinel.Lookup(name)
			if !ok {
				t.Errorf("expected %s to be cached after concurrent scans", name)
			}
		}
	})

	t.Run("concurrent mixed operations are safe", func(_ *testing.T) {
		var wg sync.WaitGroup

		// Concurrent Inspects
		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = sentinel.Inspect[User]()
			}()
		}

		// Concurrent Scans
		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = sentinel.Scan[Order]()
			}()
		}

		// Concurrent Browse
		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = sentinel.Browse()
			}()
		}

		// Concurrent Lookup
		for i := 0; i < 30; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = sentinel.Lookup("User")
			}()
		}

		wg.Wait()
	})
}

func TestPrimitiveCollectionsIgnored(t *testing.T) {
	// User.Tags is []string - should NOT create a relationship
	metadata := sentinel.Inspect[User]()

	for _, rel := range metadata.Relationships {
		if rel.Field == "Tags" {
			t.Error("primitive slice []string should not create a relationship")
		}
	}
}

// Types with pointer variations.
type WithPointers struct {
	Direct   DirectStruct            `json:"direct"`
	Pointer  *PointerStruct          `json:"pointer"`
	Slice    []SliceStruct           `json:"slice"`
	PtrSlice []*PtrSlice             `json:"ptr_slice"`
	Map      map[string]MapValue     `json:"map"`
	PtrMap   map[string]*PtrMapValue `json:"ptr_map"`
}

type DirectStruct struct{ V string }
type PointerStruct struct{ V string }
type SliceStruct struct{ V string }
type PtrSlice struct{ V string }
type MapValue struct{ V string }
type PtrMapValue struct{ V string }

func TestPointerVariations(t *testing.T) {
	metadata := sentinel.Scan[WithPointers]()

	relMap := make(map[string]sentinel.TypeRelationship)
	for _, rel := range metadata.Relationships {
		relMap[rel.Field] = rel
	}

	t.Run("direct struct field", func(t *testing.T) {
		rel, ok := relMap["Direct"]
		if !ok {
			t.Fatal("expected Direct relationship")
		}
		if rel.Kind != "reference" {
			t.Errorf("expected 'reference', got %s", rel.Kind)
		}
	})

	t.Run("pointer to struct", func(t *testing.T) {
		rel, ok := relMap["Pointer"]
		if !ok {
			t.Fatal("expected Pointer relationship")
		}
		if rel.Kind != "reference" {
			t.Errorf("expected 'reference', got %s", rel.Kind)
		}
	})

	t.Run("slice of structs", func(t *testing.T) {
		rel, ok := relMap["Slice"]
		if !ok {
			t.Fatal("expected Slice relationship")
		}
		if rel.Kind != "collection" {
			t.Errorf("expected 'collection', got %s", rel.Kind)
		}
	})

	t.Run("slice of pointers to structs", func(t *testing.T) {
		rel, ok := relMap["PtrSlice"]
		if !ok {
			t.Fatal("expected PtrSlice relationship")
		}
		if rel.Kind != "collection" {
			t.Errorf("expected 'collection', got %s", rel.Kind)
		}
	})

	t.Run("map with struct values", func(t *testing.T) {
		rel, ok := relMap["Map"]
		if !ok {
			t.Fatal("expected Map relationship")
		}
		if rel.Kind != "map" {
			t.Errorf("expected 'map', got %s", rel.Kind)
		}
	})

	t.Run("map with pointer values", func(t *testing.T) {
		rel, ok := relMap["PtrMap"]
		if !ok {
			t.Fatal("expected PtrMap relationship")
		}
		if rel.Kind != "map" {
			t.Errorf("expected 'map', got %s", rel.Kind)
		}
	})
}

func TestFieldMetadataAccuracy(t *testing.T) {
	type Detailed struct {
		ID        string    `json:"id" db:"detail_id" validate:"required"`
		Name      string    `json:"name,omitempty" validate:"min=1,max=100"`
		CreatedAt time.Time `json:"created_at" db:"created_at"`
		Data      []byte    `json:"data" encrypt:"sensitive"`
	}

	sentinel.Tag("encrypt")
	metadata := sentinel.Inspect[Detailed]()

	if len(metadata.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(metadata.Fields))
	}

	fieldMap := make(map[string]sentinel.FieldMetadata)
	for _, f := range metadata.Fields {
		fieldMap[f.Name] = f
	}

	t.Run("field types are correct", func(t *testing.T) {
		if fieldMap["ID"].Type != "string" {
			t.Errorf("expected ID type 'string', got %s", fieldMap["ID"].Type)
		}
		if fieldMap["CreatedAt"].Type != "time.Time" {
			t.Errorf("expected CreatedAt type 'time.Time', got %s", fieldMap["CreatedAt"].Type)
		}
		if fieldMap["Data"].Type != "[]uint8" {
			t.Errorf("expected Data type '[]uint8', got %s", fieldMap["Data"].Type)
		}
	})

	t.Run("json tags extracted", func(t *testing.T) {
		if fieldMap["ID"].Tags["json"] != "id" {
			t.Errorf("expected json tag 'id', got %s", fieldMap["ID"].Tags["json"])
		}
		if fieldMap["Name"].Tags["json"] != "name,omitempty" {
			t.Errorf("expected json tag 'name,omitempty', got %s", fieldMap["Name"].Tags["json"])
		}
	})

	t.Run("validate tags extracted", func(t *testing.T) {
		if fieldMap["ID"].Tags["validate"] != "required" {
			t.Errorf("expected validate tag 'required', got %s", fieldMap["ID"].Tags["validate"])
		}
		if fieldMap["Name"].Tags["validate"] != "min=1,max=100" {
			t.Errorf("expected validate tag 'min=1,max=100', got %s", fieldMap["Name"].Tags["validate"])
		}
	})

	t.Run("db tags extracted", func(t *testing.T) {
		if fieldMap["ID"].Tags["db"] != "detail_id" {
			t.Errorf("expected db tag 'detail_id', got %s", fieldMap["ID"].Tags["db"])
		}
	})

	t.Run("custom registered tags extracted", func(t *testing.T) {
		if fieldMap["Data"].Tags["encrypt"] != "sensitive" {
			t.Errorf("expected encrypt tag 'sensitive', got %s", fieldMap["Data"].Tags["encrypt"])
		}
	})
}

func TestUnexportedFieldsIgnored(t *testing.T) {
	type WithUnexported struct {
		Public  string `json:"public"`
		private string //nolint:unused
	}

	metadata := sentinel.Inspect[WithUnexported]()

	if len(metadata.Fields) != 1 {
		t.Errorf("expected 1 field (only Public), got %d", len(metadata.Fields))
	}

	if metadata.Fields[0].Name != "Public" {
		t.Errorf("expected field 'Public', got %s", metadata.Fields[0].Name)
	}
}

func TestEmptyStruct(t *testing.T) {
	type Empty struct{}

	metadata := sentinel.Inspect[Empty]()

	if metadata.TypeName != "Empty" {
		t.Errorf("expected TypeName 'Empty', got %s", metadata.TypeName)
	}

	if len(metadata.Fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(metadata.Fields))
	}

	if len(metadata.Relationships) != 0 {
		t.Errorf("expected 0 relationships, got %d", len(metadata.Relationships))
	}
}

func TestLookupNonExistent(t *testing.T) {
	_, ok := sentinel.Lookup("NonExistentType12345")
	if ok {
		t.Error("expected Lookup to return false for non-existent type")
	}
}
