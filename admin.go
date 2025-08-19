package sentinel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Admin provides exclusive write access to sentinel policies.
// Only one admin instance is allowed per process to prevent conflicting policy changes.
// Configuration can be sealed/unsealed to control when changes are allowed.
type Admin struct {
	sentinel      *Sentinel
	sealed        atomic.Bool  // Configuration is frozen once sealed
	configSession atomic.Int32 // Tracks configuration sessions
}

var (
	adminInstance *Admin
	adminMutex    sync.Mutex
	adminCreated  bool
)

// NewAdmin creates the singleton Admin instance.
// Returns an error if an admin instance already exists in this process.
func NewAdmin() (*Admin, error) {
	adminMutex.Lock()
	defer adminMutex.Unlock()

	if adminCreated {
		return nil, fmt.Errorf("sentinel: admin already exists - only one admin allowed per process")
	}

	adminCreated = true
	adminInstance = &Admin{
		sentinel: instance, // Reference to global sentinel
	}

	return adminInstance, nil
}

// SetPolicies replaces all policies with the provided set.
// This immediately invalidates cached metadata to ensure consistency.
// Returns an error if called when configuration is sealed.
func (a *Admin) SetPolicies(ctx context.Context, policies []Policy) error {
	if a.sealed.Load() {
		return fmt.Errorf("sentinel: cannot modify policies while configuration is sealed - call Unseal() first")
	}

	// Update policies
	a.sentinel.policies = policies

	// Rebuild the pipeline with new policies
	a.sentinel.pipeline = a.sentinel.buildExtractionPipeline()

	// Clear cache to ensure immediate consistency with new policies
	// TTL+LRU will handle natural expiration of future extractions
	a.sentinel.cache.Clear()

	// Emit admin event
	Logger.Admin.Emit(ctx, "ADMIN_ACTION", "Policies set", AdminEvent{
		Timestamp:   time.Now(),
		Action:      "policy_set",
		PolicyCount: len(policies),
	})
	return nil
}

// AddPolicy adds one or more policies to the current set.
// This immediately invalidates cached metadata to ensure consistency.
// Returns an error if called when configuration is sealed.
func (a *Admin) AddPolicy(ctx context.Context, policies ...Policy) error {
	if a.sealed.Load() {
		return fmt.Errorf("sentinel: cannot modify policies while configuration is sealed - call Unseal() first")
	}

	// Add to existing policies
	a.sentinel.policies = append(a.sentinel.policies, policies...)

	// Rebuild the pipeline to include new policies
	a.sentinel.pipeline = a.sentinel.buildExtractionPipeline()

	// Clear cache to ensure immediate consistency with new policies
	// TTL+LRU will handle natural expiration of future extractions
	a.sentinel.cache.Clear()

	// Emit admin event
	Logger.Admin.Emit(ctx, "ADMIN_ACTION", "Policies added", AdminEvent{
		Timestamp:   time.Now(),
		Action:      "policy_added",
		PolicyCount: len(a.sentinel.policies),
	})
	return nil
}

// GetPolicies returns a copy of the currently configured policies.
// This is read-only access for inspection purposes.
func (a *Admin) GetPolicies() []Policy {
	// Return a copy to prevent mutation
	policies := make([]Policy, len(a.sentinel.policies))
	copy(policies, a.sentinel.policies)
	return policies
}

// Seal freezes the configuration, preventing any further policy changes.
// After sealing, type inspection is allowed but policy modifications will return errors.
func (a *Admin) Seal(ctx context.Context) error {
	if a.sealed.Load() {
		return fmt.Errorf("sentinel: configuration already sealed")
	}
	a.sealed.Store(true)

	// Mark the global instance as sealed too
	instance.configSealed.Store(true)

	// Increment session counter
	a.configSession.Add(1)

	// Emit admin event
	Logger.Admin.Emit(ctx, "ADMIN_ACTION", "Configuration sealed", AdminEvent{
		Timestamp:   time.Now(),
		Action:      "sealed",
		PolicyCount: len(a.sentinel.policies),
	})
	return nil
}

// Unseal allows configuration changes again by clearing the cache and unsealing.
// This ensures proper cache invalidation when policies change.
func (a *Admin) Unseal(ctx context.Context) error {
	if !a.sealed.Load() {
		return fmt.Errorf("sentinel: configuration is not sealed")
	}

	// Clear the cache to ensure consistency with new policies
	a.sentinel.cache.Clear()

	// Unseal both admin and global instance
	a.sealed.Store(false)
	instance.configSealed.Store(false)

	// Emit admin event
	Logger.Admin.Emit(ctx, "ADMIN_ACTION", "Configuration unsealed", AdminEvent{
		Timestamp:   time.Now(),
		Action:      "unsealed",
		PolicyCount: len(a.sentinel.policies),
	})

	return nil
}

// IsSealed returns true if the configuration has been sealed.
func (a *Admin) IsSealed() bool {
	return a.sealed.Load()
}

// ConfigSession returns the current configuration session number.
// This increments each time Seal() is called.
func (a *Admin) ConfigSession() int32 {
	return a.configSession.Load()
}

// resetAdminForTesting resets the admin singleton state.
// This is only for testing purposes and should not be used in production code.
func resetAdminForTesting() {
	adminMutex.Lock()
	defer adminMutex.Unlock()

	// Reset admin fields if it exists
	if adminInstance != nil {
		adminInstance.sealed.Store(false)
		adminInstance.configSession.Store(0)
	}

	adminInstance = nil
	adminCreated = false

	// Reset sealed state
	instance.configSealed.Store(false)

	// Clear the cache to ensure clean test state
	// In production, cache would persist across policy changes due to TTL
	instance.cache.Clear()
}
