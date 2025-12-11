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

## Why Sentinel?

- **Zero dependencies** — only the Go standard library
- **Permanent caching** — types don't change at runtime, so metadata is cached forever
- **Type-safe generics** — `Inspect[T]()` catches errors at compile time
- **Relationship discovery** — understands how your types connect
- **Module-aware scanning** — recursively extracts related types within your module

## Install

```bash
go get github.com/zoobzio/sentinel@latest
```

Requires Go 1.23+.

## Quick Start

```go
type User struct {
    ID      string   `json:"id" db:"user_id"`
    Name    string   `json:"name" validate:"required"`
    Profile *Profile `json:"profile"`
    Orders  []Order  `json:"orders"`
}

// Extract metadata (cached permanently)
metadata := sentinel.Inspect[User]()

// Or scan entire type graph
metadata := sentinel.Scan[User]()  // Caches User + Profile + Order + ...

// Access field metadata
for _, field := range metadata.Fields {
    fmt.Printf("%s: %v\n", field.Name, field.Tags)
}

// Discover relationships
for _, rel := range metadata.Relationships {
    fmt.Printf("%s -> %s (%s)\n", rel.From, rel.To, rel.Kind)
}
```

## Examples

**Extract validation rules:**

```go
metadata := sentinel.Inspect[Form]()
for _, field := range metadata.Fields {
    if rules, ok := field.Tags["validate"]; ok {
        fmt.Printf("%s: %s\n", field.Name, rules)
    }
}
```

**Database column mapping:**

```go
metadata := sentinel.Inspect[Model]()
for _, field := range metadata.Fields {
    if col, ok := field.Tags["db"]; ok {
        fmt.Printf("%s -> %s\n", field.Name, col)
    }
}
```

**Export schema for code generation:**

```go
sentinel.Scan[User]()  // Cache all related types
schema := sentinel.Schema()
jsonSchema, _ := json.MarshalIndent(schema, "", "  ")
```

## Core API

```go
sentinel.Inspect[T]()           // Extract single type
sentinel.Scan[T]()              // Extract type + all related types in module
sentinel.TryInspect[T]()        // Inspect, but returns error instead of panic
sentinel.TryScan[T]()           // Scan, but returns error instead of panic
sentinel.Tag(name)              // Register custom tag
sentinel.Browse()               // List cached type names
sentinel.Lookup(name)           // Get cached metadata by name
sentinel.Schema()               // Export all cached metadata
sentinel.Reset()                // Clear cache (for testing)

sentinel.GetRelationships[T]()  // Types that T references
sentinel.GetReferencedBy[T]()   // Types that reference T
```

## Design

Sentinel uses global state by design. Go's type system is fixed at compile time—a struct's fields and tags cannot change while the program runs. This means:

- One type system = one metadata cache
- No cache invalidation needed
- Thread-safe access after initial extraction

Go does not permit methods with type parameters, so functions like `Inspect[T]()` must be package-level. Instance-based APIs are not possible for this pattern.

See [Design Philosophy](docs/1.overview.md#design-philosophy) for details.

## Documentation

- [Overview](docs/1.overview.md) — what sentinel does and why
- **Learn**
  - [Quickstart](docs/2.learn/1.quickstart.md) — get started in 5 minutes
  - [Concepts](docs/2.learn/2.concepts.md) — metadata, relationships, caching
- **Guides**
  - [Scanning](docs/3.guides/1.scanning.md) — Inspect vs Scan, module boundaries
  - [Tags](docs/3.guides/2.tags.md) — custom tag registration
  - [Testing](docs/3.guides/3.testing.md) — testing with sentinel
- **Reference**
  - [API](docs/4.reference/1.api.md) — complete function documentation
  - [Types](docs/4.reference/2.types.md) — ModelMetadata, FieldMetadata, TypeRelationship

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License — see [LICENSE](LICENSE) for details.
