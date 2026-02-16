# Implementation Plan

## Phase 1: Common Types & Pagination Infrastructure

- [x] Add `Pagination`, `Sort`, `SortField`, `SortOrder` types to `query_discovery.go`
- [x] Add `SymbolResult` type with `RefCount` and `FilePath` fields
- [x] Add `PagedResult[T]` generic type
- [x] Add internal helpers: `normalizePathPrefix`, `symbolSortColumn`, `fileSortColumn`, `sortDirection`, `escapeLike`
- [x] Add `scanSymbolResult` and `prefixSymbolCols` scan helpers
- [x] Add `UnmarshalModifiers` export to store helpers
- [x] Unit tests for pagination normalization, escapeLike, normalizePathPrefix

## Phase 2: Enumeration Endpoints

- [x] Add `SymbolFilter` type (Kinds, Visibility, Modifiers, FileID, ParentID, PathPrefix)
- [x] Implement `Symbols(filter, sort, page)` — dynamic WHERE clause from filter fields, joined with files table for FilePath and PathPrefix, subquery for ref count, `json_each` for modifier filtering
- [x] Implement `Files(pathPrefix, language, sort, page)` — query on files table with optional path prefix and language filters
- [x] Implement `Packages(pathPrefix, sort, page)` — delegates to `Symbols` with `Kinds: ["package", "module", "namespace"]` preset and optional path prefix
- [x] Verify `idx_symbols_kind` index exists (it does — defined in store.go migration)
- [x] Add `CREATE INDEX IF NOT EXISTS idx_files_language ON files(language)` to existing `schemaDDL` constant in store.go
- [x] Unit tests for each endpoint: empty DB, single result, filtered, paginated, sorted

## Phase 3: Search

- [x] Implement `SearchSymbols(pattern, filter, sort, page)` — converts `*` to `%`, uses `LIKE` in WHERE clause, combines with SymbolFilter
- [x] Handle edge cases: empty pattern (return all), pattern with no wildcards (exact match), escape literal `%` and `_` before wildcard substitution, parameterized LIKE with `ESCAPE '\'`
- [x] Unit tests: prefix, suffix, infix, multiple wildcards, combined with filters, case insensitivity

## Phase 4: Digest Endpoints

- [x] Add `ProjectSummary`, `LanguageStats` types
- [x] Implement `ProjectSummary(topN)` — aggregates across files and symbols tables, computes top-N by ref count
- [x] Add `PackageSummary` type
- [x] Implement `PackageSummary(packagePath, packageID)` — resolves path to ID if needed, gathers exported symbols, kind counts, dependency/dependent paths
- [x] Unit tests: empty project, multi-language project, package with no dependents

## Phase 5: Integration & Documentation

- [x] Verify all new methods are accessible via `Engine.Query()` (they are — QueryBuilder is returned by Engine.Query())
- [x] Integration tests: SearchSymbols→ReferencesTo, Packages→Symbols scoping, ProjectSummary→Callers

## Notes

- All new SQL queries use parameterized queries to prevent injection
- Ref count computation uses correlated subquery `(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id)` — bulk via JOIN in the main query
- PathPrefix normalization: append `/` before LIKE to prevent false matches (`"internal/store"` → `LIKE 'internal/store/%'`)
- Search is case-insensitive (SQLite LIKE default for ASCII) — this is intentional for symbol discovery
- All implementation in `query_discovery.go` with tests in `query_discovery_test.go`
