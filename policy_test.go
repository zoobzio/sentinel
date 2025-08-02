package sentinel

import (
	"strings"
	"testing"
)

func TestPolicyMatching(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		input    string
		expected bool
	}{
		{"exact match", "User", "User", true},
		{"exact no match", "User", "Admin", false},
		{"suffix match", "*Request", "CreateUserRequest", true},
		{"suffix no match", "*Request", "UserResponse", false},
		{"prefix match", "User*", "UserProfile", true},
		{"prefix no match", "User*", "AdminProfile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matches(tt.pattern, tt.input); got != tt.expected {
				t.Errorf("matches(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.expected)
			}
		})
	}
}

func TestPolicyApplication(t *testing.T) {
	// Create a test sentinel with policies
	s := New().
		WithPolicy(Policy{
			Name: "test-policy",
			Policies: []TypePolicy{
				{
					Match: "*Request",
					Ensure: map[string]string{
						"ID": "string",
					},
					Fields: []FieldPolicy{
						{
							Match: "Token",
							Apply: map[string]string{
								"encrypt": "secret",
								"redact":  "[REDACTED]",
							},
						},
					},
				},
			},
		}).
		Build()

	// Test type that should match
	type UserRequest struct {
		ID    string
		Token string
		Name  string
	}

	metadata := Inspect[UserRequest](s)

	// Check that policy was applied
	if metadata.TypeName != "UserRequest" {
		t.Errorf("expected TypeName to be UserRequest, got %s", metadata.TypeName)
	}

	// Find the Token field
	var tokenField *FieldMetadata
	for i, field := range metadata.Fields {
		if field.Name == "Token" {
			tokenField = &metadata.Fields[i]
			break
		}
	}

	if tokenField == nil {
		t.Fatal("Token field not found in metadata")
	}

	// Check that tags were applied
	if tokenField.Tags["encrypt"] != "secret" {
		t.Errorf("expected Token.encrypt to be 'secret', got %q", tokenField.Tags["encrypt"])
	}

	if tokenField.Tags["redact"] != "[REDACTED]" {
		t.Errorf("expected Token.redact to be '[REDACTED]', got %q", tokenField.Tags["redact"])
	}
}

func TestCodecApplication(t *testing.T) {
	// Create a test sentinel with codec policies
	s := New().
		WithPolicy(Policy{
			Name: "codec-policy",
			Policies: []TypePolicy{
				{
					Match:  "*Request",
					Codecs: []string{"json", "msgpack"},
				},
				{
					Match:  "*Response",
					Codecs: []string{"json", "xml"},
				},
			},
		}).
		Build()

	// Test type that should match first pattern
	type UserRequest struct {
		ID   string
		Name string
	}

	// Test type that should match second pattern
	type UserResponse struct {
		ID     string
		Name   string
		Status string
	}

	// Test type that shouldn't match any pattern
	type UserModel struct {
		ID   string
		Name string
	}

	// Check UserRequest gets correct codecs
	reqMetadata := Inspect[UserRequest](s)
	if len(reqMetadata.Codecs) != 2 {
		t.Errorf("expected UserRequest to have 2 codecs, got %d", len(reqMetadata.Codecs))
	}
	expectedCodecs := map[string]bool{"json": true, "msgpack": true}
	for _, codec := range reqMetadata.Codecs {
		if !expectedCodecs[codec] {
			t.Errorf("unexpected codec %q for UserRequest", codec)
		}
		delete(expectedCodecs, codec)
	}
	if len(expectedCodecs) > 0 {
		t.Errorf("missing codecs for UserRequest: %v", expectedCodecs)
	}

	// Check UserResponse gets correct codecs
	respMetadata := Inspect[UserResponse](s)
	if len(respMetadata.Codecs) != 2 {
		t.Errorf("expected UserResponse to have 2 codecs, got %d", len(respMetadata.Codecs))
	}
	expectedCodecs = map[string]bool{"json": true, "xml": true}
	for _, codec := range respMetadata.Codecs {
		if !expectedCodecs[codec] {
			t.Errorf("unexpected codec %q for UserResponse", codec)
		}
		delete(expectedCodecs, codec)
	}
	if len(expectedCodecs) > 0 {
		t.Errorf("missing codecs for UserResponse: %v", expectedCodecs)
	}

	// Check UserModel has no codecs
	modelMetadata := Inspect[UserModel](s)
	if len(modelMetadata.Codecs) != 0 {
		t.Errorf("expected UserModel to have no codecs, got %v", modelMetadata.Codecs)
	}
}

func TestCodecValidation(t *testing.T) {
	// Create a test sentinel with invalid codec in policy
	s := New().
		WithPolicy(Policy{
			Name: "invalid-codec-policy",
			Policies: []TypePolicy{
				{
					Match:  "*Data",
					Codecs: []string{"json", "invalid-codec", "msgpack"},
				},
			},
		}).
		Build()

	// Test type that should match
	type UserData struct {
		ID   string
		Name string
	}

	// Extract metadata - should only have valid codecs
	metadata := Inspect[UserData](s)

	// Should only have the valid codecs
	if len(metadata.Codecs) != 2 {
		t.Errorf("expected UserData to have 2 valid codecs, got %d", len(metadata.Codecs))
	}

	// Check that only valid codecs are present
	expectedCodecs := map[string]bool{"json": true, "msgpack": true}
	for _, codec := range metadata.Codecs {
		if !expectedCodecs[codec] {
			t.Errorf("unexpected codec %q for UserData", codec)
		}
		delete(expectedCodecs, codec)
	}
	if len(expectedCodecs) > 0 {
		t.Errorf("missing codecs for UserData: %v", expectedCodecs)
	}

	// Ensure invalid codec was not included
	for _, codec := range metadata.Codecs {
		if codec == "invalid-codec" {
			t.Errorf("invalid codec 'invalid-codec' should not be in metadata")
		}
	}
}

func TestPolicyValidation(t *testing.T) {
	// Test type missing required field
	type BadRequest struct {
		Name string
	}

	s := New().
		WithPolicy(Policy{
			Name: "strict-policy",
			Policies: []TypePolicy{
				{
					Match: "*Request",
					Ensure: map[string]string{
						"ID": "string",
					},
				},
			},
		}).
		WithStrictMode().
		Build()

	// This should panic in strict mode
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for missing required field")
		} else if errStr, ok := r.(string); !ok || !strings.Contains(errStr, "missing required field ID") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()

	Inspect[BadRequest](s)
}

func TestTagTemplateProcessing(t *testing.T) {
	s := &Sentinel{}

	tests := []struct {
		value     string
		fieldName string
		expected  string
	}{
		{"{snake}", "UserName", "user_name"},
		{"{lower}", "UserName", "username"},
		{"{upper}", "UserName", "USERNAME"},
		{"literal", "UserName", "literal"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := s.processTagValue(tt.value, tt.fieldName)
			if got != tt.expected {
				t.Errorf("processTagValue(%q, %q) = %q, want %q", tt.value, tt.fieldName, got, tt.expected)
			}
		})
	}
}

func TestYAMLPolicyLoading(t *testing.T) {
	yamlContent := `
name: test-policy
policies:
  - match: "*Model"
    ensure:
      ID: string
      CreatedAt: time.Time
    fields:
      - match: "*_at"
        type: time.Time
        apply:
          json: "{snake}"
`

	policy, err := LoadPolicy(strings.NewReader(yamlContent))
	if err != nil {
		t.Fatalf("failed to load policy: %v", err)
	}

	if policy.Name != "test-policy" {
		t.Errorf("expected policy name to be 'test-policy', got %q", policy.Name)
	}

	if len(policy.Policies) != 1 {
		t.Fatalf("expected 1 type policy, got %d", len(policy.Policies))
	}

	tp := policy.Policies[0]
	if tp.Match != "*Model" {
		t.Errorf("expected type match to be '*Model', got %q", tp.Match)
	}

	if len(tp.Ensure) != 2 {
		t.Errorf("expected 2 ensure fields, got %d", len(tp.Ensure))
	}

	if tp.Ensure["ID"] != "string" {
		t.Errorf("expected ID to be 'string', got %q", tp.Ensure["ID"])
	}
}

func TestPolicyValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		policy   Policy
		errorMsg string
	}{
		{
			name:     "missing name",
			policy:   Policy{},
			errorMsg: "must have a name",
		},
		{
			name:     "no type policies",
			policy:   Policy{Name: "test"},
			errorMsg: "must have at least one type policy",
		},
		{
			name: "missing type match",
			policy: Policy{
				Name: "test",
				Policies: []TypePolicy{
					{Match: ""},
				},
			},
			errorMsg: "must have a match pattern",
		},
		{
			name: "missing field match",
			policy: Policy{
				Name: "test",
				Policies: []TypePolicy{
					{
						Match: "*Model",
						Fields: []FieldPolicy{
							{Match: ""},
						},
					},
				},
			},
			errorMsg: "must have a match pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePolicy(tt.policy)
			if err == nil {
				t.Error("expected validation error")
			} else if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
			}
		})
	}
}
