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
		// Verify new fields
		if len(field.Index) != 1 || field.Index[0] != 0 {
			t.Errorf("expected Index [0], got %v", field.Index)
		}
		if field.Kind != KindScalar {
			t.Errorf("expected Kind 'scalar', got %s", field.Kind)
		}
		if field.ReflectType == nil || field.ReflectType.Kind() != reflect.String {
			t.Errorf("expected ReflectType to be string, got %v", field.ReflectType)
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

func TestExtractMetadataInternal(t *testing.T) {
	t.Run("cache hit with visited map", func(t *testing.T) {
		instance.cache.Clear()

		type CachedType struct {
			Name string `json:"name"`
		}

		// First call - populate cache
		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		typ := reflect.TypeOf(CachedType{})
		visited := make(map[string]bool)

		// First extraction
		metadata1 := s.extractMetadataInternal(typ, visited)
		if metadata1.TypeName != "CachedType" {
			t.Errorf("expected TypeName 'CachedType', got %s", metadata1.TypeName)
		}

		// Second call with visited map - should hit cache
		visited2 := make(map[string]bool)
		metadata2 := s.extractMetadataInternal(typ, visited2)
		if metadata2.TypeName != "CachedType" {
			t.Errorf("expected cached TypeName 'CachedType', got %s", metadata2.TypeName)
		}
	})

	t.Run("nil cache handling", func(t *testing.T) {
		type NoCacheType struct {
			Value int `json:"value"`
		}

		// Sentinel with nil cache
		s := &Sentinel{
			cache:          nil,
			registeredTags: instance.registeredTags,
		}

		typ := reflect.TypeOf(NoCacheType{})
		metadata := s.extractMetadataInternal(typ, nil)

		if metadata.TypeName != "NoCacheType" {
			t.Errorf("expected TypeName 'NoCacheType', got %s", metadata.TypeName)
		}
		if len(metadata.Fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(metadata.Fields))
		}
	})

	t.Run("cycle detection with visited map", func(t *testing.T) {
		instance.cache.Clear()

		type CircularA struct {
			Name string `json:"name"`
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		typ := reflect.TypeOf(CircularA{})
		visited := make(map[string]bool)

		// Mark as already visited
		visited["CircularA"] = true

		// Should return cached or empty metadata
		_ = s.extractMetadataInternal(typ, visited)

		// The type should be skipped due to already being visited
		// If cache exists, it returns cached, otherwise empty
		if visited["CircularA"] != true {
			t.Error("expected type to remain in visited map")
		}
	})

	t.Run("visited but not cached", func(t *testing.T) {
		instance.cache.Clear()

		type UncachedType struct {
			Value string `json:"value"`
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		typ := reflect.TypeOf(UncachedType{})
		visited := make(map[string]bool)

		// Mark as visited but don't cache it
		visited["UncachedType"] = true

		// Should return empty metadata since it's visited but not in cache
		metadata := s.extractMetadataInternal(typ, visited)

		if metadata.TypeName != "" {
			t.Errorf("expected empty metadata for visited but uncached type, got %s", metadata.TypeName)
		}
	})

	t.Run("visited and cached returns cached", func(t *testing.T) {
		instance.cache.Clear()

		type CycleType struct {
			Name string `json:"name"`
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		typ := reflect.TypeOf(CycleType{})

		// Pre-populate cache with metadata
		cachedMeta := Metadata{
			TypeName:    "CycleType",
			PackageName: "sentinel",
			Fields: []FieldMetadata{
				{Name: "Name", Type: "string", Tags: map[string]string{"json": "name"}},
			},
		}
		instance.cache.Set("CycleType", cachedMeta)

		// Mark as visited AND cached - simulates hitting same type twice in circular ref
		visited := make(map[string]bool)
		visited["CycleType"] = true

		// Should return cached metadata
		metadata := s.extractMetadataInternal(typ, visited)

		if metadata.TypeName != "CycleType" {
			t.Errorf("expected cached TypeName 'CycleType', got %s", metadata.TypeName)
		}
		if len(metadata.Fields) != 1 {
			t.Errorf("expected 1 field from cache, got %d", len(metadata.Fields))
		}
	})

	t.Run("cached with visited map triggers relationship scan", func(t *testing.T) {
		instance.cache.Clear()

		type Related struct {
			Value string
		}
		type Root struct {
			Related *Related
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
			modulePath:     detectModulePath(),
		}

		rootType := reflect.TypeOf(Root{})

		// First call - populate cache without visited map (Inspect mode)
		_ = s.extractMetadataInternal(rootType, nil)

		// Related should NOT be in cache yet
		if _, exists := instance.cache.Get("Related"); exists {
			t.Error("Related should not be cached after Inspect mode")
		}

		// Second call with visited map (Scan mode) - should trigger relationship scan
		visited := make(map[string]bool)
		_ = s.extractMetadataInternal(rootType, visited)

		// Now Related should be in cache
		if _, exists := instance.cache.Get("Related"); !exists {
			t.Error("Related should be cached after Scan mode on cached type")
		}
	})

	t.Run("nil type handling", func(t *testing.T) {
		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		metadata := s.extractMetadataInternal(nil, nil)

		if metadata.TypeName != "" {
			t.Errorf("expected empty metadata for nil type, got %s", metadata.TypeName)
		}
	})

	t.Run("pointer type normalization", func(t *testing.T) {
		type PointerTest struct {
			Field string `json:"field"`
		}

		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		ptrType := reflect.TypeOf(&PointerTest{})
		metadata := s.extractMetadataInternal(ptrType, nil)

		if metadata.TypeName != "PointerTest" {
			t.Errorf("expected TypeName 'PointerTest', got %s", metadata.TypeName)
		}
	})

	t.Run("non-struct type", func(t *testing.T) {
		s := &Sentinel{
			cache:          instance.cache,
			registeredTags: instance.registeredTags,
		}

		intType := reflect.TypeOf(42)
		metadata := s.extractMetadataInternal(intType, nil)

		if metadata.TypeName != "" {
			t.Errorf("expected empty metadata for int type, got %s", metadata.TypeName)
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

	t.Run("field index and kind verification", func(t *testing.T) {
		type Related struct {
			Value string
		}
		type AllKindsStruct struct {
			Scalar    string            `json:"scalar"`
			Pointer   *string           `json:"pointer"`
			Slice     []string          `json:"slice"`
			Array     [5]int            `json:"array"`
			Struct    Related           `json:"struct"`
			Map       map[string]int    `json:"map"`
			Interface interface{}       `json:"interface"`
			PtrStruct *Related          `json:"ptr_struct"`
			SlicePtr  []*Related        `json:"slice_ptr"`
		}

		fields := s.extractFieldMetadata(reflect.TypeOf(AllKindsStruct{}))
		if len(fields) != 9 {
			t.Fatalf("expected 9 fields, got %d", len(fields))
		}

		expectedKinds := []struct {
			name  string
			index int
			kind  FieldKind
		}{
			{"Scalar", 0, KindScalar},
			{"Pointer", 1, KindPointer},
			{"Slice", 2, KindSlice},
			{"Array", 3, KindSlice},
			{"Struct", 4, KindStruct},
			{"Map", 5, KindMap},
			{"Interface", 6, KindInterface},
			{"PtrStruct", 7, KindPointer},
			{"SlicePtr", 8, KindSlice},
		}

		for i, expected := range expectedKinds {
			field := fields[i]
			if field.Name != expected.name {
				t.Errorf("field %d: expected name %s, got %s", i, expected.name, field.Name)
			}
			if len(field.Index) != 1 || field.Index[0] != expected.index {
				t.Errorf("field %s: expected Index [%d], got %v", expected.name, expected.index, field.Index)
			}
			if field.Kind != expected.kind {
				t.Errorf("field %s: expected Kind %s, got %s", expected.name, expected.kind, field.Kind)
			}
			if field.ReflectType == nil {
				t.Errorf("field %s: ReflectType should not be nil", expected.name)
			}
		}
	})

	t.Run("reflect type usability", func(t *testing.T) {
		type TypeTestStruct struct {
			Name   string  `json:"name"`
			Count  int     `json:"count"`
			Active bool    `json:"active"`
			Score  float64 `json:"score"`
		}

		fields := s.extractFieldMetadata(reflect.TypeOf(TypeTestStruct{}))
		if len(fields) != 4 {
			t.Fatalf("expected 4 fields, got %d", len(fields))
		}

		// Verify ReflectType can be used for type operations
		expectedKinds := []reflect.Kind{
			reflect.String,
			reflect.Int,
			reflect.Bool,
			reflect.Float64,
		}

		for i, expectedKind := range expectedKinds {
			if fields[i].ReflectType.Kind() != expectedKind {
				t.Errorf("field %d: expected reflect.Kind %v, got %v",
					i, expectedKind, fields[i].ReflectType.Kind())
			}
		}
	})
}
