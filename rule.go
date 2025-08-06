package sentinel

import (
	"path/filepath"
	"strings"
)

// Rule represents a conditional policy rule with rich matching capabilities.
type Rule struct {
	When    *When             `yaml:"when,omitempty" json:"when,omitempty"`
	Require map[string]string `yaml:"require,omitempty" json:"require,omitempty"`
	Forbid  []string          `yaml:"forbid,omitempty" json:"forbid,omitempty"`
}

// When represents a condition that can be evaluated against metadata.
type When struct {
	// Simple field matchers
	FieldName *StringMatcher `yaml:"field.name,omitempty" json:"field.name,omitempty"`
	FieldType *StringMatcher `yaml:"field.type,omitempty" json:"field.type,omitempty"`

	// Type matchers
	TypeName *StringMatcher `yaml:"type.name,omitempty" json:"type.name,omitempty"`

	// Logical operators
	Not *When `yaml:"not,omitempty" json:"not,omitempty"`

	// Tag checks
	HasTag []string `yaml:"has_tag,omitempty" json:"has_tag,omitempty"`

	// Logical operators (slices at end for better alignment)
	All []When `yaml:"all,omitempty" json:"all,omitempty"`
	Any []When `yaml:"any,omitempty" json:"any,omitempty"`
}

// StringMatcher provides flexible string matching options.
type StringMatcher struct {
	Exact    string   `yaml:"exact,omitempty" json:"exact,omitempty"`
	Pattern  string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Contains string   `yaml:"contains,omitempty" json:"contains,omitempty"`
	OneOf    []string `yaml:"one_of,omitempty" json:"one_of,omitempty"`
}

// EvaluationContext provides data for rule evaluation.
type EvaluationContext struct {
	Field *FieldMetadata
	Type  *ModelMetadata
}

// Evaluate checks if the When condition matches the given context.
func (w *When) Evaluate(ctx *EvaluationContext) bool {
	if w == nil {
		return true // No condition means always match
	}

	// Handle logical operators first
	if len(w.All) > 0 {
		for _, condition := range w.All {
			if !condition.Evaluate(ctx) {
				return false
			}
		}
		return true
	}

	if len(w.Any) > 0 {
		for _, condition := range w.Any {
			if condition.Evaluate(ctx) {
				return true
			}
		}
		return false
	}

	if w.Not != nil {
		return !w.Not.Evaluate(ctx)
	}

	// Field-level checks (only if we have a field context)
	if ctx.Field != nil {
		if w.FieldName != nil && !w.FieldName.Matches(ctx.Field.Name) {
			return false
		}

		if w.FieldType != nil && !w.FieldType.Matches(ctx.Field.Type) {
			return false
		}

		// Tag checks
		for _, tag := range w.HasTag {
			if _, exists := ctx.Field.Tags[tag]; !exists {
				return false
			}
		}
	}

	// Type-level checks
	if ctx.Type != nil {
		if w.TypeName != nil && !w.TypeName.Matches(ctx.Type.TypeName) {
			return false
		}
	}

	return true // All conditions passed
}

// Matches checks if the string matcher matches the given value.
func (m *StringMatcher) Matches(value string) bool {
	if m == nil {
		return true
	}

	if m.Exact != "" {
		return value == m.Exact
	}

	if m.Pattern != "" {
		// Use filepath.Match for glob patterns
		matched, err := filepath.Match(m.Pattern, value)
		if err != nil {
			return false
		}
		return matched
	}

	if m.Contains != "" {
		return strings.Contains(strings.ToLower(value), strings.ToLower(m.Contains))
	}

	if len(m.OneOf) > 0 {
		for _, option := range m.OneOf {
			if value == option {
				return true
			}
		}
		return false
	}

	return true // No conditions means match
}

// UnmarshalYAML provides custom YAML unmarshaling to support simple string syntax.
func (m *StringMatcher) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try simple string first
	var str string
	if err := unmarshal(&str); err == nil {
		// Infer matcher type from string
		if strings.Contains(str, "*") {
			m.Pattern = str
		} else {
			m.Exact = str
		}
		return nil
	}

	// Fall back to full struct unmarshaling
	type rawMatcher StringMatcher
	return unmarshal((*rawMatcher)(m))
}
