//go:build testing

package testing

import (
	"testing"

	"github.com/zoobzio/sentinel"
)

// mockT captures test failures without failing the actual test.
type mockT struct {
	testing.TB
	errors []string
	helper bool
}

func (m *mockT) Helper() {
	m.helper = true
}

func (m *mockT) Error(_ ...any) {
	m.errors = append(m.errors, "error")
}

func (m *mockT) Errorf(format string, _ ...any) {
	m.errors = append(m.errors, format)
}

func (m *mockT) failed() bool {
	return len(m.errors) > 0
}

type HelperTestStruct struct {
	ID   string `json:"id" validate:"required"`
	Name string `json:"name"`
	Ref  *HelperRefStruct
}

type HelperRefStruct struct {
	Value string
}

func TestAssertMetadataValid(t *testing.T) {
	ResetCache(t)
	meta := sentinel.Inspect[HelperTestStruct]()

	t.Run("passes for valid metadata", func(t *testing.T) {
		AssertMetadataValid(t, meta)
	})

	t.Run("fails for empty TypeName", func(t *testing.T) {
		mock := &mockT{}
		invalid := sentinel.Metadata{PackageName: "pkg", ReflectType: meta.ReflectType}
		AssertMetadataValid(mock, invalid)
		if !mock.failed() {
			t.Error("expected failure for empty TypeName")
		}
	})

	t.Run("fails for empty PackageName", func(t *testing.T) {
		mock := &mockT{}
		invalid := sentinel.Metadata{TypeName: "Type", ReflectType: meta.ReflectType}
		AssertMetadataValid(mock, invalid)
		if !mock.failed() {
			t.Error("expected failure for empty PackageName")
		}
	})

	t.Run("fails for nil ReflectType", func(t *testing.T) {
		mock := &mockT{}
		invalid := sentinel.Metadata{TypeName: "Type", PackageName: "pkg"}
		AssertMetadataValid(mock, invalid)
		if !mock.failed() {
			t.Error("expected failure for nil ReflectType")
		}
	})
}

func TestAssertFieldExists(t *testing.T) {
	ResetCache(t)
	meta := sentinel.Inspect[HelperTestStruct]()

	t.Run("returns field when it exists", func(t *testing.T) {
		field := AssertFieldExists(t, meta, "ID")
		if field.Name != "ID" {
			t.Errorf("expected field name ID, got %s", field.Name)
		}
	})

	t.Run("fails when field does not exist", func(t *testing.T) {
		mock := &mockT{}
		field := AssertFieldExists(mock, meta, "NonExistent")
		if !mock.failed() {
			t.Error("expected failure for non-existent field")
		}
		if field.Name != "" {
			t.Error("expected empty field metadata on failure")
		}
	})
}

func TestAssertRelationshipExists(t *testing.T) {
	ResetCache(t)
	meta := sentinel.Inspect[HelperTestStruct]()

	t.Run("returns relationship when it exists", func(t *testing.T) {
		rel := AssertRelationshipExists(t, meta, "Ref")
		if rel.Field != "Ref" {
			t.Errorf("expected relationship field Ref, got %s", rel.Field)
		}
	})

	t.Run("fails when relationship does not exist", func(t *testing.T) {
		mock := &mockT{}
		rel := AssertRelationshipExists(mock, meta, "NonExistent")
		if !mock.failed() {
			t.Error("expected failure for non-existent relationship")
		}
		if rel.Field != "" {
			t.Error("expected empty relationship on failure")
		}
	})
}

func TestAssertTagValue(t *testing.T) {
	ResetCache(t)
	meta := sentinel.Inspect[HelperTestStruct]()
	field := AssertFieldExists(t, meta, "ID")

	t.Run("passes when tag matches", func(t *testing.T) {
		AssertTagValue(t, field, "json", "id")
	})

	t.Run("fails when tag does not exist", func(t *testing.T) {
		mock := &mockT{}
		AssertTagValue(mock, field, "nonexistent", "value")
		if !mock.failed() {
			t.Error("expected failure for non-existent tag")
		}
	})

	t.Run("fails when tag value does not match", func(t *testing.T) {
		mock := &mockT{}
		AssertTagValue(mock, field, "json", "wrong_value")
		if !mock.failed() {
			t.Error("expected failure for mismatched tag value")
		}
	})
}

func TestAssertCached(t *testing.T) {
	ResetCache(t)
	inspected := sentinel.Inspect[HelperTestStruct]()

	t.Run("returns metadata when type is cached", func(t *testing.T) {
		meta := AssertCached(t, inspected.FQDN)
		if meta.TypeName != "HelperTestStruct" {
			t.Errorf("expected TypeName HelperTestStruct, got %s", meta.TypeName)
		}
	})

	t.Run("fails when type is not cached", func(t *testing.T) {
		mock := &mockT{}
		meta := AssertCached(mock, "NonExistentType")
		if !mock.failed() {
			t.Error("expected failure for non-cached type")
		}
		if meta.TypeName != "" {
			t.Error("expected empty metadata on failure")
		}
	})
}

func TestAssertNotCached(t *testing.T) {
	ResetCache(t)

	t.Run("passes when type is not cached", func(t *testing.T) {
		AssertNotCached(t, "NonExistentType")
	})

	t.Run("fails when type is cached", func(t *testing.T) {
		inspected := sentinel.Inspect[HelperTestStruct]()
		mock := &mockT{}
		AssertNotCached(mock, inspected.FQDN)
		if !mock.failed() {
			t.Error("expected failure for cached type")
		}
	})
}

func TestResetCache(t *testing.T) {
	sentinel.Inspect[HelperTestStruct]()

	t.Run("clears the cache", func(t *testing.T) {
		ResetCache(t)
		types := sentinel.Browse()
		if len(types) != 0 {
			t.Errorf("expected empty cache after reset, got %d types", len(types))
		}
	})
}
