package sentinel

import (
	"errors"
	"testing"
)

// Test types implementing various conventions.
type ConventionUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (ConventionUser) Defaults() ConventionUser {
	return ConventionUser{
		ID:   "default-id",
		Name: "Anonymous",
	}
}

func (u ConventionUser) Validate() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func (u *ConventionUser) Clone() ConventionUser {
	return ConventionUser{
		ID:   u.ID,
		Name: u.Name,
	}
}

// Type with no conventions.
type PlainStruct struct {
	Value string `json:"value"`
}

// Type with different return signature.
type BadDefaults struct {
	Data string
}

func (BadDefaults) Defaults() (BadDefaults, error) {
	return BadDefaults{Data: "default"}, nil
}

func TestConventionDetection(t *testing.T) {
	// Set up a policy with conventions
	policy := Policy{
		Name: "test-conventions",
		Conventions: []Convention{
			{
				Name:       "defaults",
				MethodName: "Defaults",
				Params:     []string{},
				Returns:    []string{"@self"},
			},
			{
				Name:       "validator",
				MethodName: "Validate",
				Params:     []string{},
				Returns:    []string{"error"},
			},
			{
				Name:       "clone",
				MethodName: "Clone",
				Params:     []string{},
				Returns:    []string{"@self"},
			},
		},
	}

	// Clear any existing policies and set our test policy
	resetAdminForTesting()
	admin, err := NewAdmin()
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	if err := admin.SetPolicies([]Policy{policy}); err != nil {
		t.Fatalf("failed to set policies: %v", err)
	}
	if err := admin.Seal(); err != nil {
		panic(err)
	}

	// Test type with conventions
	t.Run("DetectsImplementedConventions", func(t *testing.T) {
		metadata := Inspect[ConventionUser]()

		// Should detect defaults and validator
		if len(metadata.Conventions) != 3 {
			t.Errorf("Expected 3 conventions, got %d: %v", len(metadata.Conventions), metadata.Conventions)
		}

		expectedConventions := map[string]bool{
			"defaults":  false,
			"validator": false,
			"clone":     false,
		}

		for _, conv := range metadata.Conventions {
			expectedConventions[conv] = true
		}

		for conv, found := range expectedConventions {
			if !found {
				t.Errorf("Expected convention %s not found", conv)
			}
		}
	})

	// Test type with no conventions
	t.Run("EmptyConventionsForPlainStruct", func(t *testing.T) {
		metadata := Inspect[PlainStruct]()

		if len(metadata.Conventions) != 0 {
			t.Errorf("Expected no conventions, got %d: %v", len(metadata.Conventions), metadata.Conventions)
		}
	})

	// Test type with wrong signature
	t.Run("RejectsWrongSignature", func(t *testing.T) {
		metadata := Inspect[BadDefaults]()

		// Should not detect defaults because signature doesn't match
		for _, conv := range metadata.Conventions {
			if conv == "defaults" {
				t.Error("Should not detect defaults convention with wrong signature")
			}
		}
	})
}

func TestConventionAPI(t *testing.T) {
	// Ensure policy is set
	policy := Policy{
		Name: "test-api",
		Conventions: []Convention{
			{
				Name:       "defaults",
				MethodName: "Defaults",
				Params:     []string{},
				Returns:    []string{"@self"},
			},
			{
				Name:       "validator",
				MethodName: "Validate",
				Params:     []string{},
				Returns:    []string{"error"},
			},
		},
	}
	resetAdminForTesting()
	admin, err := NewAdmin()
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	if err := admin.SetPolicies([]Policy{policy}); err != nil {
		t.Fatalf("failed to set policies: %v", err)
	}
	if err := admin.Seal(); err != nil {
		panic(err)
	}

	t.Run("HasConvention", func(t *testing.T) {
		// ConventionUser has defaults
		if !HasConvention[ConventionUser]("defaults") {
			t.Error("Expected ConventionUser to have defaults convention")
		}

		// PlainStruct does not have defaults
		if HasConvention[PlainStruct]("defaults") {
			t.Error("Expected PlainStruct to not have defaults convention")
		}
	})

	t.Run("GetConventions", func(t *testing.T) {
		conventions := GetConventions[ConventionUser]()

		if len(conventions) == 0 {
			t.Error("Expected ConventionUser to have conventions")
		}

		// Check for specific conventions
		hasDefaults := false
		hasValidator := false
		for _, conv := range conventions {
			if conv == "defaults" {
				hasDefaults = true
			}
			if conv == "validator" {
				hasValidator = true
			}
		}

		if !hasDefaults {
			t.Error("Expected ConventionUser to have defaults convention")
		}
		if !hasValidator {
			t.Error("Expected ConventionUser to have validator convention")
		}
	})
}

func TestSpecialTokens(t *testing.T) {
	// Test @self token matching
	policy := Policy{
		Name: "test-self-token",
		Conventions: []Convention{
			{
				Name:       "defaults",
				MethodName: "Defaults",
				Params:     []string{},
				Returns:    []string{"@self"},
			},
			{
				Name:       "returnsself",
				MethodName: "ReturnsSelf",
				Params:     []string{},
				Returns:    []string{"@self"},
			},
		},
	}
	resetAdminForTesting()
	admin, err := NewAdmin()
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	if err := admin.SetPolicies([]Policy{policy}); err != nil {
		t.Fatalf("failed to set policies: %v", err)
	}
	if err := admin.Seal(); err != nil {
		panic(err)
	}

	// Type that returns itself
	type SelfReturner struct {
		Value int
	}

	// This should match @self
	var _ = func(s SelfReturner) SelfReturner {
		return s
	}

	// Since we can't add methods in tests dynamically,
	// we'll test the validation logic directly
	t.Run("ValidatesSelfToken", func(t *testing.T) {
		// The @self validation is tested implicitly through ConventionUser.Defaults()
		// which returns ConventionUser (matching @self)
		metadata := Inspect[ConventionUser]()

		// If defaults was detected, @self validation worked
		hasDefaults := false
		for _, conv := range metadata.Conventions {
			if conv == "defaults" {
				hasDefaults = true
				break
			}
		}

		if !hasDefaults {
			t.Error("@self token validation failed - defaults convention not detected")
		}
	})
}
