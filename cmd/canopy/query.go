package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jward/canopy"
	"github.com/jward/canopy/internal/store"
	"github.com/spf13/cobra"
)

var (
	flagLimit  int
	flagOffset int
	flagSort   string
	flagOrder  string
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query the semantic index",
	Long:  "Run queries against an indexed codebase. All line and column numbers are 0-based.",
}

func init() {
	queryCmd.PersistentFlags().IntVar(&flagLimit, "limit", 50, "pagination limit (max 500)")
	queryCmd.PersistentFlags().IntVar(&flagOffset, "offset", 0, "pagination offset")
	queryCmd.PersistentFlags().StringVar(&flagSort, "sort", "", "sort field: name|kind|file|ref_count")
	queryCmd.PersistentFlags().StringVar(&flagOrder, "order", "asc", "sort order: asc|desc")

	queryCmd.AddCommand(symbolAtCmd)
	queryCmd.AddCommand(definitionCmd)
	queryCmd.AddCommand(referencesCmd)
	queryCmd.AddCommand(callersCmd)
	queryCmd.AddCommand(calleesCmd)
	queryCmd.AddCommand(implementationsCmd)
	queryCmd.AddCommand(symbolsCmd)
	queryCmd.AddCommand(searchCmd)
	queryCmd.AddCommand(filesCmd)
	queryCmd.AddCommand(packagesCmd)
	queryCmd.AddCommand(summaryCmd)
	queryCmd.AddCommand(packageSummaryCmd)
	queryCmd.AddCommand(depsCmd)
	queryCmd.AddCommand(dependentsCmd)
}

// --- Helpers ---

// openStore opens the Store from the --db flag path (or default).
func openStore() (*store.Store, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting cwd: %w", err)
	}
	repoRoot := findRepoRoot(cwd)
	dbPath := resolveDBPath(repoRoot)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found: %s (run 'canopy index' first)", dbPath)
	}

	return store.NewStore(dbPath)
}

// resolveFilePath converts a file argument to an absolute path.
// If the path is already absolute, it's returned as-is.
// Otherwise, it's resolved relative to the current working directory.
func resolveFilePath(file string) (string, error) {
	if filepath.IsAbs(file) {
		return file, nil
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("resolving file path %q: %w", file, err)
	}
	return abs, nil
}

// parseIntArg parses a positional argument as an integer with a clear error.
func parseIntArg(value, name string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be a non-negative integer", name, value)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid %s %q: must be non-negative", name, value)
	}
	return n, nil
}

// resolveSymbolID resolves a symbol ID from either positional args (<file> <line> <col>) or --symbol flag.
// Returns the symbol ID and the QueryBuilder to use.
func resolveSymbolID(cmd *cobra.Command, args []string, qb *canopy.QueryBuilder) (int64, error) {
	symbolFlag, _ := cmd.Flags().GetInt64("symbol")
	if symbolFlag != 0 {
		return symbolFlag, nil
	}

	if len(args) < 3 {
		return 0, fmt.Errorf("requires either <file> <line> <col> arguments or --symbol flag")
	}

	file, err := resolveFilePath(args[0])
	if err != nil {
		return 0, err
	}
	line, err := parseIntArg(args[1], "line")
	if err != nil {
		return 0, err
	}
	col, err := parseIntArg(args[2], "col")
	if err != nil {
		return 0, err
	}

	sym, err := qb.SymbolAt(file, line, col)
	if err != nil {
		return 0, fmt.Errorf("looking up symbol: %w", err)
	}
	if sym == nil {
		return 0, fmt.Errorf("no symbol found at %s:%d:%d", file, line, col)
	}
	return sym.ID, nil
}

// outputResult marshals a CLIResult to stdout in the selected format.
func outputResult(result CLIResult) error {
	if flagFormat == "text" {
		return outputResultText(result)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// outputError writes an error in the selected format and returns it so RunE
// can propagate it to Cobra. In JSON mode the error is written to stdout as a
// CLIResult envelope. In text mode it goes to stderr.
func outputError(command string, err error) error {
	errorHandled = true
	if flagFormat == "text" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return err
	}
	result := CLIResult{
		Command: command,
		Error:   err.Error(),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
	return err
}

// buildPagination creates a Pagination from CLI flags.
func buildPagination() canopy.Pagination {
	return canopy.Pagination{
		Limit:  flagLimit,
		Offset: flagOffset,
	}
}

// buildSort creates a Sort from CLI flags.
func buildSort() canopy.Sort {
	var field canopy.SortField
	switch flagSort {
	case "name":
		field = canopy.SortByName
	case "kind":
		field = canopy.SortByKind
	case "file":
		field = canopy.SortByFile
	case "ref_count":
		field = canopy.SortByRefCount
	case "external_ref_count":
		field = canopy.SortByExternalRefCount
	default:
		field = canopy.SortByName
	}

	var order canopy.SortOrder
	switch flagOrder {
	case "desc":
		order = canopy.Desc
	default:
		order = canopy.Asc
	}

	return canopy.Sort{Field: field, Order: order}
}

// symbolToCLI converts a store.Symbol to a CLISymbol.
func symbolToCLI(sym *store.Symbol, filePath string, refCount int) CLISymbol {
	return CLISymbol{
		ID:         sym.ID,
		Name:       sym.Name,
		Kind:       sym.Kind,
		Visibility: sym.Visibility,
		Modifiers:  sym.Modifiers,
		File:       filePath,
		StartLine:  sym.StartLine,
		StartCol:   sym.StartCol,
		EndLine:    sym.EndLine,
		EndCol:     sym.EndCol,
		RefCount:   refCount,
	}
}

// symbolResultToCLI converts a canopy.SymbolResult to a CLISymbol.
func symbolResultToCLI(sr canopy.SymbolResult) CLISymbol {
	return CLISymbol{
		ID:               sr.ID,
		Name:             sr.Name,
		Kind:             sr.Kind,
		Visibility:       sr.Visibility,
		Modifiers:        sr.Modifiers,
		File:             sr.FilePath,
		StartLine:        sr.StartLine,
		StartCol:         sr.StartCol,
		EndLine:          sr.EndLine,
		EndCol:           sr.EndCol,
		RefCount:         sr.RefCount,
		ExternalRefCount: sr.ExternalRefCount,
		InternalRefCount: sr.InternalRefCount,
	}
}

// locationToCLI converts a canopy.Location to a CLILocation.
func locationToCLI(loc canopy.Location, symbolID *int64) CLILocation {
	return CLILocation{
		File:      loc.File,
		StartLine: loc.StartLine,
		StartCol:  loc.StartCol,
		EndLine:   loc.EndLine,
		EndCol:    loc.EndCol,
		SymbolID:  symbolID,
	}
}

// lookupSymbolName fetches just the name of a symbol by ID.
// Returns empty string if not found; logs non-ErrNoRows errors to stderr.
func lookupSymbolName(s *store.Store, id int64) string {
	var name string
	err := s.DB().QueryRow("SELECT name FROM symbols WHERE id = ?", id).Scan(&name)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("warning: lookupSymbolName(%d): %v", id, err)
	}
	return name
}

// lookupFilePath fetches the file path for a file ID.
// Returns empty string if not found; logs non-ErrNoRows errors to stderr.
func lookupFilePath(s *store.Store, fileID *int64) string {
	if fileID == nil {
		return ""
	}
	var path string
	err := s.DB().QueryRow("SELECT path FROM files WHERE id = ?", *fileID).Scan(&path)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("warning: lookupFilePath(%d): %v", *fileID, err)
	}
	return path
}

// --- Position-Based Commands ---

var symbolAtCmd = &cobra.Command{
	Use:   "symbol-at <file> <line> <col>",
	Short: "Find the symbol at a position",
	Args:  cobra.ExactArgs(3),
	RunE:  runSymbolAt,
}

func runSymbolAt(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("symbol-at", err)
	}
	defer s.Close()

	file, err := resolveFilePath(args[0])
	if err != nil {
		return outputError("symbol-at", err)
	}
	line, err := parseIntArg(args[1], "line")
	if err != nil {
		return outputError("symbol-at", err)
	}
	col, err := parseIntArg(args[2], "col")
	if err != nil {
		return outputError("symbol-at", err)
	}

	qb := canopy.NewQueryBuilder(s)
	sym, err := qb.SymbolAt(file, line, col)
	if err != nil {
		return outputError("symbol-at", err)
	}

	if sym == nil {
		return outputResult(CLIResult{
			Command: "symbol-at",
			Results: nil,
		})
	}

	filePath := lookupFilePath(s, sym.FileID)
	cliSym := symbolToCLI(sym, filePath, 0)

	one := 1
	return outputResult(CLIResult{
		Command:    "symbol-at",
		Results:    cliSym,
		TotalCount: &one,
	})
}

var definitionCmd = &cobra.Command{
	Use:   "definition <file> <line> <col>",
	Short: "Find the definition of the symbol at a position",
	Args:  cobra.ExactArgs(3),
	RunE:  runDefinition,
}

func runDefinition(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("definition", err)
	}
	defer s.Close()

	file, err := resolveFilePath(args[0])
	if err != nil {
		return outputError("definition", err)
	}
	line, err := parseIntArg(args[1], "line")
	if err != nil {
		return outputError("definition", err)
	}
	col, err := parseIntArg(args[2], "col")
	if err != nil {
		return outputError("definition", err)
	}

	qb := canopy.NewQueryBuilder(s)
	locs, err := qb.DefinitionAt(file, line, col)
	if err != nil {
		return outputError("definition", err)
	}

	cliLocs := make([]CLILocation, len(locs))
	for i, loc := range locs {
		var symID *int64
		if sym, err := qb.SymbolAt(loc.File, loc.StartLine, loc.StartCol); err == nil && sym != nil {
			symID = &sym.ID
		}
		cliLocs[i] = locationToCLI(loc, symID)
	}

	defCount := len(cliLocs)
	return outputResult(CLIResult{
		Command:    "definition",
		Results:    cliLocs,
		TotalCount: &defCount,
	})
}

// --- Symbol ID or Position Commands ---

var referencesCmd = &cobra.Command{
	Use:   "references [<file> <line> <col>]",
	Short: "Find all references to a symbol",
	Long:  "Accepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runReferences,
}

func init() {
	referencesCmd.Flags().Int64("symbol", 0, "symbol ID to query")
}

func runReferences(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("references", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("references", err)
	}

	locs, err := qb.ReferencesTo(symID)
	if err != nil {
		return outputError("references", err)
	}

	cliLocs := make([]CLILocation, len(locs))
	for i, loc := range locs {
		cliLocs[i] = locationToCLI(loc, &symID)
	}

	refCount := len(cliLocs)
	return outputResult(CLIResult{
		Command:    "references",
		Results:    cliLocs,
		TotalCount: &refCount,
	})
}

var callersCmd = &cobra.Command{
	Use:   "callers [<file> <line> <col>]",
	Short: "Find callers of a function",
	Long:  "Accepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runCallers,
}

func init() {
	callersCmd.Flags().Int64("symbol", 0, "symbol ID to query")
}

func runCallers(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("callers", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("callers", err)
	}

	edges, err := qb.Callers(symID)
	if err != nil {
		return outputError("callers", err)
	}

	cliEdges := make([]CLICallEdge, len(edges))
	for i, e := range edges {
		cliEdges[i] = CLICallEdge{
			CallerID:   e.CallerSymbolID,
			CallerName: lookupSymbolName(s, e.CallerSymbolID),
			CalleeID:   e.CalleeSymbolID,
			CalleeName: lookupSymbolName(s, e.CalleeSymbolID),
			File:       lookupFilePath(s, e.FileID),
			Line:       e.Line,
			Col:        e.Col,
		}
	}

	callerCount := len(cliEdges)
	return outputResult(CLIResult{
		Command:    "callers",
		Results:    cliEdges,
		TotalCount: &callerCount,
	})
}

var calleesCmd = &cobra.Command{
	Use:   "callees [<file> <line> <col>]",
	Short: "Find functions called by a function",
	Long:  "Accepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runCallees,
}

func init() {
	calleesCmd.Flags().Int64("symbol", 0, "symbol ID to query")
}

func runCallees(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("callees", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("callees", err)
	}

	edges, err := qb.Callees(symID)
	if err != nil {
		return outputError("callees", err)
	}

	cliEdges := make([]CLICallEdge, len(edges))
	for i, e := range edges {
		cliEdges[i] = CLICallEdge{
			CallerID:   e.CallerSymbolID,
			CallerName: lookupSymbolName(s, e.CallerSymbolID),
			CalleeID:   e.CalleeSymbolID,
			CalleeName: lookupSymbolName(s, e.CalleeSymbolID),
			File:       lookupFilePath(s, e.FileID),
			Line:       e.Line,
			Col:        e.Col,
		}
	}

	calleeCount := len(cliEdges)
	return outputResult(CLIResult{
		Command:    "callees",
		Results:    cliEdges,
		TotalCount: &calleeCount,
	})
}

var implementationsCmd = &cobra.Command{
	Use:   "implementations [<file> <line> <col>]",
	Short: "Find implementations of an interface",
	Long:  "Accepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runImplementations,
}

func init() {
	implementationsCmd.Flags().Int64("symbol", 0, "symbol ID to query")
}

func runImplementations(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("implementations", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("implementations", err)
	}

	locs, err := qb.Implementations(symID)
	if err != nil {
		return outputError("implementations", err)
	}

	cliLocs := make([]CLILocation, len(locs))
	for i, loc := range locs {
		var implSymID *int64
		if sym, err := qb.SymbolAt(loc.File, loc.StartLine, loc.StartCol); err == nil && sym != nil {
			implSymID = &sym.ID
		}
		cliLocs[i] = locationToCLI(loc, implSymID)
	}

	implCount := len(cliLocs)
	return outputResult(CLIResult{
		Command:    "implementations",
		Results:    cliLocs,
		TotalCount: &implCount,
	})
}
