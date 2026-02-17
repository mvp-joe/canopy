package canopy

import (
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PackageDependencyGraph
// =============================================================================

func TestPackageDependencyGraph_AggregatesImportsToPackageEdges(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	// Two packages: pkg_a (2 files) and pkg_b (1 file).
	fileA1 := insertFile(t, s, "/src/pkg_a/main.go", "go")
	fileA2 := insertFile(t, s, "/src/pkg_a/util.go", "go")
	fileB := insertFile(t, s, "/src/pkg_b/handler.go", "go")

	// Package symbols.
	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA1, Name: "pkg_a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileA2, Name: "pkg_a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileB, Name: "pkg_b", Kind: "package"})
	require.NoError(t, err)

	// Both files in pkg_a import pkg_b.
	_, err = s.InsertImport(&store.Import{FileID: fileA1, Source: "pkg_b", Kind: "import"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fileA2, Source: "pkg_b", Kind: "import"})
	require.NoError(t, err)

	graph, err := q.PackageDependencyGraph()
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Should have 2 packages.
	assert.Len(t, graph.Packages, 2)

	// Should have 1 edge from pkg_a -> pkg_b.
	require.Len(t, graph.Edges, 1)
	assert.Equal(t, "pkg_a", graph.Edges[0].FromPackage)
	assert.Equal(t, "pkg_b", graph.Edges[0].ToPackage)
}

func TestPackageDependencyGraph_ImportCountReflectsDistinctFileImports(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	fileA1 := insertFile(t, s, "/src/a/one.go", "go")
	fileA2 := insertFile(t, s, "/src/a/two.go", "go")
	fileA3 := insertFile(t, s, "/src/a/three.go", "go")
	fileB := insertFile(t, s, "/src/b/main.go", "go")

	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA1, Name: "a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileA2, Name: "a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileA3, Name: "a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileB, Name: "b", Kind: "package"})
	require.NoError(t, err)

	// 3 files in package "a" each import "b".
	_, err = s.InsertImport(&store.Import{FileID: fileA1, Source: "b", Kind: "import"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fileA2, Source: "b", Kind: "import"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fileA3, Source: "b", Kind: "import"})
	require.NoError(t, err)

	graph, err := q.PackageDependencyGraph()
	require.NoError(t, err)

	require.Len(t, graph.Edges, 1)
	assert.Equal(t, 3, graph.Edges[0].ImportCount)
}

func TestPackageDependencyGraph_EmptyDatabase(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	graph, err := q.PackageDependencyGraph()
	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Empty(t, graph.Packages)
	assert.Empty(t, graph.Edges)
}

func TestPackageDependencyGraph_ExcludesExternalImports(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	fileA := insertFile(t, s, "/src/myapp/main.go", "go")

	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA, Name: "myapp", Kind: "package"})
	require.NoError(t, err)

	// Import an external package that isn't indexed.
	_, err = s.InsertImport(&store.Import{FileID: fileA, Source: "github.com/external/lib", Kind: "import"})
	require.NoError(t, err)

	graph, err := q.PackageDependencyGraph()
	require.NoError(t, err)

	// Should have 1 package but 0 edges (external import excluded).
	assert.Len(t, graph.Packages, 1)
	assert.Equal(t, "myapp", graph.Packages[0].Name)
	assert.Empty(t, graph.Edges)
}

func TestPackageDependencyGraph_FileCountAndLineCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	fileA1 := insertFileWithLines(t, s, "/src/pkg_a/main.go", "go", 100)
	fileA2 := insertFileWithLines(t, s, "/src/pkg_a/util.go", "go", 50)
	fileB := insertFileWithLines(t, s, "/src/pkg_b/handler.go", "go", 200)

	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA1, Name: "pkg_a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileA2, Name: "pkg_a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileB, Name: "pkg_b", Kind: "package"})
	require.NoError(t, err)

	graph, err := q.PackageDependencyGraph()
	require.NoError(t, err)

	// Find packages by name.
	pkgMap := map[string]PackageNode{}
	for _, p := range graph.Packages {
		pkgMap[p.Name] = p
	}

	assert.Equal(t, 2, pkgMap["pkg_a"].FileCount)
	assert.Equal(t, 150, pkgMap["pkg_a"].LineCount)
	assert.Equal(t, 1, pkgMap["pkg_b"].FileCount)
	assert.Equal(t, 200, pkgMap["pkg_b"].LineCount)
}

// =============================================================================
// CircularDependencies
// =============================================================================

func TestCircularDependencies_AcyclicReturnsEmpty(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	fileA := insertFile(t, s, "/src/a/main.go", "go")
	fileB := insertFile(t, s, "/src/b/main.go", "go")
	fileC := insertFile(t, s, "/src/c/main.go", "go")

	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA, Name: "a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileB, Name: "b", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileC, Name: "c", Kind: "package"})
	require.NoError(t, err)

	// Linear chain: a -> b -> c (no cycle).
	_, err = s.InsertImport(&store.Import{FileID: fileA, Source: "b", Kind: "import"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fileB, Source: "c", Kind: "import"})
	require.NoError(t, err)

	cycles, err := q.CircularDependencies()
	require.NoError(t, err)
	assert.Empty(t, cycles)
}

func TestCircularDependencies_SimpleABCycle(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	fileA := insertFile(t, s, "/src/a/main.go", "go")
	fileB := insertFile(t, s, "/src/b/main.go", "go")

	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA, Name: "a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileB, Name: "b", Kind: "package"})
	require.NoError(t, err)

	// a -> b and b -> a.
	_, err = s.InsertImport(&store.Import{FileID: fileA, Source: "b", Kind: "import"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fileB, Source: "a", Kind: "import"})
	require.NoError(t, err)

	cycles, err := q.CircularDependencies()
	require.NoError(t, err)
	require.Len(t, cycles, 1)

	// Cycle should contain both a and b, plus repeat of first element.
	cycle := cycles[0]
	assert.Len(t, cycle, 3, "cycle should have 3 elements (2 packages + repeated first)")
	assert.Equal(t, cycle[0], cycle[len(cycle)-1], "first element should be repeated at end")
	// Both packages should be in the cycle.
	assert.Contains(t, cycle, "a")
	assert.Contains(t, cycle, "b")
}

func TestCircularDependencies_LongerABCACycle(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	fileA := insertFile(t, s, "/src/a/main.go", "go")
	fileB := insertFile(t, s, "/src/b/main.go", "go")
	fileC := insertFile(t, s, "/src/c/main.go", "go")

	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA, Name: "a", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileB, Name: "b", Kind: "package"})
	require.NoError(t, err)
	_, err = s.InsertSymbol(&store.Symbol{FileID: &fileC, Name: "c", Kind: "package"})
	require.NoError(t, err)

	// a -> b -> c -> a (ring).
	_, err = s.InsertImport(&store.Import{FileID: fileA, Source: "b", Kind: "import"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fileB, Source: "c", Kind: "import"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fileC, Source: "a", Kind: "import"})
	require.NoError(t, err)

	cycles, err := q.CircularDependencies()
	require.NoError(t, err)
	require.Len(t, cycles, 1)

	cycle := cycles[0]
	assert.Len(t, cycle, 4, "cycle should have 4 elements (3 packages + repeated first)")
	assert.Equal(t, cycle[0], cycle[len(cycle)-1], "first element should be repeated at end")
	assert.Contains(t, cycle, "a")
	assert.Contains(t, cycle, "b")
	assert.Contains(t, cycle, "c")
}

func TestCircularDependencies_SelfLoop(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	fileA := insertFile(t, s, "/src/a/main.go", "go")

	_, err := s.InsertSymbol(&store.Symbol{FileID: &fileA, Name: "a", Kind: "package"})
	require.NoError(t, err)

	// a imports itself.
	_, err = s.InsertImport(&store.Import{FileID: fileA, Source: "a", Kind: "import"})
	require.NoError(t, err)

	cycles, err := q.CircularDependencies()
	require.NoError(t, err)
	require.Len(t, cycles, 1)

	cycle := cycles[0]
	assert.Len(t, cycle, 2, "self-loop should have 2 elements (package + repeat)")
	assert.Equal(t, "a", cycle[0])
	assert.Equal(t, "a", cycle[1])
}

func TestCircularDependencies_EmptyDatabase(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	cycles, err := q.CircularDependencies()
	require.NoError(t, err)
	assert.Empty(t, cycles)
}

// =============================================================================
// Test helpers
// =============================================================================

func insertFileWithLines(t *testing.T, s *store.Store, path, lang string, lineCount int) int64 {
	t.Helper()
	id, err := s.InsertFile(&store.File{
		Path: path, Language: lang, Hash: "h", LineCount: lineCount,
	})
	require.NoError(t, err)
	return id
}
