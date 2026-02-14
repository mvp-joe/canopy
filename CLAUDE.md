# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Canopy is a Go library for deterministic, scope-aware semantic code analysis built on tree-sitter. It bridges tree-sitter's concrete syntax tree and full LSP semantic understanding. The Go core is intentionally thin — it provides tree-sitter parsing, a SQLite store, and a Risor scripting runtime. All language-specific logic (extraction and resolution) lives in Risor scripts that receive tree-sitter objects and the Store directly, with no wrappers. The primary consumer is project-cortex, a Go-based MCP server for AI coding assistants. Target: >90% accuracy on core semantic operations across 8 languages (Go, TS/JS, Python, Rust, C/C++, Java, PHP, Ruby).

**Current status:** Planning phase. Spec is complete, spike validated Risor ↔ tree-sitter interop, implementation not yet started.

## Build Commands

```bash
go build            # Build (requires CGO_ENABLED=1 and a C compiler)
go test ./...       # Run all tests
go test ./internal/store/...  # Run tests for a specific package
go test -run TestName ./...   # Run a single test
```

CGO is required for both tree-sitter (C bindings) and SQLite (go-sqlite3).

## Architecture

### Two-Phase Pipeline

```
Source Files → Engine → tree-sitter Parse → Extraction Scripts → SQLite → Resolution Scripts → SQLite → QueryBuilder → project-cortex
```

1. **Extract**: For each changed file, parse with tree-sitter, run language-specific Risor extraction script, write to extraction tables (symbols, scopes, references, imports, etc.)
2. **Resolve**: For each language with data, run Risor resolution script against extraction tables, write to resolution tables (resolved_references, implementations, call_graph, etc.)

### Core Components

- **Engine** (`canopy/`) — Top-level orchestrator. File discovery, change detection (hash-based), script dispatch, query API for cortex.
- **Store** (`internal/store/`) — SQLite data layer. 16 tables (10 extraction + 6 resolution), WAL mode, blast radius methods for incremental re-resolution.
- **Risor Runtime** (`internal/runtime/`) — Embeds Risor VM. Exposes globals to scripts: `parse(path, language)`, `node_text(node)`, `query(pattern, node)`, `db` (Store), `log`.
- **Extraction Scripts** (`scripts/extract/{language}.risor`) — One per language. Walk CST via tree-sitter, write to extraction tables.
- **Resolution Scripts** (`scripts/resolve/{language}.risor`) — One per language. Query extraction tables, write resolution tables. No tree-sitter access.
- **Shared Utilities** (`scripts/lib/`) — Common Risor code across scripts.

### Key Design Decisions

- **Risor scripts get real Go objects** — tree-sitter Tree/Node and Store are passed directly, not wrapped. Host functions (`parse`, `node_text`, `query`) exist only where Risor's proxy system has limitations ([]byte args, free constructor functions, cursor iteration).
- **smacker/go-tree-sitter** over official go-tree-sitter — official bindings have broken CGO includes as a Go module. Validated in `.spikes/risor-treesitter/`.
- **go-sqlite3** over modernc.org/sqlite — CGO already required for tree-sitter, so CGO sqlite gives better performance.

## Risor Language Reference

`risor-context-medium.txt` in the repo root is a comprehensive Risor language reference. Read this file before writing or modifying any `.risor` scripts — it covers syntax, builtins, types, and idioms needed to write correct Risor code.

## Spec and Reference

- **Spec root:** `specs/2026-02-14_semantic-bridge/` — 8 files covering architecture, schema (16 tables, v3), interfaces, 7-phase implementation plan, test strategy, dependencies, decisions.
- **Spike:** `.spikes/risor-treesitter/` — Proof-of-concept validating Risor ↔ tree-sitter interop. Run with `cd .spikes/risor-treesitter && go build -o spike && ./spike`.
- **LSP references:** `.scratch/` — Cloned LSP implementations (gopls, tsserver, pyright, etc.) for oracle verification during development. Populated via `/clone-lsps` command.

## Testing Strategy

- **Unit tests:** Go tests for Store CRUD, schema migration, tree-sitter host functions, extraction/resolution per language.
- **Golden tests:** `testdata/{language}/level-{N}-{name}/` directories with `src/` and `golden.json`. Three tiers: extraction (Tier 1), simple resolution (Tier 2), full resolution (Tier 3). Run via `canopy test testdata/`.
- **MCP verification:** Dev-time only (not CI). LLM runs canopy, queries real LSP via MCP, iterates on Risor scripts until >90% accuracy, then writes golden fixtures.
