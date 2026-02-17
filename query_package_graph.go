package canopy

import (
	"fmt"
	"sort"
	"strings"
)

// DependencyGraph is the package-to-package dependency graph, aggregated
// from file-level imports.
type DependencyGraph struct {
	Packages []PackageNode
	Edges    []DependencyEdge
}

// PackageNode represents a package in the dependency graph.
type PackageNode struct {
	Name      string
	FileCount int
	LineCount int
}

// DependencyEdge represents a dependency between two packages with the
// number of file-level imports that contribute to it.
type DependencyEdge struct {
	FromPackage string
	ToPackage   string
	ImportCount int
}

// PackageDependencyGraph returns the package-to-package dependency graph.
// Aggregates file-level imports: for each import, determines the source file's
// package and the import target's package, then counts edges between packages.
// Packages are identified by "package"/"module"/"namespace" symbols.
func (q *QueryBuilder) PackageDependencyGraph() (*DependencyGraph, error) {
	// 1. Load all files.
	type fileInfo struct {
		id        int64
		path      string
		language  string
		lineCount int
	}
	fileRows, err := q.store.DB().Query("SELECT id, path, language, COALESCE(line_count, 0) FROM files")
	if err != nil {
		return nil, fmt.Errorf("package dependency graph: query files: %w", err)
	}
	defer fileRows.Close()

	files := map[int64]*fileInfo{}
	for fileRows.Next() {
		f := &fileInfo{}
		if err := fileRows.Scan(&f.id, &f.path, &f.language, &f.lineCount); err != nil {
			return nil, fmt.Errorf("package dependency graph: scan file: %w", err)
		}
		files[f.id] = f
	}
	if err := fileRows.Err(); err != nil {
		return nil, fmt.Errorf("package dependency graph: file rows: %w", err)
	}

	// 2. Load all package/module/namespace symbols to build file_id -> package name map.
	pkgRows, err := q.store.DB().Query(
		"SELECT id, file_id, name FROM symbols WHERE kind IN ('package', 'module', 'namespace')",
	)
	if err != nil {
		return nil, fmt.Errorf("package dependency graph: query package symbols: %w", err)
	}
	defer pkgRows.Close()

	// fileIDToPkg maps file_id -> package name.
	fileIDToPkg := map[int64]string{}
	// allPackages tracks all known package names.
	allPackages := map[string]bool{}
	for pkgRows.Next() {
		var id, fileID int64
		var name string
		if err := pkgRows.Scan(&id, &fileID, &name); err != nil {
			return nil, fmt.Errorf("package dependency graph: scan package symbol: %w", err)
		}
		fileIDToPkg[fileID] = name
		allPackages[name] = true
	}
	if err := pkgRows.Err(); err != nil {
		return nil, fmt.Errorf("package dependency graph: package symbol rows: %w", err)
	}

	// 3. Load all imports.
	type importInfo struct {
		fileID int64
		source string
		kind   string
	}
	impRows, err := q.store.DB().Query("SELECT file_id, source, kind FROM imports")
	if err != nil {
		return nil, fmt.Errorf("package dependency graph: query imports: %w", err)
	}
	defer impRows.Close()

	var imports []importInfo
	for impRows.Next() {
		var imp importInfo
		if err := impRows.Scan(&imp.fileID, &imp.source, &imp.kind); err != nil {
			return nil, fmt.Errorf("package dependency graph: scan import: %w", err)
		}
		imports = append(imports, imp)
	}
	if err := impRows.Err(); err != nil {
		return nil, fmt.Errorf("package dependency graph: import rows: %w", err)
	}

	// 4. Build import source -> target package resolution map.
	// An import resolves to an internal package if:
	//   (a) the import source exactly equals a package name, or
	//   (b) the import source's last path segment equals a package name, or
	//   (c) any file's path has a suffix matching the import source.
	// We pre-build a map for fast lookup.
	importSourceToPkg := map[string]string{}
	for _, imp := range imports {
		if _, resolved := importSourceToPkg[imp.source]; resolved {
			continue
		}
		// (a) Exact match to a known package name.
		if allPackages[imp.source] {
			importSourceToPkg[imp.source] = imp.source
			continue
		}
		// (b) Last path segment matches a package name.
		lastSeg := imp.source
		if idx := strings.LastIndex(imp.source, "/"); idx >= 0 {
			lastSeg = imp.source[idx+1:]
		}
		if lastSeg != imp.source && allPackages[lastSeg] {
			importSourceToPkg[imp.source] = lastSeg
			continue
		}
		// (c) Check if any file's path has a suffix matching the import source.
		for _, f := range files {
			if strings.HasSuffix(f.path, "/"+imp.source) || strings.HasSuffix(f.path, "/"+imp.source+"/") {
				if pkg, ok := fileIDToPkg[f.id]; ok {
					importSourceToPkg[imp.source] = pkg
					break
				}
			}
		}
	}

	// 5. Aggregate file-level imports to package-level edges.
	type edgeKey struct {
		from, to string
	}
	edgeCounts := map[edgeKey]int{}
	for _, imp := range imports {
		fromPkg, ok := fileIDToPkg[imp.fileID]
		if !ok {
			continue // file has no package symbol, skip
		}
		toPkg, ok := importSourceToPkg[imp.source]
		if !ok {
			continue // external import, skip
		}
		edgeCounts[edgeKey{from: fromPkg, to: toPkg}]++
	}

	// 6. Build package nodes with FileCount and LineCount.
	pkgFiles := map[string]int{}
	pkgLines := map[string]int{}
	for fid, f := range files {
		if pkg, ok := fileIDToPkg[fid]; ok {
			pkgFiles[pkg]++
			pkgLines[pkg] += f.lineCount
		}
	}

	var packages []PackageNode
	// Sort by name for deterministic output.
	sortedPkgNames := make([]string, 0, len(allPackages))
	for name := range allPackages {
		sortedPkgNames = append(sortedPkgNames, name)
	}
	sort.Strings(sortedPkgNames)

	for _, name := range sortedPkgNames {
		packages = append(packages, PackageNode{
			Name:      name,
			FileCount: pkgFiles[name],
			LineCount: pkgLines[name],
		})
	}

	var edges []DependencyEdge
	for ek, count := range edgeCounts {
		edges = append(edges, DependencyEdge{
			FromPackage: ek.from,
			ToPackage:   ek.to,
			ImportCount: count,
		})
	}
	// Sort edges for deterministic output.
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromPackage != edges[j].FromPackage {
			return edges[i].FromPackage < edges[j].FromPackage
		}
		return edges[i].ToPackage < edges[j].ToPackage
	})

	if packages == nil {
		packages = []PackageNode{}
	}
	if edges == nil {
		edges = []DependencyEdge{}
	}

	return &DependencyGraph{Packages: packages, Edges: edges}, nil
}

// CircularDependencies detects cycles in the package dependency graph using
// Tarjan's strongly connected components algorithm.
// Returns a list of cycles, each represented as a list of package names
// (first element repeated at end for clarity).
// Returns empty list (not nil) for acyclic graphs.
func (q *QueryBuilder) CircularDependencies() ([][]string, error) {
	graph, err := q.PackageDependencyGraph()
	if err != nil {
		return nil, fmt.Errorf("circular dependencies: %w", err)
	}

	// Build adjacency list and detect self-loops.
	adj := map[string][]string{}
	selfLoops := map[string]bool{}
	for _, edge := range graph.Edges {
		if edge.FromPackage == edge.ToPackage {
			selfLoops[edge.FromPackage] = true
		}
		adj[edge.FromPackage] = append(adj[edge.FromPackage], edge.ToPackage)
	}

	// Tarjan's SCC algorithm.
	type nodeInfo struct {
		index   int
		lowlink int
		onStack bool
	}
	info := map[string]*nodeInfo{}
	index := 0
	var stack []string
	var result [][]string

	var strongconnect func(v string)
	strongconnect = func(v string) {
		ni := &nodeInfo{index: index, lowlink: index, onStack: true}
		info[v] = ni
		index++
		stack = append(stack, v)

		for _, w := range adj[v] {
			wInfo, visited := info[w]
			if !visited {
				strongconnect(w)
				wInfo = info[w]
				if wInfo.lowlink < ni.lowlink {
					ni.lowlink = wInfo.lowlink
				}
			} else if wInfo.onStack {
				if wInfo.index < ni.lowlink {
					ni.lowlink = wInfo.index
				}
			}
		}

		if ni.lowlink == ni.index {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				info[w].onStack = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			// Only report SCCs with size > 1 (actual cycles) or self-loops.
			if len(scc) > 1 || selfLoops[scc[0]] {
				// Reverse the SCC to get a natural cycle order (Tarjan pops in reverse).
				for i, j := 0, len(scc)-1; i < j; i, j = i+1, j-1 {
					scc[i], scc[j] = scc[j], scc[i]
				}
				// Append first element to end for cycle clarity.
				scc = append(scc, scc[0])
				result = append(result, scc)
			}
		}
	}

	// Process all package nodes (including those with no edges).
	for _, pkg := range graph.Packages {
		if _, visited := info[pkg.Name]; !visited {
			strongconnect(pkg.Name)
		}
	}

	if result == nil {
		result = [][]string{}
	}

	// Sort for deterministic output.
	sort.Slice(result, func(i, j int) bool {
		return result[i][0] < result[j][0]
	})

	return result, nil
}
