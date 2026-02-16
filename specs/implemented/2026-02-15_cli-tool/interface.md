# Interface Definitions

## Engine Options

New option for providing an `fs.FS` for script loading. Added to the existing options pattern in `engine.go`.

```go
// WithScriptsFS configures the Engine to load Risor scripts from the given
// filesystem instead of from the scriptsDir path on disk. This enables
// embedding scripts via go:embed. When set, scriptsDir is ignored for
// script loading but may still be used as a label in error messages.
func WithScriptsFS(fsys fs.FS) Option
```

Engine constructor behavior changes:

```go
// New creates an Engine. Script loading priority:
//   1. If WithScriptsFS is set, use the provided fs.FS (via FSImporter + fs.ReadFile)
//   2. Otherwise, use scriptsDir on disk (via LocalImporter + os.ReadFile)
// When both WithScriptsFS and a non-empty scriptsDir are provided,
// WithScriptsFS takes precedence.
// The scriptsDir parameter may be empty when WithScriptsFS is used.
func New(dbPath string, scriptsDir string, opts ...Option) (*Engine, error)
```

---

## Runtime Changes

```go
// RuntimeOption configures a Runtime.
type RuntimeOption func(*Runtime)

// WithRuntimeFS configures the Runtime to load scripts from an fs.FS
// instead of from disk. Also configures the Risor importer to use
// FSImporter for import statement resolution.
func WithRuntimeFS(fsys fs.FS) RuntimeOption

// NewRuntime creates a Runtime. Accepts optional RuntimeOptions.
func NewRuntime(s *store.Store, scriptsDir string, opts ...RuntimeOption) *Runtime
```

Changes to existing methods:

```go
// LoadScript reads a .risor file from either the configured fs.FS or disk.
// When an fs.FS is configured, uses fs.ReadFile on the embedded filesystem.
// Otherwise, uses os.ReadFile with scriptsDir as the base directory.
func (r *Runtime) LoadScript(path string) (string, error)

// eval now wires risor.WithImporter() into the Eval call.
// When fs.FS is configured: uses importer.NewFSImporter with the fs.FS.
// When scriptsDir is set: uses importer.NewLocalImporter with the directory.
// The importer enables Risor import statements (e.g., import("lib/helpers")).
func (r *Runtime) eval(ctx context.Context, source, label string, extraGlobals map[string]any) error
```

---

## Script Embedding

The embedded scripts live in a `scripts` package within the canopy library, so any consumer can import them.

```go
// scripts/embedded.go
package scripts

import "embed"

//go:embed extract/*.risor resolve/*.risor lib
var FS embed.FS

// FS contains the embedded Risor scripts. Paths are relative to the
// scripts/ directory (e.g., "extract/go.risor", "resolve/go.risor",
// "lib/helpers.risor"). This matches what the Runtime expects.
```

Usage by any consumer (CLI, project-cortex, etc.):

```go
import "github.com/jward/canopy/scripts"

engine, err := canopy.New(dbPath, "", canopy.WithScriptsFS(scripts.FS))
```

The CLI binary (`cmd/canopy/`) imports the same package:

```go
// cmd/canopy/main.go
import "github.com/jward/canopy/scripts"

// When --scripts-dir is not set, uses scripts.FS
```

No `fs.Sub` is needed because the `//go:embed` directive embeds `extract/`, `resolve/`, and `lib/` directly (not nested under a `scripts/` prefix).

---

## QueryBuilder Construction

To support query commands that bypass the Engine:

```go
// NewQueryBuilder creates a QueryBuilder from a Store.
// Used by the CLI for query commands that don't need the Engine.
func NewQueryBuilder(s *store.Store) *QueryBuilder
```

---

## CLI Command Structure

All commands are implemented with Cobra. The root command is `canopy`.

### Root Command

```go
// canopy -- root command, no action, prints help
// Persistent flags (inherited by all subcommands):
//   --db <path>       Override DB location (default: .canopy/index.db relative to repo root)
//   --format json|text Output format (default: json)
```

### Index Command

```go
// canopy index [path]
// Indexes a repository. Path defaults to cwd.
// Flags:
//   --force            Full reindex (clear DB first)
//   --languages <list> Comma-separated language filter (e.g. "go,typescript")
//   --scripts-dir <path> Override embedded scripts with disk path
```

### Query Subcommands -- Position-Based Only

```go
// canopy query definition <file> <line> <col>
// canopy query symbol-at <file> <line> <col>
// Line and col are 0-based integers.
```

### Query Subcommands -- Position or Symbol ID

Each accepts either positional args `<file> <line> <col>` or `--symbol <id>`:

```go
// canopy query references <file> <line> <col>
// canopy query references --symbol <id>
//
// canopy query callers <file> <line> <col>
// canopy query callers --symbol <id>
//
// canopy query callees <file> <line> <col>
// canopy query callees --symbol <id>
//
// canopy query implementations <file> <line> <col>
// canopy query implementations --symbol <id>
```

### Query Subcommands -- Discovery / Search

```go
// canopy query symbols [--kind func] [--file path] [--visibility public] [--path-prefix src/]
// canopy query search <pattern>         -- glob search (e.g. "Get*User*")
// canopy query files [--language go] [--prefix src/]
// canopy query packages [--prefix ...]
// canopy query summary
// canopy query package-summary <path-or-id>
// canopy query deps <file>
// canopy query dependents <source>
```

### Shared Query Flags

All query subcommands inherit from the `query` parent:

```go
// Persistent flags on the query command:
//   --limit N          Pagination limit (default 50)
//   --offset N         Pagination offset (default 0)
//   --sort <field>     Sort field: name|kind|file|ref_count
//   --order <dir>      Sort order: asc|desc
```

---

## Output Types

JSON output types for CLI responses. These wrap QueryBuilder results with metadata for LLM consumption.

```go
// CLIResult is the top-level JSON envelope for all query commands.
type CLIResult struct {
    Command    string `json:"command"`
    Results    any    `json:"results"`
    TotalCount *int   `json:"total_count,omitempty"` // for paginated results
    Error      string `json:"error,omitempty"`
}

// CLILocation extends Location with the symbol ID for chaining.
type CLILocation struct {
    File      string `json:"file"`
    StartLine int    `json:"start_line"`
    StartCol  int    `json:"start_col"`
    EndLine   int    `json:"end_line"`
    EndCol    int    `json:"end_col"`
    SymbolID  *int64 `json:"symbol_id,omitempty"`
}

// CLISymbol is a JSON-friendly symbol representation.
type CLISymbol struct {
    ID         int64    `json:"id"`
    Name       string   `json:"name"`
    Kind       string   `json:"kind"`
    Visibility string   `json:"visibility"`
    Modifiers  []string `json:"modifiers,omitempty"`
    File       string   `json:"file,omitempty"`
    StartLine  int      `json:"start_line"`
    StartCol   int      `json:"start_col"`
    EndLine    int      `json:"end_line"`
    EndCol     int      `json:"end_col"`
    RefCount   int      `json:"ref_count,omitempty"`
}

// CLICallEdge is a JSON-friendly call graph edge.
type CLICallEdge struct {
    CallerID   int64  `json:"caller_id"`
    CallerName string `json:"caller_name,omitempty"`
    CalleeID   int64  `json:"callee_id"`
    CalleeName string `json:"callee_name,omitempty"`
    File       string `json:"file,omitempty"`
    Line       int    `json:"line"`
    Col        int    `json:"col"`
}

// CLIImport is a JSON-friendly import representation.
type CLIImport struct {
    FileID       int64   `json:"file_id"`
    FilePath     string  `json:"file_path,omitempty"`
    Source       string  `json:"source"`
    ImportedName *string `json:"imported_name,omitempty"`
    LocalAlias   *string `json:"local_alias,omitempty"`
    Kind         string  `json:"kind"`
}
```

---

## Repo Root Detection

```go
// findRepoRoot walks up from the given directory looking for a .git directory.
// Returns the directory containing .git, or the original directory if not found.
func findRepoRoot(startDir string) string
```
