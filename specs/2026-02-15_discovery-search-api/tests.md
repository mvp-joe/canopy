# Test Specifications

## Unit Tests

### Pagination & Sort Helpers

- Default pagination (offset=0, limit=50) when zero-value struct passed
- Limit capped at 500 when exceeding max
- Offset=10, limit=20 skips first 10, returns next 20
- Sort by name ascending produces alphabetical order
- Sort by ref_count descending produces most-referenced first
- TotalCount reflects full result set regardless of pagination

### Symbols (Enumeration)

- Empty database returns empty result with TotalCount=0
- Returns all symbols when no filter applied
- Filter by single kind (e.g., "function") returns only functions
- Filter by multiple kinds (e.g., ["interface", "trait"]) returns both
- Filter by visibility "public" excludes private symbols
- Filter by modifiers ["async"] returns only symbols with async modifier
- Filter by FileID restricts to that file's symbols
- Filter by ParentID returns only direct children of that symbol
- Filter by PathPrefix restricts to symbols in files under that path
- PathPrefix "internal/store" matches "internal/store/store.go" but not "internal/runtime/runtime.go"
- PathPrefix "internal/store" does NOT match "internal/store_utils/file.go" (trailing "/" normalization)
- Combined filters (kind + visibility) intersect correctly
- Combined filters (kind + PathPrefix) intersect correctly
- Pagination: first page returns correct slice, second page returns remainder
- Sort by name ascending produces alphabetical output
- Sort by kind groups symbols by kind
- Sort by file groups symbols by file path
- Sort by ref_count descending puts most-referenced first
- Inapplicable SortField (e.g. SortByRefCount on Files()) falls back to default sort
- SymbolResult includes correct FilePath for file-scoped symbols
- SymbolResult FilePath is empty for multi-file symbols (nil FileID)
- SymbolResult includes correct RefCount from resolved_references

### Files

- Empty database returns empty result
- No filter (empty pathPrefix, empty language) returns all files
- Filter by language "go" returns only Go files
- PathPrefix "internal/store" returns only files under that directory
- PathPrefix combined with language narrows correctly
- Sort by file path ascending produces alphabetical output
- Pagination works correctly

### Packages

- Returns symbols with kind "package", "module", or "namespace"
- Excludes other symbol kinds (e.g. "function", "class")
- Empty pathPrefix returns all packages
- PathPrefix "internal/" returns only packages under that path
- Sort by name ascending produces alphabetical output
- Sort by ref_count descending puts most-referenced packages first
- Pagination works correctly

### SearchSymbols

- Pattern `"Anim*"` matches "Animal", "Animation", not "inanimate"
- Pattern `"*mal"` matches "Animal", not "Animals"
- Pattern `"Anim*l"` matches "Animal", not "Animation"
- Pattern `"*Controller*"` matches "UserController", "ControllerBase"
- Pattern `"Get*User*"` matches "GetCurrentUser", "GetUserByID"
- Empty pattern returns all symbols (same as no filter)
- Pattern with no wildcards performs exact match
- Search combined with kind filter narrows results
- Search combined with PathPrefix filter narrows to that directory
- Case insensitivity: "animal" matches both "animal" and "Animal" (SQLite LIKE default)
- Pattern `"*"` returns all symbols
- Special SQL characters in pattern (`%`, `_`) are escaped before wildcard substitution
- Pagination and sorting apply to search results

### ProjectSummary

- Empty database returns zero counts
- Single-language project returns one LanguageStats entry
- Multi-language project returns per-language breakdown with correct file and symbol counts
- KindCounts within each language are accurate
- TopSymbols returns top-N by reference count
- TopN=0 returns no top symbols (but still returns stats)
- TopN exceeding total symbol count returns all symbols without error
- PackageCount reflects total package+module+namespace symbols

### PackageSummary

- Lookup by package path string resolves correctly
- Lookup by symbol ID works directly
- ExportedSymbols includes only public symbols within the package
- ExportedSymbols sorted by ref count descending
- KindCounts accurate for the package scope
- Dependencies lists import sources from files in this package
- Dependents lists packages that import this package
- Non-existent package path returns appropriate error
- Both packagePath and packageID provided: packageID takes precedence

## Integration Tests

- Get symbol ID via `SearchSymbols`, pass to `ReferencesTo` — returns correct references
- Get symbol ID via `SearchSymbols`, pass to `Implementations` — returns correct implementations
- `ProjectSummary` top symbols can be passed to existing `Callers` and `Callees` methods
- `Packages` results can scope `Symbols` via `PathPrefix` matching the package's file path

## Error Scenarios

- Invalid ParentID in filter returns empty result (not SQL error)
- Negative offset in pagination treated as 0
- Limit of 0 uses default (50), not "return nothing"
- PackageSummary with empty path and nil ID returns validation error
