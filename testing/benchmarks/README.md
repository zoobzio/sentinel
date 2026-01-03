# Benchmarks

Performance tests for sentinel core operations.

## Running

```bash
make test-bench
```

Or directly:

```bash
go test -bench=. -benchmem -benchtime=1s ./testing/benchmarks/...
```

## Benchmarks

| Benchmark | What It Measures |
|-----------|------------------|
| `BenchmarkInspectSimple` | Single-field struct extraction |
| `BenchmarkInspectComplex` | 15-field struct with multiple tags |
| `BenchmarkInspectCached` | Cache hit performance |
| `BenchmarkTagRegistration` | `Tag()` registration overhead |
| `BenchmarkConcurrentInspect` | Parallel `Inspect` calls |
| `BenchmarkInspectMemory` | Memory allocations per operation |

## Interpreting Results

```
BenchmarkInspectCached-8    50000000    25 ns/op    0 B/op    0 allocs/op
```

Key metrics:

- **ns/op** — time per operation (lower is better)
- **B/op** — bytes allocated per operation
- **allocs/op** — allocations per operation

For cached operations, expect near-zero allocations since metadata is returned from the permanent cache.

## Test Struct

Benchmarks use `BenchmarkStruct` with 15 fields and multiple tag types:

```go
type BenchmarkStruct struct {
    ID          string    `json:"id" db:"id" validate:"required,uuid"`
    Name        string    `json:"name" validate:"required,min=2,max=100"`
    Email       string    `json:"email" validate:"required,email" encrypt:"pii"`
    // ... 12 more fields
}
```

This represents a realistic production struct with common tag combinations.

## CI Integration

Benchmarks run automatically in CI. Results are uploaded as artifacts for tracking performance over time.
