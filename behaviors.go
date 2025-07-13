package catalog

import (
	"aegis/sctx"
)

// BEHAVIOR KEY TYPES - Each category gets its own typed keys

// SecurityBehaviorKey identifies security-related behaviors
type SecurityBehaviorKey string

const (
	// Core security behaviors
	AccessControlBehavior SecurityBehaviorKey = "access_control"
	RedactionBehavior     SecurityBehaviorKey = "redaction"
	EncryptionBehavior    SecurityBehaviorKey = "encryption"
	AuditBehavior         SecurityBehaviorKey = "audit"
	MaskingBehavior       SecurityBehaviorKey = "masking"
	
	// Compliance behaviors
	GDPRCompliance SecurityBehaviorKey = "gdpr"
	PCICompliance  SecurityBehaviorKey = "pci"
	HIPAACompliance SecurityBehaviorKey = "hipaa"
)

// ValidationBehaviorKey identifies validation behaviors
type ValidationBehaviorKey string

const (
	// Core validation behaviors
	RequiredValidation ValidationBehaviorKey = "required"
	FormatValidation   ValidationBehaviorKey = "format"
	RangeValidation    ValidationBehaviorKey = "range"
	LengthValidation   ValidationBehaviorKey = "length"
	PatternValidation  ValidationBehaviorKey = "pattern"
	CustomValidation   ValidationBehaviorKey = "custom"
)

// DefaultsBehaviorKey identifies default value behaviors
type DefaultsBehaviorKey string

const (
	SecureDefaults  DefaultsBehaviorKey = "secure"
	ZeroDefaults    DefaultsBehaviorKey = "zero"
	RandomDefaults  DefaultsBehaviorKey = "random"
	SystemDefaults  DefaultsBehaviorKey = "system"
)

// ScopeBehaviorKey identifies scope extraction behaviors
type ScopeBehaviorKey string

const (
	FieldScope    ScopeBehaviorKey = "field"
	ObjectScope   ScopeBehaviorKey = "object"
	ResourceScope ScopeBehaviorKey = "resource"
	TenantScope   ScopeBehaviorKey = "tenant"
)

// BEHAVIOR INPUT/OUTPUT TYPES

// SecurityInput contains data and context for security processing
type SecurityInput[T any] struct {
	Data    T
	Context sctx.SecurityContext
}

// SecurityOutput contains processed data or error
type SecurityOutput[T any] struct {
	Data  T
	Error error
}

// ValidationInput contains data to validate
type ValidationInput[T any] struct {
	Data T
}

// ValidationOutput contains validation result
type ValidationOutput[T any] struct {
	Error error
}

// ScopeInput contains data to extract scope from
type ScopeInput[T any] struct {
	Data T
}

// ScopeOutput contains extracted scope
type ScopeOutput struct {
	Scope string
}

// BEHAVIOR PROCESSORS - Type aliases for clarity

// SecurityProcessor processes security behaviors
type SecurityProcessor[T any] Processor[SecurityInput[T], SecurityOutput[T]]

// ValidationProcessor processes validation behaviors
type ValidationProcessor[T any] Processor[ValidationInput[T], ValidationOutput[T]]

// DefaultsProcessor generates default values
type DefaultsProcessor[T any] Processor[T, T]

// ScopeProcessor extracts scope information
type ScopeProcessor[T any] Processor[ScopeInput[T], ScopeOutput]

// HELPER FUNCTIONS

// GetSecurityPipeline returns the security pipeline for type T
func GetSecurityPipeline[T any]() *ServiceContract[SecurityBehaviorKey, SecurityInput[T], SecurityOutput[T]] {
	return GetContract[SecurityBehaviorKey, SecurityInput[T], SecurityOutput[T]]()
}

// GetValidationPipeline returns the validation pipeline for type T
func GetValidationPipeline[T any]() *ServiceContract[ValidationBehaviorKey, ValidationInput[T], ValidationOutput[T]] {
	return GetContract[ValidationBehaviorKey, ValidationInput[T], ValidationOutput[T]]()
}

// GetDefaultsPipeline returns the defaults pipeline for type T
func GetDefaultsPipeline[T any]() *ServiceContract[DefaultsBehaviorKey, T, T] {
	return GetContract[DefaultsBehaviorKey, T, T]()
}

// GetScopePipeline returns the scope extraction pipeline for type T
func GetScopePipeline[T any]() *ServiceContract[ScopeBehaviorKey, ScopeInput[T], ScopeOutput] {
	return GetContract[ScopeBehaviorKey, ScopeInput[T], ScopeOutput]()
}