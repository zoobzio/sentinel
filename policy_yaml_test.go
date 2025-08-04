package sentinel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadPolicy(t *testing.T) {
	t.Run("valid YAML policy", func(t *testing.T) {
		yamlData := `
name: test-policy
policies:
  - match: "*Request"
    ensure:
      ID: string
    fields:
      - match: "Password"
        apply:
          redact: "[HIDDEN]"
          encrypt: "secret"
`
		policy, err := LoadPolicy(strings.NewReader(yamlData))
		if err != nil {
			t.Fatalf("LoadPolicy failed: %v", err)
		}

		if policy.Name != "test-policy" {
			t.Errorf("expected name 'test-policy', got %s", policy.Name)
		}
		if len(policy.Policies) != 1 {
			t.Fatalf("expected 1 type policy, got %d", len(policy.Policies))
		}

		tp := policy.Policies[0]
		if tp.Match != "*Request" {
			t.Errorf("expected match '*Request', got %s", tp.Match)
		}
		if tp.Ensure["ID"] != "string" {
			t.Errorf("expected ensure ID: string, got %s", tp.Ensure["ID"])
		}
		if len(tp.Fields) != 1 {
			t.Fatalf("expected 1 field policy, got %d", len(tp.Fields))
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		yamlData := `
name: test-policy
policies
  - invalid yaml structure
`
		_, err := LoadPolicy(strings.NewReader(yamlData))
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		yamlData := `
policies:
  - match: "*Request"
`
		_, err := LoadPolicy(strings.NewReader(yamlData))
		if err == nil {
			t.Error("expected validation error for missing name")
		}
		if !strings.Contains(err.Error(), "must have a name") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty policies", func(t *testing.T) {
		yamlData := `
name: test-policy
policies: []
`
		_, err := LoadPolicy(strings.NewReader(yamlData))
		if err == nil {
			t.Error("expected validation error for empty policies")
		}
		if !strings.Contains(err.Error(), "at least one type policy") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("complex policy", func(t *testing.T) {
		yamlData := `
name: complex-policy
policies:
  - match: "*Model"
    ensure:
      ID: string
      CreatedAt: time.Time
    codecs:
      - json
      - xml
    fields:
      - match: "*_at"
        type: time.Time
        apply:
          json: "created_at"
      - match: "Email"
        require:
          validate: "email"
  - match: "*Request"
    fields:
      - match: "*Token"
        apply:
          encrypt: "secret"
          redact: "[REDACTED]"
`
		policy, err := LoadPolicy(strings.NewReader(yamlData))
		if err != nil {
			t.Fatalf("LoadPolicy failed: %v", err)
		}

		if len(policy.Policies) != 2 {
			t.Errorf("expected 2 type policies, got %d", len(policy.Policies))
		}

		// Check first type policy
		tp1 := policy.Policies[0]
		if len(tp1.Ensure) != 2 {
			t.Errorf("expected 2 ensure rules, got %d", len(tp1.Ensure))
		}
		if len(tp1.Codecs) != 2 {
			t.Errorf("expected 2 codecs, got %d", len(tp1.Codecs))
		}
		if len(tp1.Fields) != 2 {
			t.Errorf("expected 2 field policies, got %d", len(tp1.Fields))
		}
	})
}

func TestLoadPolicyFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	t.Run("valid policy file", func(t *testing.T) {
		// Create a test policy file
		policyPath := filepath.Join(tmpDir, "test-policy.yaml")
		content := `
name: file-policy
policies:
  - match: "*"
    fields:
      - match: "ID"
        apply:
          json: "id"
`
		if err := os.WriteFile(policyPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		policy, err := LoadPolicyFile(policyPath)
		if err != nil {
			t.Fatalf("LoadPolicyFile failed: %v", err)
		}

		if policy.Name != "file-policy" {
			t.Errorf("expected name 'file-policy', got %s", policy.Name)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadPolicyFile(filepath.Join(tmpDir, "non-existent.yaml"))
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("invalid YAML in file", func(t *testing.T) {
		policyPath := filepath.Join(tmpDir, "invalid.yaml")
		content := `invalid: yaml: content:`
		if err := os.WriteFile(policyPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := LoadPolicyFile(policyPath)
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

func TestLoadPolicyDir(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	t.Run("directory with valid policies", func(t *testing.T) {
		// Create test policy files
		policy1 := `
name: policy-1
policies:
  - match: "*Request"
`
		policy2 := `
name: policy-2
policies:
  - match: "*Response"
`
		// Also create a non-YAML file that should be ignored
		nonYaml := "This is not a YAML file"

		if err := os.WriteFile(filepath.Join(tmpDir, "policy1.yaml"), []byte(policy1), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "policy2.yml"), []byte(policy2), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte(nonYaml), 0o600); err != nil {
			t.Fatal(err)
		}

		// Create a subdirectory that should be ignored
		if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0o755); err != nil {
			t.Fatal(err)
		}

		policies, err := LoadPolicyDir(tmpDir)
		if err != nil {
			t.Fatalf("LoadPolicyDir failed: %v", err)
		}

		if len(policies) != 2 {
			t.Errorf("expected 2 policies, got %d", len(policies))
		}

		// Check policy names
		names := make(map[string]bool)
		for _, p := range policies {
			names[p.Name] = true
		}
		if !names["policy-1"] || !names["policy-2"] {
			t.Error("expected policies not found")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.Mkdir(emptyDir, 0o755); err != nil {
			t.Fatal(err)
		}

		policies, err := LoadPolicyDir(emptyDir)
		if err != nil {
			t.Fatalf("LoadPolicyDir failed: %v", err)
		}

		if len(policies) != 0 {
			t.Errorf("expected 0 policies, got %d", len(policies))
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := LoadPolicyDir(filepath.Join(tmpDir, "non-existent"))
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})

	t.Run("directory with invalid policy", func(t *testing.T) {
		invalidDir := filepath.Join(tmpDir, "invalid")
		if err := os.Mkdir(invalidDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create an invalid policy file (it will be skipped)
		invalidPolicy := `name: but no policies`
		if err := os.WriteFile(filepath.Join(invalidDir, "invalid.yaml"), []byte(invalidPolicy), 0o600); err != nil {
			t.Fatal(err)
		}

		policies, err := LoadPolicyDir(invalidDir)
		if err != nil {
			t.Fatalf("LoadPolicyDir failed: %v", err)
		}

		// Invalid policies are skipped, not an error
		if len(policies) != 0 {
			t.Errorf("expected 0 valid policies, got %d", len(policies))
		}
	})
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		wantErr string
	}{
		{
			name: "valid policy",
			policy: Policy{
				Name: "valid",
				Policies: []TypePolicy{
					{
						Match: "*Model",
						Fields: []FieldPolicy{
							{
								Match: "ID",
								Apply: map[string]string{"json": "id"},
							},
						},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "missing name",
			policy: Policy{
				Policies: []TypePolicy{{Match: "*"}},
			},
			wantErr: "must have a name",
		},
		{
			name: "no policies",
			policy: Policy{
				Name:     "empty",
				Policies: []TypePolicy{},
			},
			wantErr: "at least one type policy",
		},
		{
			name: "type policy missing match",
			policy: Policy{
				Name: "invalid",
				Policies: []TypePolicy{
					{
						Fields: []FieldPolicy{{Match: "ID"}},
					},
				},
			},
			wantErr: "must have a match pattern",
		},
		{
			name: "field policy missing match",
			policy: Policy{
				Name: "invalid",
				Policies: []TypePolicy{
					{
						Match: "*Model",
						Fields: []FieldPolicy{
							{
								Apply: map[string]string{"json": "id"},
							},
						},
					},
				},
			},
			wantErr: "must have a match pattern",
		},
		{
			name: "field policy with no rules",
			policy: Policy{
				Name: "invalid",
				Policies: []TypePolicy{
					{
						Match: "*Model",
						Fields: []FieldPolicy{
							{
								Match: "ID",
							},
						},
					},
				},
			},
			wantErr: "must have either require or apply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePolicy(tt.policy)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestMarshalPolicy(t *testing.T) {
	policy := Policy{
		Name: "test-policy",
		Policies: []TypePolicy{
			{
				Match:  "*Request",
				Ensure: map[string]string{"ID": "string"},
				Codecs: []string{"json"},
				Fields: []FieldPolicy{
					{
						Match:   "Password",
						Type:    "string",
						Require: map[string]string{"validate": "required"},
						Apply:   map[string]string{"redact": "[HIDDEN]"},
					},
				},
			},
		},
	}

	data, err := MarshalPolicy(policy)
	if err != nil {
		t.Fatalf("MarshalPolicy failed: %v", err)
	}

	// Verify it's valid YAML by parsing it back
	var parsed Policy
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal generated YAML: %v", err)
	}

	if parsed.Name != policy.Name {
		t.Errorf("round-trip failed: name mismatch")
	}
	if len(parsed.Policies) != len(policy.Policies) {
		t.Errorf("round-trip failed: policies count mismatch")
	}
}
