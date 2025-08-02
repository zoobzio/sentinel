package sentinel

import (
	"context"
	"fmt"
	"reflect"

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

	// Detect conventions (TODO: implement later)
	// pipeline.Register(pipz.Apply("detect-conventions", s.conventionDetector))

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
func (s *Sentinel) policyProcessor(_ context.Context, ec *ExtractionContext) (*ExtractionContext, error) {
	result := s.applyPolicies(ec)

	// Emit policy events if policies were applied
	if len(result.Applied) > 0 {
		for _, policyName := range result.Applied {
			s.logger.Emit(POLICY_APPLIED, "Policy applied", PolicyEvent{
				TypeName:   ec.Metadata.TypeName,
				PolicyName: policyName,
				Warnings:   result.Warnings,
				Errors:     result.Errors,
			})
		}
	}

	// If there are errors, emit violation events and fail the extraction
	if len(result.Errors) > 0 {
		s.logger.Emit(POLICY_VIOLATION, "Policy validation failed", ValidationEvent{
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
