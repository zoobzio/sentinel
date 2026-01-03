# Integration Tests

End-to-end tests for sentinel's scanning and relationship discovery.

## Purpose

Integration tests verify the full extraction pipeline with realistic type hierarchies, testing:

- Recursive type discovery via `Scan`
- Relationship detection across multiple types
- Cache population and lookup
- Concurrent access safety

## Running

```bash
make test-integration
```

Or directly:

```bash
go test -v -race ./testing/integration/...
```

## Test Types

The integration tests define a realistic domain model:

```
User
├── Profile (reference)
│   └── Address (reference)
├── Orders (collection)
│   └── OrderItem (collection)
└── Settings (embedding)
    └── Data (map)
```

This hierarchy exercises all relationship kinds:

| Kind | Example |
|------|---------|
| `reference` | `User.Profile` → `*Profile` |
| `collection` | `User.Orders` → `[]Order` |
| `embedding` | `User.Settings` (embedded) |
| `map` | `Settings.Metadata` → `map[string]Data` |

## Test Coverage

| Test | What It Verifies |
|------|------------------|
| `TestScanRecursiveDiscovery` | All transitive types are cached |
| `TestScanVsInspect` | `Inspect` vs `Scan` caching behaviour |
| `TestRelationshipKinds` | Correct relationship kind detection |
| `TestBrowseAfterScan` | Cache listing works after scan |
| `TestGetReferencedBy` | Reverse relationship lookup |
| `TestSchemaExport` | Full schema export |
| `TestDeeplyNestedTypes` | 5-level deep type traversal |
| `TestConcurrentScanning` | Thread safety under load |
| `TestPointerVariations` | All pointer/slice/map combinations |
| `TestFieldMetadataAccuracy` | Tag extraction accuracy |

## Adding Tests

When adding integration tests:

1. Define types in the test file (not imported from elsewhere)
2. Test the full pipeline: `Scan` → `Browse`/`Lookup` → verify relationships
3. Use FQDNs from metadata for cache lookups
4. Include concurrent access tests for new features
