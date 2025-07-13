package catalog

import (
	"fmt"
	"strings"
)

// ValidateField runs all validation rules for a field value
// Rules can be comma-separated (e.g., "required,email,min=5")
// Returns nil if all validations pass, or an error describing all failures
func ValidateField(fieldValue any, validationRules []string) error {
	if len(validationRules) == 0 {
		return nil
	}

	var errors []string

	for _, rule := range validationRules {
		// Handle rules with parameters (e.g., "min=5", "max=100")
		ruleName, _, _ := strings.Cut(rule, "=")
		
		// Look up validator
		validator, exists := GetFieldValidator(ruleName)
		if !exists {
			// Skip unknown validators - this allows forward compatibility
			continue
		}

		// Run validator
		if err := validator(fieldValue); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", ruleName, err))
		}
	}

	// Return combined error if any validations failed
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ValidateFieldWithInfo validates a field using ValidationInfo metadata
// This is a convenience function for working with catalog metadata
func ValidateFieldWithInfo(fieldValue any, info ValidationInfo) error {
	// Build rules list from ValidationInfo
	var rules []string
	
	// Add required rule if needed
	if info.Required {
		rules = append(rules, "required")
	}
	
	// Add custom rules
	rules = append(rules, info.CustomRules...)
	
	// Add constraints as rules (e.g., "min=5" becomes a rule)
	for key, value := range info.Constraints {
		rules = append(rules, fmt.Sprintf("%s=%s", key, value))
	}
	
	return ValidateField(fieldValue, rules)
}