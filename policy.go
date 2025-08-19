package sentinel

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// Convention defines a method pattern that types can implement.
type Convention struct {
	Name       string   `yaml:"name" json:"name"`       // Convention identifier (e.g., "defaults")
	MethodName string   `yaml:"method" json:"method"`   // Method name to look for
	Params     []string `yaml:"params" json:"params"`   // Expected parameter types
	Returns    []string `yaml:"returns" json:"returns"` // Expected return types
}

// Policy represents a collection of type-level policies that can be applied
// during metadata extraction.
type Policy struct {
	Name        string       `yaml:"name" json:"name"`
	Policies    []TypePolicy `yaml:"policies" json:"policies"`
	Conventions []Convention `yaml:"conventions" json:"conventions"` // Method conventions to detect
}

// TypePolicy defines requirements and field policies for types matching a pattern.
type TypePolicy struct {
	Match          string            `yaml:"match" json:"match"`                   // Type name pattern (glob)
	Classification string            `yaml:"classification" json:"classification"` // Security classification level
	Ensure         map[string]string `yaml:"ensure" json:"ensure"`                 // Required fields: name->type
	Fields         []FieldPolicy     `yaml:"fields" json:"fields"`                 // Field-level policies (legacy)
	Rules          []Rule            `yaml:"rules" json:"rules"`                   // Rule-based policies (new)
	Codecs         []string          `yaml:"codecs" json:"codecs"`                 // Supported codecs for this type
}

// FieldPolicy defines requirements for fields matching a pattern within a type.
type FieldPolicy struct {
	Require map[string]string `yaml:"require,omitempty" json:"require,omitempty"` // Tags that MUST exist
	Match   string            `yaml:"match" json:"match"`                         // Field name pattern (glob)
	Type    string            `yaml:"type,omitempty" json:"type,omitempty"`       // Required type
}

// PolicyResult contains the outcome of applying policies to metadata.
type PolicyResult struct {
	PolicyMetrics  map[string]PolicyApplicationMetrics // Per-policy metrics
	Applied        []string                            // Names of policies that were applied
	Warnings       []string                            // Non-fatal issues found
	Errors         []string                            // Fatal issues that prevent extraction
	AffectedFields []string                            // Names of fields that were changed
	FieldsModified int                                 // Number of fields that were modified
	TagsApplied    int                                 // Number of tags that were applied
}

// PolicyApplicationMetrics tracks what a specific policy changed.
type PolicyApplicationMetrics struct {
	AffectedFields []string // Fields affected by this policy
	FieldsModified int      // Fields modified by this policy
	TagsApplied    int      // Tags applied by this policy
}

// matches checks if a name matches a glob pattern.
func matches(pattern, name string) bool {
	// Exact match
	if pattern == name {
		return true
	}

	// Handle single * (matches everything)
	if pattern == "*" {
		return true
	}

	// Handle *substring* (contains)
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && len(pattern) > 1 {
		substring := pattern[1 : len(pattern)-1]
		return strings.Contains(name, substring)
	}

	// Handle *suffix
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(name, suffix)
	}

	// Handle prefix*
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(name, prefix)
	}

	// Use filepath.Match for more complex patterns
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

// applyPolicies applies all configured policies to the extraction context.
func (s *Sentinel) applyPolicies(_ context.Context, ec *ExtractionContext) PolicyResult {
	result := PolicyResult{
		Applied:        []string{},
		Warnings:       []string{},
		Errors:         []string{},
		FieldsModified: 0,
		TagsApplied:    0,
		AffectedFields: []string{},
		PolicyMetrics:  make(map[string]PolicyApplicationMetrics),
	}

	for _, policy := range s.policies {
		// Apply each type policy
		for _, typePolicy := range policy.Policies {
			if matches(typePolicy.Match, ec.Metadata.TypeName) {
				// This type matches - apply the policy
				s.applyTypePolicy(ec, &typePolicy, &result)
				result.Applied = append(result.Applied, fmt.Sprintf("%s.%s", policy.Name, typePolicy.Match))
			}
		}
	}

	return result
}

// applyTypePolicy applies a single type policy to the extraction context.
func (s *Sentinel) applyTypePolicy(ec *ExtractionContext, policy *TypePolicy, result *PolicyResult) {
	// Apply classification if specified
	if policy.Classification != "" {
		ec.Metadata.Classification = policy.Classification
	}

	// Apply codecs if specified
	if len(policy.Codecs) > 0 {
		for _, codec := range policy.Codecs {
			if IsValidCodec(codec) {
				ec.Metadata.Codecs = append(ec.Metadata.Codecs, codec)
			} else {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Invalid codec '%s' for type %s", codec, ec.Metadata.TypeName))
			}
		}
	}

	// Check required fields
	for fieldName, fieldType := range policy.Ensure {
		found := false
		for _, field := range ec.Metadata.Fields {
			if field.Name == fieldName {
				found = true
				if field.Type != fieldType {
					result.Errors = append(result.Errors,
						fmt.Sprintf("Type %s: required field %s must be type %s, got %s",
							ec.Metadata.TypeName, fieldName, fieldType, field.Type))
				}
				break
			}
		}
		if !found {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Type %s: missing required field %s (%s)",
					ec.Metadata.TypeName, fieldName, fieldType))
		}
	}

	// Apply field policies (legacy)
	for _, fieldPolicy := range policy.Fields {
		s.applyFieldPolicies(ec, &fieldPolicy, result)
	}

	// Apply rule-based policies (new)
	if len(policy.Rules) > 0 {
		s.applyRules(ec, policy.Rules, result)
	}
}

// applyFieldPolicies applies field-level policies to matching fields.
func (*Sentinel) applyFieldPolicies(ec *ExtractionContext, policy *FieldPolicy, result *PolicyResult) {
	for _, field := range ec.Metadata.Fields {
		if !matches(policy.Match, field.Name) {
			continue
		}

		// Check type requirement
		if policy.Type != "" && field.Type != policy.Type {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Field %s.%s: must be type %s, got %s",
					ec.Metadata.TypeName, field.Name, policy.Type, field.Type))
			continue
		}

		// Check required tags
		for tag, value := range policy.Require {
			existing, exists := field.Tags[tag]
			if !exists {
				result.Errors = append(result.Errors,
					fmt.Sprintf("Field %s.%s: missing required tag '%s'",
						ec.Metadata.TypeName, field.Name, tag))
			} else if value != "{any}" && existing != value {
				result.Errors = append(result.Errors,
					fmt.Sprintf("Field %s.%s: tag '%s' must be '%s', got '%s'",
						ec.Metadata.TypeName, field.Name, tag, value, existing))
			}
		}

	}
}

// applyRules applies rule-based policies to the extraction context.
func (*Sentinel) applyRules(ec *ExtractionContext, rules []Rule, result *PolicyResult) {
	evalCtx := &EvaluationContext{
		Type: &ec.Metadata,
	}

	// Apply rules to each field
	for _, field := range ec.Metadata.Fields {
		evalCtx.Field = &field

		for _, rule := range rules {
			if rule.When == nil || rule.When.Evaluate(evalCtx) {
				// Check requirements
				if rule.Require != nil {
					for tag, expected := range rule.Require {
						actual, exists := field.Tags[tag]
						if !exists {
							result.Errors = append(result.Errors,
								fmt.Sprintf("Field %s.%s: missing required tag '%s'",
									ec.Metadata.TypeName, field.Name, tag))
						} else if expected != "{any}" && actual != expected {
							result.Errors = append(result.Errors,
								fmt.Sprintf("Field %s.%s: tag '%s' must be '%s', got '%s'",
									ec.Metadata.TypeName, field.Name, tag, expected, actual))
						}
					}
				}

				// Check forbidden tags
				for _, tag := range rule.Forbid {
					if _, exists := field.Tags[tag]; exists {
						result.Errors = append(result.Errors,
							fmt.Sprintf("Field %s.%s: forbidden tag '%s'",
								ec.Metadata.TypeName, field.Name, tag))
					}
				}
			}
		}
	}

}
