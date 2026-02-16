# CLI Tool

## Summary

Adds a standalone CLI binary (`cmd/canopy/`) to the canopy library, enabling any user to index a repository and run semantic queries from the command line. The CLI is built with Cobra and exposes all QueryBuilder operations as subcommands with JSON output (for LLM chaining) and text output (for humans). A prerequisite script embedding layer allows the CLI to ship as a single binary with all Risor scripts embedded via `go:embed`, while preserving disk-based script loading for library consumers and development workflows.

## Goals

- Single binary that can index any repo and query it without writing Go code
- All QueryBuilder methods accessible via CLI subcommands (position-based and discovery/search)
- JSON output by default with symbol IDs to enable LLM query chaining
- Human-readable text output via `--format text`
- Embedded Risor scripts (`go:embed`) so the binary is self-contained
- `--scripts-dir` override for development (edit scripts, re-run immediately)
- Risor `import` statement support via `WithImporter` wiring (needed for `scripts/lib/` shared modules)
- `fs.FS` abstraction in the Engine/Runtime for library consumers who want to embed scripts
- `.canopy/index.db` default database location (repo-root detection via `.git/`)
- 0-based line and column positions throughout (matching tree-sitter convention)

## Non-Goals

- Interactive REPL or shell mode
- Watch mode or file-system event-driven re-indexing
- Remote server or daemon mode
- GUI or TUI
- Tab completion generation (can be added later via Cobra's built-in support)
- Custom output templates (JSON and text are sufficient)
- Replacing project-cortex as the primary consumer (CLI is a standalone tool, not a replacement)

## Depends On

- 2026-02-14_semantic-bridge
- 2026-02-15_discovery-search-api

## Current Status

Complete

## Key Files

- [interface.md](interface.md) - Engine options, Runtime changes, CLI command signatures
- [implementation.md](implementation.md) - Four-phase build plan
- [tests.md](tests.md) - Test specifications
- [dependencies.md](dependencies.md) - Cobra dependency
- [decisions.md](decisions.md) - DB location, script loading strategy, Cobra rationale
