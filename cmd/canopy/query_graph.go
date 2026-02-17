package main

import (
	"github.com/jward/canopy"
	"github.com/spf13/cobra"
)

// --- Graph Analysis Commands ---

var transitiveCallersCmd = &cobra.Command{
	Use:   "transitive-callers [<file> <line> <col>]",
	Short: "Find all transitive callers of a function",
	Long:  "Returns the transitive call graph of callers up to --max-depth.\nAccepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runTransitiveCallers,
}

var transitiveCalleesCmd = &cobra.Command{
	Use:   "transitive-callees [<file> <line> <col>]",
	Short: "Find all transitive callees of a function",
	Long:  "Returns the transitive call graph of callees up to --max-depth.\nAccepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runTransitiveCallees,
}

var packageGraphCmd = &cobra.Command{
	Use:   "package-graph",
	Short: "Show the package dependency graph",
	Args:  cobra.NoArgs,
	RunE:  runPackageGraph,
}

var circularDepsCmd = &cobra.Command{
	Use:   "circular-deps",
	Short: "Detect circular package dependencies",
	Args:  cobra.NoArgs,
	RunE:  runCircularDeps,
}

var unusedCmd = &cobra.Command{
	Use:   "unused",
	Short: "List symbols with zero references",
	RunE:  runUnused,
}

var hotspotsCmd = &cobra.Command{
	Use:   "hotspots",
	Short: "Show most-referenced symbols with call metrics",
	Args:  cobra.NoArgs,
	RunE:  runHotspots,
}

func init() {
	transitiveCallersCmd.Flags().Int64("symbol", 0, "symbol ID to query")
	transitiveCallersCmd.Flags().Int("max-depth", 5, "maximum traversal depth (1-100)")

	transitiveCalleesCmd.Flags().Int64("symbol", 0, "symbol ID to query")
	transitiveCalleesCmd.Flags().Int("max-depth", 5, "maximum traversal depth (1-100)")

	unusedCmd.Flags().StringVar(&flagKind, "kind", "", "filter by symbol kind (e.g. function, type)")
	unusedCmd.Flags().StringVar(&flagVisibility, "visibility", "", "filter by visibility (public, private)")
	unusedCmd.Flags().StringVar(&flagPathPrefix, "path-prefix", "", "filter by file path prefix")

	hotspotsCmd.Flags().Int("top", 10, "number of top hotspots to return")
}

func runTransitiveCallers(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("transitive-callers", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("transitive-callers", err)
	}

	maxDepth, _ := cmd.Flags().GetInt("max-depth")
	graph, err := qb.TransitiveCallers(symID, maxDepth)
	if err != nil {
		return outputError("transitive-callers", err)
	}

	if graph == nil {
		return outputResult(CLIResult{
			Command: "transitive-callers",
			Results: nil,
		})
	}

	cliGraph := callGraphToCLI(graph)
	one := 1
	return outputResult(CLIResult{
		Command:    "transitive-callers",
		Results:    cliGraph,
		TotalCount: &one,
	})
}

func runTransitiveCallees(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("transitive-callees", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("transitive-callees", err)
	}

	maxDepth, _ := cmd.Flags().GetInt("max-depth")
	graph, err := qb.TransitiveCallees(symID, maxDepth)
	if err != nil {
		return outputError("transitive-callees", err)
	}

	if graph == nil {
		return outputResult(CLIResult{
			Command: "transitive-callees",
			Results: nil,
		})
	}

	cliGraph := callGraphToCLI(graph)
	one := 1
	return outputResult(CLIResult{
		Command:    "transitive-callees",
		Results:    cliGraph,
		TotalCount: &one,
	})
}

func runPackageGraph(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("package-graph", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	graph, err := qb.PackageDependencyGraph()
	if err != nil {
		return outputError("package-graph", err)
	}

	cliGraph := dependencyGraphToCLI(graph)
	one := 1
	return outputResult(CLIResult{
		Command:    "package-graph",
		Results:    cliGraph,
		TotalCount: &one,
	})
}

func runCircularDeps(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("circular-deps", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	cycles, err := qb.CircularDependencies()
	if err != nil {
		return outputError("circular-deps", err)
	}

	cliCycles := make([]CLICycle, len(cycles))
	for i, cycle := range cycles {
		cliCycles[i] = CLICycle{Packages: cycle}
	}

	count := len(cliCycles)
	return outputResult(CLIResult{
		Command:    "circular-deps",
		Results:    cliCycles,
		TotalCount: &count,
	})
}

func runUnused(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("unused", err)
	}
	defer s.Close()

	filter := canopy.SymbolFilter{}
	if flagKind != "" {
		filter.Kinds = []string{flagKind}
	}
	if flagVisibility != "" {
		filter.Visibility = &flagVisibility
	}
	if flagPathPrefix != "" {
		filter.PathPrefix = &flagPathPrefix
	}

	qb := canopy.NewQueryBuilder(s)
	result, err := qb.UnusedSymbols(filter, buildSort(), buildPagination())
	if err != nil {
		return outputError("unused", err)
	}

	cliSyms := make([]CLISymbol, len(result.Items))
	for i, sr := range result.Items {
		cliSyms[i] = symbolResultToCLI(sr)
	}


	return outputResult(CLIResult{
		Command:    "unused",
		Results:    cliSyms,
		TotalCount: &result.TotalCount,
	})
}

func runHotspots(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("hotspots", err)
	}
	defer s.Close()

	topN, _ := cmd.Flags().GetInt("top")

	qb := canopy.NewQueryBuilder(s)
	hotspots, err := qb.Hotspots(topN)
	if err != nil {
		return outputError("hotspots", err)
	}

	cliHotspots := make([]CLIHotspot, len(hotspots))
	for i, h := range hotspots {
		cliHotspots[i] = CLIHotspot{
			Symbol:      symbolResultToCLI(h.Symbol),
			CallerCount: h.CallerCount,
			CalleeCount: h.CalleeCount,
		}
	}

	count := len(cliHotspots)
	return outputResult(CLIResult{
		Command:    "hotspots",
		Results:    cliHotspots,
		TotalCount: &count,
	})
}

// --- Converters ---

func callGraphToCLI(g *canopy.CallGraph) CLICallGraph {
	nodes := make([]CLICallGraphNode, len(g.Nodes))
	for i, n := range g.Nodes {
		nodes[i] = CLICallGraphNode{
			Symbol: symbolResultToCLI(n.Symbol),
			Depth:  n.Depth,
		}
	}

	edges := make([]CLICallGraphEdge, len(g.Edges))
	for i, e := range g.Edges {
		edges[i] = CLICallGraphEdge{
			CallerID: e.CallerID,
			CalleeID: e.CalleeID,
			File:     e.File,
			Line:     e.Line,
			Col:      e.Col,
		}
	}

	return CLICallGraph{
		Root:     g.Root,
		Nodes:    nodes,
		Edges:    edges,
		Depth:    g.Depth,
	}
}

func dependencyGraphToCLI(g *canopy.DependencyGraph) CLIDependencyGraph {
	packages := make([]CLIPackageNode, len(g.Packages))
	for i, p := range g.Packages {
		packages[i] = CLIPackageNode{
			Name:      p.Name,
			FileCount: p.FileCount,
			LineCount: p.LineCount,
		}
	}

	edges := make([]CLIDependencyEdge, len(g.Edges))
	for i, e := range g.Edges {
		edges[i] = CLIDependencyEdge{
			FromPackage: e.FromPackage,
			ToPackage:   e.ToPackage,
			ImportCount: e.ImportCount,
		}
	}

	return CLIDependencyGraph{
		Packages: packages,
		Edges:    edges,
	}
}
