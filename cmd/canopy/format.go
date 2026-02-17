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
	// Show ref count columns if any symbol has refs.
	hasRefs := false
	for _, s := range syms {
		if s.RefCount > 0 {
			hasRefs = true
			break
		}
	}
	if hasRefs {
		fmt.Fprintln(tw, "ID\tNAME\tKIND\tVISIBILITY\tREFS (EXT/TOTAL)\tFILE\tLINE")
		for _, s := range syms {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%d/%d\t%s\t%d\n",
				s.ID, s.Name, s.Kind, s.Visibility, s.ExternalRefCount, s.RefCount, s.File, s.StartLine)
		}
	} else {
		fmt.Fprintln(tw, "ID\tNAME\tKIND\tVISIBILITY\tFILE\tLINE")
		for _, s := range syms {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%d\n",
				s.ID, s.Name, s.Kind, s.Visibility, s.File, s.StartLine)
		}
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
		fmt.Fprintln(w, "Top Symbols by External References:")
		for _, sym := range summary.TopSymbols {
			fmt.Fprintf(w, "  %s (%s) - %d ext / %d total refs\n",
				sym.Name, sym.Kind, sym.ExternalRefCount, sym.RefCount)
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

// formatSymbolDetailText formats a CLISymbolDetail as readable text.
func formatSymbolDetailText(w io.Writer, detail CLISymbolDetail) {
	formatSymbolsText(w, []CLISymbol{detail.Symbol})

	if len(detail.Parameters) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Parameters:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ORD\tNAME\tTYPE\tFLAGS")
		for _, p := range detail.Parameters {
			var flags []string
			if p.IsReceiver {
				flags = append(flags, "receiver")
			}
			if p.IsReturn {
				flags = append(flags, "return")
			}
			if p.HasDefault {
				flags = append(flags, "default")
			}
			fmt.Fprintf(tw, "  %d\t%s\t%s\t%s\n",
				p.Ordinal, p.Name, p.TypeExpr, strings.Join(flags, ","))
		}
		tw.Flush()
	}

	if len(detail.Members) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Members:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tKIND\tTYPE\tVISIBILITY")
		for _, m := range detail.Members {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
				m.Name, m.Kind, m.TypeExpr, m.Visibility)
		}
		tw.Flush()
	}

	if len(detail.TypeParams) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Type Parameters:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ORD\tNAME\tCONSTRAINTS")
		for _, tp := range detail.TypeParams {
			fmt.Fprintf(tw, "  %d\t%s\t%s\n",
				tp.Ordinal, tp.Name, tp.Constraints)
		}
		tw.Flush()
	}

	if len(detail.Annotations) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Annotations:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  NAME\tARGUMENTS")
		for _, a := range detail.Annotations {
			fmt.Fprintf(tw, "  %s\t%s\n", a.Name, a.Arguments)
		}
		tw.Flush()
	}
}

// formatScopesText formats CLIScope results as aligned columns.
func formatScopesText(w io.Writer, scopes []CLIScope) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tKIND\tSTART\tEND\tSYMBOL_ID")
	for _, sc := range scopes {
		symID := ""
		if sc.SymbolID != nil {
			symID = fmt.Sprintf("%d", *sc.SymbolID)
		}
		fmt.Fprintf(tw, "%d\t%s\t%d:%d\t%d:%d\t%s\n",
			sc.ID, sc.Kind, sc.StartLine, sc.StartCol, sc.EndLine, sc.EndCol, symID)
	}
	tw.Flush()
}

// formatTypeHierarchyText formats a CLITypeHierarchy as readable text.
func formatTypeHierarchyText(w io.Writer, th CLITypeHierarchy) {
	formatSymbolsText(w, []CLISymbol{th.Symbol})

	sections := []struct {
		label string
		rels  []CLITypeRelation
	}{
		{"Implements", th.Implements},
		{"Implemented By", th.ImplementedBy},
		{"Composes", th.Composes},
		{"Composed By", th.ComposedBy},
	}
	for _, sec := range sections {
		if len(sec.rels) == 0 {
			continue
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s:\n", sec.label)
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ID\tNAME\tKIND\tRELATION_KIND")
		for _, r := range sec.rels {
			fmt.Fprintf(tw, "  %d\t%s\t%s\t%s\n",
				r.Symbol.ID, r.Symbol.Name, r.Symbol.Kind, r.Kind)
		}
		tw.Flush()
	}

	if len(th.Extensions) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Extensions:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  TYPE_SYMBOL_ID\tMEMBER_SYMBOL_ID\tKIND\tSOURCE_FILE_ID")
		for _, b := range th.Extensions {
			var typeID, fileID int64
			if b.TypeSymbolID != nil {
				typeID = *b.TypeSymbolID
			}
			if b.SourceFileID != nil {
				fileID = *b.SourceFileID
			}
			fmt.Fprintf(tw, "  %d\t%d\t%s\t%d\n",
				typeID, b.MemberSymbolID, b.Kind, fileID)
		}
		tw.Flush()
	}
}

// formatExtensionBindingsText formats CLIExtensionBinding results as aligned columns.
func formatExtensionBindingsText(w io.Writer, bindings []CLIExtensionBinding) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE_SYMBOL_ID\tMEMBER_SYMBOL_ID\tKIND\tSOURCE_FILE_ID")
	for _, b := range bindings {
		var typeID, fileID int64
		if b.TypeSymbolID != nil {
			typeID = *b.TypeSymbolID
		}
		if b.SourceFileID != nil {
			fileID = *b.SourceFileID
		}
		fmt.Fprintf(tw, "%d\t%d\t%s\t%d\n",
			typeID, b.MemberSymbolID, b.Kind, fileID)
	}
	tw.Flush()
}

// formatReexportsText formats CLIReexport results as aligned columns.
func formatReexportsText(w io.Writer, reexports []CLIReexport) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ORIGINAL_SYMBOL_ID\tEXPORTED_NAME")
	for _, r := range reexports {
		fmt.Fprintf(tw, "%d\t%s\n", r.OriginalSymbolID, r.ExportedName)
	}
	tw.Flush()
}

// formatCallGraphText formats a CLICallGraph as readable text.
func formatCallGraphText(w io.Writer, g CLICallGraph) {
	fmt.Fprintf(w, "Root: %d\n", g.Root)
	fmt.Fprintf(w, "Depth: %d\n\n", g.Depth)

	if len(g.Nodes) > 0 {
		fmt.Fprintln(w, "Nodes:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  ID\tNAME\tKIND\tDEPTH")
		for _, n := range g.Nodes {
			fmt.Fprintf(tw, "  %d\t%s\t%s\t%d\n",
				n.Symbol.ID, n.Symbol.Name, n.Symbol.Kind, n.Depth)
		}
		tw.Flush()
	}

	if len(g.Edges) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Edges:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  CALLER\tCALLEE\tFILE\tLINE")
		for _, e := range g.Edges {
			fmt.Fprintf(tw, "  %d\t%d\t%s\t%d\n",
				e.CallerID, e.CalleeID, e.File, e.Line)
		}
		tw.Flush()
	}
}

// formatDependencyGraphText formats a CLIDependencyGraph as readable text.
func formatDependencyGraphText(w io.Writer, g CLIDependencyGraph) {
	if len(g.Packages) > 0 {
		fmt.Fprintln(w, "Packages:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  PACKAGE\tFILES\tLINES")
		for _, p := range g.Packages {
			fmt.Fprintf(tw, "  %s\t%d\t%d\n", p.Name, p.FileCount, p.LineCount)
		}
		tw.Flush()
	}

	if len(g.Edges) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Dependencies:")
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  FROM\tTO\tIMPORTS")
		for _, e := range g.Edges {
			fmt.Fprintf(tw, "  %s\t%s\t%d\n", e.FromPackage, e.ToPackage, e.ImportCount)
		}
		tw.Flush()
	}
}

// formatCyclesText formats []CLICycle as readable text.
func formatCyclesText(w io.Writer, cycles []CLICycle) {
	if len(cycles) == 0 {
		fmt.Fprintln(w, "No circular dependencies found.")
		return
	}
	for _, c := range cycles {
		fmt.Fprintf(w, "Cycle: %s\n", strings.Join(c.Packages, " -> "))
	}
}

// formatHotspotsText formats []CLIHotspot as aligned columns.
func formatHotspotsText(w io.Writer, hotspots []CLIHotspot) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tKIND\tREFS (EXT/TOTAL)\tCALLERS\tCALLEES")
	for _, h := range hotspots {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%d/%d\t%d\t%d\n",
			h.Symbol.ID, h.Symbol.Name, h.Symbol.Kind,
			h.Symbol.ExternalRefCount, h.Symbol.RefCount,
			h.CallerCount, h.CalleeCount)
	}
	tw.Flush()
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
	case CLISymbolDetail:
		formatSymbolDetailText(w, v)
	case []CLIScope:
		formatScopesText(w, v)
	case CLITypeHierarchy:
		formatTypeHierarchyText(w, v)
	case []CLIExtensionBinding:
		formatExtensionBindingsText(w, v)
	case []CLIReexport:
		formatReexportsText(w, v)
	case CLICallGraph:
		formatCallGraphText(w, v)
	case CLIDependencyGraph:
		formatDependencyGraphText(w, v)
	case []CLICycle:
		formatCyclesText(w, v)
	case []CLIHotspot:
		formatHotspotsText(w, v)
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
	case CLISymbolDetail:
		return 1
	case []CLIScope:
		return len(r)
	case CLITypeHierarchy:
		return 1
	case []CLIExtensionBinding:
		return len(r)
	case []CLIReexport:
		return len(r)
	case CLICallGraph:
		return 1
	case CLIDependencyGraph:
		return 1
	case []CLICycle:
		return len(r)
	case []CLIHotspot:
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
