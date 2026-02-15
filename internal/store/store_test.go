package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())
	t.Cleanup(func() { s.Close() })
	return s
}

func ptr[T any](v T) *T { return &v }

// insertTestFile is a helper that inserts a file and returns it with ID set.
func insertTestFile(t *testing.T, s *Store, path, lang string) *File {
	t.Helper()
	f := &File{Path: path, Language: lang, Hash: "abc123", LastIndexed: time.Now().Truncate(time.Second)}
	id, err := s.InsertFile(f)
	require.NoError(t, err)
	require.Positive(t, id)
	return f
}

// insertTestSymbol inserts a symbol with minimal required fields.
func insertTestSymbol(t *testing.T, s *Store, fileID *int64, name, kind string) *Symbol {
	t.Helper()
	sym := &Symbol{
		FileID:     fileID,
		Name:       name,
		Kind:       kind,
		Visibility: "public",
		Modifiers:  []string{"async"},
		StartLine:  0, StartCol: 0, EndLine: 9, EndCol: 0,
	}
	id, err := s.InsertSymbol(sym)
	require.NoError(t, err)
	require.Positive(t, id)
	return sym
}

// =============================================================================
// Schema & Lifecycle
// =============================================================================

func TestMigrate_AllTablesExist(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	expectedTables := []string{
		"files", "symbols", "symbol_fragments", "scopes", "references_",
		"imports", "type_members", "function_parameters", "type_parameters", "annotations",
		"resolved_references", "implementations", "call_graph", "reexports",
		"extension_bindings", "type_compositions",
	}

	for _, table := range expectedTables {
		var name string
		err := s.db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		require.NoError(t, err, "table %s should exist", table)
		assert.Equal(t, table, name)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	// Running migrate again should not error.
	require.NoError(t, s.Migrate())
}

func TestMigrate_WALMode(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	var mode string
	err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	require.NoError(t, err)
	assert.Equal(t, "wal", mode)
}

// =============================================================================
// File operations
// =============================================================================

func TestFile_InsertAndRetrieve(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	now := time.Now().Truncate(time.Second)
	f := &File{Path: "/src/main.go", Language: "go", Hash: "sha256abc", LastIndexed: now}
	id, err := s.InsertFile(f)
	require.NoError(t, err)
	require.Positive(t, id)

	got, err := s.FileByPath("/src/main.go")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "/src/main.go", got.Path)
	assert.Equal(t, "go", got.Language)
	assert.Equal(t, "sha256abc", got.Hash)
}

func TestFile_ByPathNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	got, err := s.FileByPath("/nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFile_ByLanguage(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	insertTestFile(t, s, "/a.go", "go")
	insertTestFile(t, s, "/b.go", "go")
	insertTestFile(t, s, "/c.py", "python")

	goFiles, err := s.FilesByLanguage("go")
	require.NoError(t, err)
	assert.Len(t, goFiles, 2)

	pyFiles, err := s.FilesByLanguage("python")
	require.NoError(t, err)
	assert.Len(t, pyFiles, 1)
}

// =============================================================================
// Symbol operations
// =============================================================================

func TestSymbol_InsertAndQueryByFile(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")

	sym := &Symbol{
		FileID: &f.ID, Name: "Foo", Kind: "function", Visibility: "public",
		Modifiers: []string{"async", "static"}, SignatureHash: "hash1",
		StartLine: 4, StartCol: 0, EndLine: 19, EndCol: 1,
	}
	id, err := s.InsertSymbol(sym)
	require.NoError(t, err)
	require.Positive(t, id)

	symbols, err := s.SymbolsByFile(f.ID)
	require.NoError(t, err)
	require.Len(t, symbols, 1)
	assert.Equal(t, "Foo", symbols[0].Name)
	assert.Equal(t, "function", symbols[0].Kind)
	assert.Equal(t, []string{"async", "static"}, symbols[0].Modifiers)
	assert.Equal(t, 4, symbols[0].StartLine)
}

func TestSymbol_QueryByName(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	insertTestSymbol(t, s, &f.ID, "Foo", "function")
	insertTestSymbol(t, s, &f.ID, "Bar", "function")

	syms, err := s.SymbolsByName("Foo")
	require.NoError(t, err)
	require.Len(t, syms, 1)
	assert.Equal(t, "Foo", syms[0].Name)
}

func TestSymbol_QueryByKind(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	insertTestSymbol(t, s, &f.ID, "Foo", "function")
	insertTestSymbol(t, s, &f.ID, "MyStruct", "struct")

	syms, err := s.SymbolsByKind("struct")
	require.NoError(t, err)
	require.Len(t, syms, 1)
	assert.Equal(t, "MyStruct", syms[0].Name)
}

func TestSymbol_Children(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	parent := insertTestSymbol(t, s, &f.ID, "MyClass", "class")

	child := &Symbol{
		FileID: &f.ID, Name: "myMethod", Kind: "method",
		ParentSymbolID: &parent.ID,
		StartLine:      2, StartCol: 0, EndLine: 7, EndCol: 0,
	}
	_, err := s.InsertSymbol(child)
	require.NoError(t, err)

	children, err := s.SymbolChildren(parent.ID)
	require.NoError(t, err)
	require.Len(t, children, 1)
	assert.Equal(t, "myMethod", children[0].Name)
}

func TestSymbol_NilFileID(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	sym := &Symbol{Name: "mypkg", Kind: "package"}
	id, err := s.InsertSymbol(sym)
	require.NoError(t, err)
	require.Positive(t, id)

	syms, err := s.SymbolsByName("mypkg")
	require.NoError(t, err)
	require.Len(t, syms, 1)
	assert.Nil(t, syms[0].FileID)
}

// =============================================================================
// Scope operations
// =============================================================================

func TestScope_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")

	fileScope := &Scope{FileID: f.ID, Kind: "file", StartLine: 0, StartCol: 0, EndLine: 99, EndCol: 0}
	_, err := s.InsertScope(fileScope)
	require.NoError(t, err)

	funcScope := &Scope{FileID: f.ID, Kind: "function", StartLine: 4, StartCol: 0, EndLine: 19, EndCol: 0, ParentScopeID: &fileScope.ID}
	_, err = s.InsertScope(funcScope)
	require.NoError(t, err)

	blockScope := &Scope{FileID: f.ID, Kind: "block", StartLine: 9, StartCol: 0, EndLine: 14, EndCol: 0, ParentScopeID: &funcScope.ID}
	_, err = s.InsertScope(blockScope)
	require.NoError(t, err)

	scopes, err := s.ScopesByFile(f.ID)
	require.NoError(t, err)
	assert.Len(t, scopes, 3)
}

func TestScope_Chain(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")

	fileScope := &Scope{FileID: f.ID, Kind: "file", StartLine: 0, EndLine: 99}
	_, err := s.InsertScope(fileScope)
	require.NoError(t, err)

	funcScope := &Scope{FileID: f.ID, Kind: "function", StartLine: 4, EndLine: 19, ParentScopeID: &fileScope.ID}
	_, err = s.InsertScope(funcScope)
	require.NoError(t, err)

	blockScope := &Scope{FileID: f.ID, Kind: "block", StartLine: 9, EndLine: 14, ParentScopeID: &funcScope.ID}
	_, err = s.InsertScope(blockScope)
	require.NoError(t, err)

	chain, err := s.ScopeChain(blockScope.ID)
	require.NoError(t, err)
	require.Len(t, chain, 3)
	assert.Equal(t, "block", chain[0].Kind)
	assert.Equal(t, "function", chain[1].Kind)
	assert.Equal(t, "file", chain[2].Kind)
}

// =============================================================================
// Reference operations
// =============================================================================

func TestReference_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	scope := &Scope{FileID: f.ID, Kind: "file", StartLine: 0, EndLine: 99}
	_, err := s.InsertScope(scope)
	require.NoError(t, err)

	ref := &Reference{
		FileID: f.ID, ScopeID: &scope.ID, Name: "Bar",
		StartLine: 9, StartCol: 5, EndLine: 9, EndCol: 8, Context: "call",
	}
	id, err := s.InsertReference(ref)
	require.NoError(t, err)
	require.Positive(t, id)

	refs, err := s.ReferencesByFile(f.ID)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "Bar", refs[0].Name)
	assert.Equal(t, "call", refs[0].Context)
	assert.Equal(t, 9, refs[0].StartLine)
	assert.Equal(t, 5, refs[0].StartCol)
}

func TestReference_ByName(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	s.InsertReference(&Reference{FileID: f.ID, Name: "Foo", StartLine: 0, EndLine: 0, Context: "call"})
	s.InsertReference(&Reference{FileID: f.ID, Name: "Bar", StartLine: 1, EndLine: 1, Context: "type_annotation"})

	refs, err := s.ReferencesByName("Foo")
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "call", refs[0].Context)
}

func TestReference_InScope(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	scope := &Scope{FileID: f.ID, Kind: "function", StartLine: 0, EndLine: 19}
	s.InsertScope(scope)

	s.InsertReference(&Reference{FileID: f.ID, ScopeID: &scope.ID, Name: "x", StartLine: 4, EndLine: 4})
	s.InsertReference(&Reference{FileID: f.ID, Name: "y", StartLine: 24, EndLine: 24}) // no scope

	refs, err := s.ReferencesInScope(scope.ID)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "x", refs[0].Name)
}

// =============================================================================
// Import operations
// =============================================================================

func TestImport_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")

	imports := []*Import{
		{FileID: f.ID, Source: "fmt", Kind: "module", Scope: "file"},
		{FileID: f.ID, Source: "os", ImportedName: ptr("ReadFile"), Kind: "member", Scope: "file"},
		{FileID: f.ID, Source: "builtin", Kind: "builtin", Scope: "project"},
	}
	for _, imp := range imports {
		id, err := s.InsertImport(imp)
		require.NoError(t, err)
		require.Positive(t, id)
	}

	got, err := s.ImportsByFile(f.ID)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// Verify field values on the member import.
	var memberImport *Import
	for _, imp := range got {
		if imp.Kind == "member" {
			memberImport = imp
			break
		}
	}
	require.NotNil(t, memberImport)
	assert.Equal(t, "os", memberImport.Source)
	assert.Equal(t, "ReadFile", *memberImport.ImportedName)
}

// =============================================================================
// TypeMember operations
// =============================================================================

func TestTypeMember_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/types.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "MyStruct", "struct")

	members := []*TypeMember{
		{SymbolID: sym.ID, Name: "Name", Kind: "field", TypeExpr: "string", Visibility: "public"},
		{SymbolID: sym.ID, Name: "age", Kind: "field", TypeExpr: "int", Visibility: "private"},
		{SymbolID: sym.ID, Name: "io.Reader", Kind: "embedded", TypeExpr: "io.Reader"},
	}
	for _, tm := range members {
		id, err := s.InsertTypeMember(tm)
		require.NoError(t, err)
		require.Positive(t, id)
	}

	got, err := s.TypeMembers(sym.ID)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "field", got[0].Kind)
}

// =============================================================================
// FunctionParam operations
// =============================================================================

func TestFunctionParam_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "Process", "method")

	params := []*FunctionParam{
		{SymbolID: sym.ID, Name: "s", Ordinal: 0, TypeExpr: "*Server", IsReceiver: true},
		{SymbolID: sym.ID, Name: "ctx", Ordinal: 1, TypeExpr: "context.Context"},
		{SymbolID: sym.ID, Name: "input", Ordinal: 2, TypeExpr: "string"},
		{SymbolID: sym.ID, Name: "", Ordinal: 3, TypeExpr: "error", IsReturn: true},
	}
	for _, fp := range params {
		id, err := s.InsertFunctionParam(fp)
		require.NoError(t, err)
		require.Positive(t, id)
	}

	got, err := s.FunctionParams(sym.ID)
	require.NoError(t, err)
	require.Len(t, got, 4)
	// Verify ordering.
	assert.Equal(t, 0, got[0].Ordinal)
	assert.True(t, got[0].IsReceiver)
	assert.Equal(t, 3, got[3].Ordinal)
	assert.True(t, got[3].IsReturn)
}

// =============================================================================
// TypeParam operations
// =============================================================================

func TestTypeParam_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/generic.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "Map", "function")

	tps := []*TypeParam{
		{SymbolID: sym.ID, Name: "K", Ordinal: 0, ParamKind: "type", Constraints: "comparable"},
		{SymbolID: sym.ID, Name: "V", Ordinal: 1, Variance: "covariant", ParamKind: "type"},
	}
	for _, tp := range tps {
		id, err := s.InsertTypeParam(tp)
		require.NoError(t, err)
		require.Positive(t, id)
	}

	got, err := s.TypeParams(sym.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "K", got[0].Name)
	assert.Equal(t, "comparable", got[0].Constraints)
	assert.Equal(t, "covariant", got[1].Variance)
}

// =============================================================================
// Annotation operations
// =============================================================================

func TestAnnotation_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/handler.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "Handler", "class")

	ann := &Annotation{
		TargetSymbolID: sym.ID, Name: "Deprecated",
		Arguments: `{"since":"v2"}`, FileID: &f.ID, Line: 3, Col: 0,
	}
	id, err := s.InsertAnnotation(ann)
	require.NoError(t, err)
	require.Positive(t, id)

	got, err := s.AnnotationsByTarget(sym.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Deprecated", got[0].Name)
	assert.Equal(t, `{"since":"v2"}`, got[0].Arguments)
}

// =============================================================================
// SymbolFragment operations
// =============================================================================

func TestSymbolFragment_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f1 := insertTestFile(t, s, "/header.h", "cpp")
	f2 := insertTestFile(t, s, "/impl.cpp", "cpp")
	sym := insertTestSymbol(t, s, &f1.ID, "MyClass", "class")

	frags := []*SymbolFragment{
		{SymbolID: sym.ID, FileID: f1.ID, StartLine: 0, EndLine: 9, IsPrimary: true},
		{SymbolID: sym.ID, FileID: f2.ID, StartLine: 0, EndLine: 49, IsPrimary: false},
	}
	for _, frag := range frags {
		id, err := s.InsertSymbolFragment(frag)
		require.NoError(t, err)
		require.Positive(t, id)
	}

	got, err := s.SymbolFragments(sym.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)

	var primary, secondary *SymbolFragment
	for _, g := range got {
		if g.IsPrimary {
			primary = g
		} else {
			secondary = g
		}
	}
	require.NotNil(t, primary)
	require.NotNil(t, secondary)
	assert.Equal(t, f1.ID, primary.FileID)
	assert.Equal(t, f2.ID, secondary.FileID)
}

// =============================================================================
// Resolution table operations
// =============================================================================

func TestResolvedReference_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "Bar", "function")
	ref := &Reference{FileID: f.ID, Name: "Bar", StartLine: 9, EndLine: 9, Context: "call"}
	s.InsertReference(ref)

	rr := &ResolvedReference{
		ReferenceID: ref.ID, TargetSymbolID: sym.ID,
		Confidence: 0.95, ResolutionKind: "direct",
	}
	id, err := s.InsertResolvedReference(rr)
	require.NoError(t, err)
	require.Positive(t, id)

	byRef, err := s.ResolvedReferencesByRef(ref.ID)
	require.NoError(t, err)
	require.Len(t, byRef, 1)
	assert.Equal(t, 0.95, byRef[0].Confidence)
	assert.Equal(t, "direct", byRef[0].ResolutionKind)

	byTarget, err := s.ResolvedReferencesByTarget(sym.ID)
	require.NoError(t, err)
	require.Len(t, byTarget, 1)
}

func TestImplementation_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/types.go", "go")
	iface := insertTestSymbol(t, s, &f.ID, "Reader", "interface")
	typ := insertTestSymbol(t, s, &f.ID, "MyReader", "struct")

	impl := &Implementation{
		TypeSymbolID: typ.ID, InterfaceSymbolID: iface.ID,
		Kind: "implicit", FileID: &f.ID, DeclaringModule: "pkg",
	}
	id, err := s.InsertImplementation(impl)
	require.NoError(t, err)
	require.Positive(t, id)

	byType, err := s.ImplementationsByType(typ.ID)
	require.NoError(t, err)
	require.Len(t, byType, 1)
	assert.Equal(t, "implicit", byType[0].Kind)

	byIface, err := s.ImplementationsByInterface(iface.ID)
	require.NoError(t, err)
	require.Len(t, byIface, 1)
	assert.Equal(t, typ.ID, byIface[0].TypeSymbolID)
}

func TestCallEdge_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	caller := insertTestSymbol(t, s, &f.ID, "Foo", "function")
	callee := insertTestSymbol(t, s, &f.ID, "Bar", "function")

	edge := &CallEdge{
		CallerSymbolID: caller.ID, CalleeSymbolID: callee.ID,
		FileID: &f.ID, Line: 14, Col: 3,
	}
	id, err := s.InsertCallEdge(edge)
	require.NoError(t, err)
	require.Positive(t, id)

	callers, err := s.CallersByCallee(callee.ID)
	require.NoError(t, err)
	require.Len(t, callers, 1)
	assert.Equal(t, caller.ID, callers[0].CallerSymbolID)

	callees, err := s.CalleesByCaller(caller.ID)
	require.NoError(t, err)
	require.Len(t, callees, 1)
	assert.Equal(t, callee.ID, callees[0].CalleeSymbolID)
}

func TestReexport_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/index.ts", "typescript")
	sym := insertTestSymbol(t, s, &f.ID, "Component", "class")

	re := &Reexport{FileID: f.ID, OriginalSymbolID: sym.ID, ExportedName: "Component"}
	id, err := s.InsertReexport(re)
	require.NoError(t, err)
	require.Positive(t, id)

	got, err := s.ReexportsByFile(f.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Component", got[0].ExportedName)
}

func TestExtensionBinding_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/ext.swift", "swift")
	typeSym := insertTestSymbol(t, s, &f.ID, "String", "struct")
	methodSym := insertTestSymbol(t, s, &f.ID, "trimmed", "method")

	eb := &ExtensionBinding{
		MemberSymbolID:       methodSym.ID,
		ExtendedTypeExpr:     "String",
		ExtendedTypeSymbolID: &typeSym.ID,
		Kind:                 "method",
		Constraints:          "where Self: Equatable",
		IsDefaultImpl:        false,
	}
	id, err := s.InsertExtensionBinding(eb)
	require.NoError(t, err)
	require.Positive(t, id)

	got, err := s.ExtensionBindingsByType(typeSym.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "method", got[0].Kind)
	assert.Equal(t, "String", got[0].ExtendedTypeExpr)
}

func TestTypeComposition_InsertAndQuery(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/types.rb", "ruby")
	composite := insertTestSymbol(t, s, &f.ID, "MyClass", "class")
	component := insertTestSymbol(t, s, &f.ID, "Enumerable", "module")

	tc := &TypeComposition{
		CompositeSymbolID: composite.ID, ComponentSymbolID: component.ID,
		CompositionKind: "mixin_include",
	}
	id, err := s.InsertTypeComposition(tc)
	require.NoError(t, err)
	require.Positive(t, id)

	got, err := s.TypeCompositions(composite.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "mixin_include", got[0].CompositionKind)
	assert.Equal(t, component.ID, got[0].ComponentSymbolID)
}

// =============================================================================
// DeleteFileData (transactional re-index)
// =============================================================================

func TestDeleteFileData(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")

	// Populate extraction data.
	sym := insertTestSymbol(t, s, &f.ID, "Foo", "function")
	scope := &Scope{FileID: f.ID, Kind: "file", StartLine: 0, EndLine: 99}
	s.InsertScope(scope)
	s.InsertReference(&Reference{FileID: f.ID, ScopeID: &scope.ID, Name: "Bar", StartLine: 9, EndLine: 9})
	s.InsertImport(&Import{FileID: f.ID, Source: "fmt", Kind: "module", Scope: "file"})
	s.InsertTypeMember(&TypeMember{SymbolID: sym.ID, Name: "X", Kind: "field", TypeExpr: "int"})
	s.InsertFunctionParam(&FunctionParam{SymbolID: sym.ID, Name: "ctx", Ordinal: 0, TypeExpr: "context.Context"})
	s.InsertTypeParam(&TypeParam{SymbolID: sym.ID, Name: "T", Ordinal: 0, ParamKind: "type"})
	s.InsertAnnotation(&Annotation{TargetSymbolID: sym.ID, Name: "Test", FileID: &f.ID})
	s.InsertSymbolFragment(&SymbolFragment{SymbolID: sym.ID, FileID: f.ID, StartLine: 0, EndLine: 9, IsPrimary: true})

	// Populate resolution data.
	ref := &Reference{FileID: f.ID, Name: "Baz", StartLine: 14, EndLine: 14}
	s.InsertReference(ref)
	s.InsertResolvedReference(&ResolvedReference{ReferenceID: ref.ID, TargetSymbolID: sym.ID, Confidence: 1.0, ResolutionKind: "direct"})
	s.InsertCallEdge(&CallEdge{CallerSymbolID: sym.ID, CalleeSymbolID: sym.ID, FileID: &f.ID, Line: 14})
	s.InsertImplementation(&Implementation{TypeSymbolID: sym.ID, InterfaceSymbolID: sym.ID, Kind: "implicit", FileID: &f.ID})
	s.InsertReexport(&Reexport{FileID: f.ID, OriginalSymbolID: sym.ID, ExportedName: "Foo"})

	// Delete all data for the file.
	err := s.DeleteFileData(f.ID)
	require.NoError(t, err)

	// Verify everything is gone.
	syms, _ := s.SymbolsByFile(f.ID)
	assert.Empty(t, syms)

	scopes, _ := s.ScopesByFile(f.ID)
	assert.Empty(t, scopes)

	refs, _ := s.ReferencesByFile(f.ID)
	assert.Empty(t, refs)

	imports, _ := s.ImportsByFile(f.ID)
	assert.Empty(t, imports)
}

func TestDeleteFileData_ReindexWithNewData(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")

	// Initial data.
	insertTestSymbol(t, s, &f.ID, "OldFunc", "function")
	syms, _ := s.SymbolsByFile(f.ID)
	require.Len(t, syms, 1)

	// Re-index: delete old, insert new.
	require.NoError(t, s.DeleteFileData(f.ID))
	insertTestSymbol(t, s, &f.ID, "NewFunc", "function")

	syms, err := s.SymbolsByFile(f.ID)
	require.NoError(t, err)
	require.Len(t, syms, 1)
	assert.Equal(t, "NewFunc", syms[0].Name)
}

// =============================================================================
// Signature Hash
// =============================================================================

func TestSignatureHash_Deterministic(t *testing.T) {
	t.Parallel()
	members := []*TypeMember{{Name: "x", Kind: "field", TypeExpr: "int", Visibility: "public"}}
	params := []*FunctionParam{{Name: "a", Ordinal: 0, TypeExpr: "string"}}
	tps := []*TypeParam{{Name: "T", Ordinal: 0, ParamKind: "type", Constraints: "comparable"}}

	h1 := ComputeSignatureHash("Foo", "function", "public", []string{"async"}, members, params, tps)
	h2 := ComputeSignatureHash("Foo", "function", "public", []string{"async"}, members, params, tps)
	assert.Equal(t, h1, h2)
	assert.NotEmpty(t, h1)
}

func TestSignatureHash_ChangeName(t *testing.T) {
	t.Parallel()
	h1 := ComputeSignatureHash("Foo", "function", "public", nil, nil, nil, nil)
	h2 := ComputeSignatureHash("Bar", "function", "public", nil, nil, nil, nil)
	assert.NotEqual(t, h1, h2)
}

func TestSignatureHash_ChangeVisibility(t *testing.T) {
	t.Parallel()
	h1 := ComputeSignatureHash("Foo", "function", "public", nil, nil, nil, nil)
	h2 := ComputeSignatureHash("Foo", "function", "private", nil, nil, nil, nil)
	assert.NotEqual(t, h1, h2)
}

func TestSignatureHash_ChangeModifiers(t *testing.T) {
	t.Parallel()
	h1 := ComputeSignatureHash("Foo", "function", "public", []string{"async"}, nil, nil, nil)
	h2 := ComputeSignatureHash("Foo", "function", "public", []string{"static"}, nil, nil, nil)
	assert.NotEqual(t, h1, h2)
}

func TestSignatureHash_AddTypeMember(t *testing.T) {
	t.Parallel()
	h1 := ComputeSignatureHash("MyStruct", "struct", "public", nil, nil, nil, nil)
	h2 := ComputeSignatureHash("MyStruct", "struct", "public", nil,
		[]*TypeMember{{Name: "x", Kind: "field", TypeExpr: "int"}}, nil, nil)
	assert.NotEqual(t, h1, h2)
}

func TestSignatureHash_AddFunctionParam(t *testing.T) {
	t.Parallel()
	h1 := ComputeSignatureHash("Foo", "function", "public", nil, nil, nil, nil)
	h2 := ComputeSignatureHash("Foo", "function", "public", nil, nil,
		[]*FunctionParam{{Name: "a", Ordinal: 0, TypeExpr: "int"}}, nil)
	assert.NotEqual(t, h1, h2)
}

func TestSignatureHash_UnchangedSymbol(t *testing.T) {
	t.Parallel()
	members := []*TypeMember{{Name: "x", Kind: "field", TypeExpr: "int", Visibility: "public"}}
	params := []*FunctionParam{{Name: "a", Ordinal: 0, TypeExpr: "string"}}

	h1 := ComputeSignatureHash("Foo", "function", "public", []string{"async"}, members, params, nil)
	h2 := ComputeSignatureHash("Foo", "function", "public", []string{"async"}, members, params, nil)
	assert.Equal(t, h1, h2)
}

// =============================================================================
// Blast radius methods
// =============================================================================

func TestFilesReferencingSymbols(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	fC := insertTestFile(t, s, "/c.go", "go")
	symC := insertTestSymbol(t, s, &fC.ID, "Helper", "function")

	fA := insertTestFile(t, s, "/a.go", "go")
	refA := &Reference{FileID: fA.ID, Name: "Helper", StartLine: 4, EndLine: 4, Context: "call"}
	s.InsertReference(refA)
	s.InsertResolvedReference(&ResolvedReference{ReferenceID: refA.ID, TargetSymbolID: symC.ID, Confidence: 1.0, ResolutionKind: "direct"})

	fB := insertTestFile(t, s, "/b.go", "go")
	refB := &Reference{FileID: fB.ID, Name: "Helper", StartLine: 7, EndLine: 7, Context: "call"}
	s.InsertReference(refB)
	s.InsertResolvedReference(&ResolvedReference{ReferenceID: refB.ID, TargetSymbolID: symC.ID, Confidence: 1.0, ResolutionKind: "direct"})

	fileIDs, err := s.FilesReferencingSymbols([]int64{symC.ID})
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{fA.ID, fB.ID}, fileIDs)
}

func TestFilesReferencingSymbols_NoReferences(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/lonely.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "Unused", "function")

	fileIDs, err := s.FilesReferencingSymbols([]int64{sym.ID})
	require.NoError(t, err)
	assert.Empty(t, fileIDs)
}

func TestFilesImportingSource(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	fA := insertTestFile(t, s, "/a.go", "go")
	fB := insertTestFile(t, s, "/b.go", "go")
	insertTestFile(t, s, "/c.go", "go")

	s.InsertImport(&Import{FileID: fA.ID, Source: "pkg/foo", Kind: "module", Scope: "file"})
	s.InsertImport(&Import{FileID: fB.ID, Source: "pkg/foo", Kind: "module", Scope: "file"})

	fileIDs, err := s.FilesImportingSource("pkg/foo")
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{fA.ID, fB.ID}, fileIDs)
}

func TestDeleteResolutionDataForSymbols(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "Target", "function")

	// Create resolution data targeting sym.
	ref := &Reference{FileID: f.ID, Name: "Target", StartLine: 9, EndLine: 9}
	s.InsertReference(ref)
	s.InsertResolvedReference(&ResolvedReference{ReferenceID: ref.ID, TargetSymbolID: sym.ID, Confidence: 1.0})
	s.InsertCallEdge(&CallEdge{CallerSymbolID: sym.ID, CalleeSymbolID: sym.ID, FileID: &f.ID, Line: 9})
	s.InsertImplementation(&Implementation{TypeSymbolID: sym.ID, InterfaceSymbolID: sym.ID, Kind: "implicit", FileID: &f.ID})

	err := s.DeleteResolutionDataForSymbols([]int64{sym.ID})
	require.NoError(t, err)

	rr, _ := s.ResolvedReferencesByTarget(sym.ID)
	assert.Empty(t, rr)

	callers, _ := s.CallersByCallee(sym.ID)
	assert.Empty(t, callers)

	impls, _ := s.ImplementationsByType(sym.ID)
	assert.Empty(t, impls)
}

func TestDeleteResolutionDataForFiles(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")
	sym := insertTestSymbol(t, s, &f.ID, "Foo", "function")

	ref := &Reference{FileID: f.ID, Name: "Bar", StartLine: 9, EndLine: 9}
	s.InsertReference(ref)
	s.InsertResolvedReference(&ResolvedReference{ReferenceID: ref.ID, TargetSymbolID: sym.ID, Confidence: 1.0})
	s.InsertCallEdge(&CallEdge{CallerSymbolID: sym.ID, CalleeSymbolID: sym.ID, FileID: &f.ID, Line: 9})
	s.InsertReexport(&Reexport{FileID: f.ID, OriginalSymbolID: sym.ID, ExportedName: "Foo"})

	err := s.DeleteResolutionDataForFiles([]int64{f.ID})
	require.NoError(t, err)

	rr, _ := s.ResolvedReferencesByRef(ref.ID)
	assert.Empty(t, rr)

	callees, _ := s.CalleesByCaller(sym.ID)
	assert.Empty(t, callees)

	reexports, _ := s.ReexportsByFile(f.ID)
	assert.Empty(t, reexports)
}
