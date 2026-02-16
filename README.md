<p align="center">
  <img src="assets/canopy.png" alt="Canopy" width="400" />
</p>

# Canopy

Deterministic, scope-aware semantic code analysis built on [tree-sitter](https://tree-sitter.github.io/tree-sitter/). Canopy bridges tree-sitter's concrete syntax tree and full LSP semantic understanding, targeting >90% accuracy on core semantic operations across 10 languages.

## Supported Languages

Go, TypeScript, JavaScript, Python, Rust, C, C++, Java, PHP, Ruby

## How It Works

Canopy operates as a two-phase pipeline:

```
Source Files → Engine → tree-sitter Parse → Extraction Scripts → SQLite → Resolution Scripts → SQLite → QueryBuilder
```

1. **Extract** — For each source file, parse with tree-sitter, run a language-specific [Risor](https://risor.io) extraction script, and write symbols, scopes, references, imports, and type information to SQLite.
2. **Resolve** — For each language with indexed data, run a Risor resolution script that cross-references extraction data to produce resolved references, interface implementations, call graph edges, and extension bindings.

The Go core is intentionally thin — it provides tree-sitter parsing, a SQLite store, and a Risor scripting runtime. All language-specific logic lives in Risor scripts that receive tree-sitter objects and the Store directly, with no wrappers.

## Query API

The `QueryBuilder` provides seven core operations:

| Operation | Description |
|---|---|
| `DefinitionAt(file, line, col)` | Go-to-definition: find where a symbol at a position is defined |
| `ReferencesTo(file, line, col)` | Find-references: all locations referencing a symbol |
| `Implementations(file, line, col)` | Find types implementing an interface or trait |
| `Callers(file, line, col)` | Call graph: who calls this function |
| `Callees(file, line, col)` | Call graph: what does this function call |
| `Dependencies(file)` | Imports: what does this file depend on |
| `Dependents(module)` | Reverse imports: who depends on this module |

All positions are 0-based (line and column), matching tree-sitter's native convention.

## Usage

```go
e, err := canopy.New("canopy.db", "path/to/scripts")
if err != nil { ... }
defer e.Close()

ctx := context.Background()
err = e.IndexDirectory(ctx, "path/to/project")
err = e.Resolve(ctx)

q := e.Query()
locs, err := q.DefinitionAt("main.go", 9, 5)
```

### Incremental Indexing

Canopy detects unchanged files via content hashing and skips them. When a file changes, it computes a blast radius (which other files are affected) and selectively re-resolves only affected languages.

## Building

Requires Go 1.25+, CGO, and a C compiler (for tree-sitter and SQLite bindings).

```bash
go build ./...
go test ./...
```

## Architecture

```
canopy/                    # Engine — orchestrator, file discovery, change detection
internal/store/            # SQLite data layer (16 tables, WAL mode)
internal/runtime/          # Risor VM embedding, host functions
scripts/extract/{lang}.risor  # Language-specific extraction scripts
scripts/resolve/{lang}.risor  # Language-specific resolution scripts
scripts/lib/               # Shared Risor utilities
```

## LSP References

For oracle verification during development, these LSP implementations can be cloned into `.scratch/`:

| Language | LSP |
|---|---|
| Go | `go install golang.org/x/tools/gopls@latest` |
| Rust | `rustup component add rust-analyzer` |
| C/C++ | Install `clangd` via system package manager |
| Java | Eclipse JDT Language Server |
| Ruby | `gem install solargraph` |
| PHP | `npm install -g intelephense` |
