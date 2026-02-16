package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

// formatLocationsText formats CLILocation results as "file:line:col" lines.
func formatLocationsText(w io.Writer, locs []CLILocation) {
	for _, loc := range locs {
		fmt.Fprintf(w, "%s:%d:%d\n", loc.File, loc.StartLine, loc.StartCol)
	}
}

// formatSymbolsText formats CLISymbol results as aligned columns.
func formatSymbolsText(w io.Writer, syms []CLISymbol) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tKIND\tVISIBILITY\tFILE\tLINE")
	for _, s := range syms {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%d\n",
			s.ID, s.Name, s.Kind, s.Visibility, s.File, s.StartLine)
	}
	tw.Flush()
}

// formatCallEdgesText formats CLICallEdge results as aligned columns.
func formatCallEdgesText(w io.Writer, edges []CLICallEdge) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CALLER\tCALLEE\tFILE\tLINE\tCOL")
	for _, e := range edges {
		caller := fmt.Sprintf("%s (#%d)", e.CallerName, e.CallerID)
		callee := fmt.Sprintf("%s (#%d)", e.CalleeName, e.CalleeID)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\n",
			caller, callee, e.File, e.Line, e.Col)
	}
	tw.Flush()
}

// formatImportsText formats CLIImport results as aligned columns.
func formatImportsText(w io.Writer, imports []CLIImport) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SOURCE\tKIND\tFILE")
	for _, imp := range imports {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", imp.Source, imp.Kind, imp.FilePath)
	}
	tw.Flush()
}

// formatFilesText formats CLIFile results as aligned columns.
func formatFilesText(w io.Writer, files []CLIFile) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tPATH\tLANGUAGE")
	for _, f := range files {
		fmt.Fprintf(tw, "%d\t%s\t%s\n", f.ID, f.Path, f.Language)
	}
	tw.Flush()
}

// formatSummaryText formats CLIProjectSummary as readable text.
func formatSummaryText(w io.Writer, summary CLIProjectSummary) {
	fmt.Fprintln(w, "Project Summary")
	fmt.Fprintln(w, "===============")
	fmt.Fprintf(w, "Packages: %d\n", summary.PackageCount)
	fmt.Fprintln(w)

	if len(summary.Languages) > 0 {
		fmt.Fprintln(w, "Languages:")
		for _, lang := range summary.Languages {
			fmt.Fprintf(w, "  %s: %d files, %d symbols\n",
				lang.Language, lang.FileCount, lang.SymbolCount)
		}
		fmt.Fprintln(w)
	}

	if len(summary.TopSymbols) > 0 {
		fmt.Fprintln(w, "Top Symbols by References:")
		for _, sym := range summary.TopSymbols {
			fmt.Fprintf(w, "  %s (%s) - %d refs\n",
				sym.Name, sym.Kind, sym.RefCount)
		}
	}
}

// formatPackageSummaryText formats CLIPackageSummary as readable text.
func formatPackageSummaryText(w io.Writer, pkg CLIPackageSummary) {
	fmt.Fprintf(w, "Package: %s\n", pkg.Symbol.Name)
	fmt.Fprintf(w, "Path: %s\n", pkg.Path)
	fmt.Fprintf(w, "Files: %d\n", pkg.FileCount)
	fmt.Fprintln(w)

	if len(pkg.KindCounts) > 0 {
		fmt.Fprintln(w, "Symbol Kinds:")
		kinds := make([]string, 0, len(pkg.KindCounts))
		for kind := range pkg.KindCounts {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		for _, kind := range kinds {
			fmt.Fprintf(w, "  %s: %d\n", kind, pkg.KindCounts[kind])
		}
		fmt.Fprintln(w)
	}

	if len(pkg.ExportedSymbols) > 0 {
		fmt.Fprintln(w, "Exported Symbols:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ID\tNAME\tKIND")
		for _, s := range pkg.ExportedSymbols {
			fmt.Fprintf(tw, "  %d\t%s\t%s\n", s.ID, s.Name, s.Kind)
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	if len(pkg.Dependencies) > 0 {
		fmt.Fprintln(w, "Dependencies:")
		for _, dep := range pkg.Dependencies {
			fmt.Fprintf(w, "  %s\n", dep)
		}
		fmt.Fprintln(w)
	}

	if len(pkg.Dependents) > 0 {
		fmt.Fprintln(w, "Dependents:")
		for _, dep := range pkg.Dependents {
			fmt.Fprintf(w, "  %s\n", dep)
		}
	}
}

// outputResultText dispatches to the appropriate text formatter based on the
// result type. It writes to os.Stdout.
func outputResultText(result CLIResult) error {
	w := io.Writer(os.Stdout)

	switch v := result.Results.(type) {
	case []CLILocation:
		formatLocationsText(w, v)
	case []CLISymbol:
		formatSymbolsText(w, v)
	case CLISymbol:
		formatSymbolsText(w, []CLISymbol{v})
	case []CLICallEdge:
		formatCallEdgesText(w, v)
	case []CLIImport:
		formatImportsText(w, v)
	case []CLIFile:
		formatFilesText(w, v)
	case CLIProjectSummary:
		formatSummaryText(w, v)
	case CLIPackageSummary:
		formatPackageSummaryText(w, v)
	case nil:
		// No output for nil results (e.g., symbol-at with no match).
	default:
		return fmt.Errorf("unsupported result type for text format: %T", v)
	}

	// Pagination footer.
	if result.TotalCount != nil {
		count := *result.TotalCount
		shown := resultLen(result.Results)
		if shown < count {
			fmt.Fprintf(w, "\nShowing %d of %d results\n", shown, count)
		}
	}

	return nil
}

// resultLen returns the length of a result slice, or 1 for a single value.
func resultLen(v any) int {
	switch r := v.(type) {
	case []CLILocation:
		return len(r)
	case []CLISymbol:
		return len(r)
	case []CLICallEdge:
		return len(r)
	case []CLIImport:
		return len(r)
	case []CLIFile:
		return len(r)
	case nil:
		return 0
	default:
		return 1
	}
}

// validFormats lists accepted values for --format.
var validFormats = []string{"json", "text"}

// validateFormat checks that the --format flag value is recognized.
func validateFormat(format string) error {
	for _, f := range validFormats {
		if format == f {
			return nil
		}
	}
	return fmt.Errorf("invalid format %q: must be %s", format, strings.Join(validFormats, " or "))
}
