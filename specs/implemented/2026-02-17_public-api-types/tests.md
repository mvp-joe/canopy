# Tests

## Unit Tests

### Compilation Verification (existing tests)

All 519+ existing tests serve as compilation and behavioral verification. Since type aliases are the same type at compile time, every existing test that compiles and passes confirms the refactor is correct. No test code changes are needed.

- `go test ./...` must pass with zero modifications to test files
- `go build ./cmd/canopy/...` must compile successfully

### Type Alias Identity

- Verify `canopy.Symbol` and `store.Symbol` are the same type (assignable without conversion)
- Verify `canopy.Store` and `store.Store` are the same type
- Verify all 12 aliases satisfy direct assignment from their `store` counterparts

These are implicitly tested by the existing test suite: test helpers like `insertSymbol(t, s *store.Store, ...)` return `int64` IDs that are passed to `QueryBuilder` methods returning `*Symbol` (alias). If the alias were a distinct type, compilation would fail.

## Integration Tests

### External Consumer Simulation

**Given** an external Go package that imports only `github.com/jward/canopy` (not `internal/store`)
**When** it calls `engine.Store()`, `engine.Query().SymbolAt(...)`, `engine.Query().Files(...)`, etc.
**Then** all returned types (`*canopy.Symbol`, `*canopy.Store`, `canopy.PagedResult[canopy.File]`, etc.) are usable without importing `internal/store`

This is verified by compilation rather than a runtime test. The existing `cmd/canopy/` package partially validates this (it uses both `canopy.*` and `store.*` types interchangeably), but a true external-consumer test would require a separate Go module. Given that type aliases are a language-level guarantee, the compilation of the main module is sufficient.

## Error Scenarios

None. Type aliases introduce no new error conditions. They are a compile-time-only construct with no runtime representation.
