# Test Specifications

## Unit Tests

### Phase 1 -- SymbolDetail

- SymbolDetail for a function symbol returns its FunctionParams (including receiver and return type entries) with empty Members, TypeParams, and Annotations
- SymbolDetail for a struct/class symbol returns its TypeMembers with empty Params
- SymbolDetail for a generic type returns TypeParams with constraints populated
- SymbolDetail for an annotated symbol returns Annotations list with correct names and arguments
- SymbolDetail for a plain variable returns empty Params, Members, TypeParams, and Annotations
- SymbolDetail for a non-existent symbol ID returns nil, nil (no error)
- SymbolDetailAt with a valid position (pointing at a function) returns the combined SymbolDetail with the symbol and its params
- SymbolDetailAt with a position that has no symbol returns nil, nil
- SymbolDetailAt with a file path not in the database returns nil, nil

### Phase 1 -- ScopeAt

- ScopeAt at a file-level position (outside any function) returns a single scope with kind "file"
- ScopeAt inside a function body returns a chain of at least two scopes (function scope, file scope) ordered innermost to outermost
- ScopeAt inside a nested block (e.g., if-body inside a function) returns the full chain: block scope, function scope, file scope
- ScopeAt with a file path not in the database returns nil slice, nil error
- ScopeAt with a position outside any scope returns nil slice, nil error

### Phase 1 -- SymbolFilter RefCount Extensions

- Symbols with RefCountMin=1 excludes all symbols with zero resolved references
- Symbols with RefCountMax=0 returns only symbols with zero resolved references
- Symbols with RefCountMin=2 and RefCountMax=5 returns only symbols with ref count in that range
- SearchSymbols with RefCountMin set correctly filters search results by reference count
- Symbols with RefCountMin set and no symbols matching returns empty result with TotalCount=0
- TotalCount reflects the filtered count (after ref count filter), not total symbols in database

### Phase 2 -- TypeHierarchy

- TypeHierarchy for an interface symbol returns ImplementedBy containing the concrete types that implement it
- TypeHierarchy for a concrete type symbol returns Implements containing the interfaces it satisfies
- TypeHierarchy for a struct with embedding returns Composes containing the embedded types
- TypeHierarchy for a base type that is embedded returns ComposedBy containing the composing types
- TypeHierarchy for a type with extension methods returns Extensions containing the extension bindings
- TypeHierarchy for a type with no relationships returns an empty hierarchy (all lists empty, only the symbol itself populated)
- TypeHierarchy for a function symbol (not a type) returns empty hierarchy with just the symbol
- TypeHierarchy for a non-existent symbol ID returns nil, nil

### Phase 2 -- ImplementsInterfaces

- ImplementsInterfaces for a type implementing one interface returns one Location pointing to the interface definition
- ImplementsInterfaces for a type implementing multiple interfaces returns multiple Locations
- ImplementsInterfaces for a type implementing no interfaces returns empty slice

### Phase 2 -- ExtensionMethods

- ExtensionMethods for a type with extension bindings returns all bindings with member symbol IDs
- ExtensionMethods for a type with no extensions returns empty slice

### Phase 2 -- Reexports

- Reexports for a file with re-exports returns all re-exported symbols with original symbol IDs and exported names
- Reexports for a file with no re-exports returns empty slice
- Reexports for a non-existent file ID returns empty slice

### Phase 3 -- TransitiveCallers

- TransitiveCallers with depth 1 returns the same symbols as direct Callers
- TransitiveCallers with depth 3 follows multi-hop chains: if A calls B calls C, TransitiveCallers(C, 3) includes both A and B
- TransitiveCallers with depth 0 returns only the root node (the queried symbol itself) with no edges
- TransitiveCallers handles cycles without infinite loop: if A calls B and B calls A, both appear exactly once
- TransitiveCallers with depth exceeding the actual graph depth returns the full reachable set without error
- TransitiveCallers for a symbol with no callers returns only the root node
- TransitiveCallers for a non-existent symbol ID returns nil, nil

### Phase 3 -- TransitiveCallees

- TransitiveCallees with depth 1 returns the same symbols as direct Callees
- TransitiveCallees with depth 3 follows multi-hop chains: if A calls B calls C, TransitiveCallees(A, 3) includes both B and C
- TransitiveCallees handles cycles without infinite loop
- TransitiveCallees with depth 0 returns only the root node with no edges
- TransitiveCallees for a leaf function (no outgoing calls) returns only the root node

### Phase 3 -- PackageDependencyGraph

- PackageDependencyGraph aggregates file-level imports to package-level edges
- PackageDependencyGraph ImportCount on each edge reflects the number of distinct file-level imports between those two packages
- PackageDependencyGraph on an empty database returns an empty graph (no packages, no edges)
- PackageDependencyGraph excludes external imports that do not resolve to indexed packages

### Phase 3 -- CircularDependencies

- CircularDependencies returns empty list for an acyclic package graph
- CircularDependencies detects a simple A-B-A cycle (two packages importing each other)
- CircularDependencies detects a longer A-B-C-A cycle (three packages forming a ring)
- CircularDependencies detects a self-loop (package importing itself)
- CircularDependencies on an empty database returns empty list

### Phase 3 -- UnusedSymbols

- UnusedSymbols returns symbols with zero entries in resolved_references
- UnusedSymbols excludes symbols with kind "package", "module", or "namespace" (hardcoded exclusion)
- UnusedSymbols respects SymbolFilter: kind filter narrows to specific kinds (e.g., only unused functions)
- UnusedSymbols respects SymbolFilter: visibility filter (e.g., only unused private symbols)
- UnusedSymbols respects SymbolFilter: path prefix filter
- UnusedSymbols respects Pagination: first page returns correct count, second page returns remainder
- UnusedSymbols TotalCount reflects total unused symbols matching filter (before pagination)

### Phase 3 -- Hotspots

- Hotspots returns top N symbols ordered by external ref count descending
- Hotspots includes correct caller count (number of call_graph edges where symbol is callee)
- Hotspots includes correct callee count (number of call_graph edges where symbol is caller)
- Hotspots does not include TransitiveCallers field (consumers call TransitiveCallers() separately)
- Hotspots with topN larger than total symbol count returns all symbols without error
- Hotspots with topN=0 returns empty list
- Hotspots excludes symbols with zero references
- Hotspots with negative topN returns an error

## Integration Tests

- Given a Go project with structs, methods, and interfaces; when querying SymbolDetail for a method; then params include receiver (is_receiver=true) and return type (is_return=true) entries
- Given a Java project with generics; when querying SymbolDetail for a generic class; then type_params include constraints
- Given a Python project with decorators; when querying SymbolDetail for a decorated function; then annotations list includes all decorator names
- Given a multi-file Go project; when querying ScopeAt inside a nested if-block; then the scope chain includes block, function, and file scopes in order
- Given a Go project with interfaces and implementing types; when querying TypeHierarchy for the interface; then ImplementedBy contains the implementing types with correct locations
- Given a Go project with struct embedding; when querying TypeHierarchy for the embedding struct; then Composes contains the embedded type
- Given a multi-file project with a call chain A->B->C->D; when querying TransitiveCallers for D with depth 5; then returns the full chain A, B, C with correct depths
- Given a project with circular package dependencies; when querying CircularDependencies; then the cycle is reported with the correct package names
- Given a project with unused private functions; when querying UnusedSymbols with no filter; then those functions appear in results
- Given a project with heavily-referenced utility functions; when querying Hotspots with topN=5; then those utility functions appear with correct ref and call counts

## Error Scenarios

- SymbolDetail with non-existent symbol ID returns nil, nil (not an error)
- TypeHierarchy for a function symbol (not a type) returns empty hierarchy with just the symbol info
- TransitiveCallers with non-existent symbol ID returns nil, nil (not an error)
- TransitiveCallers with negative maxDepth returns an error
- TransitiveCallers with maxDepth > 100 is silently capped at 100
- TransitiveCallees with non-existent symbol ID returns nil, nil (not an error)
- TransitiveCallees with negative maxDepth returns an error
- UnusedSymbols with invalid SymbolFilter (e.g., non-existent FileID) returns empty result, not an error
- PackageDependencyGraph on empty database returns empty graph, not an error
- Hotspots with negative topN returns an error
- ScopeAt with negative line or col returns nil slice, nil error (not an error)
- Reexports with non-existent file ID returns empty slice, not an error
