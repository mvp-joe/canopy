// Package canopy provides deterministic, scope-aware semantic code analysis
// built on tree-sitter. It bridges tree-sitter's concrete syntax tree and full
// LSP semantic understanding for 9 languages: Go, TypeScript, JavaScript,
// Python, Rust, C, C++, Java, PHP, and Ruby.
//
// # Pipeline
//
// Canopy operates in two phases:
//
//  1. Extract: For each source file, parse with tree-sitter, run a
//     language-specific Risor extraction script, and write symbols, scopes,
//     references, imports, and type information to SQLite.
//
//  2. Resolve: For each language with indexed data, run a Risor resolution
//     script that cross-references extraction data to produce resolved
//     references, interface implementations, call graph edges, and extension
//     bindings.
//
// # Usage
//
// Create an Engine, index source files, resolve, and query:
//
//	e, err := canopy.New("canopy.db", "path/to/scripts")
//	if err != nil { ... }
//	defer e.Close()
//
//	ctx := context.Background()
//	err = e.IndexDirectory(ctx, "path/to/project")
//	err = e.Resolve(ctx)
//
//	q := e.Query()
//	locs, err := q.DefinitionAt("main.go", 10, 5)
//
// # Query API
//
// The [QueryBuilder] returned by [Engine.Query] provides seven operations:
//
//   - [QueryBuilder.DefinitionAt] — Go-to-definition: find where a symbol at a
//     position is defined.
//   - [QueryBuilder.ReferencesTo] — Find-references: all locations referencing a
//     symbol.
//   - [QueryBuilder.Implementations] — Find types implementing an interface or
//     trait.
//   - [QueryBuilder.Callers] — Call graph: who calls this function.
//   - [QueryBuilder.Callees] — Call graph: what does this function call.
//   - [QueryBuilder.Dependencies] — Imports: what does this file depend on.
//   - [QueryBuilder.Dependents] — Reverse imports: who depends on this module.
//
// # Incremental Indexing
//
// [Engine.IndexFiles] detects unchanged files via content hashing and skips
// them. When a file changes, canopy computes a blast radius (which other files
// are affected by the change) and selectively re-resolves only affected
// languages. Use [WithLanguages] to restrict which languages the Engine
// processes.
//
// # Scripts
//
// Language-specific logic lives in Risor scripts under the scripts directory:
//
//   - scripts/extract/{language}.risor — extraction scripts
//   - scripts/resolve/{language}.risor — resolution scripts
//   - scripts/lib/ — shared utilities
//
// Scripts receive tree-sitter objects and the SQLite Store directly via host
// functions. See the internal/runtime package for the full set of globals
// exposed to scripts.
package canopy
