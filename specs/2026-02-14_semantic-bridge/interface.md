# Interface Definitions

## Globals Exposed to Risor Scripts

These are the objects and functions available to all Risor scripts via `risor.WithGlobals()`.

Risor's proxy system can call methods on go-tree-sitter CGO objects via reflection (e.g., `node.Type()`, `node.NamedChild(i)`, `node.ChildByFieldName("name")`). The host functions below exist only where Risor proxy limitations require Go-side handling: `[]byte` arguments, free constructor functions, and awkward multi-return cursor loops.

### parse

Parses a source file with tree-sitter and returns the tree-sitter Tree object.

```go
// parse(path string, language string) *sitter.Tree
//
// The returned Tree is the actual go-tree-sitter Tree object.
// Risor scripts can call any method on it via proxy reflection:
//   tree.RootNode()
//   node.Type()
//   node.ChildCount()
//   node.NamedChild(0)
//   node.ChildByFieldName("name")
//   node.Parent()
//   node.StartPoint(), node.EndPoint()
//   node.String()  (S-expression)
//   etc.
```

### node_text

Returns the source text of a tree-sitter node as a string. Exists because Risor's proxy system cannot convert strings to `[]byte`, which `node.Content([]byte)` requires.

```go
// node_text(node *sitter.Node) string
//
// Equivalent to: string(node.Content(sourceBytes))
// The source []byte is captured on the Go side during parse().
```

### query

Runs a tree-sitter S-expression query against a node and returns structured match results. Exists because (a) `sitter.NewQuery()` and `sitter.NewQueryCursor()` are free constructor functions that Risor can't call directly, and (b) the cursor iteration pattern (`NextMatch()` returns `(*QueryMatch, bool)` as a list in Risor) is awkward to use from scripts.

```go
// query(pattern string, node *sitter.Node) []map[string]any
//
// Each map in the returned list represents one match.
// Keys are capture names from the query pattern, values are proxied Nodes.
// Example:
//   matches := query("(function_declaration name: (identifier) @name)", root)
//   for m in matches {
//       print(node_text(m["name"]))
//   }
```

### db

The Store object, exposed directly. Risor scripts call methods on it.

### log

Logging functions: `log.info(msg)`, `log.warn(msg)`, `log.error(msg)`.

---
git@github.com:mvp-joe/claude-plugins.git
## Store

```go
type Store struct {
    // unexported fields
}

// Lifecycle

func NewStore(dbPath string) (*Store, error)
func (s *Store) Migrate() error
func (s *Store) Close() error

// File operations

func (s *Store) InsertFile(f *File) (int64, error)
func (s *Store) FileByPath(path string) (*File, error)
func (s *Store) FilesByLanguage(language string) ([]*File, error)
func (s *Store) DeleteFileData(fileID int64) error

// Symbol operations

func (s *Store) InsertSymbol(sym *Symbol) (int64, error)
func (s *Store) SymbolsByFile(fileID int64) ([]*Symbol, error)
func (s *Store) SymbolsByName(name string) ([]*Symbol, error)
func (s *Store) SymbolsByKind(kind string) ([]*Symbol, error)
func (s *Store) SymbolChildren(symbolID int64) ([]*Symbol, error)

// Scope operations

func (s *Store) InsertScope(scope *Scope) (int64, error)
func (s *Store) ScopesByFile(fileID int64) ([]*Scope, error)
func (s *Store) ScopeChain(scopeID int64) ([]*Scope, error)

// Reference operations

func (s *Store) InsertReference(ref *Reference) (int64, error)
func (s *Store) ReferencesByFile(fileID int64) ([]*Reference, error)
func (s *Store) ReferencesByName(name string) ([]*Reference, error)
func (s *Store) ReferencesInScope(scopeID int64) ([]*Reference, error)

// Import operations

func (s *Store) InsertImport(imp *Import) (int64, error)
func (s *Store) ImportsByFile(fileID int64) ([]*Import, error)

// Type member operations

func (s *Store) InsertTypeMember(tm *TypeMember) (int64, error)
func (s *Store) TypeMembers(symbolID int64) ([]*TypeMember, error)

// Function parameter operations

func (s *Store) InsertFunctionParam(fp *FunctionParam) (int64, error)
func (s *Store) FunctionParams(symbolID int64) ([]*FunctionParam, error)

// Type parameter operations

func (s *Store) InsertTypeParam(tp *TypeParam) (int64, error)
func (s *Store) TypeParams(symbolID int64) ([]*TypeParam, error)

// Annotation operations

func (s *Store) InsertAnnotation(ann *Annotation) (int64, error)
func (s *Store) AnnotationsByTarget(symbolID int64) ([]*Annotation, error)

// Symbol fragment operations

func (s *Store) InsertSymbolFragment(frag *SymbolFragment) (int64, error)
func (s *Store) SymbolFragments(symbolID int64) ([]*SymbolFragment, error)

// Resolution operations

func (s *Store) InsertResolvedReference(rr *ResolvedReference) (int64, error)
func (s *Store) ResolvedReferencesByRef(referenceID int64) ([]*ResolvedReference, error)
func (s *Store) ResolvedReferencesByTarget(symbolID int64) ([]*ResolvedReference, error)

func (s *Store) InsertImplementation(impl *Implementation) (int64, error)
func (s *Store) ImplementationsByType(typeSymbolID int64) ([]*Implementation, error)
func (s *Store) ImplementationsByInterface(interfaceSymbolID int64) ([]*Implementation, error)

func (s *Store) InsertCallEdge(edge *CallEdge) (int64, error)
func (s *Store) CallersByCallee(calleeSymbolID int64) ([]*CallEdge, error)
func (s *Store) CalleesByCaller(callerSymbolID int64) ([]*CallEdge, error)

func (s *Store) InsertReexport(re *Reexport) (int64, error)
func (s *Store) ReexportsByFile(fileID int64) ([]*Reexport, error)

func (s *Store) InsertExtensionBinding(eb *ExtensionBinding) (int64, error)
func (s *Store) ExtensionBindingsByType(typeSymbolID int64) ([]*ExtensionBinding, error)

func (s *Store) InsertTypeComposition(tc *TypeComposition) (int64, error)
func (s *Store) TypeCompositions(compositeSymbolID int64) ([]*TypeComposition, error)

// Incremental resolution — blast radius computation

func (s *Store) FilesReferencingSymbols(symbolIDs []int64) ([]int64, error)  // file IDs with resolved_references targeting these symbols
func (s *Store) FilesImportingSource(source string) ([]int64, error)         // file IDs that import this module/package
func (s *Store) DeleteResolutionDataForSymbols(symbolIDs []int64) error      // remove resolved_references, call_graph, implementations, extension_bindings, reexports, type_compositions targeting these symbols
func (s *Store) DeleteResolutionDataForFiles(fileIDs []int64) error          // remove all resolution data originating from these files
```

---

## Domain Types

```go
type File struct {
    ID          int64
    Path        string
    Language    string
    Hash        string
    LastIndexed time.Time
}

type Symbol struct {
    ID             int64
    FileID         *int64  // nil for multi-file symbols (namespaces, packages)
    Name           string
    Kind           string  // function, method, class, struct, interface, trait, protocol,
                           // enum, variable, constant, type_alias, module, package, namespace,
                           // property, accessor, delegate, event, operator, record,
                           // error_set, test, actor, object, companion_object
    Visibility     string
    Modifiers      []string // ["async","static","sealed","suspend","partial",...]
    SignatureHash  string   // composite hash of name+kind+visibility+modifiers+members+params
    StartLine      int
    StartCol       int
    EndLine        int
    EndCol         int
    ParentSymbolID *int64
}

type SymbolFragment struct {
    ID        int64
    SymbolID  int64
    FileID    int64
    StartLine int
    StartCol  int
    EndLine   int
    EndCol    int
    IsPrimary bool
}

type Scope struct {
    ID            int64
    FileID        int64
    SymbolID      *int64
    Kind          string // file, block, function, class, module, namespace, comptime
    StartLine     int
    StartCol      int
    EndLine       int
    EndCol        int
    ParentScopeID *int64
}

type Reference struct {
    ID        int64
    FileID    int64
    ScopeID   *int64
    Name      string
    StartLine int
    StartCol  int
    EndLine   int
    EndCol    int
    Context   string // call, type_annotation, assignment, import, field_access,
                     // decorator, key_path, dynamic_dispatch
}

type Import struct {
    ID           int64
    FileID       int64
    Source       string
    ImportedName *string
    LocalAlias   *string
    Kind         string // module, member, builtin, extern_alias, forward_declaration
    Scope        string // file, project
}

type TypeMember struct {
    ID         int64
    SymbolID   int64
    Name       string
    Kind       string // field, method, embedded, property, event, operator, variant
    TypeExpr   string
    Visibility string
}

type FunctionParam struct {
    ID          int64
    SymbolID    int64
    Name        string
    Ordinal     int
    TypeExpr    string
    IsReceiver  bool
    IsReturn    bool
    HasDefault  bool
    DefaultExpr string
}

type TypeParam struct {
    ID          int64
    SymbolID    int64
    Name        string
    Ordinal     int
    Variance    string // covariant, contravariant
    ParamKind   string // type, value, anytype, associated_type
    Constraints string
}

type Annotation struct {
    ID               int64
    TargetSymbolID   int64
    Name             string
    ResolvedSymbolID *int64
    Arguments        string
    FileID           *int64
    Line             int
    Col              int
}

type ResolvedReference struct {
    ID             int64
    ReferenceID    int64
    TargetSymbolID int64
    Confidence     float64
    ResolutionKind string // direct, import, inheritance, interface, extension,
                          // comptime, dynamic_dispatch, companion
}

type Implementation struct {
    ID                int64
    TypeSymbolID      int64
    InterfaceSymbolID int64
    Kind              string // explicit, implicit, structural, delegation
    FileID            *int64
    DeclaringModule   string
}

type CallEdge struct {
    ID             int64
    CallerSymbolID int64
    CalleeSymbolID int64
    FileID         *int64
    Line           int
    Col            int
}

type Reexport struct {
    ID               int64
    FileID           int64
    OriginalSymbolID int64
    ExportedName     string
}

type ExtensionBinding struct {
    ID                    int64
    MemberSymbolID        int64
    ExtendedTypeExpr      string
    ExtendedTypeSymbolID  *int64
    Kind                  string // method, property, subscript
    Constraints           string
    IsDefaultImpl         bool
}

type TypeComposition struct {
    ID                int64
    CompositeSymbolID int64
    ComponentSymbolID int64
    CompositionKind   string // error_set_merge, mixin_include, type_union, protocol_composition
}
```

---

## Engine (public API for cortex)

```go
type Engine struct {
    // unexported fields
}

func New(dbPath string, scriptsDir string, opts ...Option) (*Engine, error)

func (e *Engine) IndexFiles(ctx context.Context, paths []string) error
func (e *Engine) IndexDirectory(ctx context.Context, root string) error
func (e *Engine) Resolve(ctx context.Context) error
func (e *Engine) Query() *QueryBuilder
func (e *Engine) Store() *Store
func (e *Engine) Close() error

type Option func(*Engine)

func WithLanguages(languages ...string) Option
```

---

## QueryBuilder (cortex-facing query API)

```go
type QueryBuilder struct {
    // unexported fields
}

type Location struct {
    File      string
    StartLine int
    StartCol  int
    EndLine   int
    EndCol    int
}

func (q *QueryBuilder) DefinitionAt(file string, line, col int) ([]Location, error)
func (q *QueryBuilder) ReferencesTo(symbolID int64) ([]Location, error)
func (q *QueryBuilder) Implementations(symbolID int64) ([]Location, error)
func (q *QueryBuilder) Callers(symbolID int64) ([]*CallEdge, error)
func (q *QueryBuilder) Callees(symbolID int64) ([]*CallEdge, error)
func (q *QueryBuilder) Dependencies(fileID int64) ([]*Import, error)
func (q *QueryBuilder) Dependents(source string) ([]*Import, error)
```
