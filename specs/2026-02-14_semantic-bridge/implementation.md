# Implementation Plan

## Phase 1: Foundation

- [x] Initialize Go module structure (packages: `canopy`, `internal/store`, `internal/runtime`)
- [x] Add tree-sitter dependency (`github.com/smacker/go-tree-sitter`)
- [x] Add tree-sitter grammar dependencies for all initial languages
- [x] Add Risor dependency (`github.com/risor-io/risor`)
- [x] Add SQLite driver dependency (`github.com/mattn/go-sqlite3`)
- [x] Implement Store with schema migration (`Migrate()` — all 16 tables + indexes)
- [x] Implement Store CRUD methods for extraction tables (files, symbols, scopes, references, imports, type_members, function_parameters, type_parameters, annotations, symbol_fragments)
- [x] Implement Store CRUD methods for resolution tables (resolved_references, implementations, call_graph, reexports, extension_bindings, type_compositions)
- [x] Implement `DeleteFileData` (transactional wipe of all rows for a file)
- [x] Implement `signature_hash` computation (composite hash of name+kind+visibility+modifiers+members+params)
- [x] Implement blast radius Store methods: `FilesReferencingSymbols`, `FilesImportingSource`, `DeleteResolutionDataForSymbols`, `DeleteResolutionDataForFiles`
- [x] Implement Engine skeleton with `New()`, `Close()`, database lifecycle
- [x] Implement `Engine.IndexFiles()` — file hashing, change detection, extraction script dispatch per language
- [x] Implement `Engine.IndexDirectory()` — file discovery, language detection, delegates to IndexFiles
- [x] Implement `Engine.Resolve()` — runs resolution scripts per language, blast radius computation for incremental re-resolution
- [x] Implement `parse(path, language)` — tree-sitter parsing, returns Tree object; captures source []byte for node_text
- [x] Implement `node_text(node)` — returns source text of a node as string ([]byte workaround)
- [x] Implement `query(pattern, node)` — wraps NewQuery/NewQueryCursor/NextMatch loop, returns list of match maps
- [x] Set up Risor runtime with globals: `parse`, `node_text`, `query`, `db` (Store), `log`
- [x] Implement language detection (file extension → language name mapping for script selection)
- [x] Implement script loading from `scripts/extract/` and `scripts/resolve/` directories
- [x] Create `scripts/` directory structure (`extract/`, `resolve/`, `lib/`)
- [x] Write unit tests for Store (round-trip insert/query for each table)
- [x] Write unit tests for schema migration
- [x] Write unit test: `parse` returns a usable tree-sitter Tree object (can call RootNode(), node.Type(), etc. via Risor proxy)
- [x] Write unit test: `node_text` returns correct source text for various node types
- [x] Write unit test: `query` returns correct matches for S-expression patterns (function declarations, identifiers, etc.)
- [x] Write unit test: `query` returns empty list for no-match pattern
- [x] Write unit test: `query` returns error for invalid pattern
- [x] Write unit tests for signature_hash computation (same symbol → same hash, changed symbol → different hash)
- [x] Write unit tests for blast radius methods (FilesReferencingSymbols, FilesImportingSource, DeleteResolutionDataForSymbols, DeleteResolutionDataForFiles)

## Phase 2: Go Extraction Script (first language)

- [x] Write `scripts/extract/go.risor`: extract package declarations as symbols
- [x] Write `scripts/extract/go.risor`: extract function and method declarations
- [x] Write `scripts/extract/go.risor`: extract struct, interface, and type declarations
- [x] Write `scripts/extract/go.risor`: extract variable and constant declarations
- [x] Write `scripts/extract/go.risor`: extract scope tree (file, function, block scopes)
- [x] Write `scripts/extract/go.risor`: extract references (all identifier uses)
- [x] Write `scripts/extract/go.risor`: extract import statements
- [x] Write `scripts/extract/go.risor`: extract type members (struct fields, interface methods, embedded types)
- [x] Write `scripts/extract/go.risor`: extract function parameters and return types
- [x] Write `scripts/extract/go.risor`: extract type parameters (generics)
- [x] Write unit tests: simple Go file with function declarations
- [x] Write unit tests: Go file with structs and interfaces
- [x] Write unit tests: Go file with imports and cross-package references
- [x] Write unit tests: Go file with nested scopes (if/for/switch blocks)
- [x] Write unit tests: Go file with methods and receivers
- [x] Write unit tests: Go file with generics
- [x] Validate extraction completeness against manually inspected Go files

## Phase 3: Go Resolution Script (first resolver)

- [x] Implement incremental resolution in Engine: compute blast radius on re-index (compare old vs new signature_hash for added/removed/changed symbols), selectively re-resolve only affected files
- [x] Write `scripts/resolve/go.risor`: single-file scope-based name resolution
- [x] Write `scripts/resolve/go.risor`: cross-file import resolution (same package)
- [x] Write `scripts/resolve/go.risor`: cross-package import resolution
- [x] Write `scripts/resolve/go.risor`: method resolution (receiver type matching)
- [x] Write `scripts/resolve/go.risor`: interface matching (structural, implicit)
- [x] Write `scripts/resolve/go.risor`: call graph edge creation from resolved call references
- [x] Write `scripts/resolve/go.risor`: extension binding for methods on types
- [x] Implement `canopy test` CLI command (read fixture dirs, run canopy, diff against golden.json, report pass/fail)
- [x] Define golden format JSON schema (definitions, references, implementations, calls)
- [x] Create `testdata/go/level-01-basic-decls/` — LLM-generated source files + Tier 1 golden.json (extraction only)
- [x] Create `testdata/go/level-02-structs-interfaces/` — Tier 1 golden.json
- [x] Create `testdata/go/level-03-imports/` — Tier 1 + Tier 2 golden.json (cross-file resolution)
- [ ] Verify Go resolution accuracy against gopls via MCP (LLM development workflow)
- [ ] Iterate on `scripts/resolve/go.risor` until >90% accuracy on go-to-definition
- [ ] Iterate on `scripts/resolve/go.risor` until >90% accuracy on find-references
- [ ] Write golden tests from MCP-verified results
- [x] Write regression tests for specific edge cases found during iteration

## Phase 4: Multi-language Extraction Scripts

### TypeScript/JavaScript
- [x] Write `scripts/extract/typescript.risor` (classes, interfaces, functions, arrow functions, type aliases, enums, modules, decorators, exports, re-exports)
- [x] Write `scripts/extract/javascript.risor` (shares patterns with TS minus type annotations)
- [x] Write unit tests for TS/JS extraction

### Python
- [x] Write `scripts/extract/python.risor` (classes, functions, decorators, imports, assignments as symbols)
- [x] Write unit tests for Python extraction

### Rust
- [x] Write `scripts/extract/rust.risor` (structs, enums, traits, impl blocks, modules, use statements)
- [x] Write unit tests for Rust extraction

### C/C++
- [x] Write `scripts/extract/c.risor` (functions, structs, typedefs, macros, includes)
- [x] Write `scripts/extract/cpp.risor` (extends C patterns with classes, namespaces, templates)
- [x] Write unit tests for C/C++ extraction

### Java
- [x] Write `scripts/extract/java.risor` (classes, interfaces, methods, annotations, packages, imports)
- [x] Write unit tests for Java extraction

### PHP
- [x] Write `scripts/extract/php.risor` (classes, traits, interfaces, functions, namespaces, use statements)
- [x] Write unit tests for PHP extraction

### Ruby
- [x] Write `scripts/extract/ruby.risor` (classes, modules, methods, mixins, blocks)
- [x] Write unit tests for Ruby extraction

## Phase 5: Multi-language Resolution Scripts

- [x] Write `scripts/resolve/typescript.risor`: scope resolution, import resolution, interface matching, re-exports
- [x] Write `scripts/resolve/javascript.risor`: scope resolution, require/import resolution, prototype chain
- [x] Write `scripts/resolve/python.risor`: scope resolution, import resolution, class hierarchy
- [x] Write `scripts/resolve/rust.risor`: scope resolution, use/mod resolution, trait impl matching
- [x] Write `scripts/resolve/c.risor`: scope resolution, include resolution, struct member access
- [x] Write `scripts/resolve/cpp.risor`: scope resolution, namespace resolution, class hierarchy, template awareness
- [x] Write `scripts/resolve/java.risor`: scope resolution, import resolution, class hierarchy, interface impl
- [x] Write `scripts/resolve/php.risor`: scope resolution, namespace/use resolution, trait inclusion
- [x] Write `scripts/resolve/ruby.risor`: scope resolution, require resolution, mixin inclusion, method lookup
- [ ] Verify TS/JS resolution accuracy against tsserver via MCP; write golden tests
- [ ] Verify Python resolution accuracy against pyright via MCP; write golden tests
- [ ] Verify Rust resolution accuracy against rust-analyzer via MCP; write golden tests
- [ ] Verify C/C++ resolution accuracy against clangd via MCP; write golden tests
- [ ] Verify Java resolution accuracy against jdtls via MCP; write golden tests
- [ ] Verify PHP resolution accuracy against phpactor via MCP; write golden tests
- [ ] Verify Ruby resolution accuracy against solargraph via MCP; write golden tests
- [ ] Iterate all resolvers to >90% accuracy

## Phase 6: Cortex Integration

- [x] Implement QueryBuilder (DefinitionAt, ReferencesTo, Implementations, Callers, Callees, Dependencies, Dependents)
- [x] Write integration tests: full pipeline (source file -> extraction -> resolution -> query)
- [x] Document API for cortex consumption
- [ ] Validate that cortex graph operations improve with canopy data
- [x] Performance benchmarking on representative project sizes

## Phase 7: LSP Checkout and Reference Setup

- [x] Create `.scratch/` directory structure
- [x] Check out gopls source (`golang.org/x/tools/gopls`)
- [x] Check out typescript-language-server source
- [x] Check out pyright source
- [x] Check out rust-analyzer source
- [x] Check out clangd source (LLVM project)
- [x] Check out eclipse.jdt.ls source
- [x] Check out phpactor source
- [x] Check out solargraph source
- [ ] Document reference setup for LLM teams

## Notes

- Phase 7 (LSP checkout) can happen anytime and is independent of other phases
- Phases 4 and 5 can be parallelized across LLM teams (each language is independent)
- Phase 3 must complete before Phase 5 because it establishes the script patterns and development workflow
- Phase 6 depends on at least Phase 3 (Go scripts) for initial integration testing
- Shared Risor utilities (`scripts/lib/`) should emerge naturally from Phase 2-3 and be extracted as patterns repeat
