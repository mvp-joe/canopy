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
	syms, err := e.store.SymbolsByName("greet")
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
	syms, err := e.store.SymbolsByName("Stringer")
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
		syms, err := e.store.SymbolsByName(name)
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
	f, err := e.store.FileByPath(path)
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
	mainFile, err := e.store.FileByPath(filepath.Join(root, "main.go"))
	require.NoError(t, err)
	require.NotNil(t, mainFile)

	pkgFile, err := e.store.FileByPath(filepath.Join(sub, "pkg.go"))
	require.NoError(t, err)
	require.NotNil(t, pkgFile)

	// Verify the Do function symbol was extracted.
	syms, err := e.store.SymbolsByName("Do")
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

	syms, err := e.store.SymbolsByName("alpha")
	require.NoError(t, err)
	require.NotEmpty(t, syms, "alpha should exist after first index")

	// Second version: rename alpha to beta.
	require.NoError(t, os.WriteFile(path, []byte(`package main

func beta() {}
`), 0644))
	require.NoError(t, e.IndexFiles(ctx, []string{path}))

	// alpha should be gone, beta should exist.
	syms, err = e.store.SymbolsByName("alpha")
	require.NoError(t, err)
	assert.Empty(t, syms, "alpha should be gone after reindex")

	syms, err = e.store.SymbolsByName("beta")
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
	f1, err := e.store.FileByPath(path)
	require.NoError(t, err)
	require.NotNil(t, f1)
	id1 := f1.ID

	// Second index without changes — should skip.
	require.NoError(t, e.IndexFiles(ctx, []string{path}))

	f2, err := e.store.FileByPath(path)
	require.NoError(t, err)
	require.NotNil(t, f2)

	// Same file ID means it was not re-inserted.
	assert.Equal(t, id1, f2.ID)
}

// =============================================================================
// Self-indexing discovery API test
// =============================================================================

// containsSymbolNamed checks if any SymbolResult in items has the given name.
func containsSymbolNamed(items []SymbolResult, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

// TestDiscovery_SelfIndex indexes canopy's own source files and exercises
// every discovery API method against real extraction output.
func TestDiscovery_SelfIndex(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	modRoot := findModuleRoot(t)

	// Index a curated set of canopy's own Go files.
	files := []string{
		filepath.Join(modRoot, "engine.go"),
		filepath.Join(modRoot, "query.go"),
		filepath.Join(modRoot, "query_discovery.go"),
		filepath.Join(modRoot, "internal", "store", "store.go"),
		filepath.Join(modRoot, "internal", "store", "types.go"),
		filepath.Join(modRoot, "internal", "store", "extraction.go"),
	}
	require.NoError(t, e.IndexFiles(ctx, files))
	require.NoError(t, e.Resolve(ctx))

	q := e.Query()

	t.Run("Symbols_NoFilter", func(t *testing.T) {
		result, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0, "should find symbols in canopy source")
		// Every item should have a non-empty name
		for _, item := range result.Items {
			assert.NotEmpty(t, item.Name)
		}
	})

	t.Run("Symbols_FilterByKind_Struct", func(t *testing.T) {
		result, err := q.Symbols(SymbolFilter{Kinds: []string{"struct"}}, Sort{Field: SortByName, Order: Asc}, Pagination{Limit: intP(500)})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0)
		// Should find key structs
		assert.True(t, containsSymbolNamed(result.Items, "Engine"), "should find Engine struct")
		assert.True(t, containsSymbolNamed(result.Items, "Store"), "should find Store struct")
		assert.True(t, containsSymbolNamed(result.Items, "QueryBuilder"), "should find QueryBuilder struct")
		assert.True(t, containsSymbolNamed(result.Items, "Symbol"), "should find Symbol struct")
		// All should be structs
		for _, item := range result.Items {
			assert.Equal(t, "struct", item.Kind)
		}
	})

	t.Run("Symbols_FilterByKind_Function", func(t *testing.T) {
		result, err := q.Symbols(SymbolFilter{Kinds: []string{"function"}}, Sort{Field: SortByName, Order: Asc}, Pagination{Limit: intP(500)})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0)
		assert.True(t, containsSymbolNamed(result.Items, "New"), "should find New() engine constructor")
		for _, item := range result.Items {
			assert.Equal(t, "function", item.Kind)
		}
	})

	t.Run("Symbols_FilterByVisibility_Public", func(t *testing.T) {
		vis := "public"
		result, err := q.Symbols(SymbolFilter{Visibility: &vis}, Sort{}, Pagination{Limit: intP(500)})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0)
		for _, item := range result.Items {
			assert.Equal(t, "public", item.Visibility)
		}
	})

	t.Run("Symbols_FilterByPathPrefix", func(t *testing.T) {
		// Use absolute path prefix since we indexed absolute paths
		storePrefix := filepath.Join(modRoot, "internal", "store") + "/"
		result, err := q.Symbols(SymbolFilter{PathPrefix: &storePrefix}, Sort{}, Pagination{Limit: intP(500)})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0, "should find symbols in internal/store/")
		// All results should have file paths under internal/store/
		for _, item := range result.Items {
			assert.Contains(t, item.FilePath, "internal/store/",
				"symbol %s should be in internal/store/, got %s", item.Name, item.FilePath)
		}
	})

	t.Run("Files_NoFilter", func(t *testing.T) {
		result, err := q.Files("", "", Sort{Field: SortByFile, Order: Asc}, Pagination{})
		require.NoError(t, err)
		assert.Equal(t, 6, result.TotalCount, "should have indexed exactly 6 files")
	})

	t.Run("Files_FilterByLanguage", func(t *testing.T) {
		result, err := q.Files("", "go", Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Equal(t, 6, result.TotalCount, "all indexed files are Go")
	})

	t.Run("Files_FilterByPathPrefix", func(t *testing.T) {
		storePrefix := filepath.Join(modRoot, "internal", "store")
		result, err := q.Files(storePrefix, "", Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalCount, "should find 3 store files")
	})

	t.Run("Packages", func(t *testing.T) {
		result, err := q.Packages("", Sort{Field: SortByName, Order: Asc}, Pagination{})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0, "should find package symbols")
		// Should find the store and canopy packages
		assert.True(t, containsSymbolNamed(result.Items, "store"), "should find store package")
		assert.True(t, containsSymbolNamed(result.Items, "canopy"), "should find canopy package")
		// All should be package kinds
		for _, item := range result.Items {
			assert.Contains(t, []string{"package", "module", "namespace"}, item.Kind)
		}
	})

	t.Run("SearchSymbols_Prefix", func(t *testing.T) {
		result, err := q.SearchSymbols("Query*", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0, "Query* should match something")
		assert.True(t, containsSymbolNamed(result.Items, "QueryBuilder"), "should find QueryBuilder")
	})

	t.Run("SearchSymbols_Contains", func(t *testing.T) {
		result, err := q.SearchSymbols("*Store*", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0, "*Store* should match something")
		assert.True(t, containsSymbolNamed(result.Items, "Store"), "should find Store")
		assert.True(t, containsSymbolNamed(result.Items, "NewStore"), "should find NewStore")
	})

	t.Run("SearchSymbols_ExactMatch", func(t *testing.T) {
		result, err := q.SearchSymbols("New", SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Greater(t, result.TotalCount, 0, "exact match 'New' should find the constructor")
		assert.True(t, containsSymbolNamed(result.Items, "New"))
	})

	t.Run("SearchSymbols_WithKindFilter", func(t *testing.T) {
		result, err := q.SearchSymbols("*Engine*", SymbolFilter{Kinds: []string{"struct"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(result.Items, "Engine"))
		// All results should be structs
		for _, item := range result.Items {
			assert.Equal(t, "struct", item.Kind)
		}
	})

	t.Run("ProjectSummary", func(t *testing.T) {
		summary, err := q.ProjectSummary(5)
		require.NoError(t, err)

		// Should have Go language stats
		require.NotEmpty(t, summary.Languages)
		assert.Equal(t, "go", summary.Languages[0].Language)
		assert.Equal(t, 6, summary.Languages[0].FileCount)
		assert.Greater(t, summary.Languages[0].SymbolCount, 0)

		// Should have kind counts for real Go kinds
		kindCounts := summary.Languages[0].KindCounts
		assert.Greater(t, kindCounts["function"], 0, "should have functions")
		assert.Greater(t, kindCounts["struct"], 0, "should have structs")
		assert.Greater(t, kindCounts["package"], 0, "should have package symbols")

		// Package count
		assert.Greater(t, summary.PackageCount, 0)

		// Top symbols should have ref counts after resolution
		if len(summary.TopSymbols) > 0 {
			assert.Greater(t, summary.TopSymbols[0].RefCount, 0,
				"top symbol should have ref count > 0 after resolution")
		}
	})

	t.Run("PackageSummary_Store", func(t *testing.T) {
		storePrefix := filepath.Join(modRoot, "internal", "store")
		summary, err := q.PackageSummary(storePrefix, nil)
		require.NoError(t, err)

		assert.Equal(t, "store", summary.Symbol.Name)
		assert.Equal(t, 3, summary.FileCount, "store has 3 indexed files")

		// Should have exported symbols
		assert.Greater(t, len(summary.ExportedSymbols), 0, "store should have exported symbols")
		// Check for specific exports
		exportNames := make([]string, len(summary.ExportedSymbols))
		for i, sym := range summary.ExportedSymbols {
			exportNames[i] = sym.Name
		}
		assert.Contains(t, exportNames, "Store", "should export Store")
		assert.Contains(t, exportNames, "NewStore", "should export NewStore")
		assert.Contains(t, exportNames, "Symbol", "should export Symbol")

		// Kind counts should be populated
		assert.Greater(t, len(summary.KindCounts), 0)

		// Dependencies (imports from store files)
		assert.Greater(t, len(summary.Dependencies), 0, "store should have imports")
	})

	t.Run("Pagination_Across_Real_Data", func(t *testing.T) {
		// Get total count
		all, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Limit: intP(500)})
		require.NoError(t, err)
		total := all.TotalCount

		// Page through results and collect all names
		var allNames []string
		for offset := 0; offset < total; offset += 10 {
			page, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Offset: offset, Limit: intP(10)})
			require.NoError(t, err)
			assert.Equal(t, total, page.TotalCount, "total count should be stable across pages")
			for _, item := range page.Items {
				allNames = append(allNames, item.Name)
			}
		}
		assert.Equal(t, total, len(allNames), "paginating through all pages should yield total count items")
	})

	t.Run("Sort_RefCount_Real_Data", func(t *testing.T) {
		result, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByRefCount, Order: Desc}, Pagination{Limit: intP(10)})
		require.NoError(t, err)
		// Results should be in descending ref count order
		for i := 1; i < len(result.Items); i++ {
			assert.GreaterOrEqual(t, result.Items[i-1].RefCount, result.Items[i].RefCount,
				"ref count should be descending: %s(%d) >= %s(%d)",
				result.Items[i-1].Name, result.Items[i-1].RefCount,
				result.Items[i].Name, result.Items[i].RefCount)
		}
	})
}

// =============================================================================
// Incremental update discovery tests
// =============================================================================

// TestDiscovery_Incremental tests that the discovery API returns correct results
// after files are added, modified, and have symbols removed, using the full
// pipeline: IndexFiles → Resolve → discovery queries.
func TestDiscovery_Incremental(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	dir := t.TempDir()
	q := e.Query()

	// --- Phase 1: Initial state — two files, cross-file reference ---

	libPath := filepath.Join(dir, "lib.go")
	require.NoError(t, os.WriteFile(libPath, []byte(`package main

type Server struct {
	Host string
	Port int
}

func NewServer() *Server {
	return &Server{Host: "localhost", Port: 8080}
}

func (s *Server) Start() error {
	return nil
}
`), 0644))

	mainPath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(mainPath, []byte(`package main

func main() {
	s := NewServer()
	s.Start()
}
`), 0644))

	require.NoError(t, e.IndexFiles(ctx, []string{libPath, mainPath}))
	require.NoError(t, e.Resolve(ctx))

	t.Run("Phase1_InitialState", func(t *testing.T) {
		// Files: 2 files
		files, err := q.Files("", "", Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Equal(t, 2, files.TotalCount)

		// Symbols: should find Server, NewServer, Start, main
		structs, err := q.Symbols(SymbolFilter{Kinds: []string{"struct"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(structs.Items, "Server"))

		funcs, err := q.Symbols(SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(funcs.Items, "NewServer"))
		assert.True(t, containsSymbolNamed(funcs.Items, "main"))

		// Search for *Server* — should find Server, NewServer
		search, err := q.SearchSymbols("*Server*", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(search.Items, "Server"))
		assert.True(t, containsSymbolNamed(search.Items, "NewServer"))

		// ProjectSummary
		summary, err := q.ProjectSummary(10)
		require.NoError(t, err)
		assert.Equal(t, 2, summary.Languages[0].FileCount)

		// NewServer should have ref count > 0 (called from main)
		serverSearch, err := q.SearchSymbols("NewServer", SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		require.Len(t, serverSearch.Items, 1)
		assert.Greater(t, serverSearch.Items[0].RefCount, 0,
			"NewServer should have references from main.go")
	})

	// --- Phase 2: Add a new file ---

	utilPath := filepath.Join(dir, "util.go")
	require.NoError(t, os.WriteFile(utilPath, []byte(`package main

func FormatAddr(host string, port int) string {
	return host + ":" + string(rune(port))
}

func Validate(s *Server) bool {
	return s.Host != "" && s.Port > 0
}
`), 0644))

	require.NoError(t, e.IndexFiles(ctx, []string{utilPath}))
	require.NoError(t, e.Resolve(ctx))

	t.Run("Phase2_AddFile", func(t *testing.T) {
		// Files: now 3
		files, err := q.Files("", "", Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Equal(t, 3, files.TotalCount)

		// New functions should be discoverable
		funcs, err := q.Symbols(SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{Limit: intP(100)})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(funcs.Items, "FormatAddr"), "new function should appear")
		assert.True(t, containsSymbolNamed(funcs.Items, "Validate"), "new function should appear")

		// Old symbols still present
		assert.True(t, containsSymbolNamed(funcs.Items, "NewServer"), "old function should persist")
		assert.True(t, containsSymbolNamed(funcs.Items, "main"), "old function should persist")

		// Search finds new function
		search, err := q.SearchSymbols("Format*", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(search.Items, "FormatAddr"))

		// ProjectSummary reflects 3 files
		summary, err := q.ProjectSummary(10)
		require.NoError(t, err)
		assert.Equal(t, 3, summary.Languages[0].FileCount)
	})

	// --- Phase 3: Modify a file — rename function, add struct ---

	require.NoError(t, os.WriteFile(libPath, []byte(`package main

type Server struct {
	Host string
	Port int
}

type Config struct {
	Debug bool
}

func CreateServer() *Server {
	return &Server{Host: "localhost", Port: 8080}
}

func (s *Server) Start() error {
	return nil
}
`), 0644))

	require.NoError(t, e.IndexFiles(ctx, []string{libPath}))
	require.NoError(t, e.Resolve(ctx))

	t.Run("Phase3_ModifyFile", func(t *testing.T) {
		// NewServer should be gone, CreateServer should exist
		search, err := q.SearchSymbols("NewServer", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.False(t, containsSymbolNamed(search.Items, "NewServer"),
			"renamed function should be gone")

		search, err = q.SearchSymbols("CreateServer", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(search.Items, "CreateServer"),
			"new function name should appear")

		// New struct Config should be discoverable
		structs, err := q.Symbols(SymbolFilter{Kinds: []string{"struct"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(structs.Items, "Config"), "new struct should appear")
		assert.True(t, containsSymbolNamed(structs.Items, "Server"), "old struct should persist")

		// ProjectSummary struct count should increase
		summary, err := q.ProjectSummary(10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, summary.Languages[0].KindCounts["struct"], 2,
			"should have at least Server and Config")

		// main.go still calls NewServer() which no longer exists —
		// after resolution, NewServer ref from main.go should not resolve
		// (CreateServer won't have refs from main.go since main.go still says NewServer)
		createSearch, err := q.SearchSymbols("CreateServer", SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		require.Len(t, createSearch.Items, 1)
		assert.Equal(t, 0, createSearch.Items[0].RefCount,
			"CreateServer should have 0 refs since main.go still calls NewServer")
	})

	// --- Phase 4: Update caller to match renamed function ---

	require.NoError(t, os.WriteFile(mainPath, []byte(`package main

func main() {
	s := CreateServer()
	s.Start()
	FormatAddr(s.Host, s.Port)
}
`), 0644))

	require.NoError(t, e.IndexFiles(ctx, []string{mainPath}))
	require.NoError(t, e.Resolve(ctx))

	t.Run("Phase4_UpdateCaller", func(t *testing.T) {
		// CreateServer should now have refs
		search, err := q.SearchSymbols("CreateServer", SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		require.Len(t, search.Items, 1)
		assert.Greater(t, search.Items[0].RefCount, 0,
			"CreateServer should now have references from updated main.go")

		// FormatAddr should also have refs now
		search, err = q.SearchSymbols("FormatAddr", SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{})
		require.NoError(t, err)
		require.Len(t, search.Items, 1)
		assert.Greater(t, search.Items[0].RefCount, 0,
			"FormatAddr should have references from main.go")

		// Top symbols should include referenced functions
		summary, err := q.ProjectSummary(20)
		require.NoError(t, err)
		assert.Greater(t, len(summary.TopSymbols), 0, "should have symbols with refs")
	})

	// --- Phase 5: Remove symbols from a file ---

	require.NoError(t, os.WriteFile(utilPath, []byte(`package main

func FormatAddr(host string, port int) string {
	return host + ":" + string(rune(port))
}
`), 0644))

	require.NoError(t, e.IndexFiles(ctx, []string{utilPath}))
	require.NoError(t, e.Resolve(ctx))

	t.Run("Phase5_RemoveSymbols", func(t *testing.T) {
		// Validate should be gone
		search, err := q.SearchSymbols("Validate", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.False(t, containsSymbolNamed(search.Items, "Validate"),
			"removed function should be gone")

		// FormatAddr should still exist
		search, err = q.SearchSymbols("FormatAddr", SymbolFilter{}, Sort{}, Pagination{})
		require.NoError(t, err)
		assert.True(t, containsSymbolNamed(search.Items, "FormatAddr"),
			"kept function should persist")

		// Total function count should have decreased
		funcs, err := q.Symbols(SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{Limit: intP(100)})
		require.NoError(t, err)
		assert.False(t, containsSymbolNamed(funcs.Items, "Validate"))
	})

	// --- Phase 6: Final consistency check ---

	t.Run("Phase6_FinalConsistency", func(t *testing.T) {
		// Files still 3
		files, err := q.Files("", "", Sort{}, Pagination{})
		require.NoError(t, err)
		assert.Equal(t, 3, files.TotalCount)

		// Pagination should be consistent with total
		all, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Limit: intP(500)})
		require.NoError(t, err)
		total := all.TotalCount

		var collected int
		for offset := 0; offset < total; offset += 5 {
			page, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Offset: offset, Limit: intP(5)})
			require.NoError(t, err)
			assert.Equal(t, total, page.TotalCount)
			collected += len(page.Items)
		}
		assert.Equal(t, total, collected, "pagination should cover all symbols")

		// Sort by ref count should still work
		sorted, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByRefCount, Order: Desc}, Pagination{Limit: intP(10)})
		require.NoError(t, err)
		for i := 1; i < len(sorted.Items); i++ {
			assert.GreaterOrEqual(t, sorted.Items[i-1].RefCount, sorted.Items[i].RefCount)
		}
	})
}

// TestIntegration_IndexDirectory_RemovesStaleFiles verifies that IndexDirectory
// cleans up database records for files that have been deleted from disk.
func TestIntegration_IndexDirectory_RemovesStaleFiles(t *testing.T) {
	e := newIntegrationEngine(t, WithLanguages("go"))
	ctx := context.Background()
	root := t.TempDir()

	// Create two Go files and index them.
	mainPath := writeGoFile(t, root, "main.go", `package main

func main() { helper() }
`)
	helperPath := writeGoFile(t, root, "helper.go", `package main

func helper() string { return "hi" }
`)

	require.NoError(t, e.IndexDirectory(ctx, root))
	require.NoError(t, e.Resolve(ctx))

	// Both files should be indexed.
	f1, err := e.store.FileByPath(mainPath)
	require.NoError(t, err)
	require.NotNil(t, f1, "main.go should be indexed")

	f2, err := e.store.FileByPath(helperPath)
	require.NoError(t, err)
	require.NotNil(t, f2, "helper.go should be indexed")

	// helper() symbol should exist.
	syms, err := e.store.SymbolsByName("helper")
	require.NoError(t, err)
	require.NotEmpty(t, syms, "helper symbol should exist")

	// Delete helper.go from disk.
	require.NoError(t, os.Remove(helperPath))

	// Re-index the directory.
	require.NoError(t, e.IndexDirectory(ctx, root))

	// helper.go should no longer be in the database.
	f2, err = e.store.FileByPath(helperPath)
	require.NoError(t, err)
	assert.Nil(t, f2, "helper.go should be removed from DB after deletion")

	// helper() symbol should be gone.
	syms, err = e.store.SymbolsByName("helper")
	require.NoError(t, err)
	assert.Empty(t, syms, "helper symbol should be cleaned up")

	// main.go should still be indexed.
	f1, err = e.store.FileByPath(mainPath)
	require.NoError(t, err)
	require.NotNil(t, f1, "main.go should still be indexed")
}
