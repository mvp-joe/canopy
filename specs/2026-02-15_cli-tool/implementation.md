# Implementation Plan

## Phase 1: Script Loading Abstraction (fs.FS + Importer Wiring)

- [ ] Add `RuntimeOption` type and `WithRuntimeFS(fs.FS)` option to `internal/runtime/runtime.go`
- [ ] Update `NewRuntime` signature to accept `...RuntimeOption`
- [ ] Add `fsys fs.FS` field to `Runtime` struct
- [ ] Update `LoadScript` to read from `fs.FS` when configured (via `fs.ReadFile`), falling back to `os.ReadFile` with `scriptsDir`
- [ ] Wire `risor.WithImporter()` into `eval()`: use `importer.NewFSImporter` when `fsys` is set, `importer.NewLocalImporter` when `scriptsDir` is set
- [ ] Pass global names to importer options so imported modules can reference host globals
- [ ] Add `WithScriptsFS(fs.FS)` option to Engine (`engine.go`)
- [ ] Update `Engine.New()` to pass `WithRuntimeFS` to `NewRuntime` when `WithScriptsFS` is configured; note: current code creates Runtime before applying options, so options must be processed first or Runtime creation deferred
- [ ] Update `Engine.New()` to accept empty `scriptsDir` when `WithScriptsFS` is set
- [ ] Add `NewQueryBuilder(store)` constructor for CLI query commands
- [ ] Unit tests: Runtime loads script from `fstest.MapFS`
- [ ] Unit tests: Runtime falls back to disk when no fs.FS configured
- [ ] Unit tests: Risor `import` statement works with `FSImporter` (import a module from embedded FS)
- [ ] Unit tests: Risor `import` statement works with `LocalImporter` (import a module from disk)
- [ ] Unit tests: Engine with `WithScriptsFS` can run extraction on a simple file
- [ ] Verify existing tests still pass (backward compatibility)

## Phase 2: CLI Skeleton + Index Command

- [ ] Add `github.com/spf13/cobra` dependency
- [ ] Create `scripts/embedded.go` with `//go:embed extract/*.risor resolve/*.risor lib` exporting `scripts.FS`; if `scripts/lib/` has no non-hidden files yet, add a placeholder `.risor` file so the embed directive compiles
- [ ] Create `cmd/canopy/main.go` with root command, importing `scripts.FS`
- [ ] Implement `findRepoRoot()` -- walk up from cwd to find `.git/`
- [ ] Implement root command with `--db` and `--format` persistent flags
- [ ] Implement `canopy index [path]` command
- [ ] Wire `--force` flag (delete DB file before indexing)
- [ ] Wire `--languages` flag (comma-split, pass to `WithLanguages`)
- [ ] Wire `--scripts-dir` flag (use disk path instead of embedded FS)
- [ ] Default script loading from embedded FS when `--scripts-dir` not set
- [ ] Create `.canopy/` directory if it does not exist when indexing
- [ ] Call `Engine.IndexDirectory` + `Engine.Resolve` in sequence
- [ ] Print timing/summary to stderr, keep stdout clean for piping
- [ ] Integration test: build binary, index a test fixture directory, verify `.canopy/index.db` created
- [ ] Integration test: `--force` flag clears and reindexes
- [ ] Integration test: `--languages` flag filters correctly

## Phase 3: Query Subcommands

- [ ] Implement `query` parent command with shared persistent flags (`--limit`, `--offset`, `--sort`, `--order`)
- [ ] Implement helper: open Store read-only from `--db` flag path
- [ ] Implement helper: `resolveSymbolID(cmd, args)` -- resolves `<file> <line> <col>` to symbol ID via `SymbolAt`, or reads `--symbol` flag
- [ ] Implement helper: parse `<line>` and `<col>` args as int with clear error on non-numeric
- [ ] Implement `canopy query symbol-at <file> <line> <col>`
- [ ] Implement `canopy query definition <file> <line> <col>`
- [ ] Implement `canopy query references` (position or `--symbol`)
- [ ] Implement `canopy query callers` (position or `--symbol`)
- [ ] Implement `canopy query callees` (position or `--symbol`)
- [ ] Implement `canopy query implementations` (position or `--symbol`)
- [ ] Implement `canopy query symbols` with filter flags (`--kind`, `--file`, `--visibility`, `--path-prefix`); `--file <path>` must resolve to file ID via `store.FileByPath()` before setting `SymbolFilter.FileID`
- [ ] Implement `canopy query search <pattern>`
- [ ] Implement `canopy query files` with `--language` and `--prefix` flags
- [ ] Implement `canopy query packages` with `--prefix` flag
- [ ] Implement `canopy query summary`
- [ ] Implement `canopy query package-summary <path-or-id>` (detect numeric vs path argument)
- [ ] Implement `canopy query deps <file>` (resolve path to file ID via `store.FileByPath()` before calling `Dependencies`)
- [ ] Implement `canopy query dependents <source>`
- [ ] Integration test: index then query via CLI, verify JSON output structure
- [ ] Integration test: `--symbol` flag works for references, callers, callees, implementations

## Phase 4: Output Formatting

- [ ] Implement JSON formatter: `CLIResult` envelope wrapping all query results
- [ ] Ensure all JSON results include symbol IDs (locations include `symbol_id`, edges include caller/callee IDs)
- [ ] Implement text formatter for position results (e.g., `file.go:10:5`)
- [ ] Implement text formatter for symbol lists (tabular: name, kind, visibility, file, line)
- [ ] Implement text formatter for summary/stats
- [ ] Implement text formatter for import/dependency lists
- [ ] Errors: JSON format outputs `{"error": "..."}` to stdout; text format prints to stderr
- [ ] Integration test: `--format text` produces readable output
- [ ] Integration test: `--format json` output is valid JSON parseable by `jq`
- [ ] Integration test: JSON output includes symbol IDs for chaining

## Notes

- The CLI binary lives at `cmd/canopy/main.go`. Build with `go build ./cmd/canopy/`.
- Embedded scripts live in `scripts/embedded.go` as a library package (`scripts.FS`). The `//go:embed extract/*.risor resolve/*.risor lib` directive embeds only `.risor` files (avoiding Go test files in those directories). Paths are direct (e.g., `extract/go.risor`), matching what the Runtime expects. No `fs.Sub` needed.
- Query commands do NOT need to create an Engine -- they only need a Store and QueryBuilder. The Engine is only needed for `index`.
- The `--db` flag defaults to `.canopy/index.db` relative to the detected repo root. The repo root is found by walking up from cwd looking for `.git/`. If no `.git/` is found, cwd is used.
- All line and column numbers in CLI arguments are 0-based, matching the library convention.
- Risor importer global names should include all host function names (`parse`, `node_text`, `query`, `log`, `db`, `insert_symbol`, etc.) so imported modules can reference them.
