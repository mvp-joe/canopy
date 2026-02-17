package canopy

import (
	"fmt"
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TransitiveCallers
// =============================================================================

func TestTransitiveCallers_Depth1MatchesDirectCallers(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)
	c := insertSymbol(t, s, &fID, "C", "function", "public", nil)

	// A calls C, B calls C
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: c, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: b, CalleeSymbolID: c, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallers(c, 1)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Equal(t, c, graph.Root)
	// Root + 2 callers
	assert.Len(t, graph.Nodes, 3)
	assert.Len(t, graph.Edges, 2)
	assert.Equal(t, 1, graph.Depth)

	// Verify the callers are A and B
	callerNames := make(map[string]bool)
	for _, n := range graph.Nodes {
		if n.Depth == 1 {
			callerNames[n.Symbol.Name] = true
		}
	}
	assert.True(t, callerNames["A"])
	assert.True(t, callerNames["B"])
}

func TestTransitiveCallers_Depth3FollowsMultiHopChains(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)
	c := insertSymbol(t, s, &fID, "C", "function", "public", nil)

	// A -> B -> C
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: b, CalleeSymbolID: c, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallers(c, 3)
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Root (C) + B (depth 1) + A (depth 2)
	assert.Len(t, graph.Nodes, 3)
	assert.Equal(t, 2, graph.Depth)

	// Verify depths
	depthByName := make(map[string]int)
	for _, n := range graph.Nodes {
		depthByName[n.Symbol.Name] = n.Depth
	}
	assert.Equal(t, 0, depthByName["C"])
	assert.Equal(t, 1, depthByName["B"])
	assert.Equal(t, 2, depthByName["A"])

	// Both edges should be present
	assert.Len(t, graph.Edges, 2)
}

func TestTransitiveCallers_Depth0ReturnsRootOnly(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)

	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallers(b, 0)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 1)
	assert.Equal(t, "B", graph.Nodes[0].Symbol.Name)
	assert.Equal(t, 0, graph.Nodes[0].Depth)
	assert.Empty(t, graph.Edges)
	assert.Equal(t, 0, graph.Depth)
}

func TestTransitiveCallers_HandlesCyclesWithoutInfiniteLoop(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)

	// A calls B, B calls A (cycle)
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: b, CalleeSymbolID: a, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallers(a, 10)
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Both nodes appear exactly once
	assert.Len(t, graph.Nodes, 2)
	nameSet := make(map[string]bool)
	for _, n := range graph.Nodes {
		nameSet[n.Symbol.Name] = true
	}
	assert.True(t, nameSet["A"])
	assert.True(t, nameSet["B"])

	// Both edges present
	assert.Len(t, graph.Edges, 2)
}

func TestTransitiveCallers_DepthExceedingGraphReturnsFullSet(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)
	c := insertSymbol(t, s, &fID, "C", "function", "public", nil)

	// A -> B -> C
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: b, CalleeSymbolID: c, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	// maxDepth=50 but graph only has depth 2
	graph, err := q.TransitiveCallers(c, 50)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 3)
	assert.Equal(t, 2, graph.Depth)
}

func TestTransitiveCallers_NoCallersReturnsRootOnly(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)

	graph, err := q.TransitiveCallers(a, 5)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 1)
	assert.Equal(t, "A", graph.Nodes[0].Symbol.Name)
	assert.Equal(t, 0, graph.Nodes[0].Depth)
	assert.Empty(t, graph.Edges)
	assert.Equal(t, 0, graph.Depth)
}

func TestTransitiveCallers_NonExistentSymbolReturnsNil(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	graph, err := q.TransitiveCallers(99999, 5)
	require.NoError(t, err)
	assert.Nil(t, graph)
}

func TestTransitiveCallers_NegativeDepthReturnsError(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	graph, err := q.TransitiveCallers(1, -1)
	require.Error(t, err)
	assert.Nil(t, graph)
}

func TestTransitiveCallers_MaxDepthCappedAt100(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/src/main.go", "go")
	sym := insertSymbol(t, s, &fID, "A", "function", "public", nil)

	// maxDepth > 100 should be silently capped, not error
	graph, err := q.TransitiveCallers(sym, 200)
	require.NoError(t, err)
	require.NotNil(t, graph)
	assert.Equal(t, sym, graph.Root)
}

func TestTransitiveCallers_EdgeFilePathResolved(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/src/main.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)

	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 5, Col: 3})
	require.NoError(t, err)

	graph, err := q.TransitiveCallers(b, 1)
	require.NoError(t, err)
	require.NotNil(t, graph)

	require.Len(t, graph.Edges, 1)
	assert.Equal(t, "/src/main.go", graph.Edges[0].File)
	assert.Equal(t, 5, graph.Edges[0].Line)
	assert.Equal(t, 3, graph.Edges[0].Col)
}

func TestTransitiveCallers_NilFileIDEdge(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)

	// Edge with nil FileID
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: nil, Line: 1, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallers(b, 1)
	require.NoError(t, err)
	require.NotNil(t, graph)

	require.Len(t, graph.Edges, 1)
	assert.Equal(t, "", graph.Edges[0].File)
}

// =============================================================================
// TransitiveCallees
// =============================================================================

func TestTransitiveCallees_Depth1MatchesDirectCallees(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)
	c := insertSymbol(t, s, &fID, "C", "function", "public", nil)

	// A calls B, A calls C
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: c, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallees(a, 1)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Equal(t, a, graph.Root)
	assert.Len(t, graph.Nodes, 3)
	assert.Len(t, graph.Edges, 2)
	assert.Equal(t, 1, graph.Depth)

	calleeNames := make(map[string]bool)
	for _, n := range graph.Nodes {
		if n.Depth == 1 {
			calleeNames[n.Symbol.Name] = true
		}
	}
	assert.True(t, calleeNames["B"])
	assert.True(t, calleeNames["C"])
}

func TestTransitiveCallees_Depth3FollowsChains(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)
	c := insertSymbol(t, s, &fID, "C", "function", "public", nil)

	// A -> B -> C
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: b, CalleeSymbolID: c, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallees(a, 3)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 3)
	assert.Equal(t, 2, graph.Depth)

	depthByName := make(map[string]int)
	for _, n := range graph.Nodes {
		depthByName[n.Symbol.Name] = n.Depth
	}
	assert.Equal(t, 0, depthByName["A"])
	assert.Equal(t, 1, depthByName["B"])
	assert.Equal(t, 2, depthByName["C"])

	assert.Len(t, graph.Edges, 2)
}

func TestTransitiveCallees_HandlesCyclesWithoutInfiniteLoop(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)

	// A calls B, B calls A
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: b, CalleeSymbolID: a, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallees(a, 10)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 2)
	nameSet := make(map[string]bool)
	for _, n := range graph.Nodes {
		nameSet[n.Symbol.Name] = true
	}
	assert.True(t, nameSet["A"])
	assert.True(t, nameSet["B"])

	assert.Len(t, graph.Edges, 2)
}

func TestTransitiveCallees_Depth0ReturnsRootOnly(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	a := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	b := insertSymbol(t, s, &fID, "B", "function", "public", nil)

	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: a, CalleeSymbolID: b, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)

	graph, err := q.TransitiveCallees(a, 0)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 1)
	assert.Equal(t, "A", graph.Nodes[0].Symbol.Name)
	assert.Equal(t, 0, graph.Nodes[0].Depth)
	assert.Empty(t, graph.Edges)
	assert.Equal(t, 0, graph.Depth)
}

func TestTransitiveCallees_LeafFunctionReturnsRootOnly(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	leaf := insertSymbol(t, s, &fID, "leaf", "function", "public", nil)

	graph, err := q.TransitiveCallees(leaf, 5)
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.Len(t, graph.Nodes, 1)
	assert.Equal(t, "leaf", graph.Nodes[0].Symbol.Name)
	assert.Empty(t, graph.Edges)
	assert.Equal(t, 0, graph.Depth)
}

func TestTransitiveCallees_NonExistentSymbolReturnsNil(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	graph, err := q.TransitiveCallees(99999, 5)
	require.NoError(t, err)
	assert.Nil(t, graph)
}

func TestTransitiveCallees_NegativeDepthReturnsError(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	graph, err := q.TransitiveCallees(1, -1)
	require.Error(t, err)
	assert.Nil(t, graph)
}

// =============================================================================
// UnusedSymbols
// =============================================================================

func TestUnusedSymbols_ReturnsZeroRefSymbols(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "Referenced", "function", "public", nil)
	insertSymbol(t, s, &fID, "Unused", "function", "public", nil)

	// Add a reference to sym1 only
	insertResolvedRef(t, s, fID, sym1)

	result, err := q.UnusedSymbols(SymbolFilter{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Unused", result.Items[0].Name)
	assert.Equal(t, 0, result.Items[0].RefCount)
}

func TestUnusedSymbols_ExcludesPackageModuleNamespace(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "mypkg", "package", "public", nil)
	insertSymbol(t, s, &fID, "mymod", "module", "public", nil)
	insertSymbol(t, s, &fID, "myns", "namespace", "public", nil)
	insertSymbol(t, s, &fID, "UnusedFunc", "function", "public", nil)

	result, err := q.UnusedSymbols(SymbolFilter{}, Pagination{})
	require.NoError(t, err)
	// Only UnusedFunc should appear, package/module/namespace excluded
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "UnusedFunc", result.Items[0].Name)
}

func TestUnusedSymbols_FilterByKind(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "UnusedFunc", "function", "public", nil)
	insertSymbol(t, s, &fID, "UnusedStruct", "struct", "public", nil)

	result, err := q.UnusedSymbols(SymbolFilter{Kinds: []string{"function"}}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "UnusedFunc", result.Items[0].Name)
}

func TestUnusedSymbols_FilterByVisibility(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "PublicUnused", "function", "public", nil)
	insertSymbol(t, s, &fID, "privateUnused", "function", "private", nil)

	result, err := q.UnusedSymbols(SymbolFilter{Visibility: strPtr("private")}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "privateUnused", result.Items[0].Name)
}

func TestUnusedSymbols_FilterByPathPrefix(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "internal/store/store.go", "go")
	fID2 := insertFile(t, s, "internal/runtime/runtime.go", "go")
	insertSymbol(t, s, &fID1, "StoreHelper", "function", "public", nil)
	insertSymbol(t, s, &fID2, "RuntimeHelper", "function", "public", nil)

	result, err := q.UnusedSymbols(SymbolFilter{PathPrefix: strPtr("internal/store")}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "StoreHelper", result.Items[0].Name)
}

func TestUnusedSymbols_Pagination(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	for i := range 5 {
		insertSymbol(t, s, &fID, fmt.Sprintf("Unused%d", i), "function", "public", nil)
	}

	// First page
	result, err := q.UnusedSymbols(SymbolFilter{}, Pagination{Offset: 0, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 2)

	// Second page
	result, err = q.UnusedSymbols(SymbolFilter{}, Pagination{Offset: 2, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 2)

	// Last page (partial)
	result, err = q.UnusedSymbols(SymbolFilter{}, Pagination{Offset: 4, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 1)
}

func TestUnusedSymbols_TotalCountReflectsFilter(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "UnusedFunc1", "function", "public", nil)
	insertSymbol(t, s, &fID, "UnusedFunc2", "function", "public", nil)
	insertSymbol(t, s, &fID, "UnusedStruct", "struct", "public", nil)

	// Without kind filter: 3 total unused
	all, err := q.UnusedSymbols(SymbolFilter{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 3, all.TotalCount)

	// With kind filter: TotalCount should be 2 (only functions)
	filtered, err := q.UnusedSymbols(SymbolFilter{Kinds: []string{"function"}}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, filtered.TotalCount)
	assert.Len(t, filtered.Items, 2)
}

// =============================================================================
// Hotspots
// =============================================================================

func TestHotspots_TopNByExternalRefCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "main.go", "go")
	fID2 := insertFile(t, s, "other.go", "go")

	sym1 := insertSymbol(t, s, &fID1, "Popular", "function", "public", nil)
	sym2 := insertSymbol(t, s, &fID1, "Medium", "function", "public", nil)
	sym3 := insertSymbol(t, s, &fID1, "Low", "function", "public", nil)

	// Add external refs (from other.go) in varying amounts
	for range 5 {
		insertResolvedRef(t, s, fID2, sym1)
	}
	for range 3 {
		insertResolvedRef(t, s, fID2, sym2)
	}
	insertResolvedRef(t, s, fID2, sym3)

	result, err := q.Hotspots(2)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "Popular", result[0].Symbol.Name)
	assert.Equal(t, 5, result[0].Symbol.ExternalRefCount)
	assert.Equal(t, "Medium", result[1].Symbol.Name)
	assert.Equal(t, 3, result[1].Symbol.ExternalRefCount)
}

func TestHotspots_IncludesCallerCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")

	target := insertSymbol(t, s, &fID, "Target", "function", "public", nil)
	caller1 := insertSymbol(t, s, &fID, "Caller1", "function", "public", nil)
	caller2 := insertSymbol(t, s, &fID, "Caller2", "function", "public", nil)

	// Add a reference so Target shows up as a hotspot
	insertResolvedRef(t, s, fID, target)

	// Add call edges: two callers call Target
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: caller1, CalleeSymbolID: target, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: caller2, CalleeSymbolID: target, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	result, err := q.Hotspots(10)
	require.NoError(t, err)

	// Find Target in results
	var targetResult *HotspotResult
	for _, r := range result {
		if r.Symbol.Name == "Target" {
			targetResult = r
			break
		}
	}
	require.NotNil(t, targetResult)
	assert.Equal(t, 2, targetResult.CallerCount)
}

func TestHotspots_IncludesCalleeCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")

	caller := insertSymbol(t, s, &fID, "Caller", "function", "public", nil)
	callee1 := insertSymbol(t, s, &fID, "Callee1", "function", "public", nil)
	callee2 := insertSymbol(t, s, &fID, "Callee2", "function", "public", nil)

	// Add a reference so Caller shows up as a hotspot
	insertResolvedRef(t, s, fID, caller)

	// Caller calls two functions
	_, err := s.InsertCallEdge(&store.CallEdge{CallerSymbolID: caller, CalleeSymbolID: callee1, FileID: &fID, Line: 1, Col: 0})
	require.NoError(t, err)
	_, err = s.InsertCallEdge(&store.CallEdge{CallerSymbolID: caller, CalleeSymbolID: callee2, FileID: &fID, Line: 2, Col: 0})
	require.NoError(t, err)

	result, err := q.Hotspots(10)
	require.NoError(t, err)

	var callerResult *HotspotResult
	for _, r := range result {
		if r.Symbol.Name == "Caller" {
			callerResult = r
			break
		}
	}
	require.NotNil(t, callerResult)
	assert.Equal(t, 2, callerResult.CalleeCount)
}

func TestHotspots_TopNLargerThanTotal(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym := insertSymbol(t, s, &fID, "Only", "function", "public", nil)
	insertResolvedRef(t, s, fID, sym)

	result, err := q.Hotspots(100)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Only", result[0].Symbol.Name)
}

func TestHotspots_TopNZeroReturnsEmpty(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym := insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertResolvedRef(t, s, fID, sym)

	result, err := q.Hotspots(0)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestHotspots_ExcludesZeroRefSymbols(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym := insertSymbol(t, s, &fID, "Referenced", "function", "public", nil)
	insertSymbol(t, s, &fID, "Unreferenced", "function", "public", nil)

	insertResolvedRef(t, s, fID, sym)

	result, err := q.Hotspots(10)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Referenced", result[0].Symbol.Name)
}

func TestHotspots_NegativeTopNReturnsError(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	result, err := q.Hotspots(-1)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "topN must be non-negative")
}
