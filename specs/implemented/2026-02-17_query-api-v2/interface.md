# Interface Definitions

All new methods are added to the existing `QueryBuilder` struct in the `canopy` package. All positions remain 0-based per the tree-sitter convention. Store types (`store.FunctionParam`, `store.TypeMember`, etc.) are referenced directly from `internal/store/types.go` -- not redefined here.

---

## Phase 1: Symbol Detail & Structural Queries

### New Types

```go
// SymbolDetail is a combined response that bundles a symbol with all of its
// structural metadata. One call replaces four separate Store lookups.
type SymbolDetail struct {
    Symbol      SymbolResult           // the symbol itself with ref counts
    Parameters  []*store.FunctionParam // function/method params, receiver, returns (empty for non-functions)
    Members     []*store.TypeMember    // struct fields, class methods, interface contracts (empty for non-types)
    TypeParams  []*store.TypeParam     // generic type parameters with constraints (empty if non-generic)
    Annotations []*store.Annotation    // decorators, annotations, attributes (empty if none)
}
```

### Extended SymbolFilter

Two new fields are added to the existing `SymbolFilter` struct:

```go
type SymbolFilter struct {
    // ... existing fields unchanged ...
    Kinds       []string // match any of these kinds
    Visibility  *string  // exact match
    Modifiers   []string // symbol must have ALL of these modifiers
    FileID      *int64   // restrict to a single file
    ParentID    *int64   // restrict to direct children of this symbol
    PathPrefix  *string  // restrict to symbols in files under this path

    // New in v2
    RefCountMin *int     // only symbols with ref_count >= this value
    RefCountMax *int     // only symbols with ref_count <= this value
}
```

### New QueryBuilder Methods

```go
// SymbolDetail returns a combined response with the symbol and all its
// structural metadata (parameters, members, type parameters, annotations).
// Returns nil with no error if the symbol ID does not exist.
func (q *QueryBuilder) SymbolDetail(symbolID int64) (*SymbolDetail, error)

// SymbolDetailAt is a position-based convenience that resolves the narrowest
// symbol at (file, line, col) and returns its SymbolDetail.
// Line and col are 0-based. Returns nil with no error if no symbol exists.
func (q *QueryBuilder) SymbolDetailAt(file string, line, col int) (*SymbolDetail, error)

// ScopeAt returns the scope chain at a position, ordered from innermost to
// outermost. Finds the narrowest scope containing (file, line, col), then
// walks parent_scope_id to the file scope.
// Line and col are 0-based. Returns nil slice, nil error if no scope contains
// the position or file is not indexed.
func (q *QueryBuilder) ScopeAt(file string, line, col int) ([]*store.Scope, error)
```

### CLI Commands

```
canopy query symbol-detail <file> <line> <col>
canopy query symbol-detail --symbol <id>
canopy query scope-at <file> <line> <col>
canopy query symbols --ref-count-min <n> --ref-count-max <n>   (extended filters)
```

---

## Phase 2: Type Hierarchy & Resolution Data

### New Types

```go
// TypeRelation represents a relationship between two types in a hierarchy.
// Location is derived from Symbol.SymbolResult (FilePath, StartLine, etc.).
type TypeRelation struct {
    Symbol   SymbolResult
    Kind     string   // "inheritance", "interface_impl", "composition", "embedding", "implicit"
}

// TypeHierarchy is a complete hierarchy view for a single type, combining
// data from implementations, type_compositions, and extension_bindings tables.
type TypeHierarchy struct {
    Symbol        SymbolResult             // the queried type
    Implements    []*TypeRelation          // interfaces/traits this type implements
    ImplementedBy []*TypeRelation          // concrete types implementing this interface/trait
    Composes      []*TypeRelation          // parent types (inherited, embedded, composed)
    ComposedBy    []*TypeRelation          // child types that inherit/embed/compose this type
    Extensions    []*store.ExtensionBinding // extension methods, trait impls, default impls
}
```

### New QueryBuilder Methods

```go
// TypeHierarchy returns the full type hierarchy for a symbol: what it
// implements, what implements it, what it composes, what composes it,
// and its extension methods. Combines data from implementations,
// type_compositions, and extension_bindings tables.
func (q *QueryBuilder) TypeHierarchy(symbolID int64) (*TypeHierarchy, error)

// ImplementsInterfaces returns the interfaces/traits that a concrete type
// implements. This is the inverse of the existing Implementations method
// (which returns types implementing a given interface).
// Returns locations of the interface declarations.
func (q *QueryBuilder) ImplementsInterfaces(typeSymbolID int64) ([]Location, error)

// ExtensionMethods returns extension bindings for a type: extension methods,
// trait implementations, default implementations, protocol conformances.
func (q *QueryBuilder) ExtensionMethods(typeSymbolID int64) ([]*store.ExtensionBinding, error)

// Reexports returns re-exported symbols from a file.
func (q *QueryBuilder) Reexports(fileID int64) ([]*store.Reexport, error)
```

### CLI Commands

```
canopy query type-hierarchy <file> <line> <col>
canopy query type-hierarchy --symbol <id>
canopy query implements <file> <line> <col>
canopy query implements --symbol <id>
canopy query extensions <file> <line> <col>
canopy query extensions --symbol <id>
canopy query reexports <file>
```

---

## Phase 3: Graph Traversal & Analytical Queries

### New Types

```go
// CallGraph represents a transitive call graph rooted at a symbol.
// Nodes and edges are bulk-loaded then traversed with BFS -- no recursive
// SQL or N+1 queries.
type CallGraph struct {
    Root  int64           // starting symbol ID
    Nodes []CallGraphNode // all symbols reachable within depth
    Edges []CallGraphEdge // all edges in the subgraph
    Depth int             // actual max depth reached (may be < maxDepth if graph is shallow)
}

// CallGraphNode is a symbol in the call graph with its distance from the root.
type CallGraphNode struct {
    Symbol SymbolResult
    Depth  int // BFS depth from root (0 = root itself)
}

// CallGraphEdge is a single caller-callee relationship in the call graph.
type CallGraphEdge struct {
    CallerID int64
    CalleeID int64
    File     string
    Line     int
    Col      int
}

// DependencyGraph is the package-to-package dependency graph, aggregated
// from file-level imports.
type DependencyGraph struct {
    Packages []PackageNode
    Edges    []DependencyEdge
}

// PackageNode represents a package in the dependency graph.
type PackageNode struct {
    Name      string
    FileCount int
    LineCount int
}

// DependencyEdge represents a dependency between two packages with the
// number of file-level imports that contribute to it.
type DependencyEdge struct {
    FromPackage string
    ToPackage   string
    ImportCount int // number of file-level imports between these packages
}

// HotspotResult represents a heavily-referenced symbol with fan-in/fan-out
// metrics from the call graph. Consumers can call TransitiveCallers()
// separately if they need transitive depth.
type HotspotResult struct {
    Symbol      SymbolResult
    CallerCount int // direct callers (fan-in from call_graph)
    CalleeCount int // direct callees (fan-out from call_graph)
}
```

### New QueryBuilder Methods

```go
// TransitiveCallers returns all transitive callers of a symbol up to maxDepth.
// Bulk-loads all call_graph edges into memory, then walks callers with BFS.
// maxDepth of 0 returns only the root node (no traversal). Negative returns error.
// Capped at 100. Returns nil, nil if symbolID does not exist.
func (q *QueryBuilder) TransitiveCallers(symbolID int64, maxDepth int) (*CallGraph, error)

// TransitiveCallees returns all transitive callees of a symbol up to maxDepth.
// Bulk-loads all call_graph edges into memory, then walks callees with BFS.
// maxDepth of 0 returns only the root node (no traversal). Negative returns error.
// Capped at 100. Returns nil, nil if symbolID does not exist.
func (q *QueryBuilder) TransitiveCallees(symbolID int64, maxDepth int) (*CallGraph, error)

// PackageDependencyGraph returns the package-to-package dependency graph.
// Aggregates file-level imports: for each import, determines the source file's
// package and the import target's package, then counts edges between packages.
// Packages are identified by the "package"/"module"/"namespace" symbols in the
// symbols table.
func (q *QueryBuilder) PackageDependencyGraph() (*DependencyGraph, error)

// CircularDependencies detects cycles in the package dependency graph.
// Returns a list of cycles, each represented as a list of package names
// forming the cycle (first element repeated at end for clarity).
// Uses Tarjan's or Johnson's algorithm on the PackageDependencyGraph result.
func (q *QueryBuilder) CircularDependencies() ([][]string, error)

// UnusedSymbols returns symbols with zero resolved references.
// Hardcoded exclusion of kinds "package", "module", "namespace" (never
// meaningfully referenced). Supports the same SymbolFilter and Pagination
// as Symbols().
func (q *QueryBuilder) UnusedSymbols(filter SymbolFilter, page Pagination) (*PagedResult[SymbolResult], error)

// Hotspots returns the top-N most-referenced symbols with fan-in and fan-out
// metrics. Sorts by external reference count descending.
// topN of 0 returns empty list. Negative returns error.
func (q *QueryBuilder) Hotspots(topN int) ([]*HotspotResult, error)
```

### CLI Commands

```
canopy query transitive-callers <file> <line> <col> --max-depth 5
canopy query transitive-callers --symbol <id> --max-depth 5
canopy query transitive-callees <file> <line> <col> --max-depth 5
canopy query transitive-callees --symbol <id> --max-depth 5
canopy query package-graph
canopy query circular-deps
canopy query unused --kind <kind> --visibility <vis> --path-prefix <prefix>
canopy query hotspots --top 10
```

---

## Summary of All New QueryBuilder Methods

| Phase | Method | Input | Output |
|-------|--------|-------|--------|
| 1 | `SymbolDetail` | `symbolID int64` | `*SymbolDetail` |
| 1 | `SymbolDetailAt` | `file string, line, col int` | `*SymbolDetail` |
| 1 | `ScopeAt` | `file string, line, col int` | `[]*store.Scope` |
| 2 | `TypeHierarchy` | `symbolID int64` | `*TypeHierarchy` |
| 2 | `ImplementsInterfaces` | `typeSymbolID int64` | `[]Location` |
| 2 | `ExtensionMethods` | `typeSymbolID int64` | `[]*store.ExtensionBinding` |
| 2 | `Reexports` | `fileID int64` | `[]*store.Reexport` |
| 3 | `TransitiveCallers` | `symbolID int64, maxDepth int` | `*CallGraph` |
| 3 | `TransitiveCallees` | `symbolID int64, maxDepth int` | `*CallGraph` |
| 3 | `PackageDependencyGraph` | _(none)_ | `*DependencyGraph` |
| 3 | `CircularDependencies` | _(none)_ | `[][]string` |
| 3 | `UnusedSymbols` | `filter SymbolFilter, page Pagination` | `*PagedResult[SymbolResult]` |
| 3 | `Hotspots` | `topN int` | `[]*HotspotResult` |
