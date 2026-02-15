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

// extractJSSource writes JavaScript source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractJSSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	jsFile := filepath.Join(dir, "test.js")
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

// ---------- JavaScript Tests ----------

func TestJS_FunctionDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`function greet(name) {
  return "Hello, " + name;
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fn *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fn = s
		}
	}
	require.NotNil(t, fn, "expected function symbol")
	assert.Equal(t, "greet", fn.Name)
	assert.Equal(t, 1, fn.StartLine)

	// Check parameters
	params, err := env.store.FunctionParams(fn.ID)
	require.NoError(t, err)
	require.Len(t, params, 1)
	assert.Equal(t, "name", params[0].Name)
	assert.Equal(t, "", params[0].TypeExpr) // no types in JS
}

func TestJS_ArrowFunction(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`const add = (a, b) => a + b;
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fn *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fn = s
		}
	}
	require.NotNil(t, fn, "expected function symbol for arrow function")
	assert.Equal(t, "add", fn.Name)

	params, err := env.store.FunctionParams(fn.ID)
	require.NoError(t, err)
	require.Len(t, params, 2)
	assert.Equal(t, "a", params[0].Name)
	assert.Equal(t, "b", params[1].Name)
}

func TestJS_ClassWithMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`class Animal {
  constructor(name) {
    this.name = name;
  }
  speak() {
    return this.name + " speaks";
  }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
	}
	require.NotNil(t, classSym)
	assert.Equal(t, "Animal", classSym.Name)

	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	memberNames := map[string]string{}
	for _, m := range members {
		memberNames[m.Name] = m.Kind
	}
	assert.Equal(t, "method", memberNames["constructor"])
	assert.Equal(t, "method", memberNames["speak"])

	// Method should be linked to class
	var speakSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "method" && s.Name == "speak" {
			speakSym = s
		}
	}
	require.NotNil(t, speakSym)
	require.NotNil(t, speakSym.ParentSymbolID)
	assert.Equal(t, classSym.ID, *speakSym.ParentSymbolID)

	// Constructor params
	var ctorSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "method" && s.Name == "constructor" {
			ctorSym = s
		}
	}
	require.NotNil(t, ctorSym)
	ctorParams, err := env.store.FunctionParams(ctorSym.ID)
	require.NoError(t, err)
	require.Len(t, ctorParams, 1)
	assert.Equal(t, "name", ctorParams[0].Name)
}

func TestJS_VariableDeclarations(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`var x = 1;
let y = "hello";
const z = true;
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	symsByName := map[string]*store.Symbol{}
	for _, s := range syms {
		symsByName[s.Name] = s
	}

	require.Contains(t, symsByName, "x")
	assert.Equal(t, "variable", symsByName["x"].Kind)

	require.Contains(t, symsByName, "y")
	assert.Equal(t, "variable", symsByName["y"].Kind)

	require.Contains(t, symsByName, "z")
	assert.Equal(t, "constant", symsByName["z"].Kind)
}

func TestJS_ES6Imports(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`import { foo, bar } from './module';
import * as utils from './utils';
import defaultExport from './default';
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 4) // 2 named + 1 namespace + 1 default

	impByKind := map[string][]*store.Import{}
	for _, imp := range imports {
		impByKind[imp.Kind] = append(impByKind[imp.Kind], imp)
	}

	require.Len(t, impByKind["named"], 2)
	require.Len(t, impByKind["namespace"], 1)
	require.Len(t, impByKind["default"], 1)

	assert.Equal(t, "utils", *impByKind["namespace"][0].LocalAlias)
	assert.Equal(t, "defaultExport", *impByKind["default"][0].ImportedName)
}

func TestJS_RequireImports(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`const fs = require('fs');
const { readFile } = require('fs');
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	for _, imp := range imports {
		assert.Equal(t, "fs", imp.Source)
		assert.Equal(t, "require", imp.Kind)
	}

	names := map[string]bool{}
	for _, imp := range imports {
		require.NotNil(t, imp.ImportedName)
		names[*imp.ImportedName] = true
	}
	assert.True(t, names["fs"])
	assert.True(t, names["readFile"])
}

func TestJS_ExportedFunction(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`export function hello() {}
function internal() {}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "function" {
			vis[s.Name] = s.Visibility
		}
	}
	assert.Equal(t, "public", vis["hello"])
	assert.Equal(t, "private", vis["internal"])
}

func TestJS_ReExport(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`export { qux } from './other';
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 1)

	imp := imports[0]
	assert.Equal(t, "./other", imp.Source)
	assert.Equal(t, "reexport", imp.Kind)
	require.NotNil(t, imp.ImportedName)
	assert.Equal(t, "qux", *imp.ImportedName)
}

func TestJS_ScopeTree(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`function process(x) {
  if (x > 0) {
    for (let i = 0; i < x; i++) {
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
	assert.Equal(t, 1, kinds["file"])
	assert.Equal(t, 1, kinds["function"])
	assert.GreaterOrEqual(t, kinds["block"], 2, "expected at least 2 block scopes (if + for)")
}

func TestJS_References_Call(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`function helper() {}

function main() {
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

func TestJS_References_FieldAccess(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`console.log("hello");
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Name == "log" && r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to log")
}

func TestJS_References_NewExpression(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`class Foo {}
const x = new Foo();
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var newRef *store.Reference
	for _, r := range refs {
		if r.Name == "Foo" && r.Context == "call" {
			newRef = r
			break
		}
	}
	require.NotNil(t, newRef, "expected call reference from new Foo()")
}

func TestJS_ComprehensiveFile(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJSSource(`import { EventEmitter } from 'events';

export class Server {
  constructor(name) {
    this.name = name;
  }

  handle(req) {
    return "handled: " + req;
  }
}

export function createServer(name) {
  return new Server(name);
}

const DEFAULT_PORT = 8080;
let debug = false;
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["class"], "Server")
	assert.Contains(t, kinds["function"], "createServer")
	assert.Contains(t, kinds["constant"], "DEFAULT_PORT")
	assert.Contains(t, kinds["variable"], "debug")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(imports), 1)

	// Verify scopes
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 3) // file + function + class

	// Verify references
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)

	// Verify visibility
	vis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "method" {
			continue
		}
		vis[s.Name] = s.Visibility
	}
	assert.Equal(t, "public", vis["Server"])
	assert.Equal(t, "public", vis["createServer"])
	assert.Equal(t, "private", vis["DEFAULT_PORT"])
	assert.Equal(t, "private", vis["debug"])
}

func TestJS_EndToEnd_ViaEngine(t *testing.T) {
	dir := t.TempDir()
	jsFile := filepath.Join(dir, "app.js")
	require.NoError(t, os.WriteFile(jsFile, []byte(`
export function greet(name) {
  return "Hello, " + name;
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

	fileID, err := s.InsertFile(&store.File{
		Path:     jsFile,
		Language: "javascript",
	})
	require.NoError(t, err)

	extras := map[string]any{
		"file_path": jsFile,
		"file_id":   fileID,
	}
	err = rt.RunScript(context.Background(), filepath.Join("extract", "javascript.risor"), extras)
	require.NoError(t, err)

	syms, err := s.SymbolsByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(syms), 1)
}
