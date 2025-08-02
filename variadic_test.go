package sentinel

import (
	"testing"
)

func TestVariadicWithPolicy(t *testing.T) {
	policy1 := Policy{
		Name: "policy1",
		Policies: []TypePolicy{
			{
				Match: "*Request",
				Fields: []FieldPolicy{
					{
						Match: "Token",
						Apply: map[string]string{
							"encrypt": "secret",
						},
					},
				},
			},
		},
	}

	policy2 := Policy{
		Name: "policy2",
		Policies: []TypePolicy{
			{
				Match: "*Response",
				Fields: []FieldPolicy{
					{
						Match: "Data",
						Apply: map[string]string{
							"redact": "[HIDDEN]",
						},
					},
				},
			},
		},
	}

	policy3 := Policy{
		Name: "policy3",
		Policies: []TypePolicy{
			{
				Match: "*Model",
				Ensure: map[string]string{
					"ID": "string",
				},
			},
		},
	}

	t.Run("single policy", func(t *testing.T) {
		s := New().WithPolicy(policy1).Build()
		if len(s.policies) != 1 {
			t.Errorf("expected 1 policy, got %d", len(s.policies))
		}
		if s.policies[0].Name != "policy1" {
			t.Errorf("expected policy1, got %s", s.policies[0].Name)
		}
	})

	t.Run("multiple policies at once", func(t *testing.T) {
		s := New().WithPolicy(policy1, policy2, policy3).Build()
		if len(s.policies) != 3 {
			t.Errorf("expected 3 policies, got %d", len(s.policies))
		}

		expectedNames := []string{"policy1", "policy2", "policy3"}
		for i, expected := range expectedNames {
			if s.policies[i].Name != expected {
				t.Errorf("expected policy %s at index %d, got %s", expected, i, s.policies[i].Name)
			}
		}
	})

	t.Run("chained policy calls", func(t *testing.T) {
		s := New().
			WithPolicy(policy1).
			WithPolicy(policy2).
			WithPolicy(policy3).
			Build()

		if len(s.policies) != 3 {
			t.Errorf("expected 3 policies, got %d", len(s.policies))
		}

		expectedNames := []string{"policy1", "policy2", "policy3"}
		for i, expected := range expectedNames {
			if s.policies[i].Name != expected {
				t.Errorf("expected policy %s at index %d, got %s", expected, i, s.policies[i].Name)
			}
		}
	})

	t.Run("mixed single and multiple", func(t *testing.T) {
		s := New().
			WithPolicy(policy1).
			WithPolicy(policy2, policy3).
			Build()

		if len(s.policies) != 3 {
			t.Errorf("expected 3 policies, got %d", len(s.policies))
		}

		expectedNames := []string{"policy1", "policy2", "policy3"}
		for i, expected := range expectedNames {
			if s.policies[i].Name != expected {
				t.Errorf("expected policy %s at index %d, got %s", expected, i, s.policies[i].Name)
			}
		}
	})

	t.Run("variadic policies applied correctly", func(t *testing.T) {
		type TestRequest struct {
			ID    string
			Token string
		}

		type TestResponse struct {
			Data string
		}

		s := New().WithPolicy(policy1, policy2).Build()

		// Test Request matches policy1
		reqMetadata := Inspect[TestRequest](s)
		var tokenField *FieldMetadata
		for i, field := range reqMetadata.Fields {
			if field.Name == "Token" {
				tokenField = &reqMetadata.Fields[i]
				break
			}
		}

		if tokenField == nil {
			t.Fatal("Token field not found")
		}

		if tokenField.Tags["encrypt"] != "secret" {
			t.Errorf("expected encrypt tag 'secret', got %q", tokenField.Tags["encrypt"])
		}

		// Test Response matches policy2
		respMetadata := Inspect[TestResponse](s)
		var dataField *FieldMetadata
		for i, field := range respMetadata.Fields {
			if field.Name == "Data" {
				dataField = &respMetadata.Fields[i]
				break
			}
		}

		if dataField == nil {
			t.Fatal("Data field not found")
		}

		if dataField.Tags["redact"] != "[HIDDEN]" {
			t.Errorf("expected redact tag '[HIDDEN]', got %q", dataField.Tags["redact"])
		}
	})
}
