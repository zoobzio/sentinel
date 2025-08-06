package sentinel

import (
	"reflect"
	"testing"
)

func TestPolicyStructure(t *testing.T) {
	t.Run("Policy fields", func(t *testing.T) {
		policy := Policy{
			Name: "test-policy",
			Policies: []TypePolicy{
				{
					Match:  "*Request",
					Ensure: map[string]string{"ID": "string"},
					Fields: []FieldPolicy{
						{
							Match: "Password",
						},
					},
				},
			},
		}

		if policy.Name != "test-policy" {
			t.Errorf("expected name 'test-policy', got %s", policy.Name)
		}
		if len(policy.Policies) != 1 {
			t.Errorf("expected 1 type policy, got %d", len(policy.Policies))
		}
	})

	t.Run("TypePolicy fields", func(t *testing.T) {
		tp := TypePolicy{
			Match:  "*Model",
			Ensure: map[string]string{"ID": "string", "CreatedAt": "time.Time"},
			Fields: []FieldPolicy{{Match: "ID"}},
			Rules:  []Rule{{}}, // Empty rule
			Codecs: []string{"json", "xml"},
		}

		if tp.Match != "*Model" {
			t.Errorf("expected match '*Model', got %s", tp.Match)
		}
		if len(tp.Ensure) != 2 {
			t.Errorf("expected 2 ensure entries, got %d", len(tp.Ensure))
		}
		if len(tp.Codecs) != 2 {
			t.Errorf("expected 2 codecs, got %d", len(tp.Codecs))
		}
	})

	t.Run("FieldPolicy fields", func(t *testing.T) {
		fp := FieldPolicy{
			Match:   "Email",
			Type:    "string",
			Require: map[string]string{"validate": "email"},
		}

		if fp.Match != "Email" {
			t.Errorf("expected match 'Email', got %s", fp.Match)
		}
		if fp.Type != "string" {
			t.Errorf("expected type 'string', got %s", fp.Type)
		}
	})
}

func TestMatches(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		// Exact matches
		{"User", "User", true},
		{"User", "UserModel", false},
		{"User", "user", false},

		// Suffix matches with *
		{"*Request", "UserRequest", true},
		{"*Request", "LoginRequest", true},
		{"*Request", "Request", true},
		{"*Request", "RequestUser", false},
		{"*Request", "UserResponse", false},

		// Prefix matches with *
		{"User*", "UserModel", true},
		{"User*", "UserRequest", true},
		{"User*", "User", true},
		{"User*", "ModelUser", false},
		{"User*", "user", false},

		// Contains matches with *
		{"*User*", "UserModel", true},
		{"*User*", "ModelUser", true},
		{"*User*", "ModelUserRequest", true},
		{"*User*", "User", true},
		{"*User*", "Model", false},

		// Empty patterns
		{"", "", true},
		{"", "User", false},
		{"User", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.name, func(t *testing.T) {
			got := matches(tt.pattern, tt.name)
			if got != tt.want {
				t.Errorf("matches(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
			}
		})
	}
}

func TestApplyPolicies(t *testing.T) {
	t.Run("no policies", func(t *testing.T) {
		s := &Sentinel{
			policies: []Policy{},
		}

		ec := &ExtractionContext{
			Type: reflect.TypeOf(struct{ Name string }{}),
			Metadata: ModelMetadata{
				TypeName: "TestStruct",
				Fields: []FieldMetadata{
					{Name: "Name", Type: "string", Tags: map[string]string{}},
				},
			},
		}

		result := s.applyPolicies(ec)
		if len(result.Applied) != 0 {
			t.Errorf("expected no policies applied, got %v", result.Applied)
		}
		if len(result.Errors) != 0 {
			t.Errorf("expected no errors, got %v", result.Errors)
		}
	})

	t.Run("ensure fields", func(t *testing.T) {
		s := &Sentinel{
			policies: []Policy{
				{
					Name: "test-policy",
					Policies: []TypePolicy{
						{
							Match: "User",
							Ensure: map[string]string{
								"ID":        "string",
								"CreatedAt": "time.Time",
							},
						},
					},
				},
			},
		}

		// Missing required field
		ec := &ExtractionContext{
			Type: reflect.TypeOf(struct{ Name string }{}),
			Metadata: ModelMetadata{
				TypeName: "User",
				Fields: []FieldMetadata{
					{Name: "Name", Type: "string"},
				},
			},
		}

		result := s.applyPolicies(ec)
		if len(result.Errors) != 2 {
			t.Errorf("expected 2 errors for missing fields, got %d: %v", len(result.Errors), result.Errors)
		}

		// With required fields but wrong type
		ec2 := &ExtractionContext{
			Type: reflect.TypeOf(struct {
				ID        int
				CreatedAt string
			}{}),
			Metadata: ModelMetadata{
				TypeName: "User",
				Fields: []FieldMetadata{
					{Name: "ID", Type: "int"},
					{Name: "CreatedAt", Type: "string"},
				},
			},
		}

		result2 := s.applyPolicies(ec2)
		if len(result2.Errors) != 2 {
			t.Errorf("expected 2 errors for wrong types, got %d: %v", len(result2.Errors), result2.Errors)
		}
	})

	t.Run("codecs applied", func(t *testing.T) {
		s := &Sentinel{
			policies: []Policy{
				{
					Name: "codec-policy",
					Policies: []TypePolicy{
						{
							Match:  "API*",
							Codecs: []string{"json", "xml"},
						},
					},
				},
			},
		}

		ec := &ExtractionContext{
			Type: reflect.TypeOf(struct{ Name string }{}),
			Metadata: ModelMetadata{
				TypeName: "APIRequest",
			},
		}

		result := s.applyPolicies(ec)
		if len(result.Applied) != 1 {
			t.Errorf("expected 1 policy applied, got %v", result.Applied)
		}

		if len(ec.Metadata.Codecs) != 2 {
			t.Errorf("expected 2 codecs, got %d", len(ec.Metadata.Codecs))
		}
		if ec.Metadata.Codecs[0] != "json" || ec.Metadata.Codecs[1] != "xml" {
			t.Errorf("expected codecs [json, xml], got %v", ec.Metadata.Codecs)
		}
	})
}
