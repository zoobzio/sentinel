package sentinel

import (
	"strings"
	"testing"
)

func TestStringMatcher(t *testing.T) {
	tests := []struct {
		name     string
		matcher  StringMatcher
		value    string
		expected bool
	}{
		// Exact matching
		{"exact match", StringMatcher{Exact: "User"}, "User", true},
		{"exact no match", StringMatcher{Exact: "User"}, "Admin", false},

		// Pattern matching
		{"pattern suffix", StringMatcher{Pattern: "*Request"}, "CreateUserRequest", true},
		{"pattern prefix", StringMatcher{Pattern: "User*"}, "UserProfile", true},
		{"pattern no match", StringMatcher{Pattern: "*Request"}, "Response", false},

		// Contains matching
		{"contains match", StringMatcher{Contains: "email"}, "UserEmail", true},
		{"contains case insensitive", StringMatcher{Contains: "EMAIL"}, "useremail", true},
		{"contains no match", StringMatcher{Contains: "phone"}, "email", false},

		// OneOf matching
		{"one of match", StringMatcher{OneOf: []string{"admin", "user", "guest"}}, "admin", true},
		{"one of no match", StringMatcher{OneOf: []string{"admin", "user"}}, "guest", false},

		// Empty matcher (matches everything)
		{"empty matcher", StringMatcher{}, "anything", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.matcher.Matches(tt.value); got != tt.expected {
				t.Errorf("Matches() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWhenEvaluation(t *testing.T) {
	// Test metadata
	field := &FieldMetadata{
		Name: "UserEmail",
		Type: "string",
		Tags: map[string]string{
			"json":     "user_email",
			"validate": "required,email",
		},
	}

	typeMetadata := &ModelMetadata{
		TypeName: "UserRequest",
		Fields:   []FieldMetadata{*field},
	}

	ctx := &EvaluationContext{
		Field: field,
		Type:  typeMetadata,
	}

	tests := []struct {
		name     string
		when     When
		expected bool
	}{
		// Simple field name matching
		{
			"field name exact",
			When{FieldName: &StringMatcher{Exact: "UserEmail"}},
			true,
		},
		{
			"field name pattern",
			When{FieldName: &StringMatcher{Pattern: "*Email"}},
			true,
		},
		{
			"field name contains",
			When{FieldName: &StringMatcher{Contains: "email"}},
			true,
		},

		// Type matching
		{
			"field type match",
			When{FieldType: &StringMatcher{Exact: "string"}},
			true,
		},
		{
			"field type no match",
			When{FieldType: &StringMatcher{Exact: "int"}},
			false,
		},

		// Tag checks
		{
			"has tag",
			When{HasTag: []string{"json"}},
			true,
		},
		{
			"missing tag",
			When{HasTag: []string{"encrypt"}},
			false,
		},
		{
			"multiple tags",
			When{HasTag: []string{"json", "validate"}},
			true,
		},

		// Type name matching
		{
			"type name pattern",
			When{TypeName: &StringMatcher{Pattern: "*Request"}},
			true,
		},

		// Logical operators - ALL
		{
			"all conditions true",
			When{
				All: []When{
					{FieldName: &StringMatcher{Contains: "Email"}},
					{FieldType: &StringMatcher{Exact: "string"}},
				},
			},
			true,
		},
		{
			"all conditions one false",
			When{
				All: []When{
					{FieldName: &StringMatcher{Contains: "Email"}},
					{FieldType: &StringMatcher{Exact: "int"}},
				},
			},
			false,
		},

		// Logical operators - ANY
		{
			"any conditions true",
			When{
				Any: []When{
					{FieldName: &StringMatcher{Contains: "Phone"}},
					{FieldName: &StringMatcher{Contains: "Email"}},
				},
			},
			true,
		},
		{
			"any conditions all false",
			When{
				Any: []When{
					{FieldName: &StringMatcher{Contains: "Phone"}},
					{FieldType: &StringMatcher{Exact: "int"}},
				},
			},
			false,
		},

		// Logical operators - NOT
		{
			"not true condition",
			When{
				Not: &When{FieldType: &StringMatcher{Exact: "int"}},
			},
			true,
		},
		{
			"not false condition",
			When{
				Not: &When{FieldType: &StringMatcher{Exact: "string"}},
			},
			false,
		},

		// Complex nested conditions
		{
			"complex nested",
			When{
				All: []When{
					{FieldName: &StringMatcher{Contains: "Email"}},
					{
						Any: []When{
							{HasTag: []string{"json"}},
							{HasTag: []string{"xml"}},
						},
					},
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.when.Evaluate(ctx); got != tt.expected {
				t.Errorf("Evaluate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRuleApplication(t *testing.T) {
	// Create sentinel with rule-based policies
	policy := Policy{
		Name: "test-rules",
		Policies: []TypePolicy{
			{
				Match: "*Request",
				Rules: []Rule{
					// Simple rule - apply encryption to password fields
					{
						When: &When{
							FieldName: &StringMatcher{Contains: "password"},
						},
						Apply: map[string]string{
							"encrypt": "bcrypt",
							"no_log":  "true",
						},
					},
					// Complex rule - require validation on string fields
					{
						When: &When{
							All: []When{
								{FieldType: &StringMatcher{Exact: "string"}},
								{
									Not: &When{
										FieldName: &StringMatcher{Contains: "password"},
									},
								},
							},
						},
						Require: map[string]string{
							"validate": "{any}",
						},
					},
					// Tag-based rule
					{
						When: &When{
							HasTag: []string{"json"},
						},
						Apply: map[string]string{
							"api_field": "{snake}",
						},
					},
				},
			},
		},
	}

	s := New().WithPolicy(policy).Build()

	// Test type
	type LoginRequest struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}

	metadata := Inspect[LoginRequest](s)

	// Verify password field got encryption tags
	var passwordField *FieldMetadata
	var usernameField *FieldMetadata
	for i, field := range metadata.Fields {
		if field.Name == "Password" {
			passwordField = &metadata.Fields[i]
		}
		if field.Name == "Username" {
			usernameField = &metadata.Fields[i]
		}
	}

	if passwordField == nil {
		t.Fatal("Password field not found")
	}

	// Check password encryption
	if passwordField.Tags["encrypt"] != "bcrypt" {
		t.Errorf("expected Password.encrypt to be 'bcrypt', got %q", passwordField.Tags["encrypt"])
	}
	if passwordField.Tags["no_log"] != "true" {
		t.Errorf("expected Password.no_log to be 'true', got %q", passwordField.Tags["no_log"])
	}

	// Check api_field was applied to fields with json tag
	if passwordField.Tags["api_field"] != "password" {
		t.Errorf("expected Password.api_field to be 'password', got %q", passwordField.Tags["api_field"])
	}
	if usernameField.Tags["api_field"] != "username" {
		t.Errorf("expected Username.api_field to be 'username', got %q", usernameField.Tags["api_field"])
	}
}

func TestRuleValidation(t *testing.T) {
	// Test that rules can enforce requirements
	policy := Policy{
		Name: "strict-rules",
		Policies: []TypePolicy{
			{
				Match: "*",
				Rules: []Rule{
					{
						When: &When{
							FieldName: &StringMatcher{Contains: "email"},
							FieldType: &StringMatcher{Exact: "string"},
						},
						Require: map[string]string{
							"validate": "email",
						},
					},
				},
			},
		},
	}

	s := New().WithPolicy(policy).WithStrictMode().Build()

	type BadUser struct {
		Email string `json:"email"` // Missing required validate tag
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing required tag")
		} else if errStr, ok := r.(string); !ok || !strings.Contains(errStr, "missing required tag 'validate'") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()

	Inspect[BadUser](s)
}

func TestYAMLUnmarshaling(t *testing.T) {
	yamlContent := `
name: security-policy
policies:
  - match: "*"
    rules:
      # Simple string becomes exact match
      - when:
          field.name: "Password"
        apply:
          no_log: "true"
      
      # Pattern with wildcard
      - when:
          field.name: "*Email*"
          field.type: "string"
        apply:
          validate: "email"
      
      # Complex matcher
      - when:
          field.name:
            contains: "token"
        apply:
          sensitive: "true"
      
      # Logical operators
      - when:
          any:
            - field.name: "APIKey"
            - field.name: "SecretKey"
          not:
            has_tag: ["public"]
        apply:
          encrypt: "aes256"
`

	policy, err := LoadPolicy(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("failed to load policy: %v", err)
	}

	if policy.Name != "security-policy" {
		t.Errorf("expected policy name 'security-policy', got %q", policy.Name)
	}

	if len(policy.Policies) != 1 {
		t.Fatalf("expected 1 type policy, got %d", len(policy.Policies))
	}

	rules := policy.Policies[0].Rules
	if len(rules) != 4 {
		t.Fatalf("expected 4 rules, got %d", len(rules))
	}

	// Check first rule (simple string)
	if rules[0].When.FieldName.Exact != "Password" {
		t.Errorf("expected exact match 'Password', got %+v", rules[0].When.FieldName)
	}

	// Check second rule (pattern)
	if rules[1].When.FieldName.Pattern != "*Email*" {
		t.Errorf("expected pattern '*Email*', got %+v", rules[1].When.FieldName)
	}

	// Check third rule (complex matcher)
	if rules[2].When.FieldName.Contains != "token" {
		t.Errorf("expected contains 'token', got %+v", rules[2].When.FieldName)
	}

	// Check fourth rule (logical operators)
	if len(rules[3].When.Any) != 2 {
		t.Errorf("expected 2 conditions in any, got %d", len(rules[3].When.Any))
	}
	if rules[3].When.Not == nil {
		t.Error("expected not condition to be present")
	}
}
