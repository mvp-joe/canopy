# Query API v2

## Summary

Expands the QueryBuilder to expose all extraction and resolution data currently stored in SQLite but inaccessible through the public query API, and adds graph traversal and analytical queries. The current QueryBuilder has 14 methods covering position-based navigation (SymbolAt, DefinitionAt, ReferencesTo), single-level call graph (Callers, Callees), dependency listing (Dependencies, Dependents), discovery/search (Symbols, SearchSymbols, Files, Packages), and digests (ProjectSummary, PackageSummary). Ten Store-level accessors have no QueryBuilder consumer: TypeMembers, FunctionParams, TypeParams, AnnotationsByTarget, ScopeChain, ScopesByFile, ImplementationsByType, ExtensionBindingsByType, TypeCompositions, and ReexportsByFile. This spec adds methods to surface all of that data, plus transitive graph traversals and analytical queries (unused symbols, circular dependencies, hotspots), enabling project-cortex and AI agents to answer all 13 categories of codebase questions without falling back to raw SQL.

## Goals

- **Symbol detail**: Single combined response returning a symbol with its parameters, members, type parameters, and annotations (eliminates N+1 round-trips for "tell me everything about this symbol")
- **Scope queries**: Expose scope chain at a position for visibility and scoping questions
- **Type hierarchy**: Full hierarchy view showing implements, implemented-by, composes, composed-by, and extension methods for any type
- **Inverse implementation lookup**: Answer "what interfaces does this concrete type implement?" (inverse of existing Implementations method)
- **Extension methods**: Surface extension bindings (trait impls, Swift extensions, Kotlin extensions) for a type
- **Re-exports**: Expose re-exported symbols from a file
- **Transitive call graph**: BFS traversal of callers/callees up to configurable depth, with bulk-loaded edges to avoid N+1
- **Package dependency graph**: Aggregated package-to-package dependency graph derived from file-level imports
- **Circular dependency detection**: Cycle detection in the package dependency graph
- **Unused symbol detection**: Symbols with zero resolved references, filtered by kind and visibility
- **Hotspot analysis**: Most-referenced symbols with fan-in/fan-out metrics
- **Extended filtering**: RefCount range filters on SymbolFilter for analytical queries
- **CLI commands for all new methods**: Every QueryBuilder method gets a corresponding `canopy query` subcommand
- **Answer all 13 question categories**: Orientation, navigation, understanding symbols, call graph, dependencies, type hierarchy, refactoring, code smells, impact analysis, cross-cutting, comparative, type queries, scope and visibility

## Non-Goals

- Data flow or taint analysis (requires runtime/control-flow information beyond tree-sitter)
- New extraction or resolution logic (this spec only adds query-time methods over existing data)
- Changes to Risor scripts (all new logic is Go-side QueryBuilder code)
- New database tables or schema changes (queries operate on existing 17 tables)
- Streaming or real-time results
- Fuzzy/typo-tolerant search (covered by existing SearchSymbols glob matching)

## Depends On

- implemented/2026-02-14_semantic-bridge
- implemented/2026-02-15_discovery-search-api

## Current Status

Planning

## Key Files

- [interface.md](interface.md) - Complete type signatures and method signatures for all new QueryBuilder methods, organized by phase
- [implementation.md](implementation.md) - Three-phase implementation plan with granular checkboxes
- [tests.md](tests.md) - Test specifications for all new methods organized by phase
