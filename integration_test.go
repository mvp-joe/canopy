package canopy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findModuleRoot walks up from cwd to find go.mod, returning the repo root.
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

// newIntegrationEngine creates an Engine backed by a temp DB and the real scripts dir.
func newIntegrationEngine(t *testing.T, opts ...Option) *Engine {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "integration.db")
	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")

	e, err := New(dbPath, scriptsDir, opts...)
	require.NoError(t, err)
	t.Cleanup(func() { e.Close() })
	return e
}

// writeGoFile writes Go source to a temp dir and returns the path.
func writeGoFile(t *testing.T, dir, name, src string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(src), 0644))
	return path
}

// TestIntegration_FullPipeline_GoDefinition tests the complete pipeline:
// source file → IndexFiles → Resolve → QueryBuilder.DefinitionAt
func TestIntegration_FullPipeline_GoDefinition(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()

	// Write two Go files: a library and a consumer.
	// Lines are 0-indexed, columns are 0-indexed in extraction output.
	libPath := writeGoFile(t, dir, "lib.go", `package main

func Helper() string {
	return "hello"
}
`)
	mainPath := writeGoFile(t, dir, "main.go", `package main

func main() {
	Helper()
}
`)

	// Index both files.
	require.NoError(t, e.IndexFiles(ctx, []string{libPath, mainPath}))

	// Resolve cross-file references.
	require.NoError(t, e.Resolve(ctx))

	// Query: go-to-definition on "Helper()" call in main.go.
	// The call `Helper()` is at line 3, col 1 (tab-indented).
	q := e.Query()
	locs, err := q.DefinitionAt(mainPath, 3, 1)
	require.NoError(t, err)
	require.NotEmpty(t, locs, "expected DefinitionAt to find Helper definition")

	assert.Equal(t, libPath, locs[0].File)
	// Helper is declared starting at line 2.
	assert.Equal(t, 2, locs[0].StartLine)
}

// TestIntegration_FullPipeline_GoReferences tests:
// source file → IndexFiles → Resolve → QueryBuilder.ReferencesTo
func TestIntegration_FullPipeline_GoReferences(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()

	path := writeGoFile(t, dir, "main.go", `package main

func greet() string {
	return "hi"
}

func main() {
	greet()
}
`)

	require.NoError(t, e.IndexFiles(ctx, []string{path}))
	require.NoError(t, e.Resolve(ctx))

	// Find the "greet" function symbol.
	syms, err := e.Store().SymbolsByName("greet")
	require.NoError(t, err)
	var greetID int64
	for _, s := range syms {
		if s.Kind == "function" {
			greetID = s.ID
			break
		}
	}
	require.NotZero(t, greetID, "expected to find greet function symbol")

	q := e.Query()
	locs, err := q.ReferencesTo(greetID)
	require.NoError(t, err)
	require.NotEmpty(t, locs, "expected at least one reference to greet")

	// The call reference should be on line 7 (0-indexed): `greet()`.
	found := false
	for _, loc := range locs {
		if loc.File == path && loc.StartLine == 7 {
			found = true
		}
	}
	assert.True(t, found, "expected a reference on the greet() call line (line 7)")
}

// TestIntegration_FullPipeline_GoImplementations tests:
// source file → IndexFiles → Resolve → QueryBuilder.Implementations
func TestIntegration_FullPipeline_GoImplementations(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()

	path := writeGoFile(t, dir, "main.go", `package main

type Stringer interface {
	String() string
}

type MyType struct{}

func (m *MyType) String() string {
	return "my"
}
`)

	require.NoError(t, e.IndexFiles(ctx, []string{path}))
	require.NoError(t, e.Resolve(ctx))

	// Find the interface symbol.
	syms, err := e.Store().SymbolsByName("Stringer")
	require.NoError(t, err)
	var ifaceID int64
	for _, s := range syms {
		if s.Kind == "interface" {
			ifaceID = s.ID
			break
		}
	}
	require.NotZero(t, ifaceID, "expected to find Stringer interface symbol")

	q := e.Query()
	locs, err := q.Implementations(ifaceID)
	require.NoError(t, err)
	require.Len(t, locs, 1, "expected exactly one implementation of Stringer")
	assert.Equal(t, path, locs[0].File)
}

// TestIntegration_FullPipeline_GoCallGraph tests:
// source file → IndexFiles → Resolve → QueryBuilder.Callers/Callees
func TestIntegration_FullPipeline_GoCallGraph(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()

	path := writeGoFile(t, dir, "main.go", `package main

func a() { b() }
func b() { c() }
func c() {}
`)

	require.NoError(t, e.IndexFiles(ctx, []string{path}))
	require.NoError(t, e.Resolve(ctx))

	// Find symbol IDs.
	findFunc := func(name string) int64 {
		syms, err := e.Store().SymbolsByName(name)
		require.NoError(t, err)
		for _, s := range syms {
			if s.Kind == "function" {
				return s.ID
			}
		}
		t.Fatalf("function %s not found", name)
		return 0
	}
	aID := findFunc("a")
	bID := findFunc("b")
	cID := findFunc("c")

	q := e.Query()

	// a calls b.
	callees, err := q.Callees(aID)
	require.NoError(t, err)
	require.Len(t, callees, 1)
	assert.Equal(t, bID, callees[0].CalleeSymbolID)

	// b is called by a.
	callers, err := q.Callers(bID)
	require.NoError(t, err)
	require.Len(t, callers, 1)
	assert.Equal(t, aID, callers[0].CallerSymbolID)

	// b calls c.
	callees, err = q.Callees(bID)
	require.NoError(t, err)
	require.Len(t, callees, 1)
	assert.Equal(t, cID, callees[0].CalleeSymbolID)
}

// TestIntegration_FullPipeline_GoDependencies tests:
// source file → IndexFiles → Resolve → QueryBuilder.Dependencies/Dependents
func TestIntegration_FullPipeline_GoDependencies(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()

	path := writeGoFile(t, dir, "main.go", `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)

	require.NoError(t, e.IndexFiles(ctx, []string{path}))

	// Look up the file to get its ID.
	f, err := e.Store().FileByPath(path)
	require.NoError(t, err)
	require.NotNil(t, f)

	q := e.Query()

	// Dependencies: main.go imports "fmt".
	deps, err := q.Dependencies(f.ID)
	require.NoError(t, err)
	require.NotEmpty(t, deps)
	found := false
	for _, d := range deps {
		if d.Source == "fmt" {
			found = true
		}
	}
	assert.True(t, found, "expected fmt in dependencies")

	// Dependents: who imports "fmt"?
	dependents, err := q.Dependents("fmt")
	require.NoError(t, err)
	require.NotEmpty(t, dependents)
	assert.Equal(t, f.ID, dependents[0].FileID)
}

// TestIntegration_IndexDirectory tests the full directory walk pipeline.
func TestIntegration_IndexDirectory(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()

	root := t.TempDir()
	sub := filepath.Join(root, "pkg")
	require.NoError(t, os.MkdirAll(sub, 0755))

	writeGoFile(t, root, "main.go", `package main

import "pkg"

func main() {
	pkg.Do()
}
`)
	writeGoFile(t, sub, "pkg.go", `package pkg

func Do() string {
	return "done"
}
`)

	require.NoError(t, e.IndexDirectory(ctx, root))
	require.NoError(t, e.Resolve(ctx))

	// Verify both files were indexed.
	mainFile, err := e.Store().FileByPath(filepath.Join(root, "main.go"))
	require.NoError(t, err)
	require.NotNil(t, mainFile)

	pkgFile, err := e.Store().FileByPath(filepath.Join(sub, "pkg.go"))
	require.NoError(t, err)
	require.NotNil(t, pkgFile)

	// Verify the Do function symbol was extracted.
	syms, err := e.Store().SymbolsByName("Do")
	require.NoError(t, err)
	require.NotEmpty(t, syms)
}

// TestIntegration_IncrementalReindex verifies that re-indexing a changed file
// works correctly (old data deleted, new data extracted).
func TestIntegration_IncrementalReindex(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()

	path := filepath.Join(dir, "main.go")

	// First version: has function "alpha".
	require.NoError(t, os.WriteFile(path, []byte(`package main

func alpha() {}
`), 0644))
	require.NoError(t, e.IndexFiles(ctx, []string{path}))

	syms, err := e.Store().SymbolsByName("alpha")
	require.NoError(t, err)
	require.NotEmpty(t, syms, "alpha should exist after first index")

	// Second version: rename alpha to beta.
	require.NoError(t, os.WriteFile(path, []byte(`package main

func beta() {}
`), 0644))
	require.NoError(t, e.IndexFiles(ctx, []string{path}))

	// alpha should be gone, beta should exist.
	syms, err = e.Store().SymbolsByName("alpha")
	require.NoError(t, err)
	assert.Empty(t, syms, "alpha should be gone after reindex")

	syms, err = e.Store().SymbolsByName("beta")
	require.NoError(t, err)
	assert.NotEmpty(t, syms, "beta should exist after reindex")
}

// TestIntegration_ChangeDetection verifies that unchanged files are skipped.
func TestIntegration_ChangeDetection(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()

	path := writeGoFile(t, dir, "main.go", `package main

func hello() {}
`)

	// First index.
	require.NoError(t, e.IndexFiles(ctx, []string{path}))

	// Get the file record.
	f1, err := e.Store().FileByPath(path)
	require.NoError(t, err)
	require.NotNil(t, f1)
	id1 := f1.ID

	// Second index without changes — should skip.
	require.NoError(t, e.IndexFiles(ctx, []string{path}))

	f2, err := e.Store().FileByPath(path)
	require.NoError(t, err)
	require.NotNil(t, f2)

	// Same file ID means it was not re-inserted.
	assert.Equal(t, id1, f2.ID)
}
