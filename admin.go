package sentinel

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Admin provides exclusive write access to sentinel policies.
// Only one admin instance is allowed per process to prevent conflicting policy changes.
// Once sealed, no further policy changes are allowed.
type Admin struct {
	sentinel *Sentinel
	sealed   atomic.Bool // Configuration is frozen once sealed
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

// GetAdmin returns the existing admin instance if it exists, nil otherwise.
// Use this to check if an admin has been created without creating one.
func GetAdmin() *Admin {
	adminMutex.Lock()
	defer adminMutex.Unlock()
	return adminInstance
}

// SetPolicies replaces all policies with the provided set.
// This immediately invalidates cached metadata to ensure consistency.
// Panics if called after Seal().
func (a *Admin) SetPolicies(policies []Policy) {
	if a.sealed.Load() {
		panic("sentinel: cannot modify policies after configuration is sealed")
	}

	// Update policies
	a.sentinel.policies = policies

	// Rebuild the pipeline with new policies
	a.sentinel.pipeline = a.sentinel.buildExtractionPipeline()

	// Clear cache to ensure immediate consistency with new policies
	// TTL+LRU will handle natural expiration of future extractions
	a.sentinel.cache.Clear()

	// Emit admin event
	Logger.Admin.Emit("ADMIN_ACTION", "Policies set", AdminEvent{
		Timestamp:   time.Now(),
		Action:      "policy_set",
		PolicyCount: len(policies),
	})
}

// AddPolicy adds one or more policies to the current set.
// This immediately invalidates cached metadata to ensure consistency.
// Panics if called after Seal().
func (a *Admin) AddPolicy(policies ...Policy) {
	if a.sealed.Load() {
		panic("sentinel: cannot modify policies after configuration is sealed")
	}

	// Add to existing policies
	a.sentinel.policies = append(a.sentinel.policies, policies...)

	// Rebuild the pipeline to include new policies
	a.sentinel.pipeline = a.sentinel.buildExtractionPipeline()

	// Clear cache to ensure immediate consistency with new policies
	// TTL+LRU will handle natural expiration of future extractions
	a.sentinel.cache.Clear()

	// Emit admin event
	Logger.Admin.Emit("ADMIN_ACTION", "Policies added", AdminEvent{
		Timestamp:   time.Now(),
		Action:      "policy_added",
		PolicyCount: len(a.sentinel.policies),
	})
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
// After sealing, type inspection is allowed but policy modifications will panic.
// This enforces proper initialization order: configure policies first, then use sentinel.
func (a *Admin) Seal() {
	if a.sealed.Load() {
		panic("sentinel: configuration already sealed")
	}
	a.sealed.Store(true)

	// Mark the global instance as sealed too
	instance.configSealed.Store(true)

	// Emit admin event
	Logger.Admin.Emit("ADMIN_ACTION", "Configuration sealed", AdminEvent{
		Timestamp:   time.Now(),
		Action:      "sealed",
		PolicyCount: len(a.sentinel.policies),
	})
}

// IsSealed returns true if the configuration has been sealed.
func (a *Admin) IsSealed() bool {
	return a.sealed.Load()
}

// resetAdminForTesting resets the admin singleton state.
// This is only for testing purposes and should not be used in production code.
func resetAdminForTesting() {
	adminMutex.Lock()
	defer adminMutex.Unlock()
	adminInstance = nil
	adminCreated = false

	// Reset sealed state
	instance.configSealed.Store(false)

	// Clear the cache to ensure clean test state
	// In production, cache would persist across policy changes due to TTL
	instance.cache.Clear()
}
