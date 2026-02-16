# Decisions

## 2026-02-15: Glob-style search over fuzzy search

**Context:** Search needs to support prefix, suffix, and infix wildcards on symbol names. Fuzzy search (Levenshtein, trigrams) was considered.

**Decision:** Start with glob-style patterns using `*` mapped to SQL `LIKE` with `%`. No fuzzy search in v1.

**Consequences:**
- (+) Zero new dependencies — SQLite `LIKE` is built in
- (+) Predictable, deterministic results — no scoring ambiguity
- (+) Fast with existing indexes (prefix matches can use index; suffix/infix do full scan but on indexed column)
- (-) No typo tolerance — `"Animl"` won't match `"Animal"`
- (+) Case-insensitive matching by default (SQLite LIKE for ASCII) — useful for symbol discovery
- Fuzzy search can be layered on later if needed (trigram extension or Go-side filtering)

## 2026-02-15: No batched graph traversal in v1

**Context:** Considered adding `TypeHierarchy`, `DependencyGraph`, and `CallGraph` methods with configurable depth. These would batch multiple levels of relationship traversal into a single call.

**Decision:** Defer batched graph traversal. Consumers walk the tree iteratively using existing single-level APIs (`Implementations`, `Callers`, `Callees`, `Dependencies`, `Dependents`), which all return symbol IDs that feed back into further queries.

**Consequences:**
- (+) Dramatically simpler API surface — no graph node types, depth caps, or cycle handling
- (+) Existing single-level methods already cover the building blocks
- (+) Avoids premature abstraction around traversal direction semantics
- (-) Multi-hop analysis requires multiple round trips from the consumer
- Can add a batched `Expand(symbolID, relationships, depth)` method later if round-trip cost becomes an issue

## 2026-02-15: Reference count as importance signal

**Context:** When sorting or summarizing symbols, need a way to surface "important" symbols first.

**Decision:** Use `COUNT(resolved_references WHERE target_symbol_id = ?)` as a reference count proxy for symbol importance. Exposed as `RefCount` on `SymbolResult` and as a sort option.

**Consequences:**
- (+) Natural measure — most-referenced types/functions define the architecture
- (+) Computed from existing data, no new extraction needed
- (+) Works across all languages
- (-) Slightly more expensive queries (subquery or join for count)
- (-) Newly added symbols have zero references even if important
- Can add more signals later (e.g., symbol kind weighting, export status)

## 2026-02-15: Use json_each for modifier filtering

**Context:** Modifiers are stored as JSON arrays in the `symbols.modifiers` column (e.g. `["async","static"]`). The `SymbolFilter.Modifiers` filter requires "symbol must have ALL of these modifiers." Two approaches: `LIKE '%"async"%'` (fragile, could match substrings) or SQLite JSON1 `json_each` function.

**Decision:** Use `json_each(modifiers)` to filter modifiers. For each required modifier, add a subquery: `EXISTS (SELECT 1 FROM json_each(symbols.modifiers) WHERE value = ?)`. The `go-sqlite3` driver includes JSON1 by default.

**Consequences:**
- (+) Correct — no false matches from substring containment
- (+) Clean SQL — standard SQLite JSON1 extension, no string hacks
- (+) Handles edge cases (modifier names that are substrings of other modifiers)
- (-) Slightly more complex SQL than simple LIKE
- (-) Implicit dependency on JSON1 being compiled into go-sqlite3 (it is by default)

## 2026-02-15: PathPrefix for package scoping instead of PackageID or PackagePath

**Context:** Package-scoped queries need to identify a package. Considered `PackagePath` (string-based, but `symbols.name` stores local names making resolution ambiguous) and `PackageID` (symbol ID-based, but relies on extraction scripts correctly setting `parent_symbol_id` to the package symbol — behavior varies across languages).

**Decision:** Use `PathPrefix *string` on `SymbolFilter` as the primary package-scoping mechanism. It matches against `files.path` via JOIN, which works reliably across all languages regardless of how extraction scripts handle `parent_symbol_id`. `ParentID` is available for structural parent filtering when needed. `PackageSummary` accepts a file path prefix for the same reason.

**Consequences:**
- (+) Works for every language — no dependency on extraction script behavior
- (+) File paths are always available and unambiguous
- (+) Simple implementation — `WHERE files.path LIKE ?` with the prefix
- (+) Natural for consumers — people think in terms of directories
- (-) Cannot distinguish two packages in the same directory (rare edge case)
