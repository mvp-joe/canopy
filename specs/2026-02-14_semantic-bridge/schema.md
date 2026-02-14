# Database Schema

Canopy uses a single SQLite database per project. The schema has two distinct table groups corresponding to the two-phase architecture:

1. **Extraction tables** -- written by Risor extraction scripts during parse phase. One script per language receives the tree-sitter CST and populates these tables. Tree-sitter instances are released immediately after extraction.

2. **Resolution tables** -- written by Risor resolution scripts during resolve phase. One script per language queries the extraction tables and writes semantic analysis results (resolved references, call graph, implementations, etc.).

Both phases use Risor scripts. Extraction scripts work with tree-sitter CST objects directly; resolution scripts work with relational data in SQLite. Scripts can be edited and re-run without recompiling — extraction scripts re-run after a parse, resolution scripts re-run without re-parsing.

## Schema Version

v3 -- pressure-tested against 13 languages (Go, TypeScript/JavaScript, Python, Rust, C/C++, Java, PHP, Ruby, C#, Zig, Kotlin, Swift, Objective-C).

---

## Core Extraction Tables

### files

Tracks every source file indexed by canopy. The `hash` column enables incremental re-indexing: only files whose content hash changed need re-extraction.

```sql
files (
  id              INTEGER PRIMARY KEY,
  path            TEXT NOT NULL UNIQUE,
  language        TEXT NOT NULL,
  hash            TEXT,
  last_indexed    TIMESTAMP
)
```

### symbols

The central table. Every named declaration in source code becomes a symbol: functions, methods, classes, structs, interfaces, variables, constants, type aliases, modules, packages, namespaces, properties, and more. The `parent_symbol_id` column encodes nesting (e.g., a method inside a class). The `signature_hash` is a composite hash of the symbol's identity and structure — name, kind, visibility, modifiers, plus hashes of its type_members, function_parameters, and type_parameters. Used for incremental resolution: if a symbol's signature_hash hasn't changed, its dependents don't need re-resolution.

```sql
symbols (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER REFERENCES files(id),   -- NULL for multi-file (namespaces, packages)
  name            TEXT NOT NULL,
  kind            TEXT NOT NULL,
    -- function, method, class, struct, interface, trait, protocol,
    -- enum, variable, constant, type_alias, module, package, namespace,
    -- property, accessor, delegate, event, operator, record,
    -- error_set, test, actor, object, companion_object
  visibility      TEXT,
  modifiers       TEXT,            -- JSON: ["async","static","sealed","suspend","partial",...]
  signature_hash  TEXT,            -- composite hash: name+kind+visibility+modifiers+members+params
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  parent_symbol_id INTEGER REFERENCES symbols(id)
)
```

### symbol_fragments

Handles symbols that span multiple files or have multiple declaration sites. For example, C++ classes declared in a header and defined in an implementation file, or partial classes in C#. The `is_primary` flag marks the canonical declaration.

```sql
symbol_fragments (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  file_id         INTEGER NOT NULL REFERENCES files(id),
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  is_primary      BOOLEAN DEFAULT FALSE
)
```

### scopes

Lexical scope tree. Every block, function, class body, module, and file creates a scope. The `parent_scope_id` column forms a tree used during name resolution. A scope may optionally be associated with a symbol (e.g., a function scope linked to its function symbol).

```sql
scopes (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  symbol_id       INTEGER REFERENCES symbols(id),
  kind            TEXT NOT NULL,
    -- file, block, function, class, module, namespace, comptime
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  parent_scope_id INTEGER REFERENCES scopes(id)
)
```

### references

Every use of a name that is not its declaration. Includes function calls, type annotations, variable reads, field accesses, decorator applications, and import references. The `context` column classifies how the name is used, which helps resolvers narrow candidates.

```sql
references (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  scope_id        INTEGER REFERENCES scopes(id),
  name            TEXT NOT NULL,
  start_line      INTEGER,
  start_col       INTEGER,
  end_line        INTEGER,
  end_col         INTEGER,
  context         TEXT
    -- call, type_annotation, assignment, import, field_access,
    -- decorator, key_path, dynamic_dispatch
)
```

### imports

Import/require/include statements. The `source` column is the module path or file path. `imported_name` is what is brought into scope (NULL for wildcard imports). `local_alias` is the local name if aliased. The `kind` column distinguishes module imports from member imports, builtins, extern aliases, and forward declarations.

```sql
imports (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  source          TEXT NOT NULL,
  imported_name   TEXT,
  local_alias     TEXT,
  kind            TEXT DEFAULT 'module',
    -- module, member, builtin, extern_alias, forward_declaration
  scope           TEXT DEFAULT 'file'
    -- file, project
)
```

### type_members

Fields, methods, embedded types, properties, events, operators, and enum variants belonging to a type symbol. Extracted directly from the CST. The `type_expr` column stores the raw type expression as text (not resolved).

```sql
type_members (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT NOT NULL,
  kind            TEXT NOT NULL,
    -- field, method, embedded, property, event, operator, variant
  type_expr       TEXT,
  visibility      TEXT
)
```

### function_parameters

Parameters and return types for function/method symbols. The `ordinal` column preserves order. Special flags mark receivers (Go), return parameters, and defaults.

```sql
function_parameters (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT,
  ordinal         INTEGER NOT NULL,
  type_expr       TEXT,
  is_receiver     BOOLEAN DEFAULT FALSE,
  is_return       BOOLEAN DEFAULT FALSE,
  has_default     BOOLEAN DEFAULT FALSE,
  default_expr    TEXT
)
```

### type_parameters

Generic/template type parameters. Supports variance annotations (covariant/contravariant), different parameter kinds (type, value, anytype, associated_type), and constraint expressions.

```sql
type_parameters (
  id              INTEGER PRIMARY KEY,
  symbol_id       INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT NOT NULL,
  ordinal         INTEGER NOT NULL,
  variance        TEXT,              -- covariant, contravariant
  param_kind      TEXT DEFAULT 'type',
    -- type, value, anytype, associated_type
  constraints     TEXT
)
```

### annotations

Decorators, attributes, and annotations attached to symbols. The `resolved_symbol_id` is populated during the resolution phase if the annotation itself can be resolved to a symbol.

```sql
annotations (
  id              INTEGER PRIMARY KEY,
  target_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  name            TEXT NOT NULL,
  resolved_symbol_id INTEGER REFERENCES symbols(id),
  arguments       TEXT,
  file_id         INTEGER REFERENCES files(id),
  line            INTEGER,
  col             INTEGER
)
```

---

## Resolution Tables

### resolved_references

The core output of resolution. Maps each reference to its target symbol with a confidence score. A single reference may resolve to multiple targets (e.g., overloaded methods, dynamic dispatch candidates). The `resolution_kind` classifies how the resolution was achieved.

```sql
resolved_references (
  id              INTEGER PRIMARY KEY,
  reference_id    INTEGER NOT NULL REFERENCES references(id),
  target_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  confidence      REAL DEFAULT 1.0,
  resolution_kind TEXT
    -- direct, import, inheritance, interface, extension,
    -- comptime, dynamic_dispatch, companion
)
```

### implementations

Records which types implement which interfaces/traits/protocols. The `kind` column distinguishes explicit (Java `implements`), implicit (Go structural), structural (TypeScript duck typing), and delegation patterns.

```sql
implementations (
  id              INTEGER PRIMARY KEY,
  type_symbol_id  INTEGER NOT NULL REFERENCES symbols(id),
  interface_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  kind            TEXT,
    -- explicit, implicit, structural, delegation
  file_id         INTEGER REFERENCES files(id),
  declaring_module TEXT
)
```

### call_graph

Direct caller-callee relationships derived from resolved call references. Used by cortex for callers/callees/impact operations.

```sql
call_graph (
  id              INTEGER PRIMARY KEY,
  caller_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  callee_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  file_id         INTEGER REFERENCES files(id),
  line            INTEGER,
  col             INTEGER
)
```

### reexports

Tracks names that are re-exported from a file under the same or different name. Common in TypeScript barrel files, Python `__init__.py`, and Rust `pub use`.

```sql
reexports (
  id              INTEGER PRIMARY KEY,
  file_id         INTEGER NOT NULL REFERENCES files(id),
  original_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  exported_name   TEXT NOT NULL
)
```

### extension_bindings

Methods, properties, or subscripts added to a type outside its original declaration. Covers Go methods on types from same package, Swift/Kotlin extensions, Rust `impl` blocks, and C# extension methods.

```sql
extension_bindings (
  id              INTEGER PRIMARY KEY,
  member_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  extended_type_expr TEXT NOT NULL,
  extended_type_symbol_id INTEGER REFERENCES symbols(id),
  kind            TEXT DEFAULT 'method',
    -- method, property, subscript
  constraints     TEXT,
  is_default_impl BOOLEAN DEFAULT FALSE
)
```

### type_compositions

Composite type relationships beyond simple inheritance: error set merges (Zig), mixin includes (Ruby), type unions (TypeScript), and protocol compositions (Swift).

```sql
type_compositions (
  id              INTEGER PRIMARY KEY,
  composite_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  component_symbol_id INTEGER NOT NULL REFERENCES symbols(id),
  composition_kind TEXT NOT NULL
    -- error_set_merge, mixin_include, type_union, protocol_composition
)
```

---

## Phase Interaction

```
Source Files
    |
    v
[Phase 1: Risor Extraction Scripts]
    |  tree-sitter parse -> CST walk -> write rows
    v
Extraction Tables (files, symbols, scopes, references,
                    imports, type_members, function_parameters,
                    type_parameters, annotations, symbol_fragments)
    |
    v
[Phase 2: Risor Resolvers]
    |  query extraction tables -> semantic analysis -> write rows
    v
Resolution Tables (resolved_references, implementations,
                    call_graph, reexports, extension_bindings,
                    type_compositions)
    |
    v
[project-cortex queries resolution tables for graph operations]
```

## Indexes

The following indexes should be created for query performance:

```sql
CREATE INDEX idx_symbols_file ON symbols(file_id);
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_kind ON symbols(kind);
CREATE INDEX idx_symbols_parent ON symbols(parent_symbol_id);
CREATE INDEX idx_symbols_hash ON symbols(signature_hash);
CREATE INDEX idx_scopes_file ON scopes(file_id);
CREATE INDEX idx_scopes_parent ON scopes(parent_scope_id);
CREATE INDEX idx_references_file ON references(file_id);
CREATE INDEX idx_references_name ON references(name);
CREATE INDEX idx_references_scope ON references(scope_id);
CREATE INDEX idx_imports_file ON imports(file_id);
CREATE INDEX idx_imports_source ON imports(source);
CREATE INDEX idx_type_members_symbol ON type_members(symbol_id);
CREATE INDEX idx_function_params_symbol ON function_parameters(symbol_id);
CREATE INDEX idx_type_params_symbol ON type_parameters(symbol_id);
CREATE INDEX idx_annotations_target ON annotations(target_symbol_id);
CREATE INDEX idx_symbol_fragments_symbol ON symbol_fragments(symbol_id);
CREATE INDEX idx_resolved_refs_reference ON resolved_references(reference_id);
CREATE INDEX idx_resolved_refs_target ON resolved_references(target_symbol_id);
CREATE INDEX idx_implementations_type ON implementations(type_symbol_id);
CREATE INDEX idx_implementations_interface ON implementations(interface_symbol_id);
CREATE INDEX idx_call_graph_caller ON call_graph(caller_symbol_id);
CREATE INDEX idx_call_graph_callee ON call_graph(callee_symbol_id);
CREATE INDEX idx_reexports_file ON reexports(file_id);
CREATE INDEX idx_reexports_original ON reexports(original_symbol_id);
CREATE INDEX idx_extension_bindings_member ON extension_bindings(member_symbol_id);
CREATE INDEX idx_extension_bindings_type ON extension_bindings(extended_type_symbol_id);
CREATE INDEX idx_type_compositions_composite ON type_compositions(composite_symbol_id);
CREATE INDEX idx_type_compositions_component ON type_compositions(component_symbol_id);
```

## Incremental Resolution

When a file changes, canopy computes a blast radius to minimize re-resolution work. The `signature_hash` on `symbols` enables efficient change detection.

### Blast radius algorithm

```
1. Re-parse file with tree-sitter
2. Run extraction script → new symbols (in staging)
3. Compare old vs new symbols by (name, kind, parent):
   - removed:   old symbols with no matching new symbol
   - added:     new symbols with no matching old symbol
   - changed:   matching symbols where signature_hash differs
   - unchanged: matching symbols where signature_hash matches
4. Compute affected files:
   blast_radius = {this_file}
   blast_radius ∪= files with resolved_references targeting (removed ∪ changed) symbols
   if removed or added:
       blast_radius ∪= files that import this file's module/package
5. Clean up stale resolution data:
   - Delete resolved_references targeting removed symbols
   - Delete call_graph edges involving removed symbols
   - Delete implementations involving removed symbols
   - Delete extension_bindings involving removed symbols
   - Delete reexports referencing removed symbols
   - Delete type_compositions referencing removed symbols
6. Replace extraction data (delete old, insert new)
7. Re-resolve only files in blast_radius
```

### What signature_hash covers

The hash is computed from the symbol's semantic identity, not its location:
- name, kind, visibility, modifiers
- type_members: name, kind, type_expr, visibility (sorted, hashed)
- function_parameters: name, ordinal, type_expr, is_receiver, is_return (sorted, hashed)
- type_parameters: name, ordinal, variance, param_kind, constraints (sorted, hashed)

Location changes (start_line, end_line, etc.) do NOT affect the hash. A function that moves to a different line but doesn't change its signature does not trigger re-resolution of dependents.
