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

// rustTestEnv wraps the test environment for Rust extraction tests.
type rustTestEnv struct {
	store *store.Store
	rt    *runtime.Runtime
	t     *testing.T
}

func newRustTestEnv(t *testing.T) *rustTestEnv {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())

	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")
	rt := runtime.NewRuntime(s, scriptsDir)

	t.Cleanup(func() { s.Close() })

	return &rustTestEnv{store: s, rt: rt, t: t}
}

// extractRustSource writes Rust source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *rustTestEnv) extractRustSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	rsFile := filepath.Join(dir, "test.rs")
	require.NoError(e.t, os.WriteFile(rsFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     rsFile,
		Language: "rust",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": rsFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "rust.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// ---------- Tests ----------

func TestRustExtract_SimpleFunctionDeclaration(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`fn foo() {}`)

	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fn *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fn = s
		}
	}

	require.NotNil(t, fn, "expected function symbol")
	assert.Equal(t, "foo", fn.Name)
	assert.Equal(t, "function", fn.Kind)
	assert.Equal(t, "private", fn.Visibility)
}

func TestRustExtract_PublicFunction(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`pub fn hello() {}`)

	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fn *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fn = s
		}
	}

	require.NotNil(t, fn, "expected function symbol")
	assert.Equal(t, "hello", fn.Name)
	assert.Equal(t, "public", fn.Visibility)
}

func TestRustExtract_StructWithFields(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Point {
    x: f64,
    y: f64,
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var structSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "struct" {
			structSym = s
		}
	}
	require.NotNil(t, structSym, "expected struct symbol")
	assert.Equal(t, "Point", structSym.Name)
	assert.Equal(t, "private", structSym.Visibility)

	members, err := env.store.TypeMembers(structSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	fieldMap := map[string]string{}
	for _, m := range members {
		assert.Equal(t, "field", m.Kind)
		fieldMap[m.Name] = m.TypeExpr
	}
	assert.Equal(t, "f64", fieldMap["x"])
	assert.Equal(t, "f64", fieldMap["y"])
}

func TestRustExtract_EnumWithVariants(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
enum Direction {
    Up,
    Down,
    Left,
    Right,
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var enumSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "enum" {
			enumSym = s
		}
	}
	require.NotNil(t, enumSym, "expected enum symbol")
	assert.Equal(t, "Direction", enumSym.Name)

	members, err := env.store.TypeMembers(enumSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 4)

	names := []string{}
	for _, m := range members {
		assert.Equal(t, "variant", m.Kind)
		names = append(names, m.Name)
	}
	assert.Contains(t, names, "Up")
	assert.Contains(t, names, "Down")
	assert.Contains(t, names, "Left")
	assert.Contains(t, names, "Right")
}

func TestRustExtract_TraitWithMethods(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
trait Shape {
    fn area(&self) -> f64;
    fn name(&self) -> &str;
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var traitSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "trait" {
			traitSym = s
		}
	}
	require.NotNil(t, traitSym, "expected trait symbol")
	assert.Equal(t, "Shape", traitSym.Name)

	members, err := env.store.TypeMembers(traitSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	names := []string{}
	for _, m := range members {
		assert.Equal(t, "method", m.Kind)
		names = append(names, m.Name)
	}
	assert.Contains(t, names, "area")
	assert.Contains(t, names, "name")
}

func TestRustExtract_ImplBlockWithMethods(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Point {
    x: f64,
    y: f64,
}

impl Point {
    pub fn new(x: f64, y: f64) -> Self {
        Point { x, y }
    }

    pub fn distance(&self) -> f64 {
        (self.x * self.x + self.y * self.y).sqrt()
    }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var structSym *store.Symbol
	var newFn, distFn *store.Symbol
	for _, s := range syms {
		switch {
		case s.Kind == "struct":
			structSym = s
		case s.Name == "new":
			newFn = s
		case s.Name == "distance":
			distFn = s
		}
	}

	require.NotNil(t, structSym, "expected struct symbol")
	require.NotNil(t, newFn, "expected 'new' function")
	require.NotNil(t, distFn, "expected 'distance' method")

	// 'new' is a function (no self param), linked to Point
	assert.Equal(t, "function", newFn.Kind)
	assert.Equal(t, "public", newFn.Visibility)
	require.NotNil(t, newFn.ParentSymbolID, "new should have parent_symbol_id")
	assert.Equal(t, structSym.ID, *newFn.ParentSymbolID)

	// 'distance' is a method (has &self), linked to Point
	assert.Equal(t, "method", distFn.Kind)
	assert.Equal(t, "public", distFn.Visibility)
	require.NotNil(t, distFn.ParentSymbolID, "distance should have parent_symbol_id")
	assert.Equal(t, structSym.ID, *distFn.ParentSymbolID)

	// Verify self parameter
	params, err := env.store.FunctionParams(distFn.ID)
	require.NoError(t, err)

	var selfParam *store.FunctionParam
	for _, p := range params {
		if p.IsReceiver {
			selfParam = p
		}
	}
	require.NotNil(t, selfParam, "expected self parameter")
	assert.Equal(t, "self", selfParam.Name)
	assert.Equal(t, "&self", selfParam.TypeExpr)
}

func TestRustExtract_UseStatements(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
use std::collections::HashMap;
use std::io::{self, Read};
use std::fmt;
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)

	impBySource := map[string]*store.Import{}
	for _, imp := range imports {
		impBySource[imp.Source] = imp
	}

	// use std::collections::HashMap
	hashMapImp := impBySource["std::collections::HashMap"]
	require.NotNil(t, hashMapImp, "expected HashMap import")
	require.NotNil(t, hashMapImp.ImportedName)
	assert.Equal(t, "HashMap", *hashMapImp.ImportedName)

	// use std::io::{self} -> imports std::io
	ioImp := impBySource["std::io"]
	require.NotNil(t, ioImp, "expected std::io import")
	require.NotNil(t, ioImp.ImportedName)
	assert.Equal(t, "io", *ioImp.ImportedName)

	// use std::io::{Read}
	readImp := impBySource["std::io::Read"]
	require.NotNil(t, readImp, "expected std::io::Read import")
	require.NotNil(t, readImp.ImportedName)
	assert.Equal(t, "Read", *readImp.ImportedName)

	// use std::fmt
	fmtImp := impBySource["std::fmt"]
	require.NotNil(t, fmtImp, "expected std::fmt import")
	require.NotNil(t, fmtImp.ImportedName)
	assert.Equal(t, "fmt", *fmtImp.ImportedName)
}

func TestRustExtract_ModuleDeclaration(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
mod utils {
    pub fn helper() {}
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var modSym, fnSym *store.Symbol
	for _, s := range syms {
		switch s.Kind {
		case "module":
			modSym = s
		case "function":
			fnSym = s
		}
	}

	require.NotNil(t, modSym, "expected module symbol")
	assert.Equal(t, "utils", modSym.Name)
	assert.Equal(t, "private", modSym.Visibility)

	require.NotNil(t, fnSym, "expected function symbol inside module")
	assert.Equal(t, "helper", fnSym.Name)
	assert.Equal(t, "public", fnSym.Visibility)
}

func TestRustExtract_ScopeTree(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
fn process(x: i32) {
    if x > 0 {
        for i in 0..x {
        }
    }
}
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}
	assert.Equal(t, 1, kinds["file"], "expected 1 file scope")
	assert.Equal(t, 1, kinds["function"], "expected 1 function scope")
	assert.GreaterOrEqual(t, kinds["block"], 2, "expected at least 2 block scopes (if + for)")

	// Verify nesting: file -> function -> block
	var fileScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "file" {
			fileScope = s
			break
		}
	}
	require.NotNil(t, fileScope)
	assert.Nil(t, fileScope.ParentScopeID, "file scope should have no parent")

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

func TestRustExtract_References_Call(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
fn helper() {}

fn main() {
    helper();
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
}

func TestRustExtract_References_TypeAnnotation(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Foo {}

fn create() -> Foo {
    Foo {}
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

func TestRustExtract_References_FieldAccess(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Point { x: f64, y: f64 }

fn get_x(p: Point) -> f64 {
    p.x
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Name == "x" && r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to x")
}

func TestRustExtract_FunctionParamsWithTypes(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
fn add(a: i32, b: i32) -> i32 {
    a + b
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
	assert.Equal(t, "i32", regularParams[0].TypeExpr)
	assert.Equal(t, 0, regularParams[0].Ordinal)
	assert.Equal(t, "b", regularParams[1].Name)
	assert.Equal(t, "i32", regularParams[1].TypeExpr)
	assert.Equal(t, 1, regularParams[1].Ordinal)

	require.Len(t, returnParams, 1)
	assert.Equal(t, "i32", returnParams[0].TypeExpr)
	assert.True(t, returnParams[0].IsReturn)
}

func TestRustExtract_SelfParameter(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Foo {}

impl Foo {
    fn method(&self, x: i32) {}
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var methodSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methodSym = s
			break
		}
	}
	require.NotNil(t, methodSym, "expected method symbol")

	params, err := env.store.FunctionParams(methodSym.ID)
	require.NoError(t, err)

	var selfParam *store.FunctionParam
	var regularParams []*store.FunctionParam
	for _, p := range params {
		if p.IsReceiver {
			selfParam = p
		} else if !p.IsReturn {
			regularParams = append(regularParams, p)
		}
	}

	require.NotNil(t, selfParam, "expected self parameter")
	assert.Equal(t, "self", selfParam.Name)
	assert.Equal(t, "&self", selfParam.TypeExpr)
	assert.True(t, selfParam.IsReceiver)
	assert.Equal(t, 0, selfParam.Ordinal)

	require.Len(t, regularParams, 1)
	assert.Equal(t, "x", regularParams[0].Name)
	assert.Equal(t, "i32", regularParams[0].TypeExpr)
	assert.Equal(t, 1, regularParams[0].Ordinal)
}

func TestRustExtract_GenericsWithTraitBounds(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
fn process<T: Clone + Send>(val: T) -> T {
    val
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

	tps, err := env.store.TypeParams(fnSym.ID)
	require.NoError(t, err)
	require.Len(t, tps, 1)

	assert.Equal(t, "T", tps[0].Name)
	assert.Equal(t, "type", tps[0].ParamKind)
	assert.Equal(t, 0, tps[0].Ordinal)
	assert.Equal(t, "Clone + Send", tps[0].Constraints)
}

func TestRustExtract_TypeAlias(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
type Pair = (i32, i32);
type Result<T> = std::result::Result<T, Error>;
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
	assert.True(t, names["Pair"])
	assert.True(t, names["Result"])
}

func TestRustExtract_ConstantsAndStatics(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
const MAX_SIZE: usize = 100;
static COUNTER: i32 = 0;
pub const VERSION: &str = "1.0";
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	symsByName := map[string]*store.Symbol{}
	for _, s := range syms {
		symsByName[s.Name] = s
	}

	maxSym := symsByName["MAX_SIZE"]
	require.NotNil(t, maxSym, "expected MAX_SIZE constant")
	assert.Equal(t, "constant", maxSym.Kind)
	assert.Equal(t, "private", maxSym.Visibility)

	counterSym := symsByName["COUNTER"]
	require.NotNil(t, counterSym, "expected COUNTER static")
	assert.Equal(t, "variable", counterSym.Kind)
	assert.Equal(t, "private", counterSym.Visibility)

	versionSym := symsByName["VERSION"]
	require.NotNil(t, versionSym, "expected VERSION constant")
	assert.Equal(t, "constant", versionSym.Kind)
	assert.Equal(t, "public", versionSym.Visibility)
}

func TestRustExtract_Visibility(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
fn private_fn() {}
pub fn public_fn() {}
pub(crate) fn crate_fn() {}
pub(super) fn super_fn() {}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vis := map[string]string{}
	for _, s := range syms {
		vis[s.Name] = s.Visibility
	}

	assert.Equal(t, "private", vis["private_fn"])
	assert.Equal(t, "public", vis["public_fn"])
	assert.Equal(t, "pub(crate)", vis["crate_fn"])
	assert.Equal(t, "pub(super)", vis["super_fn"])
}

func TestRustExtract_AsyncFunction(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
pub async fn fetch_data() -> String {
    String::new()
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
	require.NotNil(t, fnSym, "expected function symbol")
	assert.Equal(t, "fetch_data", fnSym.Name)
	assert.Equal(t, "public", fnSym.Visibility)

	// Verify return type
	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)

	var retParam *store.FunctionParam
	for _, p := range params {
		if p.IsReturn {
			retParam = p
		}
	}
	require.NotNil(t, retParam, "expected return type")
	assert.Equal(t, "String", retParam.TypeExpr)
}

func TestRustExtract_ImplScope(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Foo {}

impl Foo {
    fn bar(&self) {}
}
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}

	assert.Equal(t, 1, kinds["file"], "expected 1 file scope")
	assert.Equal(t, 1, kinds["impl"], "expected 1 impl scope")
	assert.Equal(t, 1, kinds["function"], "expected 1 function scope")

	// impl scope should be child of file scope
	var fileScope, implScope, funcScope *store.Scope
	for _, s := range scopes {
		switch s.Kind {
		case "file":
			fileScope = s
		case "impl":
			implScope = s
		case "function":
			funcScope = s
		}
	}

	require.NotNil(t, fileScope)
	require.NotNil(t, implScope)
	require.NotNil(t, funcScope)

	require.NotNil(t, implScope.ParentScopeID)
	assert.Equal(t, fileScope.ID, *implScope.ParentScopeID)

	require.NotNil(t, funcScope.ParentScopeID)
	assert.Equal(t, implScope.ID, *funcScope.ParentScopeID)
}

func TestRustExtract_MacroDefinition(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
macro_rules! my_macro {
    ($x:expr) => { $x + 1 };
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var macroSym *store.Symbol
	for _, s := range syms {
		if s.Name == "my_macro" {
			macroSym = s
			break
		}
	}

	require.NotNil(t, macroSym, "expected macro symbol")
	assert.Equal(t, "function", macroSym.Kind)
	assert.Equal(t, "public", macroSym.Visibility)
}

func TestRustExtract_TraitImplMethods(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Circle {
    radius: f64,
}

trait Shape {
    fn area(&self) -> f64;
}

impl Shape for Circle {
    fn area(&self) -> f64 {
        3.14 * self.radius * self.radius
    }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var circleSym, areaSym *store.Symbol
	for _, s := range syms {
		switch {
		case s.Kind == "struct":
			circleSym = s
		case s.Name == "area" && s.Kind == "method":
			areaSym = s
		}
	}

	require.NotNil(t, circleSym, "expected Circle struct")
	require.NotNil(t, areaSym, "expected area method")

	// area method should be linked to Circle (the impl type)
	require.NotNil(t, areaSym.ParentSymbolID, "area should have parent_symbol_id")
	assert.Equal(t, circleSym.ID, *areaSym.ParentSymbolID)
}

func TestRustExtract_ScopedCallReference(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
fn main() {
    let s = String::from("hello");
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var callRef *store.Reference
	for _, r := range refs {
		if r.Name == "from" && r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to String::from")
}

func TestRustExtract_ComprehensiveFile(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
use std::collections::HashMap;
use std::fmt;

const VERSION: &str = "1.0.0";

static DEBUG: bool = false;

struct Config {
    host: String,
    port: u16,
}

enum Status {
    Active,
    Inactive,
}

trait Service {
    fn start(&self);
    fn stop(&self);
}

impl Config {
    pub fn new(host: String, port: u16) -> Self {
        Config { host, port }
    }

    pub fn addr(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }
}

type Callback = fn(i32) -> bool;

fn main() {
    let cfg = Config::new(String::from("localhost"), 8080);
    cfg.addr();
}
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["constant"], "VERSION")
	assert.Contains(t, kinds["variable"], "DEBUG")
	assert.Contains(t, kinds["struct"], "Config")
	assert.Contains(t, kinds["enum"], "Status")
	assert.Contains(t, kinds["trait"], "Service")
	assert.Contains(t, kinds["type_alias"], "Callback")
	assert.Contains(t, kinds["function"], "main")
	assert.Contains(t, kinds["function"], "new")
	assert.Contains(t, kinds["method"], "addr")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	// Verify Config has 2 fields
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
	require.Len(t, members, 2)

	// Verify addr method is linked to Config
	var addrSym *store.Symbol
	for _, s := range syms {
		if s.Name == "addr" {
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
	assert.GreaterOrEqual(t, len(scopes), 3) // file + impl + functions

	// Verify references exist
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)
}

func TestRustExtract_ModScope(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
mod inner {
    pub fn do_thing() {}
}
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}
	assert.Equal(t, 1, kinds["file"])
	assert.Equal(t, 1, kinds["module"], "expected 1 module scope")
}

func TestRustExtract_UseWildcard(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
use std::io::*;
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)

	// The scoped_use_list won't match here; need to handle use_wildcard differently
	// Let's check what we get
	require.GreaterOrEqual(t, len(imports), 1, "expected at least 1 import from wildcard use")
}

func TestRustExtract_StructGenericTypeParams(t *testing.T) {
	env := newRustTestEnv(t)
	fileID := env.extractRustSource(`
struct Container<T: Clone> {
    items: Vec<T>,
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var structSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "struct" {
			structSym = s
		}
	}
	require.NotNil(t, structSym)

	tps, err := env.store.TypeParams(structSym.ID)
	require.NoError(t, err)
	require.Len(t, tps, 1)
	assert.Equal(t, "T", tps[0].Name)
	assert.Equal(t, "Clone", tps[0].Constraints)
}
