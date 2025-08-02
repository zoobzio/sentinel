package sentinel

import (
	"fmt"
	"strings"
	"testing"
)

func TestPolicySystemDemo(t *testing.T) {
	// Define test types
	type UserRequest struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	// Create a sentinel with security policies
	s := New().
		WithPolicy(Policy{
			Name: "security-policy",
			Policies: []TypePolicy{
				{
					Match: "*Request",
					Ensure: map[string]string{
						"ID": "string",
					},
					Fields: []FieldPolicy{
						{
							Match: "Password",
							Apply: map[string]string{
								"encrypt": "secret",
								"redact":  "[HIDDEN]",
								"no_log":  "true",
							},
						},
						{
							Match: "Email",
							Apply: map[string]string{
								"validate": "required,email",
								"encrypt":  "pii",
							},
						},
					},
				},
			},
		}).
		Build()

	// Extract metadata
	metadata := Inspect[UserRequest](s)

	fmt.Printf("\nGenerated metadata for %s:\n", metadata.TypeName)
	fmt.Printf("Package: %s\n", metadata.PackageName)
	fmt.Printf("Fields:\n")

	for _, field := range metadata.Fields {
		fmt.Printf("  %s (%s):\n", field.Name, field.Type)
		for tag, value := range field.Tags {
			fmt.Printf("    %s: %s\n", tag, value)
		}
	}

	// Verify policies were applied
	var passwordField *FieldMetadata
	for i, field := range metadata.Fields {
		if field.Name == "Password" {
			passwordField = &metadata.Fields[i]
			break
		}
	}

	if passwordField == nil {
		t.Fatal("Password field not found")
	}

	// Check that policy tags were applied
	if passwordField.Tags["encrypt"] != "secret" {
		t.Errorf("Expected encrypt=secret, got %s", passwordField.Tags["encrypt"])
	}

	if passwordField.Tags["redact"] != "[HIDDEN]" {
		t.Errorf("Expected redact=[HIDDEN], got %s", passwordField.Tags["redact"])
	}

	if passwordField.Tags["no_log"] != "true" {
		t.Errorf("Expected no_log=true, got %s", passwordField.Tags["no_log"])
	}

	fmt.Printf("\n✅ Policy system working correctly!\n")
}

func TestYAMLPolicyDemo(t *testing.T) {
	yamlPolicy := `
name: organization-standards
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
          
  - match: "*Request"
    fields:
      - match: "*Token"
        apply:
          encrypt: "secret"
          redact: "[REDACTED]"
`

	policy, err := LoadPolicy(strings.NewReader(yamlPolicy))
	if err != nil {
		t.Fatalf("Failed to load YAML policy: %v", err)
	}

	s := New().
		WithPolicy(policy).
		Build()

	type LoginRequest struct {
		AuthToken string `json:"auth_token"`
		UserID    string `json:"user_id"`
	}

	metadata := Inspect[LoginRequest](s)

	fmt.Printf("\nYAML Policy Demo - %s:\n", metadata.TypeName)
	for _, field := range metadata.Fields {
		if field.Name == "AuthToken" {
			fmt.Printf("  %s tags: %+v\n", field.Name, field.Tags)

			if field.Tags["encrypt"] != "secret" {
				t.Errorf("Expected encrypt=secret from YAML policy")
			}
			if field.Tags["redact"] != "[REDACTED]" {
				t.Errorf("Expected redact=[REDACTED] from YAML policy")
			}
		}
	}

	fmt.Printf("✅ YAML policy system working!\n")
}
