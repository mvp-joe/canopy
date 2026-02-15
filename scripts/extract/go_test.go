package go_extract_test

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

// scriptDir returns the absolute path to the scripts/extract/ directory.
func scriptDir(t *testing.T) string {
	t.Helper()
	// We're running from scripts/extract/, so the script is in cwd.
	// But go test may run from the module root. Find it via the go.risor file.
	dir, err := filepath.Abs(".")
	require.NoError(t, err)
	// Check if go.risor is in current dir
	if _, err := os.Stat(filepath.Join(dir, "go.risor")); err == nil {
		return dir
	}
	// Try from module root
	modRoot := findModuleRoot(t)
	return filepath.Join(modRoot, "scripts", "extract")
}

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

// setup creates a temp database, Store, and Runtime.
// Returns the Store, a function to run the extraction script on source code,
// and a cleanup function.
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

	// Scripts directory is the repo root's scripts/ dir
	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")
	rt := runtime.NewRuntime(s, scriptsDir)

	t.Cleanup(func() { s.Close() })

	return &testEnv{store: s, rt: rt, t: t}
}

// extractGoSource writes Go source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractGoSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	goFile := filepath.Join(dir, "test.go")
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

// ---------- Tests ----------

func TestExtract_SimpleFunctionDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func Hello() {
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	// Should have package + function = 2 symbols
	var pkg, fn *store.Symbol
	for _, s := range syms {
		switch s.Kind {
		case "package":
			pkg = s
		case "function":
			fn = s
		}
	}

	require.NotNil(t, pkg, "expected package symbol")
	assert.Equal(t, "main", pkg.Name)

	require.NotNil(t, fn, "expected function symbol")
	assert.Equal(t, "Hello", fn.Name)
	assert.Equal(t, "function", fn.Kind)
	assert.Equal(t, "public", fn.Visibility)
	assert.Equal(t, 3, fn.StartLine) // line 3
}

func TestExtract_MultipleFunctions(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func Foo() {}
func bar() {}
func Baz() {}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var funcs []*store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			funcs = append(funcs, s)
		}
	}
	require.Len(t, funcs, 3)

	names := map[string]string{}
	for _, f := range funcs {
		names[f.Name] = f.Visibility
	}
	assert.Equal(t, "public", names["Foo"])
	assert.Equal(t, "private", names["bar"])
	assert.Equal(t, "public", names["Baz"])
}

func TestExtract_MethodWithReceiver(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type Server struct {
	Host string
}

func (s *Server) Start() {}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var structSym, methodSym *store.Symbol
	for _, s := range syms {
		switch {
		case s.Kind == "struct":
			structSym = s
		case s.Kind == "method":
			methodSym = s
		}
	}
	require.NotNil(t, structSym, "expected struct symbol")
	require.NotNil(t, methodSym, "expected method symbol")
	assert.Equal(t, "Start", methodSym.Name)
	assert.Equal(t, "public", methodSym.Visibility)

	// Method should be linked to the struct via parent_symbol_id
	require.NotNil(t, methodSym.ParentSymbolID, "method should have parent_symbol_id")
	assert.Equal(t, structSym.ID, *methodSym.ParentSymbolID)

	// Receiver should be a function parameter with is_receiver=true
	params, err := env.store.FunctionParams(methodSym.ID)
	require.NoError(t, err)

	var receiverParam *store.FunctionParam
	for _, p := range params {
		if p.IsReceiver {
			receiverParam = p
			break
		}
	}
	require.NotNil(t, receiverParam, "expected receiver parameter")
	assert.Equal(t, "s", receiverParam.Name)
	assert.Equal(t, "*Server", receiverParam.TypeExpr)
}

func TestExtract_StructWithFields(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type Config struct {
	Host     string
	Port     int
	Verbose  bool
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var structSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "struct" {
			structSym = s
			break
		}
	}
	require.NotNil(t, structSym)
	assert.Equal(t, "Config", structSym.Name)

	members, err := env.store.TypeMembers(structSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3)

	fieldNames := map[string]string{}
	for _, m := range members {
		assert.Equal(t, "field", m.Kind)
		fieldNames[m.Name] = m.TypeExpr
	}
	assert.Equal(t, "string", fieldNames["Host"])
	assert.Equal(t, "int", fieldNames["Port"])
	assert.Equal(t, "bool", fieldNames["Verbose"])
}

func TestExtract_StructWithEmbeddedType(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type Base struct{}

type Extended struct {
	Base
	Name string
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var extSym *store.Symbol
	for _, s := range syms {
		if s.Name == "Extended" {
			extSym = s
			break
		}
	}
	require.NotNil(t, extSym)

	members, err := env.store.TypeMembers(extSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	var embedded, field *store.TypeMember
	for _, m := range members {
		switch m.Kind {
		case "embedded":
			embedded = m
		case "field":
			field = m
		}
	}
	require.NotNil(t, embedded, "expected embedded member")
	assert.Equal(t, "Base", embedded.Name)
	require.NotNil(t, field, "expected field member")
	assert.Equal(t, "Name", field.Name)
}

func TestExtract_InterfaceWithMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type Reader interface {
	Read(p []byte) (n int, err error)
	Close() error
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var ifaceSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "interface" {
			ifaceSym = s
			break
		}
	}
	require.NotNil(t, ifaceSym)
	assert.Equal(t, "Reader", ifaceSym.Name)

	members, err := env.store.TypeMembers(ifaceSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	names := []string{}
	for _, m := range members {
		assert.Equal(t, "method", m.Kind)
		names = append(names, m.Name)
	}
	assert.Contains(t, names, "Read")
	assert.Contains(t, names, "Close")
}

func TestExtract_VariableDeclarations(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

var GlobalVar int
var privateVar string
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vars := map[string]*store.Symbol{}
	for _, s := range syms {
		if s.Kind == "variable" {
			vars[s.Name] = s
		}
	}
	require.Len(t, vars, 2)
	assert.Equal(t, "public", vars["GlobalVar"].Visibility)
	assert.Equal(t, "private", vars["privateVar"].Visibility)
}

func TestExtract_ConstantDeclarations(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

const MaxRetries = 3
const defaultTimeout = 30
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	consts := map[string]*store.Symbol{}
	for _, s := range syms {
		if s.Kind == "constant" {
			consts[s.Name] = s
		}
	}
	require.Len(t, consts, 2)
	assert.Equal(t, "public", consts["MaxRetries"].Visibility)
	assert.Equal(t, "private", consts["defaultTimeout"].Visibility)
}

func TestExtract_SingleImport(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 1)

	imp := imports[0]
	assert.Equal(t, "fmt", imp.Source)
	require.NotNil(t, imp.ImportedName)
	assert.Equal(t, "fmt", *imp.ImportedName)
	assert.Equal(t, "module", imp.Kind)
}

func TestExtract_GroupedImports(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

import (
	"fmt"
	"os"
	"strings"
)
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 3)

	sources := []string{}
	for _, imp := range imports {
		sources = append(sources, imp.Source)
	}
	assert.Contains(t, sources, "fmt")
	assert.Contains(t, sources, "os")
	assert.Contains(t, sources, "strings")
}

func TestExtract_AliasedImport(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

import (
	myio "io"
	_ "net/http/pprof"
	. "math"
)
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 3)

	impBySource := map[string]*store.Import{}
	for _, imp := range imports {
		impBySource[imp.Source] = imp
	}

	// Aliased import
	ioImp := impBySource["io"]
	require.NotNil(t, ioImp)
	require.NotNil(t, ioImp.LocalAlias)
	assert.Equal(t, "myio", *ioImp.LocalAlias)

	// Blank import
	pprofImp := impBySource["net/http/pprof"]
	require.NotNil(t, pprofImp)
	require.NotNil(t, pprofImp.LocalAlias)
	assert.Equal(t, "_", *pprofImp.LocalAlias)

	// Dot import
	mathImp := impBySource["math"]
	require.NotNil(t, mathImp)
	require.NotNil(t, mathImp.LocalAlias)
	assert.Equal(t, ".", *mathImp.LocalAlias)
}

func TestExtract_PackageDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package mypackage
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var pkgSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "package" {
			pkgSym = s
			break
		}
	}
	require.NotNil(t, pkgSym)
	assert.Equal(t, "mypackage", pkgSym.Name)
}

func TestExtract_ScopeTree(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func Process(x int) {
	if x > 0 {
		for i := 0; i < x; i++ {
		}
	}
}
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	// Should have: file scope, function scope, if scope, for scope
	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}
	assert.Equal(t, 1, kinds["file"], "expected 1 file scope")
	assert.Equal(t, 1, kinds["function"], "expected 1 function scope")
	assert.GreaterOrEqual(t, kinds["block"], 2, "expected at least 2 block scopes (if + for)")

	// Verify scope nesting: file -> function -> block(if) -> block(for)
	var fileScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "file" {
			fileScope = s
			break
		}
	}
	require.NotNil(t, fileScope)
	assert.Nil(t, fileScope.ParentScopeID, "file scope should have no parent")

	// Function scope should have file as parent
	var funcScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "function" {
			funcScope = s
			break
		}
	}
	require.NotNil(t, funcScope)
	require.NotNil(t, funcScope.ParentScopeID)
	assert.Equal(t, fileScope.ID, *funcScope.ParentScopeID)
}

func TestExtract_References_Call(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func helper() {}

func main() {
	helper()
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var callRef *store.Reference
	for _, r := range refs {
		if r.Name == "helper" && r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to helper")
	assert.Equal(t, "call", callRef.Context)
}

func TestExtract_References_TypeAnnotation(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type Foo struct{}

func create() Foo {
	return Foo{}
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Name == "Foo" && r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Foo")
}

func TestExtract_References_FieldAccess(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Name == "Println" && r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to Println")
}

func TestExtract_FunctionParams(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func Add(a int, b int) int {
	return a + b
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fnSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fnSym = s
			break
		}
	}
	require.NotNil(t, fnSym)

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)

	// Should have: a, b (regular params) + 1 return type
	var regularParams, returnParams []*store.FunctionParam
	for _, p := range params {
		if p.IsReturn {
			returnParams = append(returnParams, p)
		} else {
			regularParams = append(regularParams, p)
		}
	}

	require.Len(t, regularParams, 2)
	assert.Equal(t, "a", regularParams[0].Name)
	assert.Equal(t, "int", regularParams[0].TypeExpr)
	assert.Equal(t, 0, regularParams[0].Ordinal)
	assert.Equal(t, "b", regularParams[1].Name)
	assert.Equal(t, "int", regularParams[1].TypeExpr)
	assert.Equal(t, 1, regularParams[1].Ordinal)

	require.Len(t, returnParams, 1)
	assert.Equal(t, "int", returnParams[0].TypeExpr)
	assert.True(t, returnParams[0].IsReturn)
}

func TestExtract_ReturnTypes(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func Divide(a, b int) (int, error) {
	return a / b, nil
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fnSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fnSym = s
			break
		}
	}
	require.NotNil(t, fnSym)

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)

	var returnParams []*store.FunctionParam
	for _, p := range params {
		if p.IsReturn {
			returnParams = append(returnParams, p)
		}
	}

	require.Len(t, returnParams, 2)
	types := []string{returnParams[0].TypeExpr, returnParams[1].TypeExpr}
	assert.Contains(t, types, "int")
	assert.Contains(t, types, "error")
}

func TestExtract_Generics(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type Set[T comparable] struct {
	items map[T]bool
}

func Map[T any, U any](s []T, f func(T) U) []U {
	return nil
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	// Check struct type params
	var setSym, mapFnSym *store.Symbol
	for _, s := range syms {
		if s.Name == "Set" {
			setSym = s
		}
		if s.Name == "Map" && s.Kind == "function" {
			mapFnSym = s
		}
	}

	require.NotNil(t, setSym)
	setTPs, err := env.store.TypeParams(setSym.ID)
	require.NoError(t, err)
	require.Len(t, setTPs, 1)
	assert.Equal(t, "T", setTPs[0].Name)
	assert.Equal(t, "comparable", setTPs[0].Constraints)
	assert.Equal(t, 0, setTPs[0].Ordinal)

	require.NotNil(t, mapFnSym)
	fnTPs, err := env.store.TypeParams(mapFnSym.ID)
	require.NoError(t, err)
	require.Len(t, fnTPs, 2)
	assert.Equal(t, "T", fnTPs[0].Name)
	assert.Equal(t, "any", fnTPs[0].Constraints)
	assert.Equal(t, "U", fnTPs[1].Name)
	assert.Equal(t, "any", fnTPs[1].Constraints)
}

func TestExtract_ExportedVsUnexported(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func PublicFunc() {}
func privateFunc() {}
type PublicStruct struct{}
type privateStruct struct{}
var PublicVar int
var privateVar int
const PublicConst = 1
const privateConst = 2
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "package" {
			continue
		}
		vis[s.Name] = s.Visibility
	}

	assert.Equal(t, "public", vis["PublicFunc"])
	assert.Equal(t, "private", vis["privateFunc"])
	assert.Equal(t, "public", vis["PublicStruct"])
	assert.Equal(t, "private", vis["privateStruct"])
	assert.Equal(t, "public", vis["PublicVar"])
	assert.Equal(t, "private", vis["privateVar"])
	assert.Equal(t, "public", vis["PublicConst"])
	assert.Equal(t, "private", vis["privateConst"])
}

func TestExtract_TypeAlias(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type StringSlice []string
type Handler func(int) error
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var aliases []*store.Symbol
	for _, s := range syms {
		if s.Kind == "type_alias" {
			aliases = append(aliases, s)
		}
	}
	require.Len(t, aliases, 2)

	names := map[string]bool{}
	for _, a := range aliases {
		names[a.Name] = true
	}
	assert.True(t, names["StringSlice"])
	assert.True(t, names["Handler"])
}

func TestExtract_InterfaceEmbedded(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

type Reader interface {
	Read(p []byte) (int, error)
}

type ReadCloser interface {
	Reader
	Close() error
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var rcSym *store.Symbol
	for _, s := range syms {
		if s.Name == "ReadCloser" {
			rcSym = s
			break
		}
	}
	require.NotNil(t, rcSym)

	members, err := env.store.TypeMembers(rcSym.ID)
	require.NoError(t, err)

	var embedded, method *store.TypeMember
	for _, m := range members {
		switch m.Kind {
		case "embedded":
			embedded = m
		case "method":
			method = m
		}
	}
	require.NotNil(t, embedded, "expected embedded Reader")
	assert.Equal(t, "Reader", embedded.Name)
	require.NotNil(t, method, "expected method Close")
	assert.Equal(t, "Close", method.Name)
}

func TestExtract_ComprehensiveFile(t *testing.T) {
	// A more complex file exercising multiple features together.
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

import (
	"fmt"
	"io"
)

const Version = "1.0.0"

var Debug bool

type Config struct {
	Host    string
	Port    int
	io.Reader
}

type Handler interface {
	Handle(req string) (string, error)
}

func NewConfig(host string, port int) *Config {
	return &Config{Host: host, Port: port}
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["package"], "main")
	assert.Contains(t, kinds["constant"], "Version")
	assert.Contains(t, kinds["variable"], "Debug")
	assert.Contains(t, kinds["struct"], "Config")
	assert.Contains(t, kinds["interface"], "Handler")
	assert.Contains(t, kinds["function"], "NewConfig")
	assert.Contains(t, kinds["method"], "Addr")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	// Verify Config has embedded io.Reader
	var configSym *store.Symbol
	for _, s := range syms {
		if s.Name == "Config" {
			configSym = s
			break
		}
	}
	require.NotNil(t, configSym)
	members, err := env.store.TypeMembers(configSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3) // Host, Port, Reader embedded

	// Verify Addr method is linked to Config
	var addrSym *store.Symbol
	for _, s := range syms {
		if s.Name == "Addr" {
			addrSym = s
			break
		}
	}
	require.NotNil(t, addrSym)
	require.NotNil(t, addrSym.ParentSymbolID)
	assert.Equal(t, configSym.ID, *addrSym.ParentSymbolID)

	// Verify scope tree exists
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 3) // file + 2 functions/methods

	// Verify references exist
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)
}

func TestExtract_EmptyFile(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	// Only package symbol
	require.Len(t, syms, 1)
	assert.Equal(t, "package", syms[0].Kind)
	assert.Equal(t, "main", syms[0].Name)
}

func TestExtract_SwitchScope(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func classify(x int) string {
	switch {
	case x > 0:
		return "positive"
	case x < 0:
		return "negative"
	default:
		return "zero"
	}
}
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}
	assert.Equal(t, 1, kinds["file"])
	assert.Equal(t, 1, kinds["function"])
	assert.GreaterOrEqual(t, kinds["block"], 1, "expected at least 1 block scope for switch")
}

func TestExtract_NamedReturnValues(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractGoSource(`package main

func Divide(a, b float64) (result float64, err error) {
	return a / b, nil
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fnSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fnSym = s
			break
		}
	}
	require.NotNil(t, fnSym)

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)

	var returnParams []*store.FunctionParam
	for _, p := range params {
		if p.IsReturn {
			returnParams = append(returnParams, p)
		}
	}
	require.Len(t, returnParams, 2)

	// Named return values should have names
	returnNames := map[string]string{}
	for _, p := range returnParams {
		returnNames[p.Name] = p.TypeExpr
	}
	assert.Equal(t, "float64", returnNames["result"])
	assert.Equal(t, "error", returnNames["err"])
}

// TestExtract_EndToEnd_ViaEngine tests the full pipeline via the Engine,
// the way extraction actually runs in production.
func TestExtract_EndToEnd_ViaEngine(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(goFile, []byte(`package main

func Hello() string {
	return "hello"
}
`), 0644))

	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())
	defer s.Close()

	rt := runtime.NewRuntime(s, scriptsDir)

	// Simulate what the Engine does
	fileID, err := s.InsertFile(&store.File{
		Path:     goFile,
		Language: "go",
	})
	require.NoError(t, err)

	extras := map[string]any{
		"file_path": goFile,
		"file_id":   fileID,
	}
	err = rt.RunScript(context.Background(), filepath.Join("extract", "go.risor"), extras)
	require.NoError(t, err)

	// Verify extraction worked
	syms, err := s.SymbolsByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(syms), 2) // package + function
}
