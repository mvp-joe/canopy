# Canopy CLI Reference

Canopy is a deterministic, scope-aware semantic code analysis CLI tool. It indexes source code using tree-sitter and produces a SQLite database for semantic queries.

**Binary**: `canopy` (installed via `go install github.com/jward/canopy/cmd/canopy@latest` or built locally)

## Supported Languages

Canopy supports 10 languages:
- Go
- TypeScript
- JavaScript
- Python
- Rust
- C
- C++ (cpp)
- Java
- PHP
- Ruby

## Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--db` | string | `.canopy/index.db` relative to repo root | Database path |
| `--format` | string | `json` | Output format: `json` or `text` |

The `--db` flag accepts both absolute and relative paths. Relative paths are resolved relative to the detected repository root.

The default database location is `<repo-root>/.canopy/index.db` where repo root is found by walking up from the target directory looking for a `.git` directory. If no `.git` is found, the target directory itself is used as the root.

## Commands

### `canopy index [path]`

Index a repository for semantic analysis. Parses source files with tree-sitter, runs extraction and resolution scripts, and writes results to the SQLite database.

**Arguments:**
- `path` (optional): Directory to index. Defaults to current directory (`.`).

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Delete database and reindex from scratch |
| `--languages` | string | `""` | Comma-separated language filter (e.g. `go,typescript`) |
| `--scripts-dir` | string | `""` | Load scripts from disk path instead of embedded |
| `--parallel` | bool | `false` | Enable parallel extraction (worker pool with batched writes) |

**Behavior:**
- Creates `.canopy/` directory if it doesn't exist
- With `--force`: deletes existing database before indexing
- Auto-detects script changes: if embedded scripts differ from what built the DB, automatically wipes and rebuilds (equivalent to `--force`)
- Prints timing summary to stderr: `Indexed <path> in <total> (extract: <time>, resolve: <time>)`
- Prints database path to stderr: `Database: <path>`

**Exit codes:**
- 0: Success
- 1: Error (invalid path, indexing failure, etc.)

**Examples:**
```bash
canopy index                        # Index current directory
canopy index /path/to/project       # Index specific directory
canopy index --force                # Force full reindex
canopy index --languages go,python  # Only index Go and Python files
canopy index --parallel             # Use parallel extraction
canopy index --db /tmp/my.db        # Custom database location
```

### `canopy query`

Parent command for all query subcommands. All query commands share these flags:

**Persistent Query Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--limit` | int | `50` | Pagination limit (max 500) |
| `--offset` | int | `0` | Pagination offset |
| `--sort` | string | `""` | Sort field: `name`, `kind`, `file`, `ref_count` |
| `--order` | string | `asc` | Sort order: `asc` or `desc` |

**Note:** All line and column numbers in queries and results are **0-based** (matching tree-sitter convention).

### `canopy query symbol-at <file> <line> <col>`

Find the symbol at a specific position in a file.

**Arguments:** `<file>` `<line>` `<col>` (all required, exactly 3 args)

**JSON output:**
```json
{
  "command": "symbol-at",
  "results": {
    "id": 42,
    "name": "MyFunction",
    "kind": "function",
    "visibility": "public",
    "file": "/absolute/path/to/file.go",
    "start_line": 10,
    "start_col": 0,
    "end_line": 15,
    "end_col": 1
  },
  "total_count": 1
}
```

Returns `"results": null` if no symbol is found at the position.

### `canopy query definition <file> <line> <col>`

Find the definition of the symbol at a position. Follows references to their definitions.

**Arguments:** `<file>` `<line>` `<col>` (all required, exactly 3 args)

**JSON output:**
```json
{
  "command": "definition",
  "results": [
    {
      "file": "/absolute/path/to/definition.go",
      "start_line": 5,
      "start_col": 0,
      "end_line": 5,
      "end_col": 12,
      "symbol_id": 42
    }
  ],
  "total_count": 1
}
```

### `canopy query references [<file> <line> <col>]`

Find all references to a symbol.

**Arguments:** Either `<file> <line> <col>` OR `--symbol <id>` (one is required)

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--symbol` | int64 | `0` | Symbol ID to query (alternative to positional args) |

**JSON output:**
```json
{
  "command": "references",
  "results": [
    {
      "file": "/path/to/file.go",
      "start_line": 20,
      "start_col": 4,
      "end_line": 20,
      "end_col": 16,
      "symbol_id": 42
    }
  ],
  "total_count": 2
}
```

### `canopy query callers [<file> <line> <col>]`

Find callers of a function.

**Arguments:** Either `<file> <line> <col>` OR `--symbol <id>`

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--symbol` | int64 | `0` | Symbol ID to query |

**JSON output:**
```json
{
  "command": "callers",
  "results": [
    {
      "caller_id": 10,
      "caller_name": "main",
      "callee_id": 42,
      "callee_name": "MyFunction",
      "file": "/path/to/file.go",
      "line": 25,
      "col": 4
    }
  ],
  "total_count": 1
}
```

### `canopy query callees [<file> <line> <col>]`

Find functions called by a function.

**Arguments:** Either `<file> <line> <col>` OR `--symbol <id>`

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--symbol` | int64 | `0` | Symbol ID to query |

**JSON output:** Same structure as `callers`.

### `canopy query implementations [<file> <line> <col>]`

Find implementations of an interface.

**Arguments:** Either `<file> <line> <col>` OR `--symbol <id>`

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--symbol` | int64 | `0` | Symbol ID to query |

**JSON output:** Same structure as `definition` (array of locations with optional symbol_id).

### `canopy query symbols`

List symbols with optional filters.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--kind` | string | `""` | Filter by symbol kind (e.g. `function`, `type`, `class`, `method`, `variable`, `interface`, `module`, `package`) |
| `--file` | string | `""` | Filter by file path |
| `--visibility` | string | `""` | Filter by visibility: `public` or `private` |
| `--path-prefix` | string | `""` | Filter by file path prefix |

Plus the shared pagination and sort flags.

**JSON output:**
```json
{
  "command": "symbols",
  "results": [
    {
      "id": 1,
      "name": "MyFunc",
      "kind": "function",
      "visibility": "public",
      "file": "/path/to/file.go",
      "start_line": 10,
      "start_col": 0,
      "end_line": 20,
      "end_col": 1,
      "ref_count": 5,
      "external_ref_count": 3,
      "internal_ref_count": 2
    }
  ],
  "total_count": 42
}
```

### `canopy query search <pattern>`

Search symbols by glob pattern. Use `*` as wildcard.

**Arguments:** `<pattern>` (required, exactly 1 arg)

**Examples:**
```bash
canopy query search "Get*"        # All symbols starting with Get
canopy query search "*User*"      # All symbols containing User
canopy query search "*.go"        # Would match symbols with .go in name
```

**JSON output:** Same structure as `symbols`.

### `canopy query files`

List indexed files.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `""` | Filter by language |
| `--prefix` | string | `""` | Filter by path prefix |

**JSON output:**
```json
{
  "command": "files",
  "results": [
    {
      "id": 1,
      "path": "/absolute/path/to/file.go",
      "language": "go",
      "line_count": 150
    }
  ],
  "total_count": 10
}
```

### `canopy query packages`

List packages/modules/namespaces.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--prefix` | string | `""` | Filter by path prefix |

**JSON output:** Same structure as `symbols`.

### `canopy query summary`

Show project-level summary statistics.

**JSON output:**
```json
{
  "command": "summary",
  "results": {
    "languages": [
      {
        "language": "go",
        "file_count": 5,
        "line_count": 500,
        "symbol_count": 50,
        "kind_counts": {
          "function": 20,
          "type": 10,
          "variable": 20
        }
      }
    ],
    "package_count": 3,
    "top_symbols": [
      {
        "id": 1,
        "name": "MostReferenced",
        "kind": "function",
        "visibility": "public",
        "file": "/path/to/file.go",
        "start_line": 10,
        "start_col": 0,
        "end_line": 20,
        "end_col": 1,
        "ref_count": 15,
        "external_ref_count": 12,
        "internal_ref_count": 3
      }
    ]
  }
}
```

### `canopy query package-summary <path-or-id>`

Show summary for a specific package.

**Arguments:** Either a path prefix (string) or a symbol ID (integer). Exactly 1 arg required.

**JSON output:**
```json
{
  "command": "package-summary",
  "results": {
    "symbol": { "id": 1, "name": "pkg", "kind": "package", ... },
    "path": "github.com/example/pkg",
    "file_count": 3,
    "exported_symbols": [ ... ],
    "kind_counts": { "function": 5, "type": 2 },
    "dependencies": ["fmt", "os"],
    "dependents": ["main"]
  }
}
```

### `canopy query deps <file>`

List imports/dependencies of a file.

**Arguments:** `<file>` (required, exactly 1 arg)

**JSON output:**
```json
{
  "command": "deps",
  "results": [
    {
      "file_id": 1,
      "file_path": "/path/to/file.go",
      "source": "fmt",
      "kind": "package"
    }
  ],
  "total_count": 3
}
```

### `canopy query dependents <source>`

List files that import a given source.

**Arguments:** `<source>` (required, exactly 1 arg). This is the import source string (e.g., `fmt`, `./utils`, `react`).

**JSON output:** Same structure as `deps`.

## Error Output

### JSON format (default)

Errors are returned as a JSON envelope on stdout:
```json
{
  "command": "symbol-at",
  "error": "database not found: .canopy/index.db (run 'canopy index' first)"
}
```

The process also exits with code 1 on error.

### Text format

Errors are printed to stderr:
```
Error: database not found: .canopy/index.db (run 'canopy index' first)
```

## Text Format Output

When `--format text` is used:

- **Locations** (definition, references, implementations): One per line as `file:line:col`
- **Symbols** (symbols, search, packages): Aligned columns: `ID  NAME  KIND  VISIBILITY  FILE  LINE` (with ref count columns if any symbol has refs)
- **Call edges** (callers, callees): Aligned columns: `CALLER  CALLEE  FILE  LINE  COL`
- **Imports** (deps, dependents): Aligned columns: `SOURCE  KIND  FILE`
- **Files**: Aligned columns: `ID  PATH  LANGUAGE`
- **Summary**: Human-readable project overview
- **Package summary**: Human-readable package overview with sections for kinds, exported symbols, dependencies, dependents

Pagination footer: `Showing X of Y results` appears when results are truncated.
