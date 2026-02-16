package canopy

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestQueryBuilder(t *testing.T) (*QueryBuilder, *store.Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())
	t.Cleanup(func() { s.Close() })
	return &QueryBuilder{store: s}, s
}

func TestSymbolAt_ReturnsCorrectSymbol(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	// Outer symbol: a struct spanning lines 0-19.
	outerID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "MyStruct", Kind: "struct", Visibility: "public",
		StartLine: 0, StartCol: 0, EndLine: 19, EndCol: 1,
	})
	require.NoError(t, err)

	// Inner symbol: a method inside the struct spanning lines 4-9.
	innerID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "DoWork", Kind: "method", Visibility: "public",
		StartLine: 4, StartCol: 1, EndLine: 9, EndCol: 1,
		ParentSymbolID: &outerID,
	})
	require.NoError(t, err)

	// Query a position inside the inner symbol -- should return the narrowest match.
	sym, err := q.SymbolAt("/test.go", 6, 5)
	require.NoError(t, err)
	require.NotNil(t, sym)
	assert.Equal(t, innerID, sym.ID)
	assert.Equal(t, "DoWork", sym.Name)
	assert.Equal(t, "method", sym.Kind)

	// Query a position inside the outer symbol but outside the inner one.
	sym, err = q.SymbolAt("/test.go", 14, 0)
	require.NoError(t, err)
	require.NotNil(t, sym)
	assert.Equal(t, outerID, sym.ID)
	assert.Equal(t, "MyStruct", sym.Name)
	assert.Equal(t, "struct", sym.Kind)
}

func TestSymbolAt_NoSymbol(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	_, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	// File exists but no symbols at this location.
	sym, err := q.SymbolAt("/test.go", 49, 0)
	require.NoError(t, err)
	assert.Nil(t, sym)
}

func TestSymbolAt_NoFile(t *testing.T) {
	q, _ := newTestQueryBuilder(t)

	// File doesn't exist at all.
	sym, err := q.SymbolAt("/nonexistent.go", 0, 0)
	require.NoError(t, err)
	assert.Nil(t, sym)
}

func TestDefinitionAt_NoFile(t *testing.T) {
	q, _ := newTestQueryBuilder(t)

	locs, err := q.DefinitionAt("/nonexistent.go", 0, 0)
	require.NoError(t, err)
	assert.Empty(t, locs)
}

func TestDefinitionAt_WithResolvedReference(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	// Set up: file → reference → resolved_reference → target symbol.
	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "Foo", Kind: "function", Visibility: "public",
		StartLine: 4, StartCol: 0, EndLine: 9, EndCol: 1,
	})
	require.NoError(t, err)

	refID, err := s.InsertReference(&store.Reference{
		FileID: fID, Name: "Foo",
		StartLine: 19, StartCol: 5, EndLine: 19, EndCol: 8,
		Context: "call",
	})
	require.NoError(t, err)

	_, err = s.InsertResolvedReference(&store.ResolvedReference{
		ReferenceID: refID, TargetSymbolID: symID, Confidence: 1.0, ResolutionKind: "direct",
	})
	require.NoError(t, err)

	locs, err := q.DefinitionAt("/test.go", 19, 6)
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Equal(t, "/test.go", locs[0].File)
	assert.Equal(t, 4, locs[0].StartLine)
	assert.Equal(t, 0, locs[0].StartCol)
}

func TestDefinitionAt_PositionOutsideReference(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	_, err = s.InsertReference(&store.Reference{
		FileID: fID, Name: "Foo",
		StartLine: 19, StartCol: 5, EndLine: 19, EndCol: 8,
		Context: "call",
	})
	require.NoError(t, err)

	// Position before the reference span.
	locs, err := q.DefinitionAt("/test.go", 19, 2)
	require.NoError(t, err)
	assert.Empty(t, locs)
}

func TestReferencesTo(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "Bar", Kind: "function", Visibility: "public",
		StartLine: 0, StartCol: 0, EndLine: 4, EndCol: 1,
	})
	require.NoError(t, err)

	ref1ID, err := s.InsertReference(&store.Reference{
		FileID: fID, Name: "Bar",
		StartLine: 9, StartCol: 2, EndLine: 9, EndCol: 5,
		Context: "call",
	})
	require.NoError(t, err)

	ref2ID, err := s.InsertReference(&store.Reference{
		FileID: fID, Name: "Bar",
		StartLine: 14, StartCol: 0, EndLine: 14, EndCol: 3,
		Context: "call",
	})
	require.NoError(t, err)

	for _, refID := range []int64{ref1ID, ref2ID} {
		_, err = s.InsertResolvedReference(&store.ResolvedReference{
			ReferenceID: refID, TargetSymbolID: symID, Confidence: 1.0, ResolutionKind: "direct",
		})
		require.NoError(t, err)
	}

	locs, err := q.ReferencesTo(symID)
	require.NoError(t, err)
	assert.Len(t, locs, 2)
}

func TestImplementations(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	ifaceID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "Reader", Kind: "interface", Visibility: "public",
		StartLine: 0, StartCol: 0, EndLine: 4, EndCol: 1,
	})
	require.NoError(t, err)

	typeID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "MyReader", Kind: "struct", Visibility: "public",
		StartLine: 9, StartCol: 0, EndLine: 14, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertImplementation(&store.Implementation{
		TypeSymbolID: typeID, InterfaceSymbolID: ifaceID, Kind: "implicit", FileID: &fID,
	})
	require.NoError(t, err)

	locs, err := q.Implementations(ifaceID)
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Equal(t, "/test.go", locs[0].File)
	assert.Equal(t, 9, locs[0].StartLine)
}

func TestCallers(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	callerID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "main", Kind: "function",
		StartLine: 0, StartCol: 0, EndLine: 9, EndCol: 1,
	})
	require.NoError(t, err)

	calleeID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "helper", Kind: "function",
		StartLine: 11, StartCol: 0, EndLine: 19, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertCallEdge(&store.CallEdge{
		CallerSymbolID: callerID, CalleeSymbolID: calleeID, FileID: &fID, Line: 4, Col: 2,
	})
	require.NoError(t, err)

	edges, err := q.Callers(calleeID)
	require.NoError(t, err)
	require.Len(t, edges, 1)
	assert.Equal(t, callerID, edges[0].CallerSymbolID)
}

func TestCallees(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	callerID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "main", Kind: "function",
		StartLine: 0, StartCol: 0, EndLine: 9, EndCol: 1,
	})
	require.NoError(t, err)

	calleeID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "helper", Kind: "function",
		StartLine: 11, StartCol: 0, EndLine: 19, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertCallEdge(&store.CallEdge{
		CallerSymbolID: callerID, CalleeSymbolID: calleeID, FileID: &fID, Line: 4, Col: 2,
	})
	require.NoError(t, err)

	edges, err := q.Callees(callerID)
	require.NoError(t, err)
	require.Len(t, edges, 1)
	assert.Equal(t, calleeID, edges[0].CalleeSymbolID)
}

func TestDependencies(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/test.go", Language: "go", Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	_, err = s.InsertImport(&store.Import{
		FileID: fID, Source: "fmt", Kind: "module", Scope: "file",
	})
	require.NoError(t, err)

	imports, err := q.Dependencies(fID)
	require.NoError(t, err)
	require.Len(t, imports, 1)
	assert.Equal(t, "fmt", imports[0].Source)
}

func TestDependents(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	f1ID, err := s.InsertFile(&store.File{
		Path: "/a.go", Language: "go", Hash: "a", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	f2ID, err := s.InsertFile(&store.File{
		Path: "/b.go", Language: "go", Hash: "b", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	for _, fID := range []int64{f1ID, f2ID} {
		_, err = s.InsertImport(&store.Import{
			FileID: fID, Source: "mylib/pkg", Kind: "module", Scope: "file",
		})
		require.NoError(t, err)
	}

	imports, err := q.Dependents("mylib/pkg")
	require.NoError(t, err)
	assert.Len(t, imports, 2)

	// No dependents for an unknown source.
	imports, err = q.Dependents("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, imports)
}

func TestDependents_SuffixMatch(t *testing.T) {
	q, s := newTestQueryBuilder(t)

	fID, err := s.InsertFile(&store.File{
		Path: "/a.go", Language: "go", Hash: "a", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	_, err = s.InsertImport(&store.Import{
		FileID: fID, Source: "github.com/example/util", Kind: "module", Scope: "file",
	})
	require.NoError(t, err)

	// Exact match still works.
	imports, err := q.Dependents("github.com/example/util")
	require.NoError(t, err)
	assert.Len(t, imports, 1, "exact match should find the import")

	// Suffix match: just the last path segment.
	imports, err = q.Dependents("util")
	require.NoError(t, err)
	assert.Len(t, imports, 1, "suffix 'util' should match 'github.com/example/util'")

	// Suffix match: partial path.
	imports, err = q.Dependents("example/util")
	require.NoError(t, err)
	assert.Len(t, imports, 1, "suffix 'example/util' should match")

	// No match.
	imports, err = q.Dependents("nope")
	require.NoError(t, err)
	assert.Empty(t, imports)
}
