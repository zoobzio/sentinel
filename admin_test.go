package sentinel

import (
	"reflect"
	"testing"
)

func TestAdminUnsealReseal(t *testing.T) {
	t.Run("BasicUnsealReseal", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// Should not be sealed initially
		if admin.IsSealed() {
			t.Error("Admin should not be sealed initially")
		}

		// Seal it
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		if !admin.IsSealed() {
			t.Error("Admin should be sealed after Seal()")
		}

		// Unseal it
		if err := admin.Unseal(); err != nil {
			t.Fatalf("Failed to unseal: %v", err)
		}

		if admin.IsSealed() {
			t.Error("Admin should not be sealed after Unseal()")
		}

		// Should be able to add policies after unsealing
		policy := Policy{
			Name: "test-policy",
			Policies: []TypePolicy{
				{Match: "*", Classification: "test"},
			},
		}
		if err := admin.AddPolicy(policy); err != nil {
			t.Fatalf("Failed to add policy after unseal: %v", err)
		}

		// Seal again
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed to reseal: %v", err)
		}

		// Should not be able to add policies when sealed
		if err := admin.AddPolicy(policy); err == nil {
			t.Error("Expected error when adding policy to sealed admin")
		}
	})

	t.Run("ConfigSessionTracking", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// Initial session should be 0
		if session := admin.ConfigSession(); session != 0 {
			t.Errorf("Expected initial session to be 0, got %d", session)
		}

		// Seal - should increment session
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}
		if session := admin.ConfigSession(); session != 1 {
			t.Errorf("Expected session to be 1 after first seal, got %d", session)
		}

		// Unseal and reseal - should increment again
		if err := admin.Unseal(); err != nil {
			t.Fatalf("Failed to unseal: %v", err)
		}
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed to reseal: %v", err)
		}
		if session := admin.ConfigSession(); session != 2 {
			t.Errorf("Expected session to be 2 after reseal, got %d", session)
		}
	})

	t.Run("CacheClearOnUnseal", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// Seal and inspect a type to cache it
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		type TestType struct {
			Value string `json:"value"`
		}

		// This should cache the metadata
		_ = Inspect[TestType]()

		// Verify it's cached
		if _, exists := instance.cache.Get(getTypeName(reflect.TypeOf(TestType{}))); !exists {
			t.Error("Expected metadata to be cached")
		}

		// Unseal - should clear cache
		if err := admin.Unseal(); err != nil {
			t.Fatalf("Failed to unseal: %v", err)
		}

		// Cache should be cleared
		if _, exists := instance.cache.Get(getTypeName(reflect.TypeOf(TestType{}))); exists {
			t.Error("Expected cache to be cleared after unseal")
		}
	})

	t.Run("MultipleUnsealResealCycles", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// Multiple cycles should work when not enforcing one config period
		for i := 0; i < 3; i++ {
			if err := admin.Seal(); err != nil {
				t.Fatalf("Failed to seal on iteration %d: %v", i, err)
			}
			if err := admin.Unseal(); err != nil {
				t.Fatalf("Failed to unseal on iteration %d: %v", i, err)
			}
		}

		// Final seal
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed final seal: %v", err)
		}

		if session := admin.ConfigSession(); session != 4 {
			t.Errorf("Expected session to be 4 after 4 seals, got %d", session)
		}
	})

	t.Run("UnsealNotSealedError", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// Should error if trying to unseal when not sealed
		if err := admin.Unseal(); err == nil {
			t.Error("Expected error when unsealing non-sealed admin")
		}
	})

	t.Run("SealAlreadySealedError", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// First seal should work
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		// Second seal should error
		if err := admin.Seal(); err == nil {
			t.Error("Expected error when sealing already sealed admin")
		}
	})

	t.Run("GlobalConfigSealedSync", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// Global should not be sealed initially
		if IsConfigSealed() {
			t.Error("Global config should not be sealed initially")
		}

		// Seal admin
		if err := admin.Seal(); err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		// Global should be sealed
		if !IsConfigSealed() {
			t.Error("Global config should be sealed after admin seal")
		}

		// Unseal admin
		if err := admin.Unseal(); err != nil {
			t.Fatalf("Failed to unseal: %v", err)
		}

		// Global should be unsealed
		if IsConfigSealed() {
			t.Error("Global config should not be sealed after admin unseal")
		}
	})
}

func TestAutoSeal(t *testing.T) {
	t.Run("AutoSealOnFirstInspect", func(t *testing.T) {
		resetAdminForTesting()

		// No admin created - just inspect
		type SimpleType struct {
			Value string `json:"value"`
		}

		// Should auto-seal and allow inspection
		metadata := Inspect[SimpleType]()
		if metadata.TypeName != "SimpleType" {
			t.Errorf("Expected TypeName 'SimpleType', got %s", metadata.TypeName)
		}

		// Should be sealed now
		if !IsConfigSealed() {
			t.Error("Expected config to be auto-sealed after first inspect")
		}
	})

	t.Run("AutoSealWithExistingAdmin", func(t *testing.T) {
		resetAdminForTesting()
		admin, err := NewAdmin()
		if err != nil {
			t.Fatalf("failed to create admin: %v", err)
		}

		// Add a policy before auto-seal
		policy := Policy{
			Name: "test-policy",
			Policies: []TypePolicy{
				{Match: "*", Classification: "test"},
			},
		}
		if err := admin.AddPolicy(policy); err != nil {
			t.Fatalf("Failed to add policy: %v", err)
		}

		// Admin exists but not sealed
		if admin.IsSealed() {
			t.Error("Admin should not be sealed yet")
		}

		// Inspect should trigger auto-seal
		type TestType struct {
			Value string `json:"value"`
		}
		metadata := Inspect[TestType]()

		// Should have the classification from policy
		if metadata.Classification != "test" {
			t.Errorf("Expected classification 'test', got '%s'", metadata.Classification)
		}

		// Admin should be sealed now
		if !admin.IsSealed() {
			t.Error("Admin should be auto-sealed after inspect")
		}

		// Global should be sealed
		if !IsConfigSealed() {
			t.Error("Global config should be sealed")
		}

		// Should not be able to modify policies
		if err := admin.AddPolicy(policy); err == nil {
			t.Error("Expected error when adding policy after auto-seal")
		}
	})
}
