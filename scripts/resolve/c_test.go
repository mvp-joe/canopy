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

// cTestEnv wraps the shared testEnv with C-specific helpers.
type cTestEnv struct {
	*testEnv
}

func newCTestEnv(t *testing.T) *cTestEnv {
	return &cTestEnv{testEnv: newTestEnv(t)}
}

// extractCSource writes C source to a temp .c file, inserts a file record,
// and runs the C extraction script. Returns the file ID.
func (e *cTestEnv) extractCSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	cFile := filepath.Join(dir, filename)
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

// resolveC runs the C resolution script.
func (e *cTestEnv) resolveC() {
	e.t.Helper()
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "c.risor"), nil)
	require.NoError(e.t, err)
}

// --- Tests ---

func TestCResolve_SameFileFunctionCall(t *testing.T) {
	env := newCTestEnv(t)
	env.extractCSource(`void helper() {}

void main() {
    helper();
}
`, "main.c")

	env.resolveC()

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

func TestCResolve_StructFieldAccess(t *testing.T) {
	env := newCTestEnv(t)
	env.extractCSource(`struct Point {
    int x;
    int y;
};

void use_point() {
    struct Point p;
    p.x = 10;
}
`, "point.c")

	env.resolveC()

	// Find the field_access reference to "x"
	refs, err := env.store.ReferencesByName("x")
	require.NoError(t, err)
	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to x")

	resolved, err := env.store.ResolvedReferencesByRef(fieldRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected field access to x to be resolved")

	// Should resolve to the Point struct (which owns the field)
	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Point", targetSym.Name)
	assert.Equal(t, "struct", targetSym.Kind)
}

func TestCResolve_CrossFileIncludeResolution(t *testing.T) {
	env := newCTestEnv(t)

	// Header file defines a function
	env.extractCSource(`int add(int a, int b) {
    return a + b;
}
`, "math_utils.h")

	// Source file includes header and calls the function
	env.extractCSource(`#include "math_utils.h"

int main() {
    add(1, 2);
    return 0;
}
`, "main.c")

	env.resolveC()

	refs, err := env.store.ReferencesByName("add")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to add")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected add call to be resolved via include")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "add", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
	assert.Equal(t, "import", resolved[0].ResolutionKind)
}

func TestCResolve_CallGraphEdges(t *testing.T) {
	env := newCTestEnv(t)
	env.extractCSource(`void helper() {}

void main() {
    helper();
}
`, "main.c")

	env.resolveC()

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

func TestCResolve_UnresolvedReference(t *testing.T) {
	env := newCTestEnv(t)
	env.extractCSource(`void main() {
    nonexistent();
}
`, "main.c")

	env.resolveC()

	refs, err := env.store.ReferencesByName("nonexistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonexistent should not be resolved")
	}
}

func TestCResolve_TypedefResolution(t *testing.T) {
	env := newCTestEnv(t)
	env.extractCSource(`typedef int myint;

myint add(myint a, myint b) {
    return a + b;
}
`, "types.c")

	env.resolveC()

	// Find type_annotation references to "myint"
	refs, err := env.store.ReferencesByName("myint")
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to myint")

	resolved, err := env.store.ResolvedReferencesByRef(typeRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected myint type reference to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "myint", targetSym.Name)
	assert.Equal(t, "type_alias", targetSym.Kind)
}

func TestCResolve_CrossFileFunction(t *testing.T) {
	env := newCTestEnv(t)

	// File 1: defines a utility function
	env.extractCSource(`int multiply(int a, int b) {
    return a * b;
}
`, "util.c")

	// File 2: calls the function (no include, just cross-file)
	env.extractCSource(`int main() {
    multiply(3, 4);
    return 0;
}
`, "main.c")

	env.resolveC()

	refs, err := env.store.ReferencesByName("multiply")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to multiply")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected multiply call to be resolved cross-file")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "multiply", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestCResolve_TypedefTypeAnnotation(t *testing.T) {
	env := newCTestEnv(t)
	// In C, typedef names produce type_identifier references when used as types.
	// Bare struct/enum names in "struct Foo" form don't (the name is part of the
	// specifier), but typedef'd names do.
	env.extractCSource(`typedef struct {
    int x;
    int y;
} Point;

Point origin() {
    Point p;
    return p;
}
`, "point.c")

	env.resolveC()

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
	assert.Equal(t, "type_alias", targetSym.Kind)
}
