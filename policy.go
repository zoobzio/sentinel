package sentinel

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Policy represents a collection of type-level policies that can be applied
// during metadata extraction.
type Policy struct {
	Name     string       `yaml:"name" json:"name"`
	Policies []TypePolicy `yaml:"policies" json:"policies"`
}

// TypePolicy defines requirements and field policies for types matching a pattern.
type TypePolicy struct {
	Match  string            `yaml:"match" json:"match"`   // Type name pattern (glob)
	Ensure map[string]string `yaml:"ensure" json:"ensure"` // Required fields: name->type
	Fields []FieldPolicy     `yaml:"fields" json:"fields"` // Field-level policies (legacy)
	Rules  []Rule            `yaml:"rules" json:"rules"`   // Rule-based policies (new)
	Codecs []string          `yaml:"codecs" json:"codecs"` // Supported codecs for this type
}

// FieldPolicy defines requirements for fields matching a pattern within a type.
type FieldPolicy struct {
	Require map[string]string `yaml:"require,omitempty" json:"require,omitempty"` // Tags that MUST exist
	Apply   map[string]string `yaml:"apply,omitempty" json:"apply,omitempty"`     // Tags to ALWAYS add/override
	Match   string            `yaml:"match" json:"match"`                         // Field name pattern (glob)
	Type    string            `yaml:"type,omitempty" json:"type,omitempty"`       // Required type
}

// PolicyResult contains the outcome of applying policies to metadata.
type PolicyResult struct {
	Applied  []string // Names of policies that were applied
	Warnings []string // Non-fatal issues found
	Errors   []string // Fatal issues that prevent extraction
}

// matches checks if a name matches a glob pattern.
func matches(pattern, name string) bool {
	// Simple glob matching - can be enhanced later
	if pattern == name {
		return true
	}

	// Handle * prefix
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(name, suffix)
	}

	// Handle * suffix
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
func (s *Sentinel) applyPolicies(ctx *ExtractionContext) PolicyResult {
	result := PolicyResult{
		Applied:  []string{},
		Warnings: []string{},
		Errors:   []string{},
	}

	for _, policy := range s.policies {
		// Apply each type policy
		for _, typePolicy := range policy.Policies {
			if matches(typePolicy.Match, ctx.Metadata.TypeName) {
				// This type matches - apply the policy
				s.applyTypePolicy(ctx, &typePolicy, &result)
				result.Applied = append(result.Applied, fmt.Sprintf("%s.%s", policy.Name, typePolicy.Match))
			}
		}
	}

	return result
}

// applyTypePolicy applies a single type policy to the extraction context.
func (s *Sentinel) applyTypePolicy(ctx *ExtractionContext, policy *TypePolicy, result *PolicyResult) {
	// Apply codecs if specified
	if len(policy.Codecs) > 0 {
		for _, codec := range policy.Codecs {
			if IsValidCodec(codec) {
				ctx.Metadata.Codecs = append(ctx.Metadata.Codecs, codec)
			} else {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Invalid codec '%s' for type %s", codec, ctx.Metadata.TypeName))
			}
		}
	}

	// Check required fields
	for fieldName, fieldType := range policy.Ensure {
		found := false
		for _, field := range ctx.Metadata.Fields {
			if field.Name == fieldName {
				found = true
				if field.Type != fieldType {
					result.Errors = append(result.Errors,
						fmt.Sprintf("Type %s: required field %s must be type %s, got %s",
							ctx.Metadata.TypeName, fieldName, fieldType, field.Type))
				}
				break
			}
		}
		if !found {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Type %s: missing required field %s (%s)",
					ctx.Metadata.TypeName, fieldName, fieldType))
		}
	}

	// Apply field policies (legacy)
	for _, fieldPolicy := range policy.Fields {
		s.applyFieldPolicies(ctx, &fieldPolicy, result)
	}

	// Apply rule-based policies (new)
	if len(policy.Rules) > 0 {
		s.applyRules(ctx, policy.Rules, result)
	}
}

// applyFieldPolicies applies field-level policies to matching fields.
func (s *Sentinel) applyFieldPolicies(ctx *ExtractionContext, policy *FieldPolicy, result *PolicyResult) {
	for i, field := range ctx.Metadata.Fields {
		if !matches(policy.Match, field.Name) {
			continue
		}

		// Check type requirement
		if policy.Type != "" && field.Type != policy.Type {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Field %s.%s: must be type %s, got %s",
					ctx.Metadata.TypeName, field.Name, policy.Type, field.Type))
			continue
		}

		// Check required tags
		for tag, value := range policy.Require {
			existing, exists := field.Tags[tag]
			if !exists {
				result.Errors = append(result.Errors,
					fmt.Sprintf("Field %s.%s: missing required tag '%s'",
						ctx.Metadata.TypeName, field.Name, tag))
			} else if value != "{any}" && existing != value {
				result.Errors = append(result.Errors,
					fmt.Sprintf("Field %s.%s: tag '%s' must be '%s', got '%s'",
						ctx.Metadata.TypeName, field.Name, tag, value, existing))
			}
		}

		// Apply override tags
		for tag, value := range policy.Apply {
			// Special processing for template values
			processedValue := s.processTagValue(value, field.Name)

			// Initialize tags map if needed
			if ctx.Metadata.Fields[i].Tags == nil {
				ctx.Metadata.Fields[i].Tags = make(map[string]string)
			}

			// Apply the tag
			ctx.Metadata.Fields[i].Tags[tag] = processedValue
		}
	}
}

// processTagValue handles special template values in tags.
func (*Sentinel) processTagValue(value, fieldName string) string {
	switch value {
	case "{snake}":
		// Convert PascalCase/camelCase to snake_case
		return toSnakeCase(fieldName)
	case "{lower}":
		return strings.ToLower(fieldName)
	case "{upper}":
		return strings.ToUpper(fieldName)
	default:
		return value
	}
}

// toSnakeCase converts a string to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// applyRules applies rule-based policies to the extraction context.
func (s *Sentinel) applyRules(ctx *ExtractionContext, rules []Rule, result *PolicyResult) {
	evalCtx := &EvaluationContext{
		Type: &ctx.Metadata,
	}

	// Apply rules to each field
	for i, field := range ctx.Metadata.Fields {
		evalCtx.Field = &field

		for _, rule := range rules {
			if rule.When == nil || rule.When.Evaluate(evalCtx) {
				// Apply tags
				if rule.Apply != nil {
					if ctx.Metadata.Fields[i].Tags == nil {
						ctx.Metadata.Fields[i].Tags = make(map[string]string)
					}
					for k, v := range rule.Apply {
						ctx.Metadata.Fields[i].Tags[k] = s.processTagValue(v, field.Name)
					}
				}

				// Check requirements
				if rule.Require != nil {
					for tag, expected := range rule.Require {
						actual, exists := field.Tags[tag]
						if !exists {
							result.Errors = append(result.Errors,
								fmt.Sprintf("Field %s.%s: missing required tag '%s'",
									ctx.Metadata.TypeName, field.Name, tag))
						} else if expected != "{any}" && actual != expected {
							result.Errors = append(result.Errors,
								fmt.Sprintf("Field %s.%s: tag '%s' must be '%s', got '%s'",
									ctx.Metadata.TypeName, field.Name, tag, expected, actual))
						}
					}
				}

				// Check forbidden tags
				for _, tag := range rule.Forbid {
					if _, exists := field.Tags[tag]; exists {
						result.Errors = append(result.Errors,
							fmt.Sprintf("Field %s.%s: forbidden tag '%s'",
								ctx.Metadata.TypeName, field.Name, tag))
					}
				}
			}
		}
	}

}
