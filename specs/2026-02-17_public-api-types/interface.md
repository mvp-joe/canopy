# Interface

## New File: `types.go`

A single new file in the `canopy` package containing type aliases for all `internal/store` types that appear in the public API surface.

```go
package canopy

import "github.com/jward/canopy/internal/store"

// Type aliases for internal store types exposed in the public API.
// These are Go type aliases (=), not new types -- they are identical
// to the internal types at compile time. External consumers use these
// names; no conversion is needed anywhere.

type Store = store.Store
type Symbol = store.Symbol
type File = store.File
type Scope = store.Scope
type CallEdge = store.CallEdge
type Import = store.Import
type FunctionParam = store.FunctionParam
type TypeMember = store.TypeMember
type TypeParam = store.TypeParam
type Annotation = store.Annotation
type ExtensionBinding = store.ExtensionBinding
type Reexport = store.Reexport
```

## Updated Signatures

All changes below are mechanical: replace `store.X` with `X` in public type positions. Since these are type aliases (`=`), not new types, the `store.` prefix is simply dropped. Internal logic is unchanged.

### engine.go

```go
// Before:
func (e *Engine) Store() *store.Store
func NewQueryBuilder(s *store.Store) *QueryBuilder

// After:
func (e *Engine) Store() *Store
func NewQueryBuilder(s *Store) *QueryBuilder
```

The `QueryBuilder` struct field `store *store.Store` remains unchanged -- it is unexported and not part of the public API.

### query.go

```go
// Before:
func (q *QueryBuilder) SymbolAt(file string, line, col int) (*store.Symbol, error)
func (q *QueryBuilder) Callers(symbolID int64) ([]*store.CallEdge, error)
func (q *QueryBuilder) Callees(symbolID int64) ([]*store.CallEdge, error)
func (q *QueryBuilder) Dependencies(fileID int64) ([]*store.Import, error)
func (q *QueryBuilder) Dependents(source string) ([]*store.Import, error)

// After:
func (q *QueryBuilder) SymbolAt(file string, line, col int) (*Symbol, error)
func (q *QueryBuilder) Callers(symbolID int64) ([]*CallEdge, error)
func (q *QueryBuilder) Callees(symbolID int64) ([]*CallEdge, error)
func (q *QueryBuilder) Dependencies(fileID int64) ([]*Import, error)
func (q *QueryBuilder) Dependents(source string) ([]*Import, error)
```

Internal variables inside method bodies that use `store.Import{}`, `&store.Import{}`, etc. are updated to `Import{}`, `&Import{}`. The `store.SymbolCols` and `store.UnmarshalModifiers` references remain as-is -- they are not aliased (internal-only helpers).

### query_detail.go

```go
// Before:
type SymbolDetail struct {
    Symbol      SymbolResult
    Parameters  []*store.FunctionParam
    Members     []*store.TypeMember
    TypeParams  []*store.TypeParam
    Annotations []*store.Annotation
}
func (q *QueryBuilder) ScopeAt(file string, line, col int) ([]*store.Scope, error)

// After:
type SymbolDetail struct {
    Symbol      SymbolResult
    Parameters  []*FunctionParam
    Members     []*TypeMember
    TypeParams  []*TypeParam
    Annotations []*Annotation
}
func (q *QueryBuilder) ScopeAt(file string, line, col int) ([]*Scope, error)
```

Internal nil-initialization lines (e.g., `params = []*store.FunctionParam{}`) are updated to use the alias names.

### query_discovery.go

```go
// Before:
type SymbolResult struct {
    store.Symbol
    ...
}
func (q *QueryBuilder) Files(...) (*PagedResult[store.File], error)

// After:
type SymbolResult struct {
    Symbol  // embedded alias
    ...
}
func (q *QueryBuilder) Files(...) (*PagedResult[File], error)
```

Internal variable declarations like `var items []store.File` and struct literals like `store.File{}` are updated to `File` / `File{}`.

### query_hierarchy.go

```go
// Before:
type TypeHierarchy struct {
    ...
    Extensions    []*store.ExtensionBinding
}
func (q *QueryBuilder) ExtensionMethods(typeSymbolID int64) ([]*store.ExtensionBinding, error)
func (q *QueryBuilder) Reexports(fileID int64) ([]*store.Reexport, error)

// After:
type TypeHierarchy struct {
    ...
    Extensions    []*ExtensionBinding
}
func (q *QueryBuilder) ExtensionMethods(typeSymbolID int64) ([]*ExtensionBinding, error)
func (q *QueryBuilder) Reexports(fileID int64) ([]*Reexport, error)
```

Internal nil-initialization lines updated similarly.

### query_graph.go

The `callGraphData` struct has unexported fields using `*store.CallEdge` -- these are not part of the public API but should be updated for consistency. The `resolveCallGraphEdge` helper function signature also uses `*store.CallEdge` internally.

```go
// Before (unexported):
type callGraphData struct {
    ...
    edgesByCaller map[int64][]*store.CallEdge
    edgesByCallee map[int64][]*store.CallEdge
    ...
}
func resolveCallGraphEdge(edge *store.CallEdge, filePaths map[int64]string) CallGraphEdge

// After:
type callGraphData struct {
    ...
    edgesByCaller map[int64][]*CallEdge
    edgesByCallee map[int64][]*CallEdge
    ...
}
func resolveCallGraphEdge(edge *CallEdge, filePaths map[int64]string) CallGraphEdge
```

## Types NOT Aliased

These `store` types are used only internally and never appear in public signatures:

- `store.SymbolCols` -- SQL column list constant, used in raw queries
- `store.UnmarshalModifiers` -- JSON deserialization helper, used in scan functions
- `store.ComputeSignatureHash` -- hash computation, used in `engine.go` blast radius
- `store.Reference` -- only used internally in queries, never returned
- `store.ResolvedReference` -- only used internally in queries, never returned
- `store.Implementation` -- only used internally, locations are returned instead
- `store.TypeComposition` -- only used internally in hierarchy building
- `store.SymbolFragment` -- not used in any query API method
- `store.NewStore` -- constructor, used only by Engine and CLI

## Import Cleanup

After replacing `store.X` with alias names in each file, some files may no longer reference the `store` package directly. Remove unused imports where the `store` import becomes unnecessary. Files that still use `store.SymbolCols`, `store.UnmarshalModifiers`, or direct `store.Store` method calls (e.g., `q.store.FileByPath`) will retain the import.

Expected import changes:
- `query.go` -- keeps `store` import (uses `store.SymbolCols`, `q.store.*` methods)
- `query_detail.go` -- may drop `store` import if only type references remain (but still uses `store.UnmarshalModifiers` via `scanSymbolResult` in discovery)
- `query_discovery.go` -- keeps `store` import (uses `store.UnmarshalModifiers`, `q.store.*`)
- `query_hierarchy.go` -- may drop `store` import depending on internal usage
- `query_graph.go` -- may drop `store` import depending on internal usage
- `engine.go` -- keeps `store` import (uses `store.NewStore`, `store.ComputeSignatureHash`, etc.)
