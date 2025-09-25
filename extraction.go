package sentinel

import (
	"context"
	"reflect"
	"time"

	"github.com/zoobzio/tracez"
)

// extractMetadata performs the complete metadata extraction for a type.
func (s *Sentinel) extractMetadata(ctx context.Context, t reflect.Type, _ interface{}) ModelMetadata {
	start := time.Now()

	// Start extraction span if tracer configured
	var span *tracez.ActiveSpan
	if s.tracer != nil {
		ctx, span = s.tracer.StartSpan(ctx, ExtractMetadataSpan)
		defer span.Finish()
		span.SetTag("type", getTypeName(t))
		span.SetTag("package", t.PkgPath())
	}

	// Initialize metadata with basic reflection
	metadata := ModelMetadata{
		TypeName:    getTypeName(t),
		PackageName: t.PkgPath(),
	}

	// Extract fields with span
	if s.tracer != nil {
		_, fieldSpan := s.tracer.StartSpan(ctx, ExtractFieldsSpan)
		metadata.Fields = s.extractFieldMetadata(t)
		fieldSpan.Finish()
	} else {
		metadata.Fields = s.extractFieldMetadata(t)
	}

	// Extract relationships with span
	if s.tracer != nil {
		_, relSpan := s.tracer.StartSpan(ctx, ExtractRelationshipsSpan)
		metadata.Relationships = s.extractRelationships(t)
		relSpan.Finish()
	} else {
		metadata.Relationships = s.extractRelationships(t)
	}

	duration := time.Since(start)

	// Record extraction metrics
	if s.metrics != nil {
		s.metrics.Timer(ExtractionDurationMs).Record(duration)
		s.metrics.Histogram(ExtractionFieldsCount, []float64{0, 5, 10, 20, 50, 100}).Observe(float64(len(metadata.Fields)))
		s.metrics.Histogram(ExtractionRelationshipsCount, []float64{0, 1, 3, 5, 10, 20}).Observe(float64(len(metadata.Relationships)))

		// Count unique tags
		tagSet := make(map[string]bool)
		for _, field := range metadata.Fields {
			for tag := range field.Tags {
				tagSet[tag] = true
			}
		}
		s.metrics.Histogram(ExtractionTagsCount, []float64{0, 3, 5, 10, 20}).Observe(float64(len(tagSet)))
	}

	// Emit extraction event
	if s.extractionHooks != nil {
		// Intentionally ignoring error: hook emission failures (queue full,
		// service closed) should not fail metadata extraction
		_ = s.extractionHooks.Emit(ctx, "extraction.complete", ExtractionEvent{ //nolint:errcheck
			TypeName:      metadata.TypeName,
			PackageName:   metadata.PackageName,
			FieldCount:    len(metadata.Fields),
			RelationCount: len(metadata.Relationships),
			Relationships: metadata.Relationships,
			Duration:      duration,
			FromCache:     false,
		})
	}

	return metadata
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
