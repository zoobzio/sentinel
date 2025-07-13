package catalog

import (
	"reflect"
	"strings"
	"sync"
)

// ModelMetadata contains comprehensive information about a user model
// This defines what struct tags and capabilities the system supports
type ModelMetadata struct {
	TypeName    string            `json:"type_name"`
	PackageName string            `json:"package_name"`
	Fields      []FieldMetadata   `json:"fields"`
	Functions   []FunctionInfo    `json:"functions"`
	Examples    map[string]any    `json:"examples,omitempty"`
	Description string            `json:"description,omitempty"`
	JSONSchema  map[string]any    `json:"json_schema,omitempty"`
}

// FieldMetadata captures all supported field-level metadata
type FieldMetadata struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	JSONName     string            `json:"json_name,omitempty"`
	DBColumn     string            `json:"db_column,omitempty"`
	Description  string            `json:"description,omitempty"`
	Example      any               `json:"example,omitempty"`
	
	// Security & Access Control
	Scopes       []string          `json:"scopes,omitempty"`
	
	// Validation Rules
	Validation   ValidationInfo    `json:"validation,omitempty"`
	
	// Encryption & Data Protection
	Encryption   EncryptionInfo    `json:"encryption,omitempty"`
	
	// Data Handling
	Redaction    RedactionInfo     `json:"redaction,omitempty"`
	
	// Additional tag-based metadata
	Tags         map[string]string `json:"tags,omitempty"`
}

// ValidationInfo captures validation requirements from tags
type ValidationInfo struct {
	Required     bool              `json:"required,omitempty"`
	CustomRules  []string          `json:"custom_rules,omitempty"`
	Constraints  map[string]string `json:"constraints,omitempty"`
}

// EncryptionInfo defines field-level encryption requirements  
type EncryptionInfo struct {
	Type         string   `json:"type,omitempty"`         // "pii", "financial", "medical", "homomorphic"
	Algorithm    string   `json:"algorithm,omitempty"`    // "AES-256", "ChaCha20", "homomorphic"
	KeyRotation  bool     `json:"key_rotation,omitempty"`
	DataResidency []string `json:"data_residency,omitempty"` // "us-west", "eu-central"
}

// RedactionInfo defines how fields should be redacted for unauthorized users
type RedactionInfo struct {
	Strategy string `json:"strategy,omitempty"` // "mask", "zero", "custom"
	Value    string `json:"value,omitempty"`    // Custom redaction value
}

// FunctionInfo captures methods attached to the model
type FunctionInfo struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`       // "method", "receiver"
	Signature  string            `json:"signature"`
	Convention string            `json:"convention,omitempty"` // "ScopeProvider", etc.
	Tags       map[string]string `json:"tags,omitempty"`
}

// Global metadata cache - reflect once, use everywhere
var (
	metadataCache = make(map[string]ModelMetadata)
	typeNameCache = make(map[string]reflect.Type)
	cacheMutex    sync.RWMutex
	
	// Transformer registry - stores type-specific behavior
	transformerRegistry = make(map[string]any)
	transformerMutex    sync.RWMutex
	
	// Tag registry - tracks which tags to extract
	registeredTags = make(map[string]bool)
	tagMutex       sync.RWMutex
	
	// Field manipulator registry - stores field accessors for types
	manipulatorRegistry = make(map[string]any) // stores map[string]*FieldManipulator[T]
	manipulatorMutex    sync.RWMutex
)

// GetModelMetadata retrieves cached metadata by type name
func GetModelMetadata(typeName string) (ModelMetadata, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	metadata, exists := metadataCache[typeName]
	return metadata, exists
}

// ExtractAndCacheMetadata performs comprehensive reflection on a type and caches the result
func ExtractAndCacheMetadata[T any](example T) ModelMetadata {
	t := reflect.TypeOf(example)
	typeName := getTypeName(t)
	
	// Check cache first
	cacheMutex.RLock()
	if cached, exists := metadataCache[typeName]; exists {
		cacheMutex.RUnlock()
		return cached
	}
	cacheMutex.RUnlock()
	
	// Extract comprehensive metadata
	metadata := extractMetadata(t, example)
	
	// Cache the result
	cacheMutex.Lock()
	metadataCache[typeName] = metadata
	cacheMutex.Unlock()
	
	return metadata
}

// extractMetadata performs the actual reflection and metadata extraction
func extractMetadata(t reflect.Type, example any) ModelMetadata {
	metadata := ModelMetadata{
		TypeName:    getTypeName(t),
		PackageName: t.PkgPath(),
		Fields:      extractFieldMetadata(t),
		Functions:   extractFunctionMetadata(t, example),
		Examples:    make(map[string]any),
	}
	
	return metadata
}

// extractFieldMetadata extracts comprehensive field information using pipz contracts
func extractFieldMetadata(t reflect.Type) []FieldMetadata {
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
		
		// Base field metadata
		fieldMeta := FieldMetadata{
			Name:     field.Name,
			Type:     field.Type.String(),
			JSONName: extractJSONName(field),
			DBColumn: field.Tag.Get("db"),
			Description: field.Tag.Get("desc"),
			Tags:       extractAllTags(field),
		}
		
		// Extract example if provided
		if example := field.Tag.Get("example"); example != "" {
			fieldMeta.Example = example
		}
		
		// Basic tag extraction (packages will enrich with domain-specific metadata)
		fieldMeta.Scopes = extractScopes(field)
		fieldMeta.Validation = extractValidationInfo(field)
		fieldMeta.Encryption = extractEncryptionInfo(field)
		fieldMeta.Redaction = extractRedactionInfo(field)
		
		fields = append(fields, fieldMeta)
	}
	
	return fields
}

// extractFunctionMetadata discovers methods and conventions implemented by the type
func extractFunctionMetadata(t reflect.Type, example any) []FunctionInfo {
	var functions []FunctionInfo
	
	// Check for convention interfaces
	if scopeProvider, ok := example.(interface{ GetRequiredScopes() []string }); ok {
		_ = scopeProvider // Use the interface
		functions = append(functions, FunctionInfo{
			Name:       "GetRequiredScopes",
			Type:       "method",
			Convention: "ScopeProvider",
			Signature:  "() []string",
		})
	}
	
	// TODO: Add detection for other conventions as they're implemented
	// - ValidationProvider, AuditProvider, etc.
	
	return functions
}

// Helper functions for tag extraction

func getTypeName(t reflect.Type) string {
	if t == nil {
		return "nil"
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return "nil"
	}
	return t.Name()
}

func extractJSONName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return ""
	}
	return strings.Split(jsonTag, ",")[0]
}

func extractScopes(field reflect.StructField) []string {
	scopeTag := field.Tag.Get("scope")
	if scopeTag == "" {
		return nil
	}
	return strings.Split(scopeTag, ",")
}

func extractValidationInfo(field reflect.StructField) ValidationInfo {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return ValidationInfo{}
	}
	
	info := ValidationInfo{
		Constraints: make(map[string]string),
	}
	
	parts := strings.Split(validateTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "required" {
			info.Required = true
		} else if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			info.Constraints[kv[0]] = kv[1]
		} else {
			// Custom validation rule
			info.CustomRules = append(info.CustomRules, part)
		}
	}
	
	return info
}

func extractEncryptionInfo(field reflect.StructField) EncryptionInfo {
	encryptTag := field.Tag.Get("encrypt")
	if encryptTag == "" {
		return EncryptionInfo{}
	}
	
	info := EncryptionInfo{
		Type: encryptTag,
	}
	
	// Extract additional encryption metadata
	if algo := field.Tag.Get("encrypt_algo"); algo != "" {
		info.Algorithm = algo
	}
	
	if residency := field.Tag.Get("data_residency"); residency != "" {
		info.DataResidency = strings.Split(residency, ",")
	}
	
	return info
}

func extractRedactionInfo(field reflect.StructField) RedactionInfo {
	redactTag := field.Tag.Get("redact")
	if redactTag == "" {
		return RedactionInfo{}
	}
	
	return RedactionInfo{
		Strategy: "custom",
		Value:    redactTag,
	}
}

func extractAllTags(field reflect.StructField) map[string]string {
	tags := make(map[string]string)
	
	// Always extract json tag for serialization
	if value, ok := field.Tag.Lookup("json"); ok {
		tags["json"] = value
	}
	
	// Extract all registered tags
	tagMutex.RLock()
	defer tagMutex.RUnlock()
	
	for tagName := range registeredTags {
		if value, ok := field.Tag.Lookup(tagName); ok {
			tags[tagName] = value
		}
	}
	
	return tags
}