package sentinel

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRuleStructure(t *testing.T) {
	t.Run("Rule fields", func(t *testing.T) {
		rule := Rule{
			When: &When{
				FieldName: &StringMatcher{Exact: "Password"},
			},
			Require: map[string]string{
				"validate": "required",
			},
			Forbid: []string{"log", "export"},
		}

		if rule.When == nil {
			t.Error("expected When condition")
		}
		if len(rule.Require) != 1 {
			t.Errorf("expected 1 require rule, got %d", len(rule.Require))
		}
		if len(rule.Forbid) != 2 {
			t.Errorf("expected 2 forbid rules, got %d", len(rule.Forbid))
		}
	})

	t.Run("When conditions", func(t *testing.T) {
		when := When{
			FieldName: &StringMatcher{Exact: "Email"},
			FieldType: &StringMatcher{Pattern: "*string"},
			TypeName:  &StringMatcher{Contains: "User"},
			HasTag:    []string{"json", "validate"},
			Not: &When{
				FieldName: &StringMatcher{Exact: "Internal"},
			},
			All: []When{
				{FieldName: &StringMatcher{Pattern: "*ID"}},
			},
			Any: []When{
				{HasTag: []string{"encrypt"}},
			},
		}

		if when.FieldName.Exact != "Email" {
			t.Errorf("expected FieldName.Exact 'Email', got %s", when.FieldName.Exact)
		}
		if len(when.HasTag) != 2 {
			t.Errorf("expected 2 tags, got %d", len(when.HasTag))
		}
	})
}

func TestStringMatcher(t *testing.T) {
	tests := []struct {
		name    string
		matcher StringMatcher
		value   string
		want    bool
	}{
		// Exact matching
		{
			name:    "exact match",
			matcher: StringMatcher{Exact: "UserID"},
			value:   "UserID",
			want:    true,
		},
		{
			name:    "exact no match",
			matcher: StringMatcher{Exact: "UserID"},
			value:   "UserId",
			want:    false,
		},
		// Pattern matching
		{
			name:    "pattern suffix",
			matcher: StringMatcher{Pattern: "*Request"},
			value:   "UserRequest",
			want:    true,
		},
		{
			name:    "pattern prefix",
			matcher: StringMatcher{Pattern: "User*"},
			value:   "UserModel",
			want:    true,
		},
		{
			name:    "pattern no match",
			matcher: StringMatcher{Pattern: "*Request"},
			value:   "Response",
			want:    false,
		},
		// Contains matching
		{
			name:    "contains match",
			matcher: StringMatcher{Contains: "User"},
			value:   "ModelUserRequest",
			want:    true,
		},
		{
			name:    "contains case insensitive",
			matcher: StringMatcher{Contains: "user"},
			value:   "ModelUserRequest",
			want:    true,
		},
		{
			name:    "contains no match",
			matcher: StringMatcher{Contains: "Admin"},
			value:   "UserModel",
			want:    false,
		},
		// OneOf matching
		{
			name:    "one_of match",
			matcher: StringMatcher{OneOf: []string{"GET", "POST", "PUT"}},
			value:   "POST",
			want:    true,
		},
		{
			name:    "one_of no match",
			matcher: StringMatcher{OneOf: []string{"GET", "POST", "PUT"}},
			value:   "DELETE",
			want:    false,
		},
		// Empty matcher
		{
			name:    "empty matcher",
			matcher: StringMatcher{},
			value:   "anything",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.matcher.Matches(tt.value)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("nil matcher", func(t *testing.T) {
		var m *StringMatcher
		if !m.Matches("anything") {
			t.Error("nil matcher should match anything")
		}
	})
}

func TestWhenEvaluate(t *testing.T) {
	tests := []struct {
		name string
		when When
		ctx  EvaluationContext
		want bool
	}{
		// Field matchers
		{
			name: "field name exact",
			when: When{FieldName: &StringMatcher{Exact: "Password"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Name: "Password"},
			},
			want: true,
		},
		{
			name: "field name pattern",
			when: When{FieldName: &StringMatcher{Pattern: "*ID"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Name: "UserID"},
			},
			want: true,
		},
		{
			name: "field name contains",
			when: When{FieldName: &StringMatcher{Contains: "mail"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Name: "EmailAddress"},
			},
			want: true,
		},
		{
			name: "field type match",
			when: When{FieldType: &StringMatcher{Exact: "string"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Type: "string"},
			},
			want: true,
		},
		{
			name: "field type no match",
			when: When{FieldType: &StringMatcher{Exact: "string"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Type: "int"},
			},
			want: false,
		},
		// Tag checks
		{
			name: "has tag",
			when: When{HasTag: []string{"json"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{
					Tags: map[string]string{"json": "field"},
				},
			},
			want: true,
		},
		{
			name: "missing tag",
			when: When{HasTag: []string{"validate"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{
					Tags: map[string]string{"json": "field"},
				},
			},
			want: false,
		},
		{
			name: "multiple tags",
			when: When{HasTag: []string{"json", "validate"}},
			ctx: EvaluationContext{
				Field: &FieldMetadata{
					Tags: map[string]string{
						"json":     "field",
						"validate": "required",
					},
				},
			},
			want: true,
		},
		// Type matchers
		{
			name: "type name pattern",
			when: When{TypeName: &StringMatcher{Pattern: "*Request"}},
			ctx: EvaluationContext{
				Type: &ModelMetadata{TypeName: "UserRequest"},
			},
			want: true,
		},
		// Logical operators
		{
			name: "all conditions true",
			when: When{
				All: []When{
					{FieldName: &StringMatcher{Pattern: "*ID"}},
					{FieldType: &StringMatcher{Exact: "string"}},
				},
			},
			ctx: EvaluationContext{
				Field: &FieldMetadata{
					Name: "UserID",
					Type: "string",
				},
			},
			want: true,
		},
		{
			name: "all conditions one false",
			when: When{
				All: []When{
					{FieldName: &StringMatcher{Pattern: "*ID"}},
					{FieldType: &StringMatcher{Exact: "int"}},
				},
			},
			ctx: EvaluationContext{
				Field: &FieldMetadata{
					Name: "UserID",
					Type: "string",
				},
			},
			want: false,
		},
		{
			name: "any conditions true",
			when: When{
				Any: []When{
					{FieldName: &StringMatcher{Exact: "Email"}},
					{FieldName: &StringMatcher{Exact: "EmailAddress"}},
				},
			},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Name: "Email"},
			},
			want: true,
		},
		{
			name: "any conditions all false",
			when: When{
				Any: []When{
					{FieldName: &StringMatcher{Exact: "Email"}},
					{FieldName: &StringMatcher{Exact: "EmailAddress"}},
				},
			},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Name: "Username"},
			},
			want: false,
		},
		{
			name: "not true condition",
			when: When{
				Not: &When{
					FieldName: &StringMatcher{Exact: "Internal"},
				},
			},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Name: "Public"},
			},
			want: true,
		},
		{
			name: "not false condition",
			when: When{
				Not: &When{
					FieldName: &StringMatcher{Exact: "Internal"},
				},
			},
			ctx: EvaluationContext{
				Field: &FieldMetadata{Name: "Internal"},
			},
			want: false,
		},
		// Complex nested conditions
		{
			name: "complex nested",
			when: When{
				All: []When{
					{TypeName: &StringMatcher{Pattern: "*Request"}},
					{
						Any: []When{
							{FieldName: &StringMatcher{Exact: "Password"}},
							{FieldName: &StringMatcher{Exact: "Secret"}},
						},
					},
				},
			},
			ctx: EvaluationContext{
				Type:  &ModelMetadata{TypeName: "LoginRequest"},
				Field: &FieldMetadata{Name: "Password"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.when.Evaluate(&tt.ctx)
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("nil when", func(t *testing.T) {
		var w *When
		ctx := EvaluationContext{}
		if !w.Evaluate(&ctx) {
			t.Error("nil When should always evaluate to true")
		}
	})
}

func TestStringMatcherUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    StringMatcher
		wantErr bool
	}{
		{
			name: "simple string with pattern",
			yaml: `"*Request"`,
			want: StringMatcher{Pattern: "*Request"},
		},
		{
			name: "simple string without pattern",
			yaml: `"UserID"`,
			want: StringMatcher{Exact: "UserID"},
		},
		{
			name: "explicit exact",
			yaml: `exact: "UserID"`,
			want: StringMatcher{Exact: "UserID"},
		},
		{
			name: "explicit pattern",
			yaml: `pattern: "*Request"`,
			want: StringMatcher{Pattern: "*Request"},
		},
		{
			name: "explicit contains",
			yaml: `contains: "User"`,
			want: StringMatcher{Contains: "User"},
		},
		{
			name: "explicit one_of",
			yaml: `one_of: ["GET", "POST", "PUT"]`,
			want: StringMatcher{OneOf: []string{"GET", "POST", "PUT"}},
		},
		{
			name:    "invalid yaml",
			yaml:    `{invalid yaml`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m StringMatcher
			err := yaml.Unmarshal([]byte(tt.yaml), &m)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if m.Exact != tt.want.Exact {
					t.Errorf("Exact = %v, want %v", m.Exact, tt.want.Exact)
				}
				if m.Pattern != tt.want.Pattern {
					t.Errorf("Pattern = %v, want %v", m.Pattern, tt.want.Pattern)
				}
				if m.Contains != tt.want.Contains {
					t.Errorf("Contains = %v, want %v", m.Contains, tt.want.Contains)
				}
				if len(m.OneOf) != len(tt.want.OneOf) {
					t.Errorf("OneOf = %v, want %v", m.OneOf, tt.want.OneOf)
				}
			}
		})
	}
}

func TestRuleApplication(t *testing.T) {
	// This would test how rules are applied in the context of policy processing.
	// For now, we're just testing the rule structures and evaluation.
	t.Run("rule with when condition", func(t *testing.T) {
		rule := Rule{
			When: &When{
				FieldName: &StringMatcher{Pattern: "*Password*"},
			},
		}

		// Simulate checking if rule applies to a field
		ctx := EvaluationContext{
			Field: &FieldMetadata{
				Name: "UserPassword",
				Type: "string",
			},
		}

		if !rule.When.Evaluate(&ctx) {
			t.Error("expected rule to match UserPassword field")
		}
	})
}

func TestRuleValidation(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		rule := Rule{
			When: &When{
				FieldName: &StringMatcher{Exact: "ID"},
			},
			Require: map[string]string{"validate": "required"},
		}

		// Rule should have at least one action
		if len(rule.Require) == 0 && len(rule.Forbid) == 0 {
			t.Error("rule should have at least one action")
		}
	})

	t.Run("empty rule", func(_ *testing.T) {
		rule := Rule{}

		// Empty rule with no actions is not very useful.
		if len(rule.Require) == 0 && len(rule.Forbid) == 0 {
			// This is expected - empty rules are allowed but not useful.
			_ = rule // Using rule to avoid unused variable warning.
		}
	})
}

func TestYAMLUnmarshaling(t *testing.T) {
	yamlContent := `
when:
  field.name:
    pattern: "*Password"
  has_tag:
    - json
  not:
    field.name: "Internal"
require:
  redact: "[HIDDEN]"
  validate: "required"
forbid:
  - log
  - export
`

	var rule Rule
	err := yaml.Unmarshal([]byte(yamlContent), &rule)
	if err != nil {
		t.Fatalf("failed to unmarshal rule: %v", err)
	}

	if rule.When == nil {
		t.Fatal("expected When condition")
	}
	if rule.When.FieldName.Pattern != "*Password" {
		t.Errorf("expected pattern '*Password', got %s", rule.When.FieldName.Pattern)
	}
	if len(rule.When.HasTag) != 1 || rule.When.HasTag[0] != "json" {
		t.Errorf("expected has_tag [json], got %v", rule.When.HasTag)
	}
	if rule.Require["redact"] != "[HIDDEN]" {
		t.Errorf("expected redact '[HIDDEN]', got %s", rule.Require["redact"])
	}
}
