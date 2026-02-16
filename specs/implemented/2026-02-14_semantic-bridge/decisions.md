# Architectural Decisions

## 2026-02-14: SQLite over in-memory state

**Context:** Canopy needs to store extracted structural data (symbols, scopes, references) and resolved semantic data (resolved references, call graph, implementations) for potentially large projects with thousands of files. The data must support incremental updates (re-index one file without re-parsing everything) and be queryable by both Go code and Risor scripts.

**Decision:** Use a single SQLite database file per project as the primary state store. All extraction and resolution data lives in SQLite tables. No tree-sitter ASTs or custom data structures are held in memory beyond the lifetime of a single file's extraction.

**Consequences:**
- (+) Massive projects are feasible: the database file can grow without memory pressure
- (+) Incremental updates are natural: delete rows for changed file, re-insert, re-resolve
- (+) Debugging is trivial: `sqlite3 project.db "SELECT * FROM symbols WHERE name = 'Foo'"`
- (+) Concurrent access: multiple Risor scripts can query the same database (WAL mode)
- (+) Cortex alignment: project-cortex already stores its graph in SQLite, so canopy's output format matches
- (+) Persistence: re-open the database without re-indexing if files haven't changed
- (-) Slightly slower than in-memory for hot-path queries (mitigated by indexes and WAL mode)
- (-) CGO dependency for go-sqlite3 (already needed for tree-sitter, so no marginal cost)

## 2026-02-14: Risor for ALL language-specific logic (extraction and resolution)

**Context:** Language-specific logic includes both structural extraction (walking the CST to find declarations, references, imports) and semantic resolution (scope-aware name resolution, interface matching, cross-file stitching). Both need to be authored, tested, and iterated rapidly, potentially by LLMs working on multiple languages concurrently.

**Decision:** All language-specific logic lives in Risor scripts — both extraction and resolution. The Go core is intentionally thin: it provides tree-sitter parsing, the SQLite Store, and the Risor runtime. Tree-sitter objects (Tree, Node, Query) and the Store are exposed directly to Risor — no wrappers, no abstraction layers. Risor can call methods on these Go objects directly, which is why we chose Risor.

**Consequences:**
- (+) No compile step for any language-specific work: edit a `.risor` file and re-run immediately
- (+) LLM iteration: LLMs edit scripts without understanding Go build tooling
- (+) Parallel development: multiple LLM teams work on different language scripts concurrently
- (+) Scripts are small and focused: one extraction + one resolution per language
- (+) Tree-sitter's full API is available to scripts (queries, traversal, node inspection) without Go wrapper maintenance
- (+) Adding a new language requires zero Go code changes — just new scripts
- (-) Runtime overhead vs pure Go (mitigated: Risor compiles to bytecode, and the bottleneck is SQLite I/O anyway)
- (-) Scripts must understand tree-sitter's Go API surface (mitigated: it's small and well-documented)

## 2026-02-14: Two-phase architecture (extract then resolve)

**Context:** Semantic analysis requires two conceptually different operations: (1) structural extraction from source code (what names are declared, where are they used, what scopes exist) and (2) semantic resolution (what does this name refer to, what implements this interface). Extraction needs the tree-sitter CST; resolution operates on the relational data in SQLite.

**Decision:** Separate the pipeline into two explicit phases:
- **Phase 1 (Extract):** Risor extraction scripts receive tree-sitter Tree objects. They walk the CST and write to extraction tables. Tree is released after extraction.
- **Phase 2 (Resolve):** Risor resolution scripts query extraction tables and write to resolution tables. No tree-sitter access needed.

**Consequences:**
- (+) Tree-sitter instances are short-lived: parse file, run extraction script, release. No AST memory accumulation.
- (+) Extraction can be validated independently of resolution
- (+) Resolution can be re-run without re-parsing (edit script, re-resolve, compare)
- (+) Debuggable intermediate state: inspect extraction tables before resolution runs
- (+) Resolution scripts are simpler: they work with relational data, not CSTs
- (-) Two passes over the data (mitigated: extraction is fast, resolution is the bottleneck)
- (-) Extraction must capture everything resolution needs (if something is missing, you have to re-extract)

## 2026-02-14: LLM-authored scripts with LSP oracle verification

**Context:** Writing accurate semantic extraction and resolution logic for 8+ languages is a massive effort. Each language has unique CST node types, scoping rules, import systems, type hierarchies, and edge cases.

**Decision:** Use LLMs to author and iterate on Risor scripts (both extraction and resolution). Verify correctness using existing MCP LSP servers (gopls, tsserver, pyright, etc.) during LLM development sessions. No custom Go oracle code — the LLM queries the LSP via MCP and compares results directly. LSPs are NOT used at runtime. The workflow is:
1. LLM writes/edits a Risor script
2. LLM runs canopy on test files
3. LLM queries the MCP LSP server for the same operations
4. LLM compares results, iterates on the script
5. Once accuracy meets threshold (>90%), LLM writes golden test fixtures (input files + expected output)
6. Golden tests run in CI — no MCP, no LSP, just input/output comparison

**Consequences:**
- (+) Leverage LLM capability for multi-language expertise
- (+) Objective correctness measurement via MCP LSP comparison
- (+) Golden tests provide regression protection without runtime LSP dependency
- (+) Zero custom LSP client code — uses existing, well-maintained MCP servers
- (+) Tight iteration loop: edit script, run, compare via MCP, fix
- (+) Bulk snapshot testing catches broad regressions
- (-) LLM-authored code needs review for correctness and maintainability
- (-) MCP LSP server availability varies by language (some may need community-maintained servers)

## 2026-02-14: Schema designed for 13+ languages before building

**Context:** Adding columns or tables to a schema after data exists is disruptive. Language-specific schema extensions fragment the design and make cross-language queries inconsistent.

**Decision:** Pressure-test the schema against 13 languages (Go, TypeScript/JavaScript, Python, Rust, C/C++, Java, PHP, Ruby, C#, Zig, Kotlin, Swift, Objective-C) before writing any code. The schema uses a unified design with kind/context enums that cover all language constructs.

**Consequences:**
- (+) No schema migrations needed when adding new languages
- (+) Consistent query patterns across all languages
- (+) Kind enums (`symbols.kind`, `type_members.kind`, `references.context`) are comprehensive
- (+) Future languages can be added with new scripts only, no schema changes
- (-) Some columns are unused for some languages (e.g., `variance` only matters for languages with declaration-site variance)
- (-) Schema is larger than needed for any single language (acceptable: SQLite handles sparse columns well)

## 2026-02-14: Expose tree-sitter and Store directly, no wrappers

**Context:** The Go core needs to provide tree-sitter CST access and database access to Risor scripts. The options are: (a) wrap everything in custom host functions like `node_children()`, `db_symbols_by_file()`, etc., or (b) pass the actual Go objects through and let Risor call methods on them directly.

**Decision:** Pass the tree-sitter Tree/Node objects and the Store object directly to Risor via proxy reflection. Provide thin host functions only where Risor's proxy system has limitations. Validated in `.spikes/risor-treesitter/`.

Host functions (Go-side):
- `parse(path, language)` — creates the tree-sitter Tree, captures source `[]byte`
- `node_text(node)` — workaround: Risor can't convert string to `[]byte` for `node.Content()`
- `query(pattern, node)` — workaround: `NewQuery`/`NewQueryCursor` are free functions Risor can't call; cursor iteration (`NextMatch()` returns `(*QueryMatch, bool)` as a list) is awkward

Everything else is direct method calls on proxied CGO objects: `node.Type()`, `node.NamedChild(i)`, `node.ChildByFieldName("name")`, `node.Parent()`, `node.StartPoint()`, `node.String()`, etc.

**Consequences:**
- (+) Minimal maintenance burden — only 3 host functions, rest is tree-sitter's own API
- (+) Scripts have full access to tree-sitter capabilities via proxy reflection
- (+) Store API changes automatically available to scripts (no wrapper to update)
- (+) Less Go code to write, test, and maintain
- (+) Spike-validated: Risor proxy works with CGO-backed go-tree-sitter objects
- (-) Scripts must know tree-sitter's Go method names (mitigated: well-documented, and LLMs know them)
- (-) Any new tree-sitter methods that take `[]byte` args would need additional host functions

## 2026-02-14: LSP-shaped golden format with frozen incremental levels

**Context:** Canopy needs a test corpus and expectation format that (a) LLMs can create and validate without compiling Go code, (b) doesn't break when new test cases are added, and (c) maps naturally to LSP ground-truth from MCP verification sessions.

**Decision:** Use a minimal LSP-shaped JSON golden format (`golden.json`) with four top-level keys: `definitions`, `references`, `implementations`, `calls`. Three validation tiers: extraction (Tier 1), simple resolution (Tier 2), full resolution (Tier 3). Corpus is organized as frozen incremental levels per language (`testdata/{lang}/level-{N}-{name}/`). Each level is never modified once written — new constructs get new levels. A `canopy test` CLI command runs validation.

**Consequences:**
- (+) Golden format mirrors LSP responses — MCP LSP output maps directly to test expectations
- (+) Frozen levels mean adding level N+1 never breaks levels 1..N
- (+) LLM generates both source files and golden.json — no Go code needed for test authoring
- (+) Single CLI command (`canopy test`) for validation — tight iteration loop
- (+) Three tiers allow incremental validation as scripts mature
- (-) Golden format is a subset of LSP — may need to extend it as canopy's capabilities grow
- (-) Name-based matching (loose) could miss subtle location bugs (acceptable tradeoff for maintainability)
