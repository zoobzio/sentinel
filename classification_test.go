package sentinel

import (
	"testing"
)

// Test types for classification.
type PublicData struct {
	Name string `json:"name"`
}

type UserProfile struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type CreditCardInfo struct {
	Number string `json:"number"`
	CVV    string `json:"cvv"`
}

type MedicalRecord struct {
	PatientID string `json:"patient_id"`
	Diagnosis string `json:"diagnosis"`
}

func TestClassificationSystem(t *testing.T) {
	// Set up a policy with various classification levels
	policy := Policy{
		Name: "data-classification",
		Policies: []TypePolicy{
			{
				Match:          "*",
				Classification: "internal", // Default classification
			},
			{
				Match:          "Public*",
				Classification: "public",
			},
			{
				Match:          "*User*",
				Classification: "pii",
			},
			{
				Match:          "*CreditCard*",
				Classification: "pci-dss",
			},
			{
				Match:          "*Medical*",
				Classification: "phi", // Protected Health Information
			},
		},
	}

	// Reset admin for testing and set our test policy
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

	t.Run("BasicClassification", func(t *testing.T) {
		// Test public data
		publicClass := GetClassification[PublicData]()
		if publicClass != "public" {
			t.Errorf("Expected PublicData to have 'public' classification, got '%s'", publicClass)
		}

		// Test PII data
		userClass := GetClassification[UserProfile]()
		if userClass != "pii" {
			t.Errorf("Expected UserProfile to have 'pii' classification, got '%s'", userClass)
		}

		// Test PCI-DSS data
		ccClass := GetClassification[CreditCardInfo]()
		if ccClass != "pci-dss" {
			t.Errorf("Expected CreditCardInfo to have 'pci-dss' classification, got '%s'", ccClass)
		}

		// Test PHI data
		medClass := GetClassification[MedicalRecord]()
		if medClass != "phi" {
			t.Errorf("Expected MedicalRecord to have 'phi' classification, got '%s'", medClass)
		}
	})

	t.Run("DefaultClassification", func(t *testing.T) {
		// Type that doesn't match specific patterns should get default
		type RandomData struct {
			Value string
		}

		classification := GetClassification[RandomData]()
		if classification != "internal" {
			t.Errorf("Expected default classification 'internal', got '%s'", classification)
		}
	})

	t.Run("HasClassificationAPI", func(t *testing.T) {
		// Test HasClassification
		if !HasClassification[UserProfile]() {
			t.Error("Expected UserProfile to have a classification")
		}

		// Test type with no classification
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}
		if err := admin.SetPolicies([]Policy{}); err != nil {
			t.Fatalf("failed to set policies: %v", err)
		} // Clear policies
		if err := admin.Seal(); err != nil {
			panic(err)
		}
		if HasClassification[PublicData]() {
			t.Error("Expected PublicData to have no classification when no policies are set")
		}

		// Restore policy
		resetAdminForTesting()
		admin, err = NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}
		if err := admin.SetPolicies([]Policy{policy}); err != nil {
			t.Fatalf("failed to set policies: %v", err)
		}
		if err := admin.Seal(); err != nil {
			panic(err)
		}
	})

	t.Run("LastMatchWins", func(t *testing.T) {
		// Test that last matching policy wins
		overridePolicy := Policy{
			Name: "override-classification",
			Policies: []TypePolicy{
				{
					Match:          "*",
					Classification: "unclassified",
				},
				{
					Match:          "*User*",
					Classification: "confidential",
				},
				{
					Match:          "User*",
					Classification: "highly-confidential", // This should win for UserProfile
				},
			},
		}

		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}
		if err := admin.SetPolicies([]Policy{overridePolicy}); err != nil {
			t.Fatalf("failed to set policies: %v", err)
		}
		if err := admin.Seal(); err != nil {
			panic(err)
		}

		classification := GetClassification[UserProfile]()
		if classification != "highly-confidential" {
			t.Errorf("Expected last match 'highly-confidential', got '%s'", classification)
		}
	})

	t.Run("EmptyClassification", func(t *testing.T) {
		// Test with no policies
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}
		if err := admin.SetPolicies([]Policy{}); err != nil {
			t.Fatalf("failed to set policies: %v", err)
		}
		if err := admin.Seal(); err != nil {
			panic(err)
		}

		classification := GetClassification[UserProfile]()
		if classification != "" {
			t.Errorf("Expected empty classification, got '%s'", classification)
		}
	})

	t.Run("ClassificationInMetadata", func(t *testing.T) {
		// Ensure classification is properly stored in metadata
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

		metadata := Inspect[CreditCardInfo]()
		if metadata.Classification != "pci-dss" {
			t.Errorf("Expected classification in metadata to be 'pci-dss', got '%s'", metadata.Classification)
		}

		// Check that it's included in JSON
		// This validates that the json tag is working
		if metadata.Classification == "" {
			t.Error("Classification should not be empty in metadata")
		}
	})
}

func TestClassificationPrecedence(t *testing.T) {
	// Test complex precedence scenarios
	t.Run("MultipleMatches", func(t *testing.T) {
		policies := []Policy{
			{
				Name: "base-policy",
				Policies: []TypePolicy{
					{
						Match:          "*",
						Classification: "unclassified",
					},
					{
						Match:          "*Data*",
						Classification: "data",
					},
				},
			},
			{
				Name: "specific-policy",
				Policies: []TypePolicy{
					{
						Match:          "*User*",
						Classification: "user-data",
					},
					{
						Match:          "UserData",
						Classification: "specific-user-data",
					},
				},
			},
		}

		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}
		if err := admin.SetPolicies(policies); err != nil {
			t.Fatalf("failed to set policies: %v", err)
		}
		if err := admin.Seal(); err != nil {
			panic(err)
		}

		type UserData struct {
			Info string
		}

		// Should match in order: *, *Data*, *User*, UserData
		// Last match (UserData) should win
		classification := GetClassification[UserData]()
		if classification != "specific-user-data" {
			t.Errorf("Expected 'specific-user-data' from most specific match, got '%s'", classification)
		}
	})
}
