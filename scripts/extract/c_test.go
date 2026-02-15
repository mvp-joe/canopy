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

// cTestEnv wraps the test environment for C extraction tests.
type cTestEnv struct {
	store *store.Store
	rt    *runtime.Runtime
	t     *testing.T
}

func newCTestEnv(t *testing.T) *cTestEnv {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())

	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")
	rt := runtime.NewRuntime(s, scriptsDir)

	t.Cleanup(func() { s.Close() })

	return &cTestEnv{store: s, rt: rt, t: t}
}

// extractCSource writes C source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *cTestEnv) extractCSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	cFile := filepath.Join(dir, "test.c")
	require.NoError(e.t, os.WriteFile(cFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     cFile,
		Language: "c",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": cFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "c.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// ---------- Tests ----------

func TestCExtract_FunctionDefinition(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`int add(int a, int b) {
    return a + b;
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
	assert.Equal(t, "add", fn.Name)
	assert.Equal(t, "function", fn.Kind)
	assert.Equal(t, "public", fn.Visibility)
	assert.Equal(t, 0, fn.StartLine)
}

func TestCExtract_FunctionDeclaration(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`void hello(int x, char *y);
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
	assert.Equal(t, "hello", fn.Name)
	assert.Equal(t, "public", fn.Visibility)
}

func TestCExtract_StructWithFields(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`struct Point {
    int x;
    int y;
};
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
	assert.Equal(t, "Point", structSym.Name)

	members, err := env.store.TypeMembers(structSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	fieldNames := map[string]string{}
	for _, m := range members {
		assert.Equal(t, "field", m.Kind)
		fieldNames[m.Name] = m.TypeExpr
	}
	assert.Equal(t, "int", fieldNames["x"])
	assert.Equal(t, "int", fieldNames["y"])
}

func TestCExtract_Typedef(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`typedef unsigned long size_t;
typedef int MyInt;
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
	assert.True(t, names["size_t"])
	assert.True(t, names["MyInt"])
}

func TestCExtract_EnumWithEnumerators(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`enum Color { RED, GREEN, BLUE };
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var enumSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "enum" {
			enumSym = s
			break
		}
	}
	require.NotNil(t, enumSym)
	assert.Equal(t, "Color", enumSym.Name)

	members, err := env.store.TypeMembers(enumSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3)

	for _, m := range members {
		assert.Equal(t, "variant", m.Kind)
	}

	memberNames := []string{}
	for _, m := range members {
		memberNames = append(memberNames, m.Name)
	}
	assert.Contains(t, memberNames, "RED")
	assert.Contains(t, memberNames, "GREEN")
	assert.Contains(t, memberNames, "BLUE")
}

func TestCExtract_IncludeDirectives(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`#include <stdio.h>
#include "myheader.h"
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	sources := []string{}
	for _, imp := range imports {
		sources = append(sources, imp.Source)
		assert.Equal(t, "header", imp.Kind)
	}
	assert.Contains(t, sources, "stdio.h")
	assert.Contains(t, sources, "myheader.h")
}

func TestCExtract_MacroDefinitions(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`#define MAX_SIZE 100
#define SQUARE(x) ((x) * (x))
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	symsByName := map[string]*store.Symbol{}
	for _, s := range syms {
		symsByName[s.Name] = s
	}

	// Value macro -> constant
	require.Contains(t, symsByName, "MAX_SIZE")
	assert.Equal(t, "constant", symsByName["MAX_SIZE"].Kind)

	// Function-like macro -> function
	require.Contains(t, symsByName, "SQUARE")
	assert.Equal(t, "function", symsByName["SQUARE"].Kind)
}

func TestCExtract_GlobalVariables(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`int global_var = 42;
const char *name = "test";
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
	assert.Contains(t, vars, "global_var")
	assert.Contains(t, vars, "name")
}

func TestCExtract_ScopeTree(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`void process(int x) {
    if (x > 0) {
        for (int i = 0; i < x; i++) {
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

	// Verify nesting
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

func TestCExtract_References_Call(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`void helper() {}

void main() {
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

func TestCExtract_References_FieldAccess(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`struct Foo { int bar; };

void test() {
    struct Foo f;
    f.bar;
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Name == "bar" && r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to bar")
}

func TestCExtract_References_ArrowFieldAccess(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`struct Node { int value; };

void test() {
    struct Node *n;
    n->value;
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Name == "value" && r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference via ->")
}

func TestCExtract_FunctionParams(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`int add(int a, int b) {
    return a + b;
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
	assert.Equal(t, "int", regularParams[0].TypeExpr)
	assert.Equal(t, 0, regularParams[0].Ordinal)
	assert.Equal(t, "b", regularParams[1].Name)
	assert.Equal(t, "int", regularParams[1].TypeExpr)
	assert.Equal(t, 1, regularParams[1].Ordinal)

	require.Len(t, returnParams, 1)
	assert.Equal(t, "int", returnParams[0].TypeExpr)
	assert.True(t, returnParams[0].IsReturn)
}

func TestCExtract_ComprehensiveFile(t *testing.T) {
	env := newCTestEnv(t)
	fileID := env.extractCSource(`#include <stdio.h>
#include "utils.h"

#define MAX_SIZE 256

typedef unsigned int uint;

enum Status { OK, ERROR };

struct Config {
    char *host;
    int port;
};

int global_count = 0;

void init(struct Config *cfg) {
    cfg->host = "localhost";
    cfg->port = 8080;
}

int main() {
    struct Config c;
    init(&c);
    printf("Host: %s\n", c.host);
    return 0;
}
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["constant"], "MAX_SIZE")
	assert.Contains(t, kinds["type_alias"], "uint")
	assert.Contains(t, kinds["enum"], "Status")
	assert.Contains(t, kinds["struct"], "Config")
	assert.Contains(t, kinds["variable"], "global_count")
	assert.Contains(t, kinds["function"], "init")
	assert.Contains(t, kinds["function"], "main")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	// Verify scope tree
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 3) // file + 2 functions

	// Verify references exist
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)
}
