# Dependencies

## New Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/smacker/go-tree-sitter` | Go bindings for tree-sitter parsing library. Provides `Parser`, `Tree`, `Node`, `Query`, `QueryCursor` types for CST access. CGO-based, bundles C sources properly (unlike official bindings which have broken CGO includes as a Go module). |
| `github.com/smacker/go-tree-sitter/golang` | Tree-sitter grammar for Go |
| `github.com/smacker/go-tree-sitter/javascript` | Tree-sitter grammar for JavaScript |
| `github.com/smacker/go-tree-sitter/typescript/typescript` | Tree-sitter grammar for TypeScript |
| `github.com/smacker/go-tree-sitter/python` | Tree-sitter grammar for Python |
| `github.com/smacker/go-tree-sitter/rust` | Tree-sitter grammar for Rust |
| `github.com/smacker/go-tree-sitter/c` | Tree-sitter grammar for C |
| `github.com/smacker/go-tree-sitter/cpp` | Tree-sitter grammar for C++ |
| `github.com/smacker/go-tree-sitter/java` | Tree-sitter grammar for Java |
| `github.com/smacker/go-tree-sitter/php` | Tree-sitter grammar for PHP |
| `github.com/smacker/go-tree-sitter/ruby` | Tree-sitter grammar for Ruby |
| `github.com/risor-io/risor` | Risor scripting language runtime. Pure Go, embeddable. Used for language-specific extraction and resolution scripts. Key API: `risor.Eval()`, `risor.WithGlobals()`, `risor.WithLocalImporter()`. Compiles to bytecode, runs on lightweight VM. |
| `github.com/mattn/go-sqlite3` | SQLite3 driver for Go's `database/sql` interface. CGO-based, requires `CGO_ENABLED=1` and a C compiler. Supports WAL mode, transactions, and all standard SQL operations. |

## Rationale

### smacker/go-tree-sitter over tree-sitter/go-tree-sitter

`github.com/smacker/go-tree-sitter` is the community-maintained Go binding that bundles C sources correctly. The official `github.com/tree-sitter/go-tree-sitter` package has broken CGO includes when used as a Go module dependency (`#include "../../src/parser.c"` doesn't resolve). Validated in `.spikes/risor-treesitter/`. The smacker package is widely used, actively maintained, and bundles grammar packages as sub-packages (e.g., `github.com/smacker/go-tree-sitter/golang`).

### Risor over alternatives (Tengo, Expr, Starlark, Lua)

- **Go-native**: Pure Go, no CGO dependency beyond what tree-sitter/sqlite already require
- **Familiar syntax**: Hybrid of Go and Python; readable by Go/Python/TS developers
- **Clean embedding API**: `risor.Eval(ctx, source, risor.WithGlobals(...))` is the entire integration surface
- **No compile step**: Scripts are loaded and executed directly; LLMs can edit and re-run instantly
- **Go struct access**: Pass Go structs to scripts; scripts can call methods and access fields directly
- **Module system**: `WithLocalImporter` allows scripts to import from a directory, enabling shared utilities

### go-sqlite3 over modernc.org/sqlite

`go-sqlite3` is CGO-based which requires a C compiler, but we already need CGO for tree-sitter. Using CGO sqlite gives better performance and full SQLite feature support. `modernc.org/sqlite` (pure Go) would avoid CGO but adds no value when CGO is already required. `go-sqlite3` is also the most widely used and battle-tested Go SQLite driver.
