package catalog


// EnrichmentContract provides type-safe metadata enrichment
type EnrichmentContract[ModelType, EnricherType comparable, MetadataType any] interface {
	Register(enricher EnricherType, handler func(ModelType) MetadataType)
	Enrich(enricher EnricherType, model ModelType) (MetadataType, bool)
	GetEnrichers() []EnricherType
	HasEnricher(enricher EnricherType) bool
}

// enrichmentAdapter wraps service contract for enrichment
type enrichmentAdapter[ModelType, EnricherType comparable, MetadataType any] struct {
	contract *ServiceContract[EnricherType, ModelType, MetadataType]
}

func (a *enrichmentAdapter[M, E, D]) Register(enricher E, handler func(M) D) {
	processor := func(input M) (D, error) {
		return handler(input), nil
	}
	a.contract.Register(enricher, processor)
}

func (a *enrichmentAdapter[M, E, D]) Enrich(enricher E, model M) (D, bool) {
	result, err := a.contract.Process(enricher, model)
	return result, err == nil
}

func (a *enrichmentAdapter[M, E, D]) GetEnrichers() []E {
	return a.contract.ListKeys()
}

func (a *enrichmentAdapter[M, E, D]) HasEnricher(enricher E) bool {
	return a.contract.HasProcessor(enricher)
}

// GetEnrichmentContract returns a type-safe enrichment contract
func GetEnrichmentContract[ModelType, EnricherType comparable, MetadataType any]() EnrichmentContract[ModelType, EnricherType, MetadataType] {
	return &enrichmentAdapter[ModelType, EnricherType, MetadataType]{
		contract: GetContract[EnricherType, ModelType, MetadataType](),
	}
}

// RegisterEnricher registers a type-safe metadata enricher
func RegisterEnricher[ModelType, EnricherType comparable, MetadataType any](
	enricher EnricherType,
	handler func(ModelType) MetadataType,
) {
	contract := GetEnrichmentContract[ModelType, EnricherType, MetadataType]()
	contract.Register(enricher, handler)
}

// EnrichType enriches a model with metadata from a specific enricher
func EnrichType[ModelType, EnricherType comparable, MetadataType any](
	model ModelType,
	enricher EnricherType,
) (MetadataType, bool) {
	contract := GetEnrichmentContract[ModelType, EnricherType, MetadataType]()
	return contract.Enrich(enricher, model)
}

// MustEnrichType enriches a model with metadata, panicking if enricher not found
func MustEnrichType[ModelType, EnricherType comparable, MetadataType any](
	model ModelType,
	enricher EnricherType,
) MetadataType {
	metadata, exists := EnrichType[ModelType, EnricherType, MetadataType](model, enricher)
	if !exists {
		panic("enricher not found for type")
	}
	return metadata
}

// GetRegisteredEnrichers returns all enrichers for a model type
func GetRegisteredEnrichers[ModelType, EnricherType comparable, MetadataType any]() []EnricherType {
	contract := GetEnrichmentContract[ModelType, EnricherType, MetadataType]()
	return contract.GetEnrichers()
}

// HasEnricher checks if an enricher is registered for a model type
func HasEnricher[ModelType, EnricherType comparable, MetadataType any](enricher EnricherType) bool {
	contract := GetEnrichmentContract[ModelType, EnricherType, MetadataType]()
	return contract.HasEnricher(enricher)
}