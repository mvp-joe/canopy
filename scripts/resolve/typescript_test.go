package go_resolve_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractTSSource writes TypeScript source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractTSSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	tsFile := filepath.Join(dir, filename)
	require.NoError(e.t, os.WriteFile(tsFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     tsFile,
		Language: "typescript",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": tsFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "typescript.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// resolveTS runs the TypeScript resolution script.
func (e *testEnv) resolveTS() {
	e.t.Helper()
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "typescript.risor"), nil)
	require.NoError(e.t, err)
}

// --- TypeScript Resolution Tests ---

func TestTSResolve_SameFileFunctionCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
function helper(): void {}

function main(): void {
  helper()
}
`, "main.ts")

	env.resolveTS()

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

func TestTSResolve_CrossFileNamedImport(t *testing.T) {
	env := newTestEnv(t)

	// File 1: utils.ts with exported function
	env.extractTSSource(`
export function greet(name: string): string {
  return "Hello, " + name
}
`, "utils.ts")

	// File 2: main.ts imports and calls greet
	env.extractTSSource(`
import { greet } from './utils'

function main(): void {
  greet("world")
}
`, "main.ts")

	env.resolveTS()

	// The "greet" call reference should be resolved
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
	require.NotEmpty(t, resolved, "expected greet call to be resolved via import")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "greet", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestTSResolve_DefaultImport(t *testing.T) {
	env := newTestEnv(t)

	// File 1: logger.ts with exported function
	env.extractTSSource(`
export function createLogger(): void {}
`, "logger.ts")

	// File 2: main.ts with default import
	env.extractTSSource(`
import Logger from './logger'

function main(): void {
  Logger()
}
`, "main.ts")

	env.resolveTS()

	refs, err := env.store.ReferencesByName("Logger")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to Logger")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Logger default import to be resolved")

	assert.Equal(t, "import", resolved[0].ResolutionKind)
}

func TestTSResolve_ClassMethodResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
class Server {
  start(): void {}
  stop(): void {}
}

function main(): void {
  const s = new Server()
  s.start()
}
`, "main.ts")

	env.resolveTS()

	// Find "start" field_access reference
	refs, err := env.store.ReferencesByName("start")
	require.NoError(t, err)
	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to start")

	resolved, err := env.store.ResolvedReferencesByRef(fieldRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected start method to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "start", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestTSResolve_InterfaceImplementation(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
interface Reader {
  read(): string
}

class FileReader {
  read(): string {
    return "data"
  }
}
`, "main.ts")

	env.resolveTS()

	ifaceSym := findSymbolByName(t, env.store, "Reader", "interface")
	require.NotNil(t, ifaceSym)

	impls, err := env.store.ImplementationsByInterface(ifaceSym.ID)
	require.NoError(t, err)
	require.Len(t, impls, 1, "expected 1 implementation of Reader")

	typeSym := findSymbolByID(t, env.store, impls[0].TypeSymbolID)
	assert.Equal(t, "FileReader", typeSym.Name)
	assert.Equal(t, "structural", impls[0].Kind)
}

func TestTSResolve_CallGraphEdge(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
function helper(): void {}

function main(): void {
  helper()
}
`, "main.ts")

	env.resolveTS()

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

func TestTSResolve_UnresolvedReference(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
function main(): void {
  nonExistent()
}
`, "main.ts")

	env.resolveTS()

	refs, err := env.store.ReferencesByName("nonExistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonExistent should not be resolved")
	}
}

func TestTSResolve_ClassExtendsResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
class Animal {
  move(): void {}
}

class Dog extends Animal {
  bark(): void {}
}

function main(): void {
  const d = new Dog()
  d.move()
}
`, "main.ts")

	env.resolveTS()

	// "move" field_access should resolve to Animal.move method
	refs, err := env.store.ReferencesByName("move")
	require.NoError(t, err)
	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to move")

	resolved, err := env.store.ResolvedReferencesByRef(fieldRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected move to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "move", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestTSResolve_ExtensionBinding(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
class Server {
  start(): void {}
  stop(): void {}
}
`, "main.ts")

	env.resolveTS()

	serverSym := findSymbolByName(t, env.store, "Server", "class")
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

func TestTSResolve_TypeAnnotationResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
interface Config {
  host: string
}

function createConfig(): Config {
  return { host: "localhost" }
}
`, "main.ts")

	env.resolveTS()

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
	assert.Equal(t, "interface", targetSym.Kind)
}

func TestTSResolve_NewExpressionResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractTSSource(`
class MyService {
  run(): void {}
}

function main(): void {
  const svc = new MyService()
}
`, "main.ts")

	env.resolveTS()

	// "new MyService()" creates a "call" context reference to "MyService"
	refs, err := env.store.ReferencesByName("MyService")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to MyService from new expression")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected MyService new expression to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "MyService", targetSym.Name)
	assert.Equal(t, "class", targetSym.Kind)
}
