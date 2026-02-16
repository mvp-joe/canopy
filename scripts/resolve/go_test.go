package go_resolve_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jward/canopy/internal/runtime"
	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root")
		}
		dir = parent
	}
}

type testEnv struct {
	store *store.Store
	rt    *runtime.Runtime
	t     *testing.T
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())

	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")
	rt := runtime.NewRuntime(s, scriptsDir)

	t.Cleanup(func() { s.Close() })

	return &testEnv{store: s, rt: rt, t: t}
}

// extractGoSource writes Go source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractGoSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	goFile := filepath.Join(dir, filename)
	require.NoError(e.t, os.WriteFile(goFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     goFile,
		Language: "go",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": goFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "go.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// resolve runs the Go resolution script.
func (e *testEnv) resolve() {
	e.t.Helper()
	extras := map[string]any{
		"files_to_resolve": runtime.MakeFilesToResolveFn(e.store, nil),
	}
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "go.risor"), extras)
	require.NoError(e.t, err)
}

// --- Tests ---

func TestResolve_SameFileFunctionCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

func helper() {}

func main() {
	helper()
}
`, "main.go")

	env.resolve()

	// Find the "helper" call reference
	refs, err := env.store.ReferencesByName("helper")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to helper")

	// Verify it resolved
	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected helper call to be resolved")

	// Verify the target is the helper function
	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "helper", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestResolve_CrossFileSamePackage(t *testing.T) {
	env := newTestEnv(t)

	// File 1: util.go with Greet function
	env.extractGoSource(`package main

func Greet(name string) string {
	return "Hello, " + name
}
`, "util.go")

	// File 2: main.go calls Greet
	env.extractGoSource(`package main

func main() {
	Greet("world")
}
`, "main.go")

	env.resolve()

	// Find "Greet" call reference in main.go
	refs, err := env.store.ReferencesByName("Greet")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to Greet")

	// Verify resolution
	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Greet call to be resolved cross-file")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Greet", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestResolve_ImportResolution_QualifiedCall(t *testing.T) {
	env := newTestEnv(t)

	// Create a "util" package with an exported function
	env.extractGoSource(`package util

func DoWork() string {
	return "done"
}
`, "util.go")

	// Create main.go that imports and calls util.DoWork
	env.extractGoSource(`package main

import "util"

func main() {
	util.DoWork()
}
`, "main.go")

	env.resolve()

	// Find "DoWork" field_access reference
	refs, err := env.store.ReferencesByName("DoWork")
	require.NoError(t, err)
	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to DoWork")

	// Verify it resolved to the util.DoWork function
	resolved, err := env.store.ResolvedReferencesByRef(fieldRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected DoWork to be resolved via import")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "DoWork", targetSym.Name)
	assert.Equal(t, "import", resolved[0].ResolutionKind)
}

func TestResolve_MethodResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

type Server struct{}

func (s *Server) Start() {}

func main() {
	s := Server{}
	s.Start()
}
`, "main.go")

	env.resolve()

	// Find "Start" field_access reference
	refs, err := env.store.ReferencesByName("Start")
	require.NoError(t, err)
	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to Start")

	// Verify it resolved
	resolved, err := env.store.ResolvedReferencesByRef(fieldRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Start method to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Start", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestResolve_InterfaceImplementation(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

type Reader interface {
	Read() string
}

type MyReader struct{}

func (r *MyReader) Read() string {
	return "data"
}
`, "main.go")

	env.resolve()

	// Find the implementation record
	ifaceSym := findSymbolByName(t, env.store, "Reader", "interface")
	require.NotNil(t, ifaceSym)

	impls, err := env.store.ImplementationsByInterface(ifaceSym.ID)
	require.NoError(t, err)
	require.Len(t, impls, 1, "expected 1 implementation of Reader")

	typeSym := findSymbolByID(t, env.store, impls[0].TypeSymbolID)
	assert.Equal(t, "MyReader", typeSym.Name)
	assert.Equal(t, "implicit", impls[0].Kind)
}

func TestResolve_CallGraphEdge(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

func helper() {}

func main() {
	helper()
}
`, "main.go")

	env.resolve()

	// Find caller/callee symbols
	mainSym := findSymbolByName(t, env.store, "main", "function")
	helperSym := findSymbolByName(t, env.store, "helper", "function")
	require.NotNil(t, mainSym)
	require.NotNil(t, helperSym)

	// Check call graph
	edges, err := env.store.CalleesByCaller(mainSym.ID)
	require.NoError(t, err)

	found := false
	for _, e := range edges {
		if e.CalleeSymbolID == helperSym.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "expected call edge from main to helper")
}

func TestResolve_ExtensionBinding(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

type Server struct{}

func (s *Server) Start() {}
func (s *Server) Stop() {}
`, "main.go")

	env.resolve()

	serverSym := findSymbolByName(t, env.store, "Server", "struct")
	require.NotNil(t, serverSym)

	bindings, err := env.store.ExtensionBindingsByType(serverSym.ID)
	require.NoError(t, err)
	require.Len(t, bindings, 2, "expected 2 extension bindings (Start + Stop)")

	names := map[string]bool{}
	for _, b := range bindings {
		sym := findSymbolByID(t, env.store, b.MemberSymbolID)
		names[sym.Name] = true
		assert.Equal(t, "method", b.Kind)
		assert.Equal(t, "Server", b.ExtendedTypeExpr)
	}
	assert.True(t, names["Start"])
	assert.True(t, names["Stop"])
}

func TestResolve_ScopeShadowing(t *testing.T) {
	// Test scope-based resolution: an inner function declaration shadows an outer one.
	// The call inside the inner function should resolve to the inner function's local helper.
	env := newTestEnv(t)
	env.extractGoSource(`package main

func outer() {}

func inner() {
	outer()
}
`, "main.go")

	env.resolve()

	// The call to "outer" inside "inner" should resolve to the file-level "outer" function.
	refs, err := env.store.ReferencesByName("outer")
	require.NoError(t, err)

	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to outer")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected outer call to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "outer", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestResolve_UnresolvedReference(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

func main() {
	nonExistent()
}
`, "main.go")

	env.resolve()

	refs, err := env.store.ReferencesByName("nonExistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	// nonExistent should NOT be resolved
	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonExistent should not be resolved")
	}
}

func TestResolve_MultipleInterfaceImplementation(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

type Writer interface {
	Write() error
}

type Closer interface {
	Close() error
}

type MyFile struct{}

func (f *MyFile) Write() error { return nil }
func (f *MyFile) Close() error { return nil }
`, "main.go")

	env.resolve()

	writerSym := findSymbolByName(t, env.store, "Writer", "interface")
	closerSym := findSymbolByName(t, env.store, "Closer", "interface")
	require.NotNil(t, writerSym)
	require.NotNil(t, closerSym)

	writerImpls, err := env.store.ImplementationsByInterface(writerSym.ID)
	require.NoError(t, err)
	require.Len(t, writerImpls, 1)

	closerImpls, err := env.store.ImplementationsByInterface(closerSym.ID)
	require.NoError(t, err)
	require.Len(t, closerImpls, 1)

	// Both should resolve to MyFile
	assert.Equal(t, writerImpls[0].TypeSymbolID, closerImpls[0].TypeSymbolID)
}

func TestResolve_TypeAnnotationResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractGoSource(`package main

type Config struct {
	Host string
}

func NewConfig() Config {
	return Config{}
}
`, "main.go")

	env.resolve()

	// Find type_annotation references to Config
	refs, err := env.store.ReferencesByName("Config")
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Config")

	resolved, err := env.store.ResolvedReferencesByRef(typeRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Config type annotation to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Config", targetSym.Name)
	assert.Equal(t, "struct", targetSym.Kind)
}

// --- Helper functions ---

func findSymbolByID(t *testing.T, s *store.Store, id int64) *store.Symbol {
	t.Helper()
	var sym store.Symbol
	var mods string
	err := s.DB().QueryRow(
		`SELECT id, file_id, name, kind, visibility, modifiers, signature_hash,
		 start_line, start_col, end_line, end_col, parent_symbol_id
		 FROM symbols WHERE id = ?`, id,
	).Scan(&sym.ID, &sym.FileID, &sym.Name, &sym.Kind, &sym.Visibility, &mods,
		&sym.SignatureHash, &sym.StartLine, &sym.StartCol, &sym.EndLine, &sym.EndCol,
		&sym.ParentSymbolID)
	require.NoError(t, err)
	return &sym
}

func findSymbolByName(t *testing.T, s *store.Store, name, kind string) *store.Symbol {
	t.Helper()
	syms, err := s.SymbolsByName(name)
	require.NoError(t, err)
	for _, sym := range syms {
		if sym.Kind == kind {
			return sym
		}
	}
	return nil
}
