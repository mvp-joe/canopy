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

// rustTestEnv wraps the shared testEnv with Rust-specific helpers.
type rustTestEnv struct {
	*testEnv
}

func newRustTestEnv(t *testing.T) *rustTestEnv {
	return &rustTestEnv{testEnv: newTestEnv(t)}
}

// extractRustSource writes Rust source to a temp .rs file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *rustTestEnv) extractRustSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	rsFile := filepath.Join(dir, filename)
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

// resolveRust runs the Rust resolution script.
func (e *rustTestEnv) resolveRust() {
	e.t.Helper()
	extras := map[string]any{
		"files_to_resolve": runtime.MakeFilesToResolveFn(e.store, nil),
	}
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "rust.risor"), extras)
	require.NoError(e.t, err)
}

// --- Tests ---

func TestRustResolve_SameFileFunctionCall(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`fn helper() {}

fn main() {
    helper();
}
`, "main.rs")

	env.resolveRust()

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

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected helper call to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "helper", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestRustResolve_MethodCallOnSelf(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`struct Server {
    port: u16,
}

impl Server {
    fn new() -> Server {
        Server { port: 8080 }
    }

    fn start(&self) {}

    fn run(&self) {
        self.start();
    }
}
`, "server.rs")

	env.resolveRust()

	// Find the "start" call reference (from self.start())
	refs, err := env.store.ReferencesByName("start")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to start")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected start method call to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "start", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestRustResolve_UseStatementResolution(t *testing.T) {
	env := newRustTestEnv(t)

	// File 1: defines a struct in a "module"
	env.extractRustSource(`pub struct Config {
    pub host: String,
}

pub fn default_config() -> Config {
    Config { host: String::new() }
}
`, "config.rs")

	// File 2: uses Config via import
	env.extractRustSource(`use crate::config::Config;

fn main() {
    let c = Config { host: String::new() };
}
`, "main.rs")

	env.resolveRust()

	// Find type_annotation references to Config in main.rs
	refs, err := env.store.ReferencesByName("Config")
	require.NoError(t, err)

	// There should be a type_annotation reference in the second file
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
	require.NotEmpty(t, resolved, "expected Config type reference to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Config", targetSym.Name)
	assert.Equal(t, "struct", targetSym.Kind)
}

func TestRustResolve_TraitImplementation(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`trait Drawable {
    fn draw(&self);
}

struct Circle {
    radius: f64,
}

impl Circle {
    fn draw(&self) {}
}
`, "shapes.rs")

	env.resolveRust()

	traitSym := findSymbolByName(t, env.store, "Drawable", "trait")
	require.NotNil(t, traitSym)

	impls, err := env.store.ImplementationsByInterface(traitSym.ID)
	require.NoError(t, err)
	require.Len(t, impls, 1, "expected 1 implementation of Drawable")

	typeSym := findSymbolByID(t, env.store, impls[0].TypeSymbolID)
	assert.Equal(t, "Circle", typeSym.Name)
	assert.Equal(t, "explicit", impls[0].Kind)
}

func TestRustResolve_CallGraphEdge(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`fn helper() {}

fn main() {
    helper();
}
`, "main.rs")

	env.resolveRust()

	mainSym := findSymbolByName(t, env.store, "main", "function")
	helperSym := findSymbolByName(t, env.store, "helper", "function")
	require.NotNil(t, mainSym)
	require.NotNil(t, helperSym)

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

func TestRustResolve_ExtensionBindings(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`struct Server {}

impl Server {
    fn start(&self) {}
    fn stop(&self) {}
}
`, "server.rs")

	env.resolveRust()

	serverSym := findSymbolByName(t, env.store, "Server", "struct")
	require.NotNil(t, serverSym)

	bindings, err := env.store.ExtensionBindingsByType(serverSym.ID)
	require.NoError(t, err)
	require.Len(t, bindings, 2, "expected 2 extension bindings (start + stop)")

	names := map[string]bool{}
	for _, b := range bindings {
		sym := findSymbolByID(t, env.store, b.MemberSymbolID)
		names[sym.Name] = true
		assert.Equal(t, "method", b.Kind)
		assert.Equal(t, "Server", b.ExtendedTypeExpr)
	}
	assert.True(t, names["start"])
	assert.True(t, names["stop"])
}

func TestRustResolve_UnresolvedReference(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`fn main() {
    nonexistent();
}
`, "main.rs")

	env.resolveRust()

	refs, err := env.store.ReferencesByName("nonexistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonexistent should not be resolved")
	}
}

func TestRustResolve_CrossFileModuleResolution(t *testing.T) {
	env := newRustTestEnv(t)

	// File 1: defines a function
	env.extractRustSource(`pub fn greet(name: &str) -> String {
    format!("Hello, {}", name)
}
`, "util.rs")

	// File 2: imports and calls greet
	env.extractRustSource(`use crate::util::greet;

fn main() {
    greet("world");
}
`, "main.rs")

	env.resolveRust()

	refs, err := env.store.ReferencesByName("greet")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to greet")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected greet call to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "greet", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestRustResolve_MultipleTraitImplementation(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`trait Readable {
    fn read(&self) -> Vec<u8>;
}

trait Writable {
    fn write(&self, data: &[u8]);
}

struct FileHandle {}

impl FileHandle {
    fn read(&self) -> Vec<u8> { vec![] }
    fn write(&self, data: &[u8]) {}
}
`, "io.rs")

	env.resolveRust()

	readableSym := findSymbolByName(t, env.store, "Readable", "trait")
	writableSym := findSymbolByName(t, env.store, "Writable", "trait")
	require.NotNil(t, readableSym)
	require.NotNil(t, writableSym)

	readableImpls, err := env.store.ImplementationsByInterface(readableSym.ID)
	require.NoError(t, err)
	require.Len(t, readableImpls, 1)

	writableImpls, err := env.store.ImplementationsByInterface(writableSym.ID)
	require.NoError(t, err)
	require.Len(t, writableImpls, 1)

	// Both should resolve to FileHandle
	assert.Equal(t, readableImpls[0].TypeSymbolID, writableImpls[0].TypeSymbolID)
}

func TestRustResolve_TypeAnnotationResolution(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`struct Point {
    x: f64,
    y: f64,
}

fn origin() -> Point {
    Point { x: 0.0, y: 0.0 }
}
`, "point.rs")

	env.resolveRust()

	refs, err := env.store.ReferencesByName("Point")
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Point")

	resolved, err := env.store.ResolvedReferencesByRef(typeRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Point type annotation to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Point", targetSym.Name)
	assert.Equal(t, "struct", targetSym.Kind)
}

func TestRustResolve_EnumVariantMethod(t *testing.T) {
	env := newRustTestEnv(t)
	env.extractRustSource(`enum Shape {
    Circle,
    Square,
}

impl Shape {
    fn area(&self) -> f64 {
        0.0
    }
}

fn compute() {
    let s = Shape::Circle;
    s.area();
}
`, "shape.rs")

	env.resolveRust()

	// Verify extension binding on enum
	shapeSym := findSymbolByName(t, env.store, "Shape", "enum")
	require.NotNil(t, shapeSym)

	bindings, err := env.store.ExtensionBindingsByType(shapeSym.ID)
	require.NoError(t, err)
	require.Len(t, bindings, 1, "expected 1 extension binding (area)")

	sym := findSymbolByID(t, env.store, bindings[0].MemberSymbolID)
	assert.Equal(t, "area", sym.Name)
	assert.Equal(t, "Shape", bindings[0].ExtendedTypeExpr)

	// Verify method call resolves
	refs, err := env.store.ReferencesByName("area")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to area")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected area call to be resolved")
}
