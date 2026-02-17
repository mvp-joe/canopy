package main

import (
	"fmt"

	"github.com/jward/canopy"
	"github.com/jward/canopy/internal/store"
	"github.com/spf13/cobra"
)

// --- Type Hierarchy Commands ---

var typeHierarchyCmd = &cobra.Command{
	Use:   "type-hierarchy [<file> <line> <col>]",
	Short: "Show the full type hierarchy for a symbol",
	Long:  "Returns implements, implemented-by, composes, composed-by, and extensions.\nAccepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runTypeHierarchy,
}

var implementsCmd = &cobra.Command{
	Use:   "implements [<file> <line> <col>]",
	Short: "Find interfaces a type implements",
	Long:  "Returns locations of interface declarations that the given type implements.\nAccepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runImplements,
}

var extensionsCmd = &cobra.Command{
	Use:   "extensions [<file> <line> <col>]",
	Short: "Find extension methods for a type",
	Long:  "Returns extension bindings (trait impls, extension methods, default impls) for a type.\nAccepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runExtensions,
}

var reexportsCmd = &cobra.Command{
	Use:   "reexports <file>",
	Short: "Find re-exported symbols from a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runReexports,
}

func init() {
	typeHierarchyCmd.Flags().Int64("symbol", 0, "symbol ID to query")
	implementsCmd.Flags().Int64("symbol", 0, "symbol ID to query")
	extensionsCmd.Flags().Int64("symbol", 0, "symbol ID to query")
}

func runTypeHierarchy(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("type-hierarchy", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("type-hierarchy", err)
	}

	th, err := qb.TypeHierarchy(symID)
	if err != nil {
		return outputError("type-hierarchy", err)
	}

	if th == nil {
		return outputResult(CLIResult{
			Command: "type-hierarchy",
			Results: nil,
		})
	}

	cliTH := typeHierarchyToCLI(th, s)
	one := 1
	return outputResult(CLIResult{
		Command:    "type-hierarchy",
		Results:    cliTH,
		TotalCount: &one,
	})
}

func runImplements(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("implements", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("implements", err)
	}

	locs, err := qb.ImplementsInterfaces(symID)
	if err != nil {
		return outputError("implements", err)
	}

	cliLocs := make([]CLILocation, len(locs))
	for i, loc := range locs {
		var ifaceSymID *int64
		if sym, symErr := qb.SymbolAt(loc.File, loc.StartLine, loc.StartCol); symErr == nil && sym != nil {
			ifaceSymID = &sym.ID
		}
		cliLocs[i] = locationToCLI(loc, ifaceSymID)
	}

	paged, totalCount := paginateSlice(cliLocs)
	return outputResult(CLIResult{
		Command:    "implements",
		Results:    paged,
		TotalCount: &totalCount,
	})
}

func runExtensions(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("extensions", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)
	symID, err := resolveSymbolID(cmd, args, qb)
	if err != nil {
		return outputError("extensions", err)
	}

	bindings, err := qb.ExtensionMethods(symID)
	if err != nil {
		return outputError("extensions", err)
	}

	cliBindings := make([]CLIExtensionBinding, len(bindings))
	for i, b := range bindings {
		cliBindings[i] = extensionBindingToCLI(b, s)
	}

	paged, totalCount := paginateSlice(cliBindings)
	return outputResult(CLIResult{
		Command:    "extensions",
		Results:    paged,
		TotalCount: &totalCount,
	})
}

func runReexports(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("reexports", err)
	}
	defer s.Close()

	file, err := resolveFilePath(args[0])
	if err != nil {
		return outputError("reexports", err)
	}

	f, err := s.FileByPath(file)
	if err != nil {
		return outputError("reexports", err)
	}
	if f == nil {
		return outputError("reexports", fmt.Errorf("file not found in index: %s", file))
	}

	qb := canopy.NewQueryBuilder(s)
	reexports, err := qb.Reexports(f.ID)
	if err != nil {
		return outputError("reexports", err)
	}

	cliReexports := make([]CLIReexport, len(reexports))
	for i, r := range reexports {
		cliReexports[i] = CLIReexport{
			FileID:           r.FileID,
			OriginalSymbolID: r.OriginalSymbolID,
			ExportedName:     r.ExportedName,
		}
	}

	paged, totalCount := paginateSlice(cliReexports)
	return outputResult(CLIResult{
		Command:    "reexports",
		Results:    paged,
		TotalCount: &totalCount,
	})
}

// --- Converters ---

func typeHierarchyToCLI(th *canopy.TypeHierarchy, s *store.Store) CLITypeHierarchy {
	cli := CLITypeHierarchy{
		Symbol: symbolResultToCLI(th.Symbol),
	}

	cli.Implements = make([]CLITypeRelation, len(th.Implements))
	for i, r := range th.Implements {
		cli.Implements[i] = CLITypeRelation{
			Symbol: symbolResultToCLI(r.Symbol),
			Kind:   r.Kind,
		}
	}

	cli.ImplementedBy = make([]CLITypeRelation, len(th.ImplementedBy))
	for i, r := range th.ImplementedBy {
		cli.ImplementedBy[i] = CLITypeRelation{
			Symbol: symbolResultToCLI(r.Symbol),
			Kind:   r.Kind,
		}
	}

	cli.Composes = make([]CLITypeRelation, len(th.Composes))
	for i, r := range th.Composes {
		cli.Composes[i] = CLITypeRelation{
			Symbol: symbolResultToCLI(r.Symbol),
			Kind:   r.Kind,
		}
	}

	cli.ComposedBy = make([]CLITypeRelation, len(th.ComposedBy))
	for i, r := range th.ComposedBy {
		cli.ComposedBy[i] = CLITypeRelation{
			Symbol: symbolResultToCLI(r.Symbol),
			Kind:   r.Kind,
		}
	}

	cli.Extensions = make([]CLIExtensionBinding, len(th.Extensions))
	for i, b := range th.Extensions {
		cli.Extensions[i] = extensionBindingToCLI(b, s)
	}

	return cli
}

func extensionBindingToCLI(b *store.ExtensionBinding, s *store.Store) CLIExtensionBinding {
	cli := CLIExtensionBinding{
		TypeSymbolID:   b.ExtendedTypeSymbolID,
		MemberSymbolID: b.MemberSymbolID,
		Kind:           b.Kind,
	}
	// Look up the member symbol's file ID.
	if sym, err := s.SymbolByID(b.MemberSymbolID); err == nil && sym != nil {
		cli.SourceFileID = sym.FileID
	}
	return cli
}
