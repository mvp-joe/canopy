# Interface Definitions

All new methods are added to the existing `QueryBuilder` struct. All positions remain 0-based per the tree-sitter convention.

---

## Common Types

```go
// Pagination controls offset+limit paging on list/search results.
type Pagination struct {
    Offset int // skip this many results (default 0)
    Limit  int // max results to return (default 50, max 500)
}

// SortField specifies how to order results.
type SortField string

const (
    SortByName      SortField = "name"
    SortByKind      SortField = "kind"
    SortByFile      SortField = "file"      // maps to files.path via JOIN for symbol queries, files.path directly for file queries
    SortByRefCount  SortField = "ref_count"  // most-referenced first
)

// SortOrder specifies ascending or descending.
type SortOrder string

const (
    Asc  SortOrder = "asc"
    Desc SortOrder = "desc"
)

// Sort controls result ordering.
// Zero-value defaults: SortByName ascending for symbol queries, SortByFile ascending for file queries.
// If an inapplicable SortField is passed for an endpoint (e.g. SortByRefCount on Files()),
// it falls back to the default sort for that endpoint.
type Sort struct {
    Field SortField
    Order SortOrder
}

// SymbolResult extends Symbol with computed fields useful for discovery.
type SymbolResult struct {
    store.Symbol
    FilePath string // resolved file path (empty for multi-file symbols)
    RefCount int    // number of resolved references targeting this symbol
}

// PagedResult wraps a page of results with total count for pagination.
type PagedResult[T any] struct {
    Items      []T
    TotalCount int // total matching results (before pagination)
}
```

---

## Enumeration

### Symbols

The primary listing/filtering endpoint. All filter fields are optional — omit to match all.

```go
// SymbolFilter specifies which symbols to include.
type SymbolFilter struct {
    Kinds       []string // match any of these kinds ("function", "interface", etc.)
    Visibility  *string  // exact match ("public", "private", "protected", etc.)
    Modifiers   []string // symbol must have ALL of these modifiers
    FileID      *int64   // restrict to a single file
    ParentID    *int64   // restrict to direct children of this symbol (via parent_symbol_id)
    PathPrefix  *string  // restrict to symbols in files under this path (e.g. "internal/store"); normalized to end with "/" before LIKE matching
}

func (q *QueryBuilder) Symbols(filter SymbolFilter, sort Sort, page Pagination) (*PagedResult[SymbolResult], error)
```

### Files

Convenience method for listing files. For more complex queries, use `Store.DB()` directly.

```go
// pathPrefix restricts to files under this directory (e.g. "internal/store"); normalized to end with "/" before LIKE matching. Empty string means all files.
// language restricts to files of this language (e.g. "go"). Empty string means all languages.
func (q *QueryBuilder) Files(pathPrefix string, language string, sort Sort, page Pagination) (*PagedResult[store.File], error)
```

### Packages

Convenience method for listing packages, modules, and namespaces. For more complex filtering, use `Symbols` with `Kinds: ["package", "module", "namespace"]`.

```go
// pathPrefix restricts to packages under this path (e.g. "internal/"); normalized to end with "/" before LIKE matching. Empty string means all packages.
func (q *QueryBuilder) Packages(pathPrefix string, sort Sort, page Pagination) (*PagedResult[SymbolResult], error)
```

---

## Search

### SearchSymbols

Glob-style search on symbol names. `*` is the wildcard (mapped to SQL `%`).
Combines with the same structured filters as `Symbols`.

```go
func (q *QueryBuilder) SearchSymbols(pattern string, filter SymbolFilter, sort Sort, page Pagination) (*PagedResult[SymbolResult], error)
```

Pattern examples:
- `"Anim*"` — prefix match
- `"*mal"` — suffix match
- `"Anim*l"` — infix wildcard
- `"*Controller*"` — contains
- `"Get*User*"` — multiple wildcards

Matching is case-insensitive (SQLite LIKE default for ASCII). Pattern `*` alone returns all symbols (equivalent to `Symbols` with no filter).

**Escaping:** Literal `%` and `_` characters in the input pattern are escaped (to `\%` and `\_` with `ESCAPE '\'`) before `*` is converted to `%`. This prevents SQL `LIKE` single-character wildcard `_` from matching unintentionally.

---

## Digest / Summary

### ProjectSummary

High-level overview of the entire indexed codebase.

```go
type ProjectSummary struct {
    Languages    []LanguageStats       // per-language breakdown
    PackageCount int
    TopSymbols   []SymbolResult        // top-N most-referenced symbols across all kinds
}

type LanguageStats struct {
    Language   string
    FileCount  int
    SymbolCount int
    KindCounts map[string]int // symbol count by kind ("function": 142, "class": 23, etc.)
}

func (q *QueryBuilder) ProjectSummary(topN int) (*ProjectSummary, error)
```

### PackageSummary

Summary of a single package/module.

```go
type PackageSummary struct {
    Symbol       SymbolResult
    Path         string              // package/module path
    FileCount    int
    ExportedSymbols []SymbolResult   // public symbols, sorted by ref count descending
    KindCounts   map[string]int      // symbol count by kind within this package
    Dependencies []string            // import source strings from files in this package
    Dependents   []string            // file paths of files outside this package that import it
}

// Accepts either a file path prefix or a symbol ID.
// packagePath is a file path prefix (e.g. "internal/store") — resolved by finding files
// under that path, then locating the package/module symbol in those files.
// If packageID is non-nil, it is used directly.
func (q *QueryBuilder) PackageSummary(packagePath string, packageID *int64) (*PackageSummary, error)
```
