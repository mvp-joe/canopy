# Implementation

Single phase. This is a mechanical refactor with no behavioral changes.

## Phase 1: Add Type Aliases and Update Signatures

- [x] Create `types.go` in the `canopy` package with 12 type aliases (Store, Symbol, File, Scope, CallEdge, Import, FunctionParam, TypeMember, TypeParam, Annotation, ExtensionBinding, Reexport)
- [x] Update `engine.go`: change `Store()` return type and `NewQueryBuilder()` parameter type to use aliases
- [x] Update `query.go`: change `SymbolAt`, `Callers`, `Callees`, `Dependencies`, `Dependents` return types; update internal `store.Import{}` literals to `Import{}`
- [x] Update `query_detail.go`: change `SymbolDetail` struct field types and `ScopeAt` return type; update internal nil-initialization literals
- [x] Update `query_discovery.go`: change `SymbolResult` embedded type and `Files()` return type; update internal `store.File` usages
- [x] Update `query_hierarchy.go`: change `TypeHierarchy.Extensions` field type, `ExtensionMethods` and `Reexports` return types; update internal literals
- [x] Update `query_graph.go`: change `callGraphData` field types and `resolveCallGraphEdge` parameter type (consistency, though unexported)
- [x] Clean up unused `store` imports from files that no longer reference the package directly
- [x] Run `go build ./...` to verify compilation
- [x] Run `go test ./...` to verify all existing tests pass unchanged
- [x] Update `cmd/canopy/` files if any break — no changes needed (type aliases are interchangeable)

## Notes

- The `cmd/canopy/` package imports `store` directly for `store.NewStore()` and type references. Since `cmd/` is part of this module (not an external consumer), it can continue importing `internal/store` directly. However, functions like `openStore() (*store.Store, error)` and converter functions like `extensionBindingToCLI(b *store.ExtensionBinding, s *store.Store)` will work unchanged because type aliases are fully interchangeable with the original type.
- No test files need modification -- they are in the `canopy` package and can import `internal/store` directly.
