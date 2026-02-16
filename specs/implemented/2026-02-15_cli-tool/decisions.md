# Architectural Decisions

## 2026-02-15: .canopy/index.db as default database location

**Context:** The CLI needs a default location for the SQLite database. Options: (a) a dotfile in the repo root like `.canopy.db`, (b) a directory like `.canopy/index.db`, (c) XDG cache directory, (d) a `canopy.db` file in the repo root.

**Decision:** Use `.canopy/index.db` inside a `.canopy/` directory at the repo root. The repo root is detected by walking up from cwd to find `.git/`. If no `.git/` is found, cwd is used as the root. The `--force` flag deletes the entire `.canopy/index.db` file before indexing; this is safe because `Engine.New()` runs `Migrate()` on every startup, recreating the schema from scratch.

**Consequences:**
- (+) `.canopy/` directory can hold future artifacts (logs, cache, config) without cluttering the repo root
- (+) Easy to gitignore with a single `.canopy/` entry
- (+) Hidden directory convention matches `.git/`, `.vscode/`, `.idea/`
- (+) Repo root detection via `.git/` is simple and covers the vast majority of use cases
- (-) Non-git repos require `--db` flag or will use cwd (acceptable for an initial release)
- (-) One extra directory to create on first index

## 2026-02-15: Embedded scripts via go:embed with disk override

**Context:** The CLI must be a self-contained binary (no external script files to distribute), but developers iterating on Risor scripts need to use disk scripts without rebuilding the binary. Library consumers (project-cortex) may want either mode.

**Decision:** The embedded scripts live in `scripts/embedded.go` as a library package exporting `scripts.FS` via `//go:embed extract/*.risor resolve/*.risor lib`. Any consumer (CLI, project-cortex, other tools) imports `github.com/jward/canopy/scripts` and passes `scripts.FS` to `WithScriptsFS`. The `--scripts-dir` CLI flag overrides embedded scripts with a disk path. Script loading priority: `--scripts-dir` flag > `WithScriptsFS` option > `scriptsDir` constructor parameter.

**Consequences:**
- (+) Single binary distribution -- no external dependencies beyond the binary itself
- (+) Any library consumer can embed scripts with one import: `canopy.WithScriptsFS(scripts.FS)`
- (+) Developers can iterate on scripts without rebuilding: `canopy index --scripts-dir ./scripts`
- (+) `embed.FS` implements `fs.FS`, so it works directly with Risor's `importer.NewFSImporter`
- (+) No `fs.Sub` needed -- paths embed directly matching Runtime expectations
- (+) Using `*.risor` globs avoids embedding Go test files from extract/resolve directories
- (+) Backward compatible -- existing `New(dbPath, scriptsDir)` callers are unaffected
- (-) Binary size increases by the size of all Risor scripts (small -- scripts are text files, ~250KB total)

## 2026-02-15: Wire risor.WithImporter into eval

**Context:** The Risor runtime's `eval()` method currently calls `risor.Eval()` without an importer. This works because existing scripts only use Risor builtin modules (like `strings`). However, `scripts/lib/` shared modules will need `import lib_helpers` to work, which requires a file-based importer.

**Decision:** Wire `risor.WithImporter()` into every `eval()` call. When `fs.FS` is configured, use `importer.NewFSImporter`. When `scriptsDir` is set, use `importer.NewLocalImporter`. Both importers receive the list of global names so that imported modules can reference host-provided globals.

**Consequences:**
- (+) Enables `scripts/lib/` shared modules -- scripts can use `import lib_helpers`
- (+) `FSImporter` and `LocalImporter` both cache compiled modules in memory, so repeated imports are fast
- (+) No behavior change for scripts that don't use import statements
- (-) Small overhead of creating an importer on each eval call (mitigated: importers are lightweight structs)

## 2026-02-15: Query commands open DB directly, not via Engine

**Context:** Query commands only need read access to the SQLite database. The Engine constructor runs schema migration, creates a Runtime with tree-sitter host functions, loads script paths, etc. -- none of which is needed for queries.

**Decision:** Query subcommands open the database with `store.NewStore()` and create a `QueryBuilder` directly via a new `NewQueryBuilder(store)` constructor, bypassing the Engine entirely. Only the `index` command creates a full Engine.

**Consequences:**
- (+) Query commands start faster -- no script loading, no tree-sitter initialization
- (+) Cleaner separation: indexing is a write path, querying is a read path
- (+) Can query a database even if scripts are missing, broken, or a different version
- (-) Requires a new `NewQueryBuilder` constructor (trivial addition)
- (-) Schema migration is not run on query -- the DB must already be migrated (enforced by requiring `index` to run first)

## 2026-02-15: 0-based line and column arguments

**Context:** The library uses 0-based line and column numbers throughout, matching tree-sitter's convention. CLI tools often use 1-based line numbers (matching editor conventions). We need to decide what the CLI accepts.

**Decision:** The CLI accepts 0-based line and column numbers, matching the library convention. This is documented in help text for each command.

**Consequences:**
- (+) No conversion layer between CLI and library -- positions pass through directly
- (+) Consistent with tree-sitter, the database, and all internal APIs
- (+) JSON output positions match CLI input positions (no confusion when chaining)
- (-) Less intuitive for humans used to 1-based editor line numbers
- (-) Users must subtract 1 from editor line numbers when using the CLI manually (documented in help text)
