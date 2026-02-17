package main

import (
	"fmt"
	"strconv"

	"github.com/jward/canopy"
	"github.com/spf13/cobra"
)

// --- Discovery / Search Commands ---

var (
	flagKind       string
	flagFile       string
	flagVisibility string
	flagPathPrefix string
	flagLanguage   string
	flagPrefix     string
)

var symbolsCmd = &cobra.Command{
	Use:   "symbols",
	Short: "List symbols with optional filters",
	RunE:  runSymbols,
}

func init() {
	symbolsCmd.Flags().StringVar(&flagKind, "kind", "", "filter by symbol kind (e.g. function, type)")
	symbolsCmd.Flags().StringVar(&flagFile, "file", "", "filter by file path")
	symbolsCmd.Flags().StringVar(&flagVisibility, "visibility", "", "filter by visibility (public, private)")
	symbolsCmd.Flags().StringVar(&flagPathPrefix, "path-prefix", "", "filter by file path prefix")
	symbolsCmd.Flags().Int("ref-count-min", 0, "minimum reference count")
	symbolsCmd.Flags().Int("ref-count-max", 0, "maximum reference count")

	searchCmd.Flags().Int("ref-count-min", 0, "minimum reference count")
	searchCmd.Flags().Int("ref-count-max", 0, "maximum reference count")
}

func runSymbols(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("symbols", err)
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
	if cmd.Flags().Changed("ref-count-min") {
		v, _ := cmd.Flags().GetInt("ref-count-min")
		filter.RefCountMin = intPtr(v)
	}
	if cmd.Flags().Changed("ref-count-max") {
		v, _ := cmd.Flags().GetInt("ref-count-max")
		filter.RefCountMax = intPtr(v)
	}
	if flagFile != "" {
		resolvedFile, resolveErr := resolveFilePath(flagFile)
		if resolveErr != nil {
			return outputError("symbols", resolveErr)
		}
		f, err := s.FileByPath(resolvedFile)
		if err != nil {
			return outputError("symbols", fmt.Errorf("looking up file %q: %w", flagFile, err))
		}
		if f == nil {
			return outputError("symbols", fmt.Errorf("file not found: %s", flagFile))
		}
		filter.FileID = &f.ID
	}

	qb := canopy.NewQueryBuilder(s)
	result, err := qb.Symbols(filter, buildSort(), buildPagination())
	if err != nil {
		return outputError("symbols", err)
	}

	cliSyms := make([]CLISymbol, len(result.Items))
	for i, sr := range result.Items {
		cliSyms[i] = symbolResultToCLI(sr)
	}

	return outputResult(CLIResult{
		Command:    "symbols",
		Results:    cliSyms,
		TotalCount: &result.TotalCount,
	})
}

var searchCmd = &cobra.Command{
	Use:   "search <pattern>",
	Short: "Search symbols by glob pattern",
	Long:  "Search for symbols matching a glob pattern. Use * as wildcard (e.g. 'Get*User*').",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("search", err)
	}
	defer s.Close()

	filter := canopy.SymbolFilter{}
	if cmd.Flags().Changed("ref-count-min") {
		v, _ := cmd.Flags().GetInt("ref-count-min")
		filter.RefCountMin = intPtr(v)
	}
	if cmd.Flags().Changed("ref-count-max") {
		v, _ := cmd.Flags().GetInt("ref-count-max")
		filter.RefCountMax = intPtr(v)
	}

	qb := canopy.NewQueryBuilder(s)
	result, err := qb.SearchSymbols(args[0], filter, buildSort(), buildPagination())
	if err != nil {
		return outputError("search", err)
	}

	cliSyms := make([]CLISymbol, len(result.Items))
	for i, sr := range result.Items {
		cliSyms[i] = symbolResultToCLI(sr)
	}

	return outputResult(CLIResult{
		Command:    "search",
		Results:    cliSyms,
		TotalCount: &result.TotalCount,
	})
}

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "List indexed files",
	RunE:  runFiles,
}

func init() {
	filesCmd.Flags().StringVar(&flagLanguage, "language", "", "filter by language")
	filesCmd.Flags().StringVar(&flagPrefix, "prefix", "", "filter by path prefix")
}

func runFiles(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("files", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	result, err := qb.Files(flagPrefix, flagLanguage, buildSort(), buildPagination())
	if err != nil {
		return outputError("files", err)
	}

	cliFiles := make([]CLIFile, len(result.Items))
	for i, f := range result.Items {
		cliFiles[i] = CLIFile{
			ID:        f.ID,
			Path:      f.Path,
			Language:  f.Language,
			LineCount: f.LineCount,
		}
	}

	return outputResult(CLIResult{
		Command:    "files",
		Results:    cliFiles,
		TotalCount: &result.TotalCount,
	})
}

var packagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "List packages/modules/namespaces",
	RunE:  runPackages,
}

func init() {
	// Reuse flagPrefix for packages; avoid double-defining by using a local flag.
	packagesCmd.Flags().String("prefix", "", "filter by path prefix")
}

func runPackages(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("packages", err)
	}
	defer s.Close()

	prefix, _ := cmd.Flags().GetString("prefix")

	qb := canopy.NewQueryBuilder(s)
	result, err := qb.Packages(prefix, buildSort(), buildPagination())
	if err != nil {
		return outputError("packages", err)
	}

	cliSyms := make([]CLISymbol, len(result.Items))
	for i, sr := range result.Items {
		cliSyms[i] = symbolResultToCLI(sr)
	}

	return outputResult(CLIResult{
		Command:    "packages",
		Results:    cliSyms,
		TotalCount: &result.TotalCount,
	})
}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show project-level summary statistics",
	RunE:  runSummary,
}

func runSummary(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("summary", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	summary, err := qb.ProjectSummary(10)
	if err != nil {
		return outputError("summary", err)
	}

	cliSummary := CLIProjectSummary{
		PackageCount: summary.PackageCount,
	}

	cliSummary.Languages = make([]CLILanguageStats, len(summary.Languages))
	for i, ls := range summary.Languages {
		cliSummary.Languages[i] = CLILanguageStats{
			Language:    ls.Language,
			FileCount:   ls.FileCount,
			LineCount:   ls.LineCount,
			SymbolCount: ls.SymbolCount,
			KindCounts:  ls.KindCounts,
		}
	}

	cliSummary.TopSymbols = make([]CLISymbol, len(summary.TopSymbols))
	for i, sr := range summary.TopSymbols {
		cliSummary.TopSymbols[i] = symbolResultToCLI(sr)
	}

	return outputResult(CLIResult{
		Command: "summary",
		Results: cliSummary,
	})
}

var packageSummaryCmd = &cobra.Command{
	Use:   "package-summary <path-or-id>",
	Short: "Show summary for a specific package",
	Long:  "Accepts either a path prefix (string) or a symbol ID (integer).",
	Args:  cobra.ExactArgs(1),
	RunE:  runPackageSummary,
}

func runPackageSummary(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("package-summary", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)

	var pkgPath string
	var pkgID *int64

	// Detect if argument is numeric (symbol ID) or string (path).
	if id, err := strconv.ParseInt(args[0], 10, 64); err == nil {
		pkgID = &id
	} else {
		pkgPath = args[0]
	}

	summary, err := qb.PackageSummary(pkgPath, pkgID)
	if err != nil {
		return outputError("package-summary", err)
	}

	cliSummary := CLIPackageSummary{
		Symbol:       symbolResultToCLI(summary.Symbol),
		Path:         summary.Path,
		FileCount:    summary.FileCount,
		KindCounts:   summary.KindCounts,
		Dependencies: summary.Dependencies,
		Dependents:   summary.Dependents,
	}

	cliSummary.ExportedSymbols = make([]CLISymbol, len(summary.ExportedSymbols))
	for i, sr := range summary.ExportedSymbols {
		cliSummary.ExportedSymbols[i] = symbolResultToCLI(sr)
	}

	return outputResult(CLIResult{
		Command: "package-summary",
		Results: cliSummary,
	})
}

var depsCmd = &cobra.Command{
	Use:   "deps <file>",
	Short: "List imports/dependencies of a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeps,
}

func runDeps(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("deps", err)
	}
	defer s.Close()

	filePath, err := resolveFilePath(args[0])
	if err != nil {
		return outputError("deps", err)
	}

	f, err := s.FileByPath(filePath)
	if err != nil {
		return outputError("deps", fmt.Errorf("looking up file %q: %w", args[0], err))
	}
	if f == nil {
		return outputError("deps", fmt.Errorf("file not found: %s", args[0]))
	}

	qb := canopy.NewQueryBuilder(s)
	imports, err := qb.Dependencies(f.ID)
	if err != nil {
		return outputError("deps", err)
	}

	cliImports := make([]CLIImport, len(imports))
	for i, imp := range imports {
		cliImports[i] = CLIImport{
			FileID:       imp.FileID,
			FilePath:     lookupFilePath(s, &imp.FileID),
			Source:       imp.Source,
			ImportedName: imp.ImportedName,
			LocalAlias:   imp.LocalAlias,
			Kind:         imp.Kind,
		}
	}

	paged, depsCount := paginateSlice(cliImports)
	return outputResult(CLIResult{
		Command:    "deps",
		Results:    paged,
		TotalCount: &depsCount,
	})
}

var dependentsCmd = &cobra.Command{
	Use:   "dependents <source>",
	Short: "List files that import a given source",
	Args:  cobra.ExactArgs(1),
	RunE:  runDependents,
}

func runDependents(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("dependents", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	imports, err := qb.Dependents(args[0])
	if err != nil {
		return outputError("dependents", err)
	}

	cliImports := make([]CLIImport, len(imports))
	for i, imp := range imports {
		cliImports[i] = CLIImport{
			FileID:       imp.FileID,
			FilePath:     lookupFilePath(s, &imp.FileID),
			Source:       imp.Source,
			ImportedName: imp.ImportedName,
			LocalAlias:   imp.LocalAlias,
			Kind:         imp.Kind,
		}
	}

	paged, depCount := paginateSlice(cliImports)
	return outputResult(CLIResult{
		Command:    "dependents",
		Results:    paged,
		TotalCount: &depCount,
	})
}

// intPtr returns a pointer to an int value.
func intPtr(i int) *int { return &i }
