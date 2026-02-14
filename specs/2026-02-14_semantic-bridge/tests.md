# Test Specifications

## Unit Tests

### Store Layer

- Insert a file record and retrieve it by path; verify all fields match
- Insert a file, then insert symbols for that file; query symbols by file ID; verify count and field values
- Insert nested scopes (file -> function -> block); query scope chain from innermost; verify parent traversal
- Insert references in a file; query by name; verify location data
- Insert imports with various kinds (module, member, builtin); query by file; verify all fields
- Insert type members for a symbol; query by symbol ID; verify kind and type_expr
- Insert function parameters with ordinals, receiver flag, return flag; query by symbol; verify ordering
- Insert type parameters with variance and constraints; query by symbol; verify fields
- Insert annotations targeting a symbol; query by target; verify name and arguments
- Insert symbol fragments across two files for one symbol; query by symbol; verify is_primary flag
- Transactional re-index: insert file data, then re-index the file with different data; verify old rows deleted and new rows present
- Insert resolved references; query by reference ID and by target symbol ID
- Insert implementations; query by type and by interface
- Insert call graph edges; query callers and callees
- Insert reexports; query by file
- Insert extension bindings; query by member and by extended type
- Insert type compositions; query by composite symbol ID; verify component symbols returned
- Schema migration on empty database; verify all 16 tables exist with correct columns
- Schema migration is idempotent (running twice does not error)
- WAL mode is enabled after migration

### Signature Hash

- Compute signature_hash for a symbol; verify deterministic (same input → same hash)
- Change symbol name; verify signature_hash changes
- Change symbol visibility; verify signature_hash changes
- Change symbol modifiers; verify signature_hash changes
- Add a type member to a symbol; verify signature_hash changes
- Add a function parameter; verify signature_hash changes
- Unchanged symbol across re-extraction; verify signature_hash stays the same

### Blast Radius Methods

- `FilesReferencingSymbols`: insert resolved_references from files A and B to symbols in file C; query with file C's symbol IDs; verify returns file IDs for A and B
- `FilesImportingSource`: insert imports from files A and B importing "pkg/foo"; query; verify returns A and B
- `DeleteResolutionDataForSymbols`: insert resolved_references, call_graph edges, implementations targeting symbols; delete; verify all removed
- `DeleteResolutionDataForFiles`: insert all resolution data originating from a file; delete; verify all removed
- Blast radius with no references: `FilesReferencingSymbols` returns empty list for an unreferenced symbol

### Tree-sitter Host Functions

- `parse` returns a tree-sitter Tree object; calling `RootNode()` on it returns a valid proxied node
- `parse` with a Go file returns a tree whose root node `Type()` is `source_file`
- `parse` with invalid source still returns a tree (tree-sitter is error-tolerant)
- Proxied node methods work: `node.NamedChild(0)`, `node.ChildByFieldName("name")`, `node.ChildCount()`, `node.Parent()`, `node.StartPoint()`
- `node_text` returns correct source text for a function name node
- `node_text` returns correct source text for a full function declaration node
- `node_text` returns empty string for a zero-width node
- `query` with `(function_declaration name: (identifier) @name)` on a Go file returns all function names as match maps
- `query` with no matches returns empty list
- `query` with invalid pattern returns an error
- `query` match maps have capture names as keys and proxied Nodes as values

### Go Extraction Script

- Simple function declaration: extracts symbol with kind=function, correct name, visibility, location
- Multiple functions in one file: extracts all as separate symbols under same file
- Method with receiver: extracts symbol with kind=method, parent_symbol_id points to receiver type symbol
- Struct declaration: extracts symbol with kind=struct and type_members for each field
- Struct with embedded type: type_member with kind=embedded
- Interface declaration: extracts symbol with kind=interface and type_members with kind=method
- Variable and constant declarations: extracts symbols with correct kinds
- Import statements: single import, grouped imports, aliased imports, dot imports
- Package declaration: extracts as module/package symbol
- Scope tree: file scope contains function scopes which contain block scopes (if/for/switch)
- References: function call creates reference with context=call
- References: type annotation creates reference with context=type_annotation
- References: field access creates reference with context=field_access
- Function parameters: name, type_expr, ordinal; receiver marked with is_receiver
- Return types: extracted as function_parameters with is_return=true
- Generics: type parameters with constraints extracted correctly
- Nested types: struct inside function has correct parent_symbol_id chain
- Exported vs unexported: visibility correctly determined from capitalization

### TypeScript Extraction Script

- Class declaration: extracts symbol, type_members for properties and methods
- Interface declaration: extracts symbol, type_members
- Function and arrow function declarations
- Enum declaration with variants as type_members
- Type alias declaration
- Import statements: named, default, namespace imports
- Export statements: named exports, default exports, re-exports
- Decorators: extracted as annotations
- Generic type parameters with constraints
- Module/namespace declarations

### Python Extraction Script

- Function and async function declarations
- Class declaration with methods and class variables
- Decorator extraction as annotations
- Import statements: import, from...import, aliased imports
- Variable assignments at module level as symbols
- Nested functions and classes with correct parent_symbol_id
- Scope tree: module, class, function scopes

### Rust Extraction Script

- Function declarations (fn, pub fn, async fn)
- Struct declarations with fields
- Enum declarations with variants
- Trait declarations with methods
- Impl blocks: extracts methods and links to type
- Module declarations (mod, pub mod)
- Use statements as imports
- Generics with trait bounds

### Other Language Extraction Scripts

- C: functions, structs, typedefs, enums, includes, macros
- C++: classes, namespaces, templates, access specifiers (public/private/protected)
- Java: classes, interfaces, annotations, packages, imports, access modifiers
- PHP: classes, traits, interfaces, namespaces, use statements, visibility
- Ruby: classes, modules, methods (def, self.def), mixins (include, extend), require

### QueryBuilder

- `DefinitionAt`: pre-populate extraction + resolution data for a Go file with function calls; query a call site location; verify returns the target function's location
- `ReferencesTo`: insert a symbol and multiple resolved_references targeting it; query; verify all reference locations returned
- `Implementations`: insert an interface symbol and implementation rows; query; verify implementing type locations returned
- `Callers`: insert call_graph edges; query by callee; verify caller locations returned
- `Callees`: insert call_graph edges; query by caller; verify callee locations returned
- `Dependencies`: insert imports for a file; query by file ID; verify all imports returned
- `Dependents`: insert imports from multiple files referencing a source; query by source; verify all importing files returned

## Integration Tests

### Full Pipeline

**Given** a Go source file with a function `Foo` that calls function `Bar` defined in the same file,
**When** the file is indexed and resolved,
**Then** the reference to `Bar` in `Foo` is resolved to `Bar`'s symbol, and a call_graph edge exists from `Foo` to `Bar`.

**Given** two Go files in the same package where file A calls a function defined in file B,
**When** both files are indexed and resolved,
**Then** the cross-file reference is resolved correctly via same-package resolution.

**Given** a Go file that imports a package and calls a function from it,
**When** all relevant files are indexed and resolved,
**Then** the import is resolved and the call reference points to the correct symbol in the imported package.

**Given** a Go struct that implements an interface (implicit, structural),
**When** indexed and resolved,
**Then** an implementations row links the struct to the interface with kind=implicit.

**Given** a TypeScript file with a class that implements an interface,
**When** indexed and resolved,
**Then** an implementations row links the class to the interface with kind=explicit.

**Given** a Python file with a class hierarchy (class B extends A),
**When** indexed and resolved,
**Then** references to methods on B that are defined in A are resolved correctly.

### Incremental Re-indexing

**Given** a file that has been indexed,
**When** the file is modified and re-indexed,
**Then** old extraction data is removed and new data replaces it; resolution is re-run for affected files.

**Given** a file that has NOT changed (same hash),
**When** IndexFiles is called with that file's path,
**Then** the file is skipped (not re-extracted).

### Incremental Resolution (Blast Radius)

**Given** files A, B, C where B and C reference symbols in A,
**When** A is modified and a symbol's signature changes (different signature_hash),
**Then** resolution data for B and C is invalidated and re-resolved, but other files are untouched.

**Given** files A and B where B references a symbol in A,
**When** A is modified and the referenced symbol is removed,
**Then** the resolved_reference in B targeting that symbol is deleted, B is re-resolved, and B's resolution reflects the missing symbol.

**Given** files A and B where B does not reference any symbols in A,
**When** A is modified with changed symbols,
**Then** B's resolution data is not touched (blast radius does not include B).

**Given** a file A that adds a new exported symbol,
**When** A is re-indexed,
**Then** only files that import A's module are candidates for re-resolution (new symbol may now resolve previously-unresolved references).

**Given** a file with unchanged signature_hash on all symbols after re-extraction,
**When** incremental resolution runs,
**Then** no resolution data is invalidated and no re-resolution occurs.

## MCP-based Verification (dev-time, not CI)

Accuracy verification happens during LLM development sessions using MCP LSP servers. This is NOT automated test infrastructure — it's a development workflow. The LLM:

1. Configures the relevant MCP LSP server (gopls, tsserver, pyright, etc.)
2. Runs canopy on test files
3. Queries the LSP via MCP for the same operations (definition, references, implementations)
4. Compares results, iterates on the Risor script
5. Writes golden test fixtures from verified results

### Accuracy targets (per language)

- go-to-definition: >90% F1 score
- find-references: >90% F1 score
- implementations: >90% F1 score (where applicable)

### Languages and MCP LSP servers

| Language | MCP LSP Server |
|----------|---------------|
| Go | gopls |
| TypeScript/JavaScript | typescript-language-server |
| Python | pyright / pylsp |
| Rust | rust-analyzer |
| C/C++ | clangd |
| Java | eclipse.jdt.ls |
| PHP | phpactor |
| Ruby | solargraph |

## Golden Test Corpus

### Corpus structure

LLM-generated source files organized by language and complexity level. Each level is a frozen snapshot — never modified once written. New levels are added as scripts handle more constructs.

```
testdata/
  go/
    level-01-basic-decls/
      src/             # source files (LLM-generated)
      golden.json      # expected output in golden format
    level-02-structs-interfaces/
      src/
      golden.json
    level-03-imports/
      src/
      golden.json
    ...
  typescript/
    level-01-basic-decls/
      src/
      golden.json
    ...
```

### Three validation tiers

Each golden.json can contain expectations at any combination of tiers:

**Tier 1: Extraction** — "did we find the right stuff in the file?"
Validates that extraction scripts correctly identify symbols, scopes, references, and imports.

**Tier 2: Simple resolution** — "do references point to the right symbols?"
Validates go-to-definition and find-references within a small set of files.

**Tier 3: Full resolution** — "do complex semantic operations work?"
Validates implementations, call graph, cross-package resolution, extension bindings.

Lower levels start with Tier 1 only. Higher levels add Tier 2 and 3 as resolution scripts mature.

### Golden format

A minimal subset of LSP-shaped data — what canopy's consumers actually care about. Not the full LSP protocol, but the spatial/relational core.

```json
{
  "definitions": [
    { "name": "Foo", "kind": "function", "file": "main.go", "line": 5 },
    { "name": "Bar", "kind": "function", "file": "util.go", "line": 3 },
    { "name": "MyStruct", "kind": "struct", "file": "types.go", "line": 1 }
  ],
  "references": [
    {
      "from": { "file": "main.go", "line": 10, "col": 5 },
      "to": { "name": "Bar", "file": "util.go", "line": 3 }
    }
  ],
  "implementations": [
    { "type": "MyStruct", "interface": "Reader" }
  ],
  "calls": [
    { "caller": "Foo", "callee": "Bar" }
  ]
}
```

Loose on purpose — line numbers but not column ranges for definitions, name-based matching where unambiguous. Strict enough to catch real bugs, loose enough that source reformatting doesn't break everything.

### How golden tests are created

1. LLM generates source files for a new level
2. LLM writes the extraction/resolution script
3. LLM queries the MCP LSP server to get ground-truth answers
4. LLM saves the LSP answers in golden format as `golden.json`
5. Golden test runner (`canopy test testdata/go/level-01-basic-decls/`) validates canopy output matches

### Test runner

A single CLI command the LLM can run locally:

```
canopy test testdata/go/level-01-basic-decls/   # test one level
canopy test testdata/go/                         # test all Go levels
canopy test testdata/                            # test everything
```

The runner:
1. Reads source files from `src/`
2. Runs canopy extraction (and resolution if golden.json has Tier 2/3 expectations)
3. Maps canopy output to golden format
4. Diffs against `golden.json`
5. Reports pass/fail per expectation

## Error Scenarios

- Parsing a file with syntax errors: extraction script should extract what it can from the partial CST; no crash
- Extraction script with a runtime error: Engine logs error and continues with other files; no crash
- Resolution script with a runtime error: Engine logs error and continues with other languages; no crash
- Database file that is locked by another process: Engine returns clear error
- File with unsupported language extension: Engine skips the file and logs a warning
- Empty file: Extraction script creates file entry but no symbols/scopes/references
- Binary file accidentally included: Engine detects non-text content and skips
- Circular imports: Resolution script handles import cycles without infinite loop
- Symbol with no file (multi-file namespace): Extraction succeeds with NULL file_id on symbols
