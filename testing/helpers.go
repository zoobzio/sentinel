// Package testing provides shared test utilities for sentinel tests.
package testing

import (
	"testing"

	"github.com/zoobzio/sentinel"
)

// AssertMetadataValid verifies that metadata has required fields populated.
func AssertMetadataValid(t testing.TB, meta sentinel.Metadata) {
	t.Helper()
	if meta.TypeName == "" {
		t.Error("expected TypeName to be non-empty")
	}
	if meta.PackageName == "" {
		t.Error("expected PackageName to be non-empty")
	}
	if meta.ReflectType == nil {
		t.Error("expected ReflectType to be non-nil")
	}
}

// AssertFieldExists verifies that a field with the given name exists in metadata.
func AssertFieldExists(t testing.TB, meta sentinel.Metadata, fieldName string) sentinel.FieldMetadata {
	t.Helper()
	for _, f := range meta.Fields {
		if f.Name == fieldName {
			return f
		}
	}
	t.Errorf("expected field %q to exist in %s", fieldName, meta.TypeName)
	return sentinel.FieldMetadata{}
}

// AssertRelationshipExists verifies that a relationship exists for the given field.
func AssertRelationshipExists(t testing.TB, meta sentinel.Metadata, fieldName string) sentinel.TypeRelationship {
	t.Helper()
	for _, r := range meta.Relationships {
		if r.Field == fieldName {
			return r
		}
	}
	t.Errorf("expected relationship for field %q in %s", fieldName, meta.TypeName)
	return sentinel.TypeRelationship{}
}

// AssertTagValue verifies that a field has the expected tag value.
func AssertTagValue(t testing.TB, field sentinel.FieldMetadata, tagName, expected string) {
	t.Helper()
	actual, ok := field.Tags[tagName]
	if !ok {
		t.Errorf("expected tag %q on field %s", tagName, field.Name)
		return
	}
	if actual != expected {
		t.Errorf("expected tag %s=%q, got %q", tagName, expected, actual)
	}
}

// AssertCached verifies that a type is present in the sentinel cache.
func AssertCached(t testing.TB, typeName string) sentinel.Metadata {
	t.Helper()
	meta, ok := sentinel.Lookup(typeName)
	if !ok {
		t.Errorf("expected type %q to be cached", typeName)
		return sentinel.Metadata{}
	}
	return meta
}

// AssertNotCached verifies that a type is not present in the sentinel cache.
func AssertNotCached(t testing.TB, typeName string) {
	t.Helper()
	_, ok := sentinel.Lookup(typeName)
	if ok {
		t.Errorf("expected type %q to not be cached", typeName)
	}
}

// ResetCache clears the sentinel cache for test isolation.
func ResetCache(t testing.TB) {
	t.Helper()
	sentinel.Reset()
}
