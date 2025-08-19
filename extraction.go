package sentinel

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/zoobzio/pipz"
)

// ExtractionContext flows through the extraction pipeline carrying all metadata.
type ExtractionContext struct {
	Type     reflect.Type
	Instance interface{}   // Zero value of the type
	Metadata ModelMetadata // The metadata being built
}

// extractionPipeline builds the internal processing pipeline for metadata extraction.
func (s *Sentinel) buildExtractionPipeline() *pipz.Sequence[*ExtractionContext] {
	pipeline := pipz.NewSequence[*ExtractionContext]("extraction")

	// Basic reflection extracts initial metadata
	pipeline.Register(pipz.Apply("basic-reflection", s.basicReflection))

	// Apply configured policies
	pipeline.Register(pipz.Apply("apply-policies", s.policyProcessor))

	// Detect conventions
	pipeline.Register(pipz.Apply("detect-conventions", s.conventionDetector))

	// Extract relationships
	pipeline.Register(pipz.Apply("extract-relationships", s.relationshipExtractor))

	// Final validation
	pipeline.Register(pipz.Apply("validate", s.validateMetadata))

	return pipeline
}

// basicReflection performs the initial struct reflection to extract metadata.
func (s *Sentinel) basicReflection(_ context.Context, ec *ExtractionContext) (*ExtractionContext, error) {
	t := ec.Type

	// Initialize metadata
	ec.Metadata = ModelMetadata{
		TypeName:    getTypeName(t),
		PackageName: t.PkgPath(),
		Fields:      s.extractFieldMetadata(t),
	}

	return ec, nil
}

// policyProcessor applies all configured policies to the metadata.
func (s *Sentinel) policyProcessor(ctx context.Context, ec *ExtractionContext) (*ExtractionContext, error) {
	result := s.applyPolicies(ctx, ec)

	// Emit policy events if policies were applied
	if len(result.Applied) > 0 {
		for _, policyName := range result.Applied {
			// Get metrics for this specific policy if available
			metrics := result.PolicyMetrics[policyName]

			Logger.Policy.Emit(ctx, "POLICY_APPLIED", "Policy applied", PolicyEvent{
				TypeName:       ec.Metadata.TypeName,
				PolicyName:     policyName,
				FieldsModified: metrics.FieldsModified,
				TagsApplied:    metrics.TagsApplied,
				AffectedFields: metrics.AffectedFields,
				Warnings:       result.Warnings,
				Errors:         result.Errors,
				Timestamp:      time.Now(),
			})
		}
	}

	// If there are errors, emit violation events and fail the extraction
	if len(result.Errors) > 0 {
		Logger.Validation.Emit(ctx, "POLICY_VIOLATION", "Policy validation failed", ValidationEvent{
			Timestamp:  time.Now(),
			TypeName:   ec.Metadata.TypeName,
			PolicyName: "policy-enforcement",
			Errors:     result.Errors,
			Fatal:      true,
		})

		// Join all errors into a single error message
		errMsg := "Policy violations: " + result.Errors[0]
		for i := 1; i < len(result.Errors); i++ {
			errMsg += "; " + result.Errors[i]
		}
		return ec, fmt.Errorf("%s", errMsg)
	}

	// Add warnings to metadata (could be used by consumers)
	// TODO: Add warnings field to ModelMetadata if needed

	return ec, nil
}

// validateMetadata performs final validation on the extracted metadata.
func (*Sentinel) validateMetadata(_ context.Context, ec *ExtractionContext) (*ExtractionContext, error) {
	// Basic validation - ensure we have a type name
	if ec.Metadata.TypeName == "" {
		return ec, fmt.Errorf("extracted metadata missing type name")
	}

	// Could add more validation here

	return ec, nil
}

// extractFieldMetadata extracts field information with registered tags.
func (s *Sentinel) extractFieldMetadata(t reflect.Type) []FieldMetadata {
	var fields []FieldMetadata

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fields
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		// Extract all tags
		tags := make(map[string]string)

		// Include registered tags
		s.tagMutex.RLock()
		for tagName := range s.registeredTags {
			if tagValue := field.Tag.Get(tagName); tagValue != "" {
				tags[tagName] = tagValue
			}
		}
		s.tagMutex.RUnlock()

		// Always include common tags
		commonTags := []string{"json", "validate", "db", "scope", "encrypt", "redact", "desc", "example"}
		for _, tagName := range commonTags {
			if tagValue := field.Tag.Get(tagName); tagValue != "" {
				tags[tagName] = tagValue
			}
		}

		fieldMeta := FieldMetadata{
			Name: field.Name,
			Type: field.Type.String(),
			Tags: tags,
		}

		fields = append(fields, fieldMeta)
	}

	return fields
}

// conventionDetector checks for method patterns defined in policies.
func (s *Sentinel) conventionDetector(_ context.Context, ec *ExtractionContext) (*ExtractionContext, error) {
	var detectedConventions []string

	// Collect all conventions from all policies
	for _, policy := range s.policies {
		for _, convention := range policy.Conventions {
			// Check both value and pointer receivers
			if s.checkConvention(ec.Type, convention) {
				detectedConventions = append(detectedConventions, convention.Name)
			}
		}
	}

	// Add detected conventions to metadata
	ec.Metadata.Conventions = detectedConventions

	return ec, nil
}

// checkConvention verifies if a type implements a convention.
func (s *Sentinel) checkConvention(t reflect.Type, convention Convention) bool {
	// Try value receiver first
	if method, found := t.MethodByName(convention.MethodName); found {
		if s.validateMethodSignature(method, t, convention) {
			return true
		}
	}

	// Try pointer receiver
	ptrType := reflect.PointerTo(t)
	if method, found := ptrType.MethodByName(convention.MethodName); found {
		if s.validateMethodSignature(method, t, convention) {
			return true
		}
	}

	return false
}

// validateMethodSignature checks if a method matches the expected convention signature.
func (*Sentinel) validateMethodSignature(method reflect.Method, receiverType reflect.Type, convention Convention) bool {
	mt := method.Type

	// Check parameters (skip receiver at index 0)
	expectedParams := len(convention.Params)
	actualParams := mt.NumIn() - 1 // -1 for receiver

	if actualParams != expectedParams {
		return false
	}

	// Validate each parameter
	for i, expectedType := range convention.Params {
		actualType := mt.In(i + 1).String() // +1 to skip receiver
		if actualType != expectedType {
			return false
		}
	}

	// Check returns
	expectedReturns := len(convention.Returns)
	actualReturns := mt.NumOut()

	if actualReturns != expectedReturns {
		return false
	}

	// Validate each return type
	receiverTypeString := receiverType.String()
	for i, expectedType := range convention.Returns {
		actualType := mt.Out(i).String()

		if expectedType == "@self" {
			// Special case: @self must match receiver type
			if actualType != receiverTypeString {
				return false
			}
		} else if actualType != expectedType {
			return false
		}
	}

	return true
}

// relationshipExtractor discovers relationships to other types within the same package domain.
func (s *Sentinel) relationshipExtractor(_ context.Context, ec *ExtractionContext) (*ExtractionContext, error) {
	var relationships []TypeRelationship
	t := ec.Type

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return ec, nil
	}

	// Get the root package for domain filtering
	rootPackage := t.PkgPath()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		// Check if field type is a struct or related type
		rel := s.extractRelationship(field, rootPackage)
		if rel != nil {
			rel.From = t.Name()
			relationships = append(relationships, *rel)
		}
	}

	ec.Metadata.Relationships = relationships
	return ec, nil
}

// extractRelationship checks if a field represents a relationship to another struct type.
func (s *Sentinel) extractRelationship(field reflect.StructField, rootPackage string) *TypeRelationship {
	ft := field.Type

	// Handle different field types
	switch ft.Kind() {
	case reflect.Struct:
		// Direct struct embedding
		if field.Anonymous {
			return s.createRelationshipIfInDomain(field, ft, RelationshipEmbedding, rootPackage)
		}
		// Regular struct field
		return s.createRelationshipIfInDomain(field, ft, RelationshipReference, rootPackage)

	case reflect.Ptr:
		// Pointer to struct
		elem := ft.Elem()
		if elem.Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, elem, RelationshipReference, rootPackage)
		}

	case reflect.Slice, reflect.Array:
		// Slice/array of structs
		elem := ft.Elem()
		// Handle []T and []*T
		if elem.Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, elem, RelationshipCollection, rootPackage)
		} else if elem.Kind() == reflect.Ptr && elem.Elem().Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, elem.Elem(), RelationshipCollection, rootPackage)
		}

	case reflect.Map:
		// Map with struct values
		val := ft.Elem()
		// Handle map[K]V and map[K]*V where V is struct
		if val.Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, val, RelationshipMap, rootPackage)
		} else if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct {
			return s.createRelationshipIfInDomain(field, val.Elem(), RelationshipMap, rootPackage)
		}
	}

	return nil
}

// createRelationshipIfInDomain creates a TypeRelationship if the target type is in the same package domain.
func (s *Sentinel) createRelationshipIfInDomain(field reflect.StructField, targetType reflect.Type, kind string, rootPackage string) *TypeRelationship {
	targetPkg := targetType.PkgPath()

	// Skip types without package (built-in types)
	if targetPkg == "" {
		return nil
	}

	// Check if in same package domain
	if !s.isInPackageDomain(targetPkg, rootPackage) {
		return nil
	}

	return &TypeRelationship{
		To:        targetType.Name(),
		Field:     field.Name,
		Kind:      kind,
		ToPackage: targetPkg,
	}
}

// isInPackageDomain checks if a target package is within the same domain as the source.
func (*Sentinel) isInPackageDomain(targetPkg, sourcePkg string) bool {
	// For now, only include exact same package
	// TODO: Consider module prefix matching for broader domain inclusion
	return targetPkg == sourcePkg
}
