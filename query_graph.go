package canopy

import (
	"fmt"
	"strings"

	"github.com/jward/canopy/internal/store"
)

// CallGraph represents a transitive call graph rooted at a symbol.
// Nodes and edges are bulk-loaded then traversed with BFS -- no recursive
// SQL or N+1 queries.
type CallGraph struct {
	Root  int64           // starting symbol ID
	Nodes []CallGraphNode // all symbols reachable within depth
	Edges []CallGraphEdge // all edges in the subgraph
	Depth int             // actual max depth reached (may be < maxDepth if graph is shallow)
}

// CallGraphNode is a symbol in the call graph with its distance from the root.
type CallGraphNode struct {
	Symbol SymbolResult
	Depth  int // BFS depth from root (0 = root itself)
}

// CallGraphEdge is a single caller-callee relationship in the call graph.
type CallGraphEdge struct {
	CallerID int64
	CalleeID int64
	File     string
	Line     int
	Col      int
}

// callGraphData holds the bulk-loaded call graph adjacency maps and file path index.
type callGraphData struct {
	forward   map[int64][]int64            // caller -> callees
	reverse   map[int64][]int64            // callee -> callers
	edgesByCaller map[int64][]*CallEdge // edges keyed by caller
	edgesByCallee map[int64][]*CallEdge // edges keyed by callee
	filePaths map[int64]string              // file ID -> path
}

// buildCallGraph bulk-loads all call edges and files into memory and builds
// forward/reverse adjacency maps. This avoids N+1 queries during BFS traversal.
func (q *QueryBuilder) buildCallGraph() (*callGraphData, error) {
	edges, err := q.store.AllCallEdges()
	if err != nil {
		return nil, fmt.Errorf("build call graph: load edges: %w", err)
	}

	filePaths, err := q.store.AllFiles()
	if err != nil {
		return nil, fmt.Errorf("build call graph: load files: %w", err)
	}

	data := &callGraphData{
		forward:       make(map[int64][]int64),
		reverse:       make(map[int64][]int64),
		edgesByCaller: make(map[int64][]*CallEdge),
		edgesByCallee: make(map[int64][]*CallEdge),
		filePaths:     filePaths,
	}

	for _, e := range edges {
		data.forward[e.CallerSymbolID] = append(data.forward[e.CallerSymbolID], e.CalleeSymbolID)
		data.reverse[e.CalleeSymbolID] = append(data.reverse[e.CalleeSymbolID], e.CallerSymbolID)
		data.edgesByCaller[e.CallerSymbolID] = append(data.edgesByCaller[e.CallerSymbolID], e)
		data.edgesByCallee[e.CalleeSymbolID] = append(data.edgesByCallee[e.CalleeSymbolID], e)
	}

	return data, nil
}

// resolveCallGraphEdge converts a CallEdge to a CallGraphEdge,
// resolving FileID to a file path string.
func resolveCallGraphEdge(edge *CallEdge, filePaths map[int64]string) CallGraphEdge {
	file := ""
	if edge.FileID != nil {
		file = filePaths[*edge.FileID]
	}
	return CallGraphEdge{
		CallerID: edge.CallerSymbolID,
		CalleeID: edge.CalleeSymbolID,
		File:     file,
		Line:     edge.Line,
		Col:      edge.Col,
	}
}

// TransitiveCallers returns all transitive callers of a symbol up to maxDepth.
// Bulk-loads all call_graph edges into memory, then walks callers with BFS.
// maxDepth of 0 returns only the root node (no traversal). Negative returns error.
// Capped at 100. Returns nil, nil if symbolID does not exist.
func (q *QueryBuilder) TransitiveCallers(symbolID int64, maxDepth int) (*CallGraph, error) {
	if maxDepth < 0 {
		return nil, fmt.Errorf("transitive callers: maxDepth must be non-negative, got %d", maxDepth)
	}
	if maxDepth > 100 {
		maxDepth = 100
	}

	rootSym, err := q.symbolResultByID(symbolID)
	if err != nil {
		return nil, fmt.Errorf("transitive callers: %w", err)
	}
	if rootSym == nil {
		return nil, nil
	}

	result := &CallGraph{
		Root:  symbolID,
		Nodes: []CallGraphNode{{Symbol: *rootSym, Depth: 0}},
		Edges: []CallGraphEdge{},
		Depth: 0,
	}

	if maxDepth == 0 {
		return result, nil
	}

	data, err := q.buildCallGraph()
	if err != nil {
		return nil, fmt.Errorf("transitive callers: %w", err)
	}

	// BFS on reverse adjacency map
	visited := map[int64]int{symbolID: 0} // symbol ID -> depth
	type bfsEntry struct {
		id    int64
		depth int
	}
	queue := []bfsEntry{{id: symbolID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Don't explore further if at maxDepth
		if current.depth >= maxDepth {
			continue
		}

		for _, callerID := range data.reverse[current.id] {
			if _, seen := visited[callerID]; !seen {
				newDepth := current.depth + 1
				visited[callerID] = newDepth
				if newDepth > result.Depth {
					result.Depth = newDepth
				}
				queue = append(queue, bfsEntry{id: callerID, depth: newDepth})
			}
		}
	}

	// Collect all visited node IDs (except root, already added)
	nodeIDs := make([]int64, 0, len(visited)-1)
	for id := range visited {
		if id != symbolID {
			nodeIDs = append(nodeIDs, id)
		}
	}

	// Bulk load symbols
	symbols, err := q.symbolResultsByIDs(nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("transitive callers: load symbols: %w", err)
	}

	for _, id := range nodeIDs {
		if sr, ok := symbols[id]; ok {
			result.Nodes = append(result.Nodes, CallGraphNode{Symbol: *sr, Depth: visited[id]})
		}
	}

	// Collect edges that connect visited nodes.
	// For reverse traversal (callers), an edge is relevant if both caller and callee
	// are in the visited set.
	edgeSeen := make(map[int64]bool)
	for id := range visited {
		for _, edge := range data.edgesByCallee[id] {
			if _, callerVisited := visited[edge.CallerSymbolID]; callerVisited {
				if !edgeSeen[edge.ID] {
					edgeSeen[edge.ID] = true
					result.Edges = append(result.Edges, resolveCallGraphEdge(edge, data.filePaths))
				}
			}
		}
	}

	return result, nil
}

// TransitiveCallees returns all transitive callees of a symbol up to maxDepth.
// Bulk-loads all call_graph edges into memory, then walks callees with BFS.
// maxDepth of 0 returns only the root node (no traversal). Negative returns error.
// Capped at 100. Returns nil, nil if symbolID does not exist.
func (q *QueryBuilder) TransitiveCallees(symbolID int64, maxDepth int) (*CallGraph, error) {
	if maxDepth < 0 {
		return nil, fmt.Errorf("transitive callees: maxDepth must be non-negative, got %d", maxDepth)
	}
	if maxDepth > 100 {
		maxDepth = 100
	}

	rootSym, err := q.symbolResultByID(symbolID)
	if err != nil {
		return nil, fmt.Errorf("transitive callees: %w", err)
	}
	if rootSym == nil {
		return nil, nil
	}

	result := &CallGraph{
		Root:  symbolID,
		Nodes: []CallGraphNode{{Symbol: *rootSym, Depth: 0}},
		Edges: []CallGraphEdge{},
		Depth: 0,
	}

	if maxDepth == 0 {
		return result, nil
	}

	data, err := q.buildCallGraph()
	if err != nil {
		return nil, fmt.Errorf("transitive callees: %w", err)
	}

	// BFS on forward adjacency map
	visited := map[int64]int{symbolID: 0}
	type bfsEntry struct {
		id    int64
		depth int
	}
	queue := []bfsEntry{{id: symbolID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		for _, calleeID := range data.forward[current.id] {
			if _, seen := visited[calleeID]; !seen {
				newDepth := current.depth + 1
				visited[calleeID] = newDepth
				if newDepth > result.Depth {
					result.Depth = newDepth
				}
				queue = append(queue, bfsEntry{id: calleeID, depth: newDepth})
			}
		}
	}

	// Collect visited node IDs (except root)
	nodeIDs := make([]int64, 0, len(visited)-1)
	for id := range visited {
		if id != symbolID {
			nodeIDs = append(nodeIDs, id)
		}
	}

	symbols, err := q.symbolResultsByIDs(nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("transitive callees: load symbols: %w", err)
	}

	for _, id := range nodeIDs {
		if sr, ok := symbols[id]; ok {
			result.Nodes = append(result.Nodes, CallGraphNode{Symbol: *sr, Depth: visited[id]})
		}
	}

	// Collect edges that connect visited nodes.
	edgeSeen := make(map[int64]bool)
	for id := range visited {
		for _, edge := range data.edgesByCaller[id] {
			if _, calleeVisited := visited[edge.CalleeSymbolID]; calleeVisited {
				if !edgeSeen[edge.ID] {
					edgeSeen[edge.ID] = true
					result.Edges = append(result.Edges, resolveCallGraphEdge(edge, data.filePaths))
				}
			}
		}
	}

	return result, nil
}

// HotspotResult represents a heavily-referenced symbol with fan-in/fan-out
// metrics from the call graph.
type HotspotResult struct {
	Symbol      SymbolResult
	CallerCount int // direct callers (fan-in from call_graph)
	CalleeCount int // direct callees (fan-out from call_graph)
}

// UnusedSymbols returns symbols with zero resolved references.
// Hardcoded exclusion of kinds "package", "module", "namespace" (never
// meaningfully referenced). Supports the same SymbolFilter and Pagination
// as Symbols().
func (q *QueryBuilder) UnusedSymbols(filter SymbolFilter, sort Sort, page Pagination) (*PagedResult[SymbolResult], error) {
	page = page.normalize()

	var where []string
	var args []any

	// Core condition: no resolved references targeting this symbol
	where = append(where, "NOT EXISTS (SELECT 1 FROM resolved_references rr WHERE rr.target_symbol_id = s.id)")

	// Hardcoded exclusion of package-like kinds
	where = append(where, "s.kind NOT IN ('package', 'module', 'namespace')")

	// Apply SymbolFilter fields
	if len(filter.Kinds) > 0 {
		placeholders := strings.Repeat("?,", len(filter.Kinds)-1) + "?"
		where = append(where, "s.kind IN ("+placeholders+")")
		for _, k := range filter.Kinds {
			args = append(args, k)
		}
	}
	if filter.Visibility != nil {
		where = append(where, "s.visibility = ?")
		args = append(args, *filter.Visibility)
	}
	if filter.FileID != nil {
		where = append(where, "s.file_id = ?")
		args = append(args, *filter.FileID)
	}
	if filter.ParentID != nil {
		where = append(where, "s.parent_symbol_id = ?")
		args = append(args, *filter.ParentID)
	}
	if filter.PathPrefix != nil {
		prefix := normalizePathPrefix(*filter.PathPrefix)
		if prefix != "" {
			where = append(where, "f.path LIKE ? ESCAPE '\\'")
			args = append(args, escapeLike(prefix)+"%")
		}
	}
	for _, mod := range filter.Modifiers {
		where = append(where, "EXISTS (SELECT 1 FROM json_each(s.modifiers) WHERE json_each.value = ?)")
		args = append(args, mod)
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	// Count query
	var totalCount int
	countSQL := `SELECT COUNT(*) FROM symbols s LEFT JOIN files f ON s.file_id = f.id ` + whereClause
	if err := q.store.DB().QueryRow(countSQL, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("unused symbols: count: %w", err)
	}

	// Data query -- ref_count and external_ref_count are always 0 for unused symbols,
	// but we use the standard scan to keep SymbolResult consistent.
	orderCol := symbolSortColumn(sort.Field)
	orderDir := sortDirection(sort.Order)

	dataSQL := fmt.Sprintf(
		`SELECT %s, COALESCE(f.path, '') AS file_path,
			0 AS ref_count,
			0 AS external_ref_count
		 FROM symbols s
		 LEFT JOIN files f ON s.file_id = f.id
		 %s
		 ORDER BY %s %s
		 LIMIT ? OFFSET ?`,
		prefixSymbolCols("s"), whereClause, orderCol, orderDir,
	)
	dataArgs := append(append([]any{}, args...), *page.Limit, page.Offset)

	rows, err := q.store.DB().Query(dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("unused symbols: query: %w", err)
	}
	defer rows.Close()

	var items []SymbolResult
	for rows.Next() {
		sr, err := scanSymbolResult(rows)
		if err != nil {
			return nil, fmt.Errorf("unused symbols: scan: %w", err)
		}
		items = append(items, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("unused symbols: rows: %w", err)
	}
	if items == nil {
		items = []SymbolResult{}
	}

	return &PagedResult[SymbolResult]{Items: items, TotalCount: totalCount}, nil
}

// Hotspots returns the top-N most-referenced symbols with fan-in and fan-out
// metrics. Sorts by external reference count descending.
// topN of 0 returns empty list. Negative returns error.
func (q *QueryBuilder) Hotspots(topN int) ([]*HotspotResult, error) {
	if topN < 0 {
		return nil, fmt.Errorf("hotspots: topN must be non-negative, got %d", topN)
	}
	if topN == 0 {
		return []*HotspotResult{}, nil
	}

	dataSQL := fmt.Sprintf(
		`SELECT %s, COALESCE(f.path, '') AS file_path,
			(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
			(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count,
			(SELECT COUNT(*) FROM call_graph cg WHERE cg.callee_symbol_id = s.id) AS caller_count,
			(SELECT COUNT(*) FROM call_graph cg WHERE cg.caller_symbol_id = s.id) AS callee_count
		 FROM symbols s
		 LEFT JOIN files f ON s.file_id = f.id
		 WHERE EXISTS (SELECT 1 FROM resolved_references rr2 WHERE rr2.target_symbol_id = s.id)
		 ORDER BY external_ref_count DESC
		 LIMIT ?`,
		prefixSymbolCols("s"),
	)

	rows, err := q.store.DB().Query(dataSQL, topN)
	if err != nil {
		return nil, fmt.Errorf("hotspots: query: %w", err)
	}
	defer rows.Close()

	var items []*HotspotResult
	for rows.Next() {
		hr, err := scanHotspotResult(rows)
		if err != nil {
			return nil, fmt.Errorf("hotspots: scan: %w", err)
		}
		items = append(items, hr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("hotspots: rows: %w", err)
	}
	if items == nil {
		items = []*HotspotResult{}
	}

	return items, nil
}

// scanHotspotResult scans a row into a HotspotResult.
// Expects columns: [SymbolCols..., file_path, ref_count, external_ref_count, caller_count, callee_count].
func scanHotspotResult(row scanner) (*HotspotResult, error) {
	var hr HotspotResult
	var mods string
	err := row.Scan(
		&hr.Symbol.ID, &hr.Symbol.FileID, &hr.Symbol.Name, &hr.Symbol.Kind,
		&hr.Symbol.Visibility, &mods, &hr.Symbol.SignatureHash,
		&hr.Symbol.StartLine, &hr.Symbol.StartCol, &hr.Symbol.EndLine, &hr.Symbol.EndCol,
		&hr.Symbol.ParentSymbolID,
		&hr.Symbol.FilePath, &hr.Symbol.RefCount, &hr.Symbol.ExternalRefCount,
		&hr.CallerCount, &hr.CalleeCount,
	)
	if err != nil {
		return nil, err
	}
	hr.Symbol.Modifiers = store.UnmarshalModifiers(mods)
	hr.Symbol.InternalRefCount = hr.Symbol.RefCount - hr.Symbol.ExternalRefCount
	return &hr, nil
}
