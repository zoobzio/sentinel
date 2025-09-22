package sentinel

import (
	"context"
	"strings"
	"testing"
)

// Test types for ERD generation.
type ERDTestUser struct {
	ID      string          `json:"id"`
	Profile *ERDTestProfile `json:"profile"`
	Orders  []ERDTestOrder  `json:"orders"`
}

type ERDTestProfile struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

type ERDTestOrder struct {
	ID     string  `json:"id"`
	UserID string  `json:"user_id"`
	Total  float64 `json:"total"`
}

func TestGenerateERD(t *testing.T) {
	// Clear cache and populate with test data
	instance.cache.Clear()

	// Inspect types to populate cache
	Inspect[ERDTestUser](context.Background())
	Inspect[ERDTestProfile](context.Background())
	Inspect[ERDTestOrder](context.Background())

	t.Run("mermaid format", func(t *testing.T) {
		erd := GenerateERD(ERDFormatMermaid)

		// Should contain mermaid header
		if !strings.Contains(erd, "erDiagram") {
			t.Error("Mermaid ERD should contain 'erDiagram' header")
		}

		// Should contain our test types
		if !strings.Contains(erd, "ERDTestUser") {
			t.Error("ERD should contain ERDTestUser")
		}
		if !strings.Contains(erd, "ERDTestProfile") {
			t.Error("ERD should contain ERDTestProfile")
		}
		if !strings.Contains(erd, "ERDTestOrder") {
			t.Error("ERD should contain ERDTestOrder")
		}
	})

	t.Run("dot format", func(t *testing.T) {
		erd := GenerateERD(ERDFormatDOT)

		// Should contain DOT header
		if !strings.Contains(erd, "digraph") {
			t.Error("DOT ERD should contain 'digraph' header")
		}

		// Should contain our test types
		if !strings.Contains(erd, "ERDTestUser") {
			t.Error("ERD should contain ERDTestUser")
		}
	})

	t.Run("default format", func(t *testing.T) {
		erd := GenerateERD("invalid")

		// Should default to mermaid
		if !strings.Contains(erd, "erDiagram") {
			t.Error("Invalid format should default to Mermaid")
		}
	})

	t.Run("empty cache", func(t *testing.T) {
		instance.cache.Clear()

		erd := GenerateERD(ERDFormatMermaid)

		// Should still generate valid but empty ERD
		if !strings.Contains(erd, "erDiagram") {
			t.Error("Empty cache should still generate valid ERD header")
		}
	})
}

func TestGenerateERDFromRoot(t *testing.T) {
	// Clear cache first
	instance.cache.Clear()

	// Inspect all types to populate cache
	Inspect[ERDTestUser](context.Background())
	Inspect[ERDTestProfile](context.Background())
	Inspect[ERDTestOrder](context.Background())

	t.Run("from user root", func(t *testing.T) {
		erd := GenerateERDFromRoot[ERDTestUser](ERDFormatMermaid)

		// Should contain mermaid header
		if !strings.Contains(erd, "erDiagram") {
			t.Error("Root ERD should contain 'erDiagram' header")
		}

		// Should contain User (root type)
		if !strings.Contains(erd, "ERDTestUser") {
			t.Error("Root ERD should contain ERDTestUser")
		}

		// Should contain related types (Profile, Order)
		if !strings.Contains(erd, "ERDTestProfile") {
			t.Error("Root ERD should contain related ERDTestProfile")
		}
		if !strings.Contains(erd, "ERDTestOrder") {
			t.Error("Root ERD should contain related ERDTestOrder")
		}
	})

	t.Run("from profile root", func(t *testing.T) {
		erd := GenerateERDFromRoot[ERDTestProfile](ERDFormatMermaid)

		// Should contain Profile
		if !strings.Contains(erd, "ERDTestProfile") {
			t.Error("Profile root ERD should contain ERDTestProfile")
		}

		// Profile has no outgoing relationships, so should be minimal
		lines := strings.Split(erd, "\n")
		nonEmptyLines := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines++
			}
		}

		// Should be a small ERD (header + profile entity)
		if nonEmptyLines > 10 {
			t.Errorf("Profile root ERD should be minimal, got %d lines", nonEmptyLines)
		}
	})

	t.Run("dot format from root", func(t *testing.T) {
		erd := GenerateERDFromRoot[ERDTestUser](ERDFormatDOT)

		// Should contain DOT header
		if !strings.Contains(erd, "digraph") {
			t.Error("Root DOT ERD should contain 'digraph' header")
		}

		// Should contain root type
		if !strings.Contains(erd, "ERDTestUser") {
			t.Error("Root DOT ERD should contain ERDTestUser")
		}
	})
}

func TestERDFormat(t *testing.T) {
	t.Run("format constants", func(t *testing.T) {
		if ERDFormatMermaid != "mermaid" {
			t.Errorf("ERDFormatMermaid should be 'mermaid', got %s", ERDFormatMermaid)
		}
		if ERDFormatDOT != "dot" {
			t.Errorf("ERDFormatDOT should be 'dot', got %s", ERDFormatDOT)
		}
	})
}
