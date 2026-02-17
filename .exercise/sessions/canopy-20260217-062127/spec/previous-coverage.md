# Previous Exercise Coverage

The previous exercise session (canopy-20260216-232119) wrote 25 test scripts covering the following areas. **Do NOT re-test these basic scenarios.** Focus on NEW commands and DEEPER scenarios from the taxonomy.

## Already Tested (basic coverage)

1. `canopy index` — basic indexing, multi-language, force reindex
2. `canopy query files` — list files, filter by language/prefix
3. `canopy query symbols` — list symbols, filter by kind/visibility/path-prefix/file
4. `canopy query symbol-at` — positional lookup
5. `canopy query definition` — go-to-definition
6. `canopy query references` — find references (positional and --symbol)
7. `canopy query callers` / `callees` — direct call graph
8. `canopy query search` — glob pattern search
9. `canopy query summary` — project summary
10. `canopy query deps` / `dependents` — file-level imports
11. `canopy query implementations` — interface implementations
12. `canopy query packages` / `package-summary` — package listing and detail
13. Error handling — missing DB, invalid args, non-existent files
14. Edge cases — empty repos, special paths, large files
15. Text format output — `--format text` for various commands
16. Pagination and sorting — `--limit`, `--offset`, `--sort`, `--order`
17. Language-specific scenarios — Go, TypeScript, Python, Java
18. Cross-file references and data correctness

## NOT Yet Tested — Focus Here

### New Commands (Query API v2 — never tested before)
- `canopy query symbol-detail` — combined symbol + params + members + type_params + annotations
- `canopy query scope-at` — scope chain at position
- `canopy query type-hierarchy` — full type hierarchy view
- `canopy query implements` — inverse of implementations (what interfaces does this type implement?)
- `canopy query extensions` — extension bindings for a type
- `canopy query reexports` — re-exported symbols from a file
- `canopy query transitive-callers` — BFS caller graph with depth
- `canopy query transitive-callees` — BFS callee graph with depth
- `canopy query package-graph` — package-to-package dependency graph
- `canopy query circular-deps` — cycle detection
- `canopy query unused` — zero-reference symbols
- `canopy query hotspots` — most-referenced symbols with fan-in/fan-out

### New Flags (Query API v2)
- `--ref-count-min` / `--ref-count-max` on `symbols` and `search` commands

### Taxonomy Categories to Explore Deeper
See taxonomy-code-questions.md for the full list. Prioritize categories 3, 4, 6, 7, 8, 9 which map directly to new commands:
- Category 3 (Understanding Symbols) → `symbol-detail`
- Category 4 (Call Graph) → `transitive-callers`, `transitive-callees`, call depth/cycles
- Category 6 (Type Hierarchy) → `type-hierarchy`, `implements`, `extensions`
- Category 7 (Refactoring) → `unused`, reference analysis, blast radius
- Category 8 (Code Smells) → `unused`, `hotspots`, `circular-deps`
- Category 9 (Impact Analysis) → `hotspots`, transitive callers, coupling analysis
- Category 13 (Scope & Visibility) → `scope-at`
