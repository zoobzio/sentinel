# sentinel

[![CI Status](https://github.com/zoobzio/sentinel/workflows/CI/badge.svg)](https://github.com/zoobzio/sentinel/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zoobzio/sentinel/graph/badge.svg?branch=main)](https://codecov.io/gh/zoobzio/sentinel)
[![Go Report Card](https://goreportcard.com/badge/github.com/zoobzio/sentinel)](https://goreportcard.com/report/github.com/zoobzio/sentinel)
[![CodeQL](https://github.com/zoobzio/sentinel/workflows/CodeQL/badge.svg)](https://github.com/zoobzio/sentinel/security/code-scanning)
[![Go Reference](https://pkg.go.dev/badge/github.com/zoobzio/sentinel.svg)](https://pkg.go.dev/github.com/zoobzio/sentinel)
[![License](https://img.shields.io/github/license/zoobzio/sentinel)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zoobzio/sentinel)](go.mod)
[![Release](https://img.shields.io/github/v/release/zoobzio/sentinel)](https://github.com/zoobzio/sentinel/releases)

Zero-dependency struct introspection for Go.

Extract struct metadata once, cache it permanently, and discover relationships between types.

## The Type Graph

```go
type User struct {
    ID      string   `json:"id" db:"id" validate:"required"`
    Email   string   `json:"email" validate:"required,email"`
    Profile *Profile
    Orders  []Order
}

metadata := sentinel.Scan[User]()
// metadata.TypeName → "User"
// metadata.FQDN     → "github.com/app/models.User"
// metadata.Fields   → []FieldMetadata (4 fields)
// metadata.Relationships → []TypeRelationship (2 relationships)

field := metadata.Fields[0]
// field.Name  → "ID"
// field.Type  → "string"
// field.Kind  → "scalar"
// field.Tags  → {"json": "id", "db": "id", "validate": "required"}
// field.Index → []int{0}
```

One call extracts metadata for `User` and every type it touches — `Profile`, `Order`, and anything they reference. All cached permanently.

```go
types := sentinel.Browse()
// [
//   "github.com/app/models.User",
//   "github.com/app/models.Profile",
//   "github.com/app/models.Order",
// ]

relationships := sentinel.GetRelationships[User]()
// []TypeRelationship (2 relationships)

rel := relationships[0]
// rel.From      → "User"
// rel.To        → "Profile"
// rel.Field     → "Profile"
// rel.Kind      → "reference"
// rel.ToPackage → "github.com/app/models"
```

Types don't change at runtime. Neither does their metadata.

## Install

```bash
go get github.com/zoobzio/sentinel@latest
```

Requires Go 1.24+.

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/zoobzio/sentinel"
)

type Order struct {
    ID     string  `json:"id" db:"order_id" validate:"required"`
    Total  float64 `json:"total" validate:"gte=0"`
    Status string  `json:"status"`
}

type User struct {
    ID     string  `json:"id" db:"user_id"`
    Email  string  `json:"email" validate:"required,email"`
    Orders []Order
}

func main() {
    // Scan extracts User, Order, and their relationship
    metadata := sentinel.Scan[User]()

    // Type information
    fmt.Println(metadata.TypeName) // "User"
    fmt.Println(metadata.FQDN)     // "main.User" (reflects actual package path)

    // Field metadata
    for _, field := range metadata.Fields {
        fmt.Printf("%s (%s): %v\n", field.Name, field.Kind, field.Tags)
    }
    // ID (scalar): map[json:id db:user_id]
    // Email (scalar): map[json:email validate:required,email]
    // Orders (slice): map[]

    // Relationships
    for _, rel := range metadata.Relationships {
        fmt.Printf("%s → %s (%s)\n", metadata.TypeName, rel.To, rel.Kind)
    }
    // User → Order (collection)

    // Everything is cached
    fmt.Println(sentinel.Browse())
    // [github.com/app/models.User github.com/app/models.Order]

    // Export the full schema
    schema := sentinel.Schema()
    fmt.Printf("%d types cached\n", len(schema))
}
```

## Capabilities

| Feature                | Description                                     | Docs                                           |
| ---------------------- | ----------------------------------------------- | ---------------------------------------------- |
| Metadata Extraction    | Fields, types, indices, categories, struct tags | [Concepts](docs/2.learn/2.concepts.md)         |
| Relationship Discovery | References, collections, embeddings, maps       | [Scanning](docs/3.guides/1.scanning.md)        |
| Permanent Caching      | Extract once, cached forever                    | [Architecture](docs/2.learn/3.architecture.md) |
| Custom Tags            | Register additional struct tags                 | [Tags](docs/3.guides/2.tags.md)                |
| Module-Aware Scanning  | Recursive extraction within module boundaries   | [Scanning](docs/3.guides/1.scanning.md)        |
| Schema Export          | `Schema()` returns all cached metadata          | [API](docs/5.reference/1.api.md)               |

## Why sentinel?

- **Zero dependencies** — only the Go standard library
- **Permanent caching** — types are immutable at runtime, so metadata is cached once
- **Type-safe generics** — `Inspect[T]()` catches type errors at compile time
- **Relationship discovery** — automatically maps references, collections, embeddings, and maps
- **Module-aware scanning** — `Scan[T]()` recursively extracts related types within your module
- **Thread-safe** — concurrent access after initial extraction

## Type-Driven Generation

Sentinel metadata enables a pattern: **define types once, generate everything else**.

Your struct definitions become the single source of truth. Downstream tools consume sentinel's metadata to generate:

- **Entity diagrams** — Visualize domain models directly from type relationships
- **Database schemas** — Generate DDL and type-safe queries from struct tags
- **API documentation** — Produce OpenAPI specs from request/response types

The [zoobzio ecosystem](https://github.com/zoobzio) implements this pattern:

- **[erd](https://github.com/zoobzio/erd)** — ERD generation from sentinel schemas. See [ERD Diagrams](docs/4.cookbook/1.erd-diagrams.md).
- **[soy](https://github.com/zoobzio/soy)** — Type-safe query building. See [Database Schemas](docs/4.cookbook/2.database-schemas.md).
- **[rocco](https://github.com/zoobzio/rocco)** — OpenAPI generation. See [API Documentation](docs/4.cookbook/3.api-documentation.md).

## Documentation

- [Overview](docs/1.overview.md) — design philosophy and architecture
- **Learn**
  - [Quickstart](docs/2.learn/1.quickstart.md) — get started in 5 minutes
  - [Concepts](docs/2.learn/2.concepts.md) — metadata, relationships, caching
  - [Architecture](docs/2.learn/3.architecture.md) — internal design and components
- **Guides**
  - [Scanning](docs/3.guides/1.scanning.md) — Inspect vs Scan, module boundaries
  - [Tags](docs/3.guides/2.tags.md) — custom tag registration
  - [Testing](docs/3.guides/3.testing.md) — testing with sentinel
- **Cookbook**
  - [ERD Diagrams](docs/4.cookbook/1.erd-diagrams.md) — visualize domain models
  - [Database Schemas](docs/4.cookbook/2.database-schemas.md) — structurally safe queries
  - [API Documentation](docs/4.cookbook/3.api-documentation.md) — automatic OpenAPI generation
- **Reference**
  - [API](docs/5.reference/1.api.md) — complete function documentation
  - [Types](docs/5.reference/2.types.md) — Metadata, FieldMetadata, TypeRelationship

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License — see [LICENSE](LICENSE) for details.
