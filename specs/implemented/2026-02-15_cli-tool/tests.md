# Test Specifications

## Unit Tests

### Script Loading from fs.FS

- Runtime with `WithRuntimeFS` loads a script from the provided `fstest.MapFS`; verify source content matches
- Runtime with `WithRuntimeFS` returns error for non-existent script path
- Runtime without `WithRuntimeFS` loads from disk via `scriptsDir` (existing behavior preserved)
- Runtime with `WithRuntimeFS` and a relative script path resolves correctly within the FS
- LoadScript with absolute path and fs.FS strips leading separator and resolves within the FS

### Risor Importer Wiring

- Runtime with `WithRuntimeFS` wires `FSImporter`; a Risor script containing `import lib_helpers` resolves the module from the fs.FS and executes it
- Runtime with `scriptsDir` wires `LocalImporter`; a Risor script containing `import lib_helpers` resolves the module from disk
- Importer receives global names so imported modules can reference host-provided globals like `db` and `log`
- Script without import statements works regardless of importer configuration (no regression)

### Engine with WithScriptsFS

- `New(dbPath, "", WithScriptsFS(embeddedFS))` creates an Engine that can run extraction scripts from the embedded FS
- `New(dbPath, scriptsDir)` without `WithScriptsFS` preserves existing disk-based behavior
- `WithScriptsFS` takes precedence over a non-empty `scriptsDir` when both are provided

### NewQueryBuilder

- `NewQueryBuilder(store)` returns a QueryBuilder that can execute `SymbolAt` queries
- `NewQueryBuilder(store)` returns a QueryBuilder that can execute discovery queries like `Symbols`

### Repo Root Detection

- `findRepoRoot` from a directory containing `.git/` returns that directory
- `findRepoRoot` from a nested subdirectory of a git repo returns the repo root (not the subdirectory)
- `findRepoRoot` from a directory with no `.git/` ancestor returns the starting directory itself

### CLI Argument Parsing

- `canopy index` with no path argument defaults to cwd
- `canopy index /some/path` passes the path correctly to `IndexDirectory`
- `canopy index --force` sets force flag to true
- `canopy index --languages go,typescript` splits correctly into `["go", "typescript"]`
- `canopy query definition file.go 10 5` parses file as string, line as 10, col as 5
- `canopy query references --symbol 42` parses symbol ID as int64(42)
- `canopy query references file.go 10 5` parses positional args when `--symbol` is not set
- `canopy query symbols --kind func --visibility public` parses both filter flags
- `canopy query search "Get*"` captures the pattern as the first positional argument
- `canopy query package-summary internal/store` parses path argument as a string
- `canopy query package-summary 42` detects numeric argument and parses as symbol ID
- `--db /custom/path.db` overrides default DB location on all commands
- `--format text` sets text output mode
- `--limit 100 --offset 50` sets pagination parameters on query commands
- `--sort ref_count --order desc` sets sort parameters on query commands

### JSON Output Structure

- Position-based results (definition, symbol-at) include `symbol_id` field in output
- Location results include `file`, `start_line`, `start_col`, `end_line`, `end_col` fields
- Paginated list results include `total_count` field in the envelope
- Error responses have `error` field and null or absent `results`
- All JSON output round-trips through `json.Marshal`/`json.Unmarshal` without loss
- Symbol results include `id`, `name`, `kind`, `visibility`, `file` fields
- Call edge results include `caller_id` and `callee_id` fields

### Text Output Format

- Definition result formats as `file:line:col` (with 0-based numbers)
- Symbol list renders as aligned columns with name, kind, file headers
- Summary renders human-readable statistics with labeled counts
- Error messages go to stderr; no error text appears on stdout

## Integration Tests

### Index Command

- Given a directory with Go source files, when `canopy index` is run, then `.canopy/index.db` is created in the repo root and contains indexed symbols
- Given a previously indexed directory with no changes, when `canopy index` is run again, then it completes quickly (incremental skip) and the DB is unchanged
- Given a previously indexed directory, when `canopy index --force` is run, then all files are reindexed
- Given a directory with Go and Python files, when `canopy index --languages go` is run, then only Go files are indexed (verify by querying files table)
- Given `--scripts-dir` pointing to a custom scripts path, when `canopy index` is run, then scripts are loaded from that path instead of embedded
- Given `--db /tmp/custom.db`, when `canopy index` is run, then the database is created at `/tmp/custom.db` and not at `.canopy/index.db`

### Query Commands -- Position-Based

- Given an indexed Go project, when `canopy query symbol-at main.go 5 5` is run, then the JSON result contains the symbol at that position with its ID
- Given an indexed Go project, when `canopy query definition main.go 15 8` is run on a reference, then the JSON result contains the definition location with a symbol ID

### Query Commands -- Symbol ID

- Given an indexed project, when a symbol ID is obtained from `symbol-at` JSON output, then `canopy query references --symbol <id>` returns reference locations with file paths
- Given an indexed project, when `canopy query callers --symbol <id>` is run for a function, then caller edges with IDs are returned
- Given an indexed project, when `canopy query callees --symbol <id>` is run for a function, then callee edges with IDs are returned
- Given an indexed project, when `canopy query implementations --symbol <id>` is run for an interface, then implementing type locations are returned

### Query Commands -- Discovery

- Given an indexed project, when `canopy query symbols --kind function` is run, then only function symbols are returned with `total_count`
- Given an indexed project, when `canopy query search "Get*"` is run, then symbols matching the glob are returned
- Given an indexed project, when `canopy query files --language go` is run, then only Go files are listed
- Given an indexed project, when `canopy query packages` is run, then package/module/namespace symbols are listed
- Given an indexed project, when `canopy query summary` is run, then project-level statistics are returned including language breakdown
- Given an indexed project, when `canopy query package-summary internal/store` is run, then package-level statistics are returned including exported symbols

### Query Commands -- Deps

- Given an indexed Go file with imports, when `canopy query deps main.go` is run, then the file's imports are listed with source strings
- Given an indexed module, when `canopy query dependents fmt` is run, then files importing `fmt` are listed

### End-to-End Chaining

- Index a project, run `canopy query search "Handle*" --format json`, parse JSON to extract a symbol ID, then run `canopy query references --symbol <id> --format json` and verify the output contains valid reference locations

### Embedded Scripts

- Build the CLI binary (`go build ./cmd/canopy/`), run it on a test project without `--scripts-dir`, and verify it successfully indexes using embedded scripts
- Run with `--scripts-dir ./scripts` pointing to the repo's scripts directory, verify same indexing results as the embedded version

## Error Scenarios

- `canopy query definition` with no arguments produces a usage error message
- `canopy query definition file.go abc 5` with non-numeric line produces a clear parse error
- `canopy query references` with neither position args nor `--symbol` flag produces a usage error
- `canopy query` on a non-existent DB path produces a clear "database not found" error
- `canopy index` on a non-existent directory produces a clear "directory not found" error
- `canopy query symbol-at file.go 99999 0` on a position with no symbol returns empty results (not an error)
- `canopy query package-summary nonexistent/path` returns an error indicating the package was not found
- Invalid `--format` value (e.g., `--format xml`) produces a validation error
- `canopy query` with no subcommand prints help/usage (not an error)
- `--limit 1000` exceeding the max limit (500) is capped to 500 or produces a validation error
- Invalid `--sort` value is ignored and falls back to the default sort for that endpoint
- `canopy query callers --symbol 999999` with a non-existent symbol ID returns empty results (not a crash)
