# Testing

Testing strategy and utilities for sentinel.

## Structure

```
testing/
├── helpers.go          # Shared test utilities
├── helpers_test.go     # Tests for the helpers themselves
├── integration/        # End-to-end tests
│   └── scan_test.go    # Full scanning and relationship tests
└── benchmarks/         # Performance tests
    └── core_test.go    # Core operation benchmarks
```

## Test Categories

### Unit Tests

Colocated with source files in the root package. Run with:

```bash
make test-unit
```

Each source file has a corresponding `_test.go`:

- `api_test.go` — public API functions
- `cache_test.go` — cache implementations
- `extraction_test.go` — metadata extraction
- `metadata_test.go` — type definitions and helpers
- `relationship_test.go` — relationship discovery

### Integration Tests

Located in `testing/integration/`. These test the full extraction pipeline with realistic type hierarchies. Run with:

```bash
make test-integration
```

### Benchmarks

Located in `testing/benchmarks/`. Run with:

```bash
make test-bench
```

## Test Helpers

The `testing` package provides domain-specific assertions:

```go
import sentineltest "github.com/zoobzio/sentinel/testing"

func TestExample(t *testing.T) {
    metadata := sentinel.Inspect[MyType]()

    sentineltest.AssertMetadataValid(t, metadata)
    field := sentineltest.AssertFieldExists(t, metadata, "ID")
    sentineltest.AssertTagValue(t, field, "json", "id")
    sentineltest.AssertCached(t, metadata.FQDN)
}
```

Available helpers:

| Function | Purpose |
|----------|---------|
| `AssertMetadataValid` | Verify required metadata fields |
| `AssertFieldExists` | Find and return a field by name |
| `AssertRelationshipExists` | Find and return a relationship by field |
| `AssertTagValue` | Verify a tag value on a field |
| `AssertCached` | Verify a type is in the cache |
| `AssertNotCached` | Verify a type is not cached |
| `ResetCache` | Clear cache for test isolation |

## Running Tests

```bash
make test              # All tests with race detector
make test-unit         # Unit tests only (short mode)
make test-integration  # Integration tests
make test-bench        # Benchmarks
make coverage          # Generate coverage report
make check             # Tests + lint
```

## Writing Tests

1. **Use helpers** — prefer `sentineltest.Assert*` over raw assertions
2. **Call `t.Helper()`** — in any test helper functions
3. **Use FQDNs** — when checking cache via `Lookup()`, use the FQDN from metadata
4. **Test concurrency** — sentinel is thread-safe; include concurrent access tests where relevant
