# 📋 Sentinel: Universal Model Metadata System

**Sentinel** is a standalone metadata extraction and caching system that enables powerful model-driven features through comprehensive struct tag analysis.

## 🎯 Core Concept

**Reflect Once, Use Everywhere**: Define your model with comprehensive struct tags, and every service in the system gets instant access to rich metadata through lazy extraction with no registration required.

## 🏗️ Architecture

```go
// User defines model with comprehensive tags
type User struct {
    Name   string `json:"name" validate:"required" desc:"Full name"`
    SSN    string `json:"ssn" scope:"admin" encrypt:"pii" redact:"XXX-XX-XXXX"`
    Salary int    `json:"salary" scope:"hr" encrypt:"financial"`
}

// Zero registration - just use clean generic API
metadata := sentinel.Inspect[User]()        // Comprehensive metadata
fields := sentinel.GetFields[User]()       // Just field info
scopes := sentinel.GetScopes[User]()       // Security scopes
container := sentinel.Wrap(userInstance)   // Transparent container
```

## 🚀 Supported Struct Tags

Sentinel recognizes and extracts comprehensive metadata from these struct tags:

### **Core Tags**
- `json:"field_name"` - JSON serialization name
- `db:"column_name"` - Database column mapping  
- `desc:"description"` - Human-readable field description
- `example:"value"` - Example value for documentation

### **Security & Access Control**
- `scope:"admin,hr"` - Required permissions for field access
- `encrypt:"pii"` - Encryption classification (`pii`, `financial`, `medical`, `homomorphic`)
- `redact:"XXX-XX-XXXX"` - Custom redaction value for unauthorized access

### **Validation**
- `validate:"required,email"` - Validation rules (supports go-playground/validator syntax)
- Custom validation tags are automatically detected

### **Advanced Options**
- `encrypt_algo:"AES-256"` - Specific encryption algorithm
- `data_residency:"us-west,eu-central"` - Geographic data requirements

## 📦 Container Pattern (Transparent)

Sentinel automatically wraps user models in containers with standard system fields:

```go
type Container[T any] struct {
    ID        string    `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Version   int       `json:"version"`
    Data      T         `json:"data"` // User's actual model
}
```

**Users never see containers** - they work directly with their models, but the system gets automatic timestamps, versioning, and ID management.

## 🎨 Convention Detection

Sentinel automatically detects when models implement framework conventions:

```go
// ScopeProvider convention
func (u User) GetRequiredScopes() []string {
    return []string{"user_data"}
}

// Automatically detected and cached in metadata
metadata.Functions // Contains: GetRequiredScopes -> ScopeProvider
```

## 🔥 Power Features Enabled

### **Auto-Generated OpenAPI Documentation**
```go
// HTTP package automatically generates comprehensive OpenAPI specs
openapi := GenerateOpenAPI[User]()
// Includes: validation rules, security requirements, examples, encryption docs
```

### **Self-Documenting Database Schemas**
```go
// Database package gets instant schema information
schema := GetDatabaseSchema[User]()
// Includes: column types, constraints, indexes, encryption requirements
```

### **Field-Level Encryption Control**
```go
// Encryption package gets field-by-field requirements
encryptionPlan := GetEncryptionPlan[User]()
// Supports: PII encryption, financial data, homomorphic computation
```

### **Intelligent Scoping & Redaction**
```go
// Cereal package gets comprehensive field access rules
scopeRules := GetScopeRules[User]()
redactionValues := GetRedactionRules[User]()
```

## 📊 Usage Examples

### Basic Usage (Zero Registration)
```go
type Product struct {
    Name     string  `json:"name" validate:"required" desc:"Product name"`
    Price    float64 `json:"price" validate:"gt=0" encrypt:"financial"`
    Category string  `json:"category" scope:"admin"`
}

// No registration needed - just use the generic API
metadata := sentinel.Inspect[Product]()           // Full metadata
fields := sentinel.GetFields[Product]()          // Field details
encryptedFields := sentinel.GetEncryptionFields[Product]() // Security info
```

### Specialized Accessors
```go
// Get only what you need
scopes := sentinel.GetScopes[User]()             // ["profile", "admin", "hr"]  
redactionRules := sentinel.GetRedactionRules[User]() // {"SSN": "XXX-XX-XXXX"}
validatedFields := sentinel.GetValidationFields[User]() // Fields with rules
hasScope := sentinel.HasConvention[User]("ScopeProvider") // true/false
```

### Consuming Metadata in Services
```go
// Cereal package using clean generic API
scopes := sentinel.GetScopes[User]()
for _, scope := range scopes {
    // Apply scope-based security
    checkUserPermission(scope)
}

// Validation package gets validation rules
validatedFields := sentinel.GetValidationFields[User]()
for _, field := range validatedFields {
    addValidationRule(field)
}

// HTTP package gets examples for OpenAPI
fields := sentinel.GetFields[User]()
for _, field := range fields {
    if field.Example != nil {
        addExampleToSchema(field.Name, field.Example)
    }
}

// Encryption package gets field-level encryption requirements
encryptedFields := sentinel.GetEncryptionFields[User]()
for _, field := range encryptedFields {
    configureEncryption(field.Name, field.Encryption.Type)
}
```

## 🧪 Testing & Development

```bash
# Run all sentinel tests
go test -v

# Test the clean generic API
go test -v -run TestGenericAPI

# View comprehensive metadata JSON output
go test -v -run TestMetadataJSON

# Test lazy metadata extraction
go test -v -run TestLazyMetadataExtraction
```

## 🎯 Benefits

- **Zero Registration**: No manual setup - just use the generic API
- **Performance**: Lazy extraction + permanent caching per type
- **Clean API**: Generic functions, no string-based lookups for users
- **Consistency**: Single source of truth for all model metadata
- **Transparency**: Users work with normal Go structs + tags
- **Extensibility**: Easy to add new tag types and capabilities
- **Self-Documenting**: Comprehensive metadata enables automatic documentation
- **Security-First**: Field-level encryption and access control built-in

## 🚀 Framework Integration

Sentinel can be integrated with various services to provide:

- **Field-level Security**: Scoping and redaction based on permissions
- **API Documentation**: Auto-generated OpenAPI specifications  
- **Database Integration**: Schema generation and encryption requirements
- **Access Control**: Permission-based field access
- **Usage Tracking**: Model usage metrics and monitoring
- **Audit Logging**: Comprehensive change tracking

**The entire application becomes self-aware of its data models.**