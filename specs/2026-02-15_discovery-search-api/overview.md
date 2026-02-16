# Discovery & Search API

## Summary

Extends the QueryBuilder with discovery, search, and summary capabilities so that consumers (project-cortex MCP tools, library users, AI agents) can explore an unfamiliar codebase top-down. The current QueryBuilder is position-driven ("what's at line 42?"); this adds the entry-point layer ("what's in this repo?") that feeds symbol IDs into the existing navigation APIs.

## Goals

- Enumeration: list symbols filtered by kind, visibility, file, package, modifiers, path prefix
- Search: glob-style pattern matching (`Anim*`, `*mal`, `Anim*l`) with `*` mapped to SQL `%`
- Digest endpoints: project overview and package summary for quick orientation
- Pagination and sorting on all list/search endpoints (offset+limit, sort by name/kind/reference-count)
- Path prefix scoping for package-level queries (works across all languages without extraction script assumptions)
- All results include symbol IDs for seamless flow into existing navigation APIs (`ReferencesTo`, `Implementations`, `Callers`, etc.)
- Convenience methods for common queries: `Files` (by path prefix/language), `Packages` (by path prefix)

## Non-Goals

- Full-text search over source code contents (this searches symbol names, not file contents)
- Fuzzy/typo-tolerant search (start with exact glob; fuzzy can be added later)
- Replacing existing position-based QueryBuilder methods (those stay as-is)
- GraphQL-style arbitrary query composition (we expose fixed, well-designed endpoints)
- Batched graph traversal (consumers walk the tree iteratively using existing single-level APIs like `Implementations`, `Callers`, `Callees`, `Dependencies`, `Dependents`)
- Real-time / streaming results

## Depends On

- 2026-02-14_semantic-bridge

## Current Status

Planning

## Key Files

- [interface.md](interface.md) - New types, filter structs, and QueryBuilder method signatures
- [implementation.md](implementation.md) - Phased build plan
- [tests.md](tests.md) - Test specifications
- [decisions.md](decisions.md) - Glob-vs-fuzzy, graph deferral, ref count, json_each, path resolution choices
