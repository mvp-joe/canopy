# Implementation Plan

## Phase 1: Common Types & Pagination Infrastructure

- [ ] Add `Pagination`, `Sort`, `SortField`, `SortOrder` types to `query.go` (or a new `query_types.go`)
- [ ] Add `SymbolResult` type with `RefCount` and `FilePath` fields
- [ ] Add `PagedResult[T]` generic type
- [ ] Add internal helper: `applyPagination(query, Pagination)` that appends `LIMIT/OFFSET` to SQL
- [ ] Add internal helper: `applySortOrder(query, Sort)` that appends `ORDER BY` clause
- [ ] Add internal helper: `countTotal(query)` that wraps a query in `SELECT COUNT(*)`
- [ ] Add internal helper for bulk symbol enrichment (file path + ref count via JOIN/CTE, not per-symbol) to avoid N+1
- [ ] Unit tests for pagination/sort helpers

## Phase 2: Enumeration Endpoints

- [ ] Add `SymbolFilter` type (Kinds, Visibility, Modifiers, FileID, ParentID, PathPrefix)
- [ ] Implement `Symbols(filter, sort, page)` — dynamic WHERE clause from filter fields, joined with files table for FilePath and PathPrefix, subquery for ref count, `json_each` for modifier filtering
- [ ] Implement `Files(pathPrefix, language, sort, page)` — query on files table with optional path prefix and language filters
- [ ] Implement `Packages(pathPrefix, sort, page)` — delegates to `Symbols` with `Kinds: ["package", "module", "namespace"]` preset and optional path prefix
- [ ] Verify `idx_symbols_kind` index exists (it does — defined in store.go migration)
- [ ] Add `CREATE INDEX IF NOT EXISTS idx_files_language ON files(language)` to existing `schemaDDL` constant in store.go
- [ ] Unit tests for each endpoint: empty DB, single result, filtered, paginated, sorted

## Phase 3: Search

- [ ] Implement `SearchSymbols(pattern, filter, sort, page)` — converts `*` to `%`, uses `LIKE` in WHERE clause, combines with SymbolFilter
- [ ] Handle edge cases: empty pattern (return all), pattern with no wildcards (exact match), escape literal `%` and `_` before wildcard substitution, parameterized LIKE with `ESCAPE '\'`
- [ ] Unit tests: prefix, suffix, infix, multiple wildcards, combined with filters, case sensitivity

## Phase 4: Digest Endpoints

- [ ] Add `ProjectSummary`, `LanguageStats` types
- [ ] Implement `ProjectSummary(topN)` — aggregates across files and symbols tables, computes top-N by ref count
- [ ] Add `PackageSummary` type
- [ ] Implement `PackageSummary(packagePath, packageID)` — resolves path to ID if needed, gathers exported symbols, kind counts, dependency/dependent paths
- [ ] Unit tests: empty project, multi-language project, package with no dependents

## Phase 5: Integration & Documentation

- [ ] Verify all new methods are accessible via `Engine.Query()`
- [ ] Add golden test fixtures exercising discovery queries end-to-end
- [ ] Update `doc.go` package documentation to reflect new QueryBuilder surface

## Notes

- All new SQL queries should use parameterized queries to prevent injection
- Ref count computation: `SELECT COUNT(*) FROM resolved_references WHERE target_symbol_id = ?` — consider a CTE or subquery join for bulk enrichment to avoid N+1
- `PackageSummary` path resolution: `packagePath` is a file path prefix (e.g. `"internal/store"`). Resolution: query `files WHERE path LIKE 'internal/store/%'` → find package/module symbol in those files → use that symbol ID. This also determines "files in this package" for Dependencies/Dependents computation
- PathPrefix normalization: if the prefix does not end with `/`, append `/` before constructing the LIKE pattern (e.g. `"internal/store"` → `LIKE 'internal/store/%'`). This prevents `"internal/store"` from matching `"internal/store_utils/file.go"`
- RefCount at scale: for `ProjectSummary.TopSymbols` and large `Symbols` queries sorted by ref_count, a correlated subquery may be slow. If performance becomes an issue, consider a denormalized `ref_count` column on `symbols` updated during resolution, or a materialized view
