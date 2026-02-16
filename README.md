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

The `QueryBuilder` provides core operations:

| Operation | Description |
|---|---|
| `SymbolAt(file, line, col)` | Find the symbol at a position (narrowest match) |
| `DefinitionAt(file, line, col)` | Go-to-definition: find where a symbol at a position is defined |
| `ReferencesTo(symbolID)` | Find-references: all locations referencing a symbol |
| `Implementations(symbolID)` | Find types implementing an interface or trait |
| `Callers(symbolID)` | Call graph: who calls this function |
| `Callees(symbolID)` | Call graph: what does this function call |
| `Dependencies(file)` | Imports: what does this file depend on |
| `Dependents(module)` | Reverse imports: who depends on this module |

All positions are 0-based (line and column), matching tree-sitter's native convention.

### Discovery API

Additional operations for browsing and searching indexed data:

| Operation | Description |
|---|---|
| `Symbols(file, filter, pagination, sort)` | List symbols with optional filtering by kind, visibility, path prefix |
| `SearchSymbols(pattern, filter, pagination, sort)` | Glob-search symbol names (`*` wildcard) |
| `Files(pagination)` | List indexed files |
| `Packages(pagination)` | List packages |
| `ProjectSummary()` | Aggregate stats: languages, files, symbols, references |
| `PackageSummary(path)` | Per-package breakdown |

All discovery methods support pagination (default limit 50, max 500) and sorting.

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

## CLI

Canopy includes a command-line tool for indexing and querying.

### Index

```bash
canopy index [path]              # Index a project (extraction + resolution)
canopy index --force [path]      # Delete DB and reindex from scratch
canopy index --languages go,rust # Index specific languages only
```

### Query

```bash
canopy query definition main.go 9 5        # Go-to-definition
canopy query symbol-at main.go 9 5         # Symbol at position
canopy query references main.go 9 5        # Find references (position)
canopy query references --symbol 42        # Find references (symbol ID)
canopy query callers main.go 9 5           # Who calls this function
canopy query callees main.go 9 5           # What does this function call
canopy query implementations main.go 9 5   # Interface implementations
canopy query symbols --kind function       # List symbols by kind
canopy query search "Parse*"               # Glob-search symbol names
canopy query files                         # List indexed files
canopy query packages                      # List packages
canopy query summary                       # Project-wide stats
canopy query package-summary mypackage     # Per-package stats
canopy query deps main.go                  # File dependencies
canopy query dependents mypackage          # Reverse import lookup
```

All output defaults to JSON. Use `--format text` for human-readable output. Query commands support `--limit`, `--offset`, `--sort`, and `--order` for pagination.

The database defaults to `.canopy/index.db` relative to the git repository root. Override with `--db`.

## Building

Requires Go 1.25+, CGO, and a C compiler (for tree-sitter and SQLite bindings).

```bash
go build ./...
go test ./...
```

Risor scripts are embedded in the binary at build time. For development, use `--scripts-dir` to load scripts from disk instead.

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

## License

Apache 2.0 — see [LICENSE](LICENSE).
