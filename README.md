# sentinel

[![CI Status](https://github.com/zoobzio/sentinel/workflows/CI/badge.svg)](https://github.com/zoobzio/sentinel/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zoobzio/sentinel/graph/badge.svg?branch=main)](https://codecov.io/gh/zoobzio/sentinel)
[![Go Report Card](https://goreportcard.com/badge/github.com/zoobzio/sentinel)](https://goreportcard.com/report/github.com/zoobzio/sentinel)
[![CodeQL](https://github.com/zoobzio/sentinel/workflows/CodeQL/badge.svg)](https://github.com/zoobzio/sentinel/security/code-scanning)
[![Go Reference](https://pkg.go.dev/badge/github.com/zoobzio/sentinel.svg)](https://pkg.go.dev/github.com/zoobzio/sentinel)
[![License](https://img.shields.io/github/license/zoobzio/sentinel)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zoobzio/sentinel)](go.mod)
[![Release](https://img.shields.io/github/v/release/zoobzio/sentinel)](https://github.com/zoobzio/sentinel/releases)

Struct metadata extraction and relationship discovery for Go with zero dependencies.

Extract comprehensive metadata from your structs once, cache it permanently, and understand type relationships in your codebase.

## Core Features

Sentinel provides runtime struct introspection with:

- **Comprehensive metadata extraction** from struct fields and tags
- **Type relationship discovery** between structs in your domain
- **Permanent caching** for optimal performance
- **Zero dependencies** - just the Go standard library

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/zoobzio/sentinel"
)

type User struct {
    ID      string   `json:"id" db:"user_id"`
    Name    string   `json:"name" validate:"required"`
    Email   string   `json:"email" validate:"email"`
    Profile *Profile `json:"profile"`
    Orders  []Order  `json:"orders"`
}

type Profile struct {
    Bio    string `json:"bio"`
    Avatar string `json:"avatar_url"`
}

type Order struct {
    ID     string  `json:"id"`
    Total  float64 `json:"total" validate:"gt=0"`
}

func main() {
    // Extract metadata (cached after first call)
    metadata := sentinel.Inspect[User]()

    fmt.Printf("Type: %s\n", metadata.TypeName)
    fmt.Printf("Package: %s\n", metadata.PackageName)
    fmt.Printf("Fields: %d\n", len(metadata.Fields))
    fmt.Printf("Relationships: %d\n", len(metadata.Relationships))

    // Discover relationships
    for _, rel := range metadata.Relationships {
        fmt.Printf("  %s -> %s (%s via field %s)\n",
            rel.From, rel.To, rel.Kind, rel.Field)
    }
}
```

## Metadata Extraction

Sentinel extracts comprehensive metadata from struct tags:

```go
type Product struct {
    ID          string  `json:"id" db:"product_id"`
    Name        string  `json:"name" validate:"required,max=100"`
    Price       float64 `json:"price" validate:"gt=0"`
    Description string  `json:"desc,omitempty" db:"description"`
    Tags        []Tag   `json:"tags"`
}

metadata := sentinel.Inspect[Product]()

// Access field metadata
for _, field := range metadata.Fields {
    fmt.Printf("Field: %s (%s)\n", field.Name, field.Type)

    // Access all tags
    for tag, value := range field.Tags {
        fmt.Printf("  %s: %s\n", tag, value)
    }
}
```

## Relationship Discovery

Sentinel automatically discovers relationships between types in the same package:

```go
// GetRelationships returns all types this type references
relationships := sentinel.GetRelationships[User]()

// GetReferencedBy returns all types that reference this type
referencedBy := sentinel.GetReferencedBy[Profile]()

// Relationship types:
// - "reference": Direct struct field or pointer
// - "collection": Slice or array of structs
// - "embedding": Anonymous embedded struct
// - "map": Map with struct values
```

## Recursive Scanning

Use `Scan[T]()` to automatically inspect a type and all related types within the same module:

```go
// Inspect only inspects a single type
metadata := sentinel.Inspect[User]()  // Only User is cached

// Scan recursively inspects all related types in the same module
metadata := sentinel.Scan[User]()
// Now User, Profile, Order, and all transitively related types are cached

// Check what was cached
types := sentinel.Browse()
fmt.Printf("Cached types: %v\n", types)
// Output: [User Profile Order OrderItem Address ...]
```

**Module boundary detection:**

- Uses first 3 path segments to determine module root
- `github.com/user/myapp/models` and `github.com/user/myapp/api` → same module ✓
- `github.com/user/myapp` and `github.com/lib/pq` → different modules ✗
- External library types are never scanned

## Custom Tag Registration

Register custom struct tags for extraction:

```go
// Register custom tags
sentinel.Tag("custom")
sentinel.Tag("myapp")

type Model struct {
    Field string `custom:"value" myapp:"metadata"`
}

// Custom tags are now extracted
metadata := sentinel.Inspect[Model]()
// metadata.Fields[0].Tags["custom"] == "value"
// metadata.Fields[0].Tags["myapp"] == "metadata"
```

## Performance

Sentinel uses permanent caching - struct metadata is extracted once and cached forever (types don't change at runtime):

```go
// First call: extracts and caches metadata
metadata1 := sentinel.Inspect[User]()  // ~microseconds

// Subsequent calls: returns from cache
metadata2 := sentinel.Inspect[User]()  // ~nanoseconds

// Scan entire module graph once
sentinel.Scan[User]()  // Caches User + all related types in module

// All subsequent lookups are instant
userMeta := sentinel.Inspect[User]()      // ~nanoseconds
profileMeta := sentinel.Inspect[Profile]() // ~nanoseconds
orderMeta := sentinel.Inspect[Order]()     // ~nanoseconds
```

## Why sentinel?

- **Zero configuration**: No setup or registration required
- **Performance focused**: Permanent caching with minimal overhead
- **Type-safe**: Generic API prevents runtime type errors
- **Relationship aware**: Understands connections between your types
- **Zero dependencies**: No external packages required
- **Well tested**: 92%+ test coverage

## Architecture: Global State Design

Sentinel uses global state by design, which is optimal for struct metadata extraction because:

- **Types are immutable after compilation**: Go's type system is fixed at compile time. A struct's fields, their types, and tags cannot change while the program is running.
- **One type system = One sentinel**: Since types don't change during runtime, there's a 1:1 relationship between your application's type system and sentinel's metadata cache.
- **No cleanup needed**: Unlike traditional caches that handle mutable data, sentinel's metadata is intentionally permanent. Once extracted, struct metadata remains valid for the entire program lifetime.
- **Thread-safe by nature**: The immutability of type information means cached metadata can be safely accessed concurrently without synchronization overhead after initial extraction.

This architectural decision eliminates unnecessary complexity around cache invalidation, lifecycle management, and instance passing that would provide no value for immutable type metadata.

## Installation

```bash
go get github.com/zoobzio/sentinel@latest
```

Requires Go 1.23 or later.

## API Reference

### Core Functions

```go
// Extract metadata for a single type (cached permanently)
func Inspect[T any]() ModelMetadata

// Recursively scan a type and all related types in the same module
func Scan[T any]() ModelMetadata

// Register a custom struct tag for extraction
func Tag(tagName string)

// Get all cached type names
func Browse() []string

// Get cached metadata by type name
func Lookup(typeName string) (ModelMetadata, bool)

// Get all cached metadata at once
func Schema() map[string]ModelMetadata
```

### Relationship Functions

```go
// Get all relationships from a type
func GetRelationships[T any]() []TypeRelationship

// Get all types that reference this type
func GetReferencedBy[T any]() []TypeRelationship

// Get relationship graph data
func GetRelationshipGraph() map[string][]TypeRelationship
```

## Examples

### Extract validation rules

```go
type Form struct {
    Email    string `validate:"required,email"`
    Password string `validate:"required,min=8"`
    Age      int    `validate:"min=18,max=120"`
}

metadata := sentinel.Inspect[Form]()
for _, field := range metadata.Fields {
    if rules, ok := field.Tags["validate"]; ok {
        fmt.Printf("%s: %s\n", field.Name, rules)
    }
}
```

### Database schema discovery

```go
type Model struct {
    ID        string `db:"id,primarykey"`
    CreatedAt time.Time `db:"created_at"`
    Name      string `db:"name,index"`
}

metadata := sentinel.Inspect[Model]()
for _, field := range metadata.Fields {
    if dbTag, ok := field.Tags["db"]; ok {
        fmt.Printf("Column: %s -> %s\n", field.Name, dbTag)
    }
}
```

### Generate API documentation

```go
type APIRequest struct {
    UserID string `json:"user_id" example:"usr_123"`
    Action string `json:"action" enum:"create,update,delete"`
    Data   any    `json:"data,omitempty"`
}

metadata := sentinel.Inspect[APIRequest]()
// Use metadata to generate OpenAPI specs, documentation, etc.
```

### Export complete schema

```go
// Option 1: Inspect types individually
sentinel.Inspect[User]()
sentinel.Inspect[Product]()
sentinel.Inspect[Order]()

// Option 2: Scan entire module from root type
sentinel.Scan[User]()  // Automatically caches User + all related types

// Get the complete schema all at once
schema := sentinel.Schema()

// Export as JSON for documentation or code generation
jsonSchema, _ := json.MarshalIndent(schema, "", "  ")
fmt.Println(string(jsonSchema))

// Use for validation, documentation generation, or API contracts
for typeName, metadata := range schema {
    fmt.Printf("Type: %s\n", typeName)
    fmt.Printf("  Fields: %d\n", len(metadata.Fields))
    fmt.Printf("  Relationships: %d\n", len(metadata.Relationships))
}
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
