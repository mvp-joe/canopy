# Project Status

> Last updated: 2026-02-17

## Current Focus

Resolution accuracy across 10 languages. Core pipeline, query API, and public API surface are solid. The next push is finding and fixing resolution gaps in the Risor scripts — deeper type hierarchies, re-exports, method dispatch, and language-specific edge cases.

## Active Work

None currently in progress.

## Up Next

1. Deeper resolution testing — work through scenarios in `TESTING-ROADMAP.md` starting with high-priority items
2. JavaScript coverage — only 2 golden levels exist, no extraction-only basics
3. C++ call graph and implementation tests — 7 resolution levels but all references-only

## Parked

- MCP verification workflow — designed for LLM-driven accuracy iteration against real LSPs, not needed until resolution scripts plateau

## Session Log

### 2026-02-17
- Hid `Store` from public API (removed type alias + `Engine.Store()` method)
- Added stale file cleanup to `IndexDirectory` (removes orphaned DB records on file deletion / branch switch)
- Moved completed specs (`query-api-v2`, `public-api-types`) to `specs/implemented/`
- Moved golden test roadmap from STATUS.md to `TESTING-ROADMAP.md`, added exercise session gap analysis
