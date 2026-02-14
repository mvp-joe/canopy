# Dependencies

## New Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/tree-sitter/go-tree-sitter` | Go bindings for tree-sitter parsing library. Provides `Parser`, `Tree`, `Node`, `Query`, `TreeCursor` types for CST access. CGO-based, requires `Close()` calls on all allocated objects. |
| `github.com/tree-sitter/tree-sitter-go/bindings/go` | Tree-sitter grammar for Go |
| `github.com/tree-sitter/tree-sitter-javascript/bindings/go` | Tree-sitter grammar for JavaScript |
| `github.com/tree-sitter/tree-sitter-typescript/bindings/go` | Tree-sitter grammar for TypeScript |
| `github.com/tree-sitter/tree-sitter-python/bindings/go` | Tree-sitter grammar for Python |
| `github.com/tree-sitter/tree-sitter-rust/bindings/go` | Tree-sitter grammar for Rust |
| `github.com/tree-sitter/tree-sitter-c/bindings/go` | Tree-sitter grammar for C |
| `github.com/tree-sitter/tree-sitter-cpp/bindings/go` | Tree-sitter grammar for C++ |
| `github.com/tree-sitter/tree-sitter-java/bindings/go` | Tree-sitter grammar for Java |
| `github.com/tree-sitter/tree-sitter-php/bindings/go` | Tree-sitter grammar for PHP |
| `github.com/tree-sitter/tree-sitter-ruby/bindings/go` | Tree-sitter grammar for Ruby |
| `github.com/risor-io/risor` | Risor scripting language runtime. Pure Go, embeddable. Used for language-specific extraction and resolution scripts. Key API: `risor.Eval()`, `risor.WithGlobals()`, `risor.WithLocalImporter()`. Compiles to bytecode, runs on lightweight VM. |
| `github.com/mattn/go-sqlite3` | SQLite3 driver for Go's `database/sql` interface. CGO-based, requires `CGO_ENABLED=1` and a C compiler. Supports WAL mode, transactions, and all standard SQL operations. |

## Rationale

### tree-sitter/go-tree-sitter over smacker/go-tree-sitter

`github.com/tree-sitter/go-tree-sitter` is the official tree-sitter Go binding maintained by the tree-sitter organization. The `smacker/go-tree-sitter` package is an older community binding. The official package has better alignment with upstream tree-sitter releases and grammar packages. Grammar packages use the `github.com/tree-sitter/tree-sitter-{lang}/bindings/go` import path convention.

### Risor over alternatives (Tengo, Expr, Starlark, Lua)

- **Go-native**: Pure Go, no CGO dependency beyond what tree-sitter/sqlite already require
- **Familiar syntax**: Hybrid of Go and Python; readable by Go/Python/TS developers
- **Clean embedding API**: `risor.Eval(ctx, source, risor.WithGlobals(...))` is the entire integration surface
- **No compile step**: Scripts are loaded and executed directly; LLMs can edit and re-run instantly
- **Go struct access**: Pass Go structs to scripts; scripts can call methods and access fields directly
- **Module system**: `WithLocalImporter` allows scripts to import from a directory, enabling shared utilities

### go-sqlite3 over modernc.org/sqlite

`go-sqlite3` is CGO-based which requires a C compiler, but we already need CGO for tree-sitter. Using CGO sqlite gives better performance and full SQLite feature support. `modernc.org/sqlite` (pure Go) would avoid CGO but adds no value when CGO is already required. `go-sqlite3` is also the most widely used and battle-tested Go SQLite driver.
