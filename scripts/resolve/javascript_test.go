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

// extractJSSource writes JavaScript source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractJSSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	jsFile := filepath.Join(dir, filename)
	require.NoError(e.t, os.WriteFile(jsFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     jsFile,
		Language: "javascript",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": jsFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "javascript.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// resolveJS runs the JavaScript resolution script.
func (e *testEnv) resolveJS() {
	e.t.Helper()
	extras := map[string]any{
		"files_to_resolve": runtime.MakeFilesToResolveFn(e.store, nil),
	}
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "javascript.risor"), extras)
	require.NoError(e.t, err)
}

// --- JavaScript Resolution Tests ---

func TestJSResolve_SameFileFunctionCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractJSSource(`
function helper() {}

function main() {
  helper()
}
`, "main.js")

	env.resolveJS()

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

func TestJSResolve_CrossFileNamedImport(t *testing.T) {
	env := newTestEnv(t)

	// File 1: utils.js with exported function
	env.extractJSSource(`
export function greet(name) {
  return "Hello, " + name
}
`, "utils.js")

	// File 2: main.js imports and calls greet
	env.extractJSSource(`
import { greet } from './utils'

function main() {
  greet("world")
}
`, "main.js")

	env.resolveJS()

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

func TestJSResolve_DefaultImport(t *testing.T) {
	env := newTestEnv(t)

	// File 1: logger.js with exported function
	env.extractJSSource(`
export function createLogger() {}
`, "logger.js")

	// File 2: main.js with default import
	env.extractJSSource(`
import Logger from './logger'

function main() {
  Logger()
}
`, "main.js")

	env.resolveJS()

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

func TestJSResolve_CommonJSRequire(t *testing.T) {
	env := newTestEnv(t)

	// File 1: utils.js with exported function
	env.extractJSSource(`
export function processData(data) {
  return data
}
`, "utils.js")

	// File 2: main.js with require()
	env.extractJSSource(`
const processData = require('./utils')

function main() {
  processData("hello")
}
`, "main.js")

	env.resolveJS()

	refs, err := env.store.ReferencesByName("processData")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to processData")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected processData require to be resolved")

	assert.Equal(t, "import", resolved[0].ResolutionKind)
}

func TestJSResolve_ClassMethodResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractJSSource(`
class Server {
  start() {}
  stop() {}
}

function main() {
  const s = new Server()
  s.start()
}
`, "main.js")

	env.resolveJS()

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

func TestJSResolve_CallGraphEdge(t *testing.T) {
	env := newTestEnv(t)
	env.extractJSSource(`
function helper() {}

function main() {
  helper()
}
`, "main.js")

	env.resolveJS()

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

func TestJSResolve_UnresolvedReference(t *testing.T) {
	env := newTestEnv(t)
	env.extractJSSource(`
function main() {
  nonExistent()
}
`, "main.js")

	env.resolveJS()

	refs, err := env.store.ReferencesByName("nonExistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonExistent should not be resolved")
	}
}

func TestJSResolve_ClassExtendsResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractJSSource(`
class Animal {
  move() {}
}

class Dog extends Animal {
  bark() {}
}

function main() {
  const d = new Dog()
  d.move()
}
`, "main.js")

	env.resolveJS()

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

func TestJSResolve_ExtensionBinding(t *testing.T) {
	env := newTestEnv(t)
	env.extractJSSource(`
class Server {
  start() {}
  stop() {}
}
`, "main.js")

	env.resolveJS()

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

func TestJSResolve_NewExpressionResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractJSSource(`
class MyService {
  run() {}
}

function main() {
  const svc = new MyService()
}
`, "main.js")

	env.resolveJS()

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
