# Implementation Plan

## Phase 1: Symbol Detail & Structural Queries

- [ ] Add `SymbolDetail` type to `query.go` (or new `query_detail.go`) composing the symbol with its params, members, type params, and annotations
- [ ] Implement `SymbolDetail(symbolID)` -- call `Store.FunctionParams`, `Store.TypeMembers`, `Store.TypeParams`, `Store.AnnotationsByTarget` for the given symbol ID, return combined `SymbolDetail` struct
- [ ] Implement `SymbolDetailAt(file, line, col)` -- call `SymbolAt` then `SymbolDetail`; return `nil, nil` when no symbol at position
- [ ] Add `ScopeAt` Store helper -- spatial query on scopes table: find the single *innermost* scope containing position using `(start_line, start_col) <= (line, col) <= (end_line, end_col)` ordered by span size ascending, limit 1. Returns a single `*Scope` (or nil if no scope contains the position).
- [ ] Implement `ScopeAt(file, line, col)` on QueryBuilder -- resolve file path to file ID via `Store.FileByPath`, call the `ScopeAt` Store helper to get the innermost scope, then pass that scope to `Store.ScopeChain` which walks `parent_scope_id` upward to produce the full chain. Return the chain ordered innermost-to-outermost.
- [ ] Extend `SymbolFilter` with `RefCountMin *int` and `RefCountMax *int` fields
- [ ] Update `Symbols()` SQL builder to add `HAVING ref_count >= ?` / `HAVING ref_count <= ?` when ref count filters set -- requires restructuring query to use a CTE or subquery so the ref count alias is available for filtering
- [ ] Update `SearchSymbols()` SQL builder with the same ref count filter logic
- [ ] Add CLI command: `canopy query symbol-detail` -- accepts `<file> <line> <col>` or `--symbol <id>`, outputs `CLISymbolDetail`
- [ ] Add CLI command: `canopy query scope-at <file> <line> <col>` -- outputs `CLIScope` array (innermost to outermost)
- [ ] Add CLI flags: `--ref-count-min`, `--ref-count-max` on `symbols` and `search` commands
- [ ] Add CLI output types: `CLISymbolDetail` (symbol + params + members + type_params + annotations), `CLIScope` (id, kind, start/end, symbol_id)
- [ ] Add text formatters for `CLISymbolDetail` and `[]CLIScope` to existing `format.go`; add cases to `outputResultText` switch
- [ ] Add unit tests for `SymbolDetail` -- function returns params, struct returns members, annotated symbol returns annotations, plain variable returns empty sub-fields
- [ ] Add unit tests for `SymbolDetailAt` -- valid position returns combined detail, invalid position returns nil
- [ ] Add unit tests for `ScopeAt` -- file-level returns single scope, nested blocks return full chain
- [ ] Add unit tests for `Symbols`/`SearchSymbols` ref count filters -- RefCountMin=1 excludes unreferenced, RefCountMax=0 returns only unreferenced, combined range works
- [ ] Add golden test fixture exercising symbol detail (e.g., Go struct with methods and members)
- [ ] Verify with adversarial exercise

## Phase 2: Type Hierarchy & Resolution Data

- [ ] Add `TypeHierarchy` type -- wraps a symbol with lists of `ImplementedBy`, `Implements`, `Composes`, `ComposedBy`, and `Extensions` relations
- [ ] Add `TypeRelation` type -- symbol ID, name, file path, and relation kind (for hierarchy entries)
- [ ] Add new Store method: `TypeComposedBy(componentSymbolID)` -- reverse query on `type_compositions` table where `component_symbol_id = ?`
- [ ] Implement `TypeHierarchy(symbolID)` -- compose `Store.ImplementationsByInterface` (→ ImplementedBy: types implementing this interface), `Store.ImplementationsByType` (→ Implements: interfaces this type satisfies), `Store.TypeCompositions` (Composes), `Store.TypeComposedBy` (ComposedBy), `Store.ExtensionBindingsByType` (Extensions); resolve each related symbol ID to name and location
- [ ] Implement `ImplementsInterfaces(typeSymbolID)` -- call `Store.ImplementationsByType`, resolve each `InterfaceSymbolID` to a `Location`
- [ ] Implement `ExtensionMethods(typeSymbolID)` -- call `Store.ExtensionBindingsByType`, return `[]*store.ExtensionBinding` with resolved member symbol locations
- [ ] Implement `Reexports(fileID)` -- call `Store.ReexportsByFile`, return `[]*store.Reexport`
- [ ] Add CLI command: `canopy query type-hierarchy` -- accepts `<file> <line> <col>` or `--symbol <id>`, outputs `CLITypeHierarchy`
- [ ] Add CLI command: `canopy query implements` -- accepts `<file> <line> <col>` or `--symbol <id>`, outputs locations of interfaces
- [ ] Add CLI command: `canopy query extensions` -- accepts `<file> <line> <col>` or `--symbol <id>`, outputs extension bindings
- [ ] Add CLI command: `canopy query reexports <file>` -- outputs reexports for a file
- [ ] Add CLI output types: `CLITypeHierarchy`, `CLITypeRelation`, `CLIExtensionBinding`, `CLIReexport`
- [ ] Add text formatters for new CLI types to existing `format.go`
- [ ] Add unit tests for `TypeHierarchy` -- interface returns ImplementedBy, concrete type returns Implements, struct with embedding returns Composes, base returns ComposedBy, type with extensions returns Extensions
- [ ] Add unit tests for `ImplementsInterfaces`, `ExtensionMethods`, `Reexports`
- [ ] Add golden test fixture for type hierarchy (Go interface + implementing structs)
- [ ] Verify with adversarial exercise

## Phase 3: Graph Traversal & Analytical Queries

- [ ] Add new Store method: `AllCallEdges()` -- `SELECT caller_symbol_id, callee_symbol_id, file_id, line, col FROM call_graph`, returns `[]CallEdge`
- [ ] Add new Store method: `AllImports()` -- `SELECT file_id, source, kind FROM imports`, returns minimal struct or `[]*Import` (only `file_id`, `source`, `kind` needed for package graph aggregation)
- [ ] Add new Store method: `SymbolByID(id)` -- `SELECT ... FROM symbols WHERE id = ?`, returns `*Symbol` (consolidates inline SQL in `symbolLocation`)
- [ ] Implement in-memory graph builder: `buildCallGraph()` -- load all call edges via `AllCallEdges()`, build forward and reverse adjacency maps (`map[int64][]int64`)
- [ ] Implement `TransitiveCallers(symbolID, maxDepth)` -- BFS on reverse adjacency map with depth tracking; return structured result with nodes, edges, and depth per node; maxDepth 0 returns root only, negative returns error, cap at 100. Note: `store.CallEdge.FileID` is `*int64` and must be resolved to a file path string when building `CallGraphEdge` objects (pre-load all files into `map[int64]string` alongside the edge bulk load).
- [ ] Implement `TransitiveCallees(symbolID, maxDepth)` -- BFS on forward adjacency map with depth tracking; same structure and depth semantics as TransitiveCallers. Same `FileID`-to-path resolution applies.
- [ ] Implement `PackageDependencyGraph()` -- load all files + imports, resolve import sources to internal files/packages where possible, aggregate file-level imports to package-level edges with import counts. Compute per-package `LineCount` by summing `COALESCE(files.line_count, 0)` for all files belonging to each package (matching existing `ProjectSummary` pattern for pre-migration databases).
- [ ] Implement `CircularDependencies()` -- run Tarjan's SCC algorithm on package dependency graph, return cycles (SCCs with size > 1 or self-loops)
- [ ] Implement `UnusedSymbols(filter, page)` -- symbols with 0 resolved_references, exclude package/module/namespace kinds, apply SymbolFilter and Pagination
- [ ] Implement `Hotspots(topN)` -- top N symbols by external ref count, include caller count and callee count from call_graph
- [ ] Add CLI command: `canopy query transitive-callers` -- accepts `<file> <line> <col>` or `--symbol <id>`, `--max-depth N` (default 5)
- [ ] Add CLI command: `canopy query transitive-callees` -- same interface as transitive-callers
- [ ] Add CLI command: `canopy query package-graph` -- outputs package dependency graph
- [ ] Add CLI command: `canopy query circular-deps` -- outputs detected cycles
- [ ] Add CLI command: `canopy query unused` -- accepts optional filter flags, outputs unused symbols
- [ ] Add CLI command: `canopy query hotspots` -- accepts `--top N` (default 10), outputs hotspot symbols
- [ ] Add CLI output types: `CLICallGraph` (nodes + edges + max_depth), `CLICallGraphNode` (symbol_id, name, depth), `CLICallGraphEdge` (caller_id, callee_id), `CLIDependencyGraph` (packages + edges), `CLIDependencyEdge` (from_package, to_package, import_count), `CLICycle` (package names in cycle), `CLIHotspot` (symbol + ref_count + caller_count + callee_count)
- [ ] Add text formatters for all new CLI types to existing `format.go`
- [ ] Add unit tests for transitive call graph traversal -- depth 1 equals direct callers, depth 3 follows chains, handles cycles without infinite loop
- [ ] Add unit tests for package dependency graph -- aggregates file imports to package level, correct import counts
- [ ] Add unit tests for cycle detection -- empty for acyclic, detects A-B-A, detects A-B-C-A
- [ ] Add unit tests for unused symbols -- 0 refs returned, excludes package/module/namespace, respects filter and pagination
- [ ] Add unit tests for hotspots -- returns top N, includes caller/callee counts
- [ ] Add golden test fixture for graph queries
- [ ] Verify with adversarial exercise

## Notes

- Each phase is independently testable and deployable. Phase 1 has no dependency on Phase 2 or 3.
- New QueryBuilder methods go in `query_detail.go` (Phase 1), `query_hierarchy.go` (Phase 2), and `query_graph.go` (Phase 3) to keep file sizes manageable. Tests follow the same naming convention.
- New CLI commands register in `cmd/canopy/query.go` `init()` via `queryCmd.AddCommand(...)`, following the existing pattern. New command implementations go in `cmd/canopy/query_detail.go`, `cmd/canopy/query_hierarchy.go`, and `cmd/canopy/query_graph.go`.
- New CLI output types go in `cmd/canopy/types.go`. New text formatters are added to the existing `cmd/canopy/format.go`.
- Phase 3 graph traversal loads ALL call_edges into memory, builds adjacency maps, and runs BFS with depth tracking. For a typical project (10K symbols, 50K edges), this is trivially fast and avoids N+1 SQL queries during traversal.
- Phase 3 cycle detection uses Tarjan's SCC algorithm (linear time O(V+E)) on the package dependency graph. An SCC with more than one node indicates a cycle.
- New Store methods introduced across phases: `ScopeAt` (Phase 1), `TypeComposedBy` (Phase 2), `SymbolByID`, `AllCallEdges`, `AllImports`, and `AllFiles` (Phase 3). `AllFiles` returns `map[int64]string` (ID→path) for bulk FileID resolution in graph traversal.
- [ ] Refactor `symbolLocation` and `referenceLocation` in `query.go` to use the new `SymbolByID` Store method (Phase 3, after `SymbolByID` is added)
- Ref count filtering in Phase 1 requires restructuring the `Symbols()`/`SearchSymbols()` SQL to make the ref count alias available in a HAVING or WHERE clause. Options: (a) wrap the existing query as a CTE, (b) repeat the subquery in a HAVING clause. Option (b) is simpler and SQLite optimizes the repeated correlated subquery.
- `maxDepth` for transitive traversals: 0 returns root only (no traversal), negative returns error, values above 100 are silently capped.
