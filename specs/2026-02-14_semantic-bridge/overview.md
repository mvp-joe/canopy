# Canopy Semantic Bridge

## Summary

Canopy is a Go library that provides deterministic, scope-aware semantic code analysis on top of tree-sitter. It fills the gap between tree-sitter's concrete syntax tree and full LSP semantic understanding. The Go core is intentionally thin — it provides tree-sitter parsing, a SQLite store, and a Risor runtime. All language-specific logic (both extraction and resolution) lives in Risor scripts that receive tree-sitter objects and the Store directly, with no wrappers. The primary consumer is project-cortex, a Go-based MCP server for AI coding assistants. Canopy targets >90% accuracy on core semantic operations across eight initial languages without requiring a running LSP at runtime.

## Goals

- Deterministic, scope-aware semantic analysis built on tree-sitter
- Multi-language support: Go, TypeScript/JavaScript, Python, Rust, C/C++, Java, PHP, Ruby
- SQLite-backed state with no ASTs held in memory after extraction
- All language-specific logic in Risor scripts — both extraction and resolution
- Thin Go core: tree-sitter objects and Store exposed directly to Risor, no wrappers
- LLM-driven development workflow with LSP oracle verification at dev time
- Integration with project-cortex as the primary consumer library
- Schema designed to support 13+ languages (including C#, Zig, Kotlin, Swift, Objective-C)

## Non-Goals

- Full LSP replacement (we target >90% accuracy on core operations, not 100%)
- Runtime LLM dependency (scripts are deterministic authored code, not LLM-generated at runtime)
- Translation of Risor scripts to Go (scripts stay as Risor unless perf demands otherwise)
- Complete type checking or type inference (we do structural matching, not full type system)
- IDE integration or language server protocol implementation (cortex handles that layer)
- Wrapping tree-sitter or Store with abstraction layers (scripts get the real objects)

## Current Status

Planning

## Key Files

- [interface.md](interface.md) - Type definitions: parse, query, Store, domain types, Engine, QueryBuilder
- [schema.md](schema.md) - Full database schema (extraction + resolution tables)
- [system-components.md](system-components.md) - Go core, Risor scripts, testing harness
- [implementation.md](implementation.md) - Seven-phase implementation plan
- [tests.md](tests.md) - Test specifications across all levels
- [dependencies.md](dependencies.md) - New Go dependencies
- [decisions.md](decisions.md) - Key architectural decisions and rationale
