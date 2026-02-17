package main

import (
	"fmt"

	"github.com/jward/canopy"
	"github.com/spf13/cobra"
)

// --- Detail Commands ---

var symbolDetailCmd = &cobra.Command{
	Use:   "symbol-detail [<file> <line> <col>]",
	Short: "Get detailed metadata for a symbol",
	Long:  "Returns symbol info plus parameters, members, type params, and annotations.\nAccepts either <file> <line> <col> positional args or --symbol <id>.",
	Args:  cobra.MaximumNArgs(3),
	RunE:  runSymbolDetail,
}

func init() {
	symbolDetailCmd.Flags().Int64("symbol", 0, "symbol ID to query")
}

func runSymbolDetail(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("symbol-detail", err)
	}
	defer s.Close()

	qb := canopy.NewQueryBuilder(s)

	var detail *canopy.SymbolDetail

	if cmd.Flags().Changed("symbol") {
		symbolFlag, _ := cmd.Flags().GetInt64("symbol")
		detail, err = qb.SymbolDetail(symbolFlag)
	} else {
		if len(args) < 3 {
			return outputError("symbol-detail", fmt.Errorf("requires either <file> <line> <col> arguments or --symbol flag"))
		}
		file, fileErr := resolveFilePath(args[0])
		if fileErr != nil {
			return outputError("symbol-detail", fileErr)
		}
		line, lineErr := parseIntArg(args[1], "line")
		if lineErr != nil {
			return outputError("symbol-detail", lineErr)
		}
		col, colErr := parseIntArg(args[2], "col")
		if colErr != nil {
			return outputError("symbol-detail", colErr)
		}
		detail, err = qb.SymbolDetailAt(file, line, col)
	}

	if err != nil {
		return outputError("symbol-detail", err)
	}

	if detail == nil {
		return outputResult(CLIResult{
			Command: "symbol-detail",
			Results: nil,
		})
	}

	cliDetail := symbolDetailToCLI(detail)
	one := 1
	return outputResult(CLIResult{
		Command:    "symbol-detail",
		Results:    cliDetail,
		TotalCount: &one,
	})
}

var scopeAtCmd = &cobra.Command{
	Use:   "scope-at <file> <line> <col>",
	Short: "Get the scope chain at a position",
	Long:  "Returns scopes from innermost to outermost at the given position.\nLine and col are 0-based.",
	Args:  cobra.ExactArgs(3),
	RunE:  runScopeAt,
}

func runScopeAt(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return outputError("scope-at", err)
	}
	defer s.Close()

	file, err := resolveFilePath(args[0])
	if err != nil {
		return outputError("scope-at", err)
	}
	line, err := parseIntArg(args[1], "line")
	if err != nil {
		return outputError("scope-at", err)
	}
	col, err := parseIntArg(args[2], "col")
	if err != nil {
		return outputError("scope-at", err)
	}

	qb := canopy.NewQueryBuilder(s)
	scopes, err := qb.ScopeAt(file, line, col)
	if err != nil {
		return outputError("scope-at", err)
	}

	cliScopes := make([]CLIScope, len(scopes))
	for i, sc := range scopes {
		cliScopes[i] = CLIScope{
			ID:        sc.ID,
			Kind:      sc.Kind,
			StartLine: sc.StartLine,
			StartCol:  sc.StartCol,
			EndLine:   sc.EndLine,
			EndCol:    sc.EndCol,
			SymbolID:  sc.SymbolID,
		}
	}

	scopeCount := len(cliScopes)
	return outputResult(CLIResult{
		Command:    "scope-at",
		Results:    cliScopes,
		TotalCount: &scopeCount,
	})
}

// symbolDetailToCLI converts a canopy.SymbolDetail to a CLISymbolDetail.
func symbolDetailToCLI(d *canopy.SymbolDetail) CLISymbolDetail {
	cli := CLISymbolDetail{
		Symbol: symbolResultToCLI(d.Symbol),
	}

	cli.Parameters = make([]CLIFunctionParam, len(d.Parameters))
	for i, p := range d.Parameters {
		cli.Parameters[i] = CLIFunctionParam{
			Name:       p.Name,
			Ordinal:    p.Ordinal,
			TypeExpr:   p.TypeExpr,
			IsReceiver: p.IsReceiver,
			IsReturn:   p.IsReturn,
			HasDefault: p.HasDefault,
		}
	}

	cli.Members = make([]CLITypeMember, len(d.Members))
	for i, m := range d.Members {
		cli.Members[i] = CLITypeMember{
			Name:       m.Name,
			Kind:       m.Kind,
			TypeExpr:   m.TypeExpr,
			Visibility: m.Visibility,
		}
	}

	cli.TypeParams = make([]CLITypeParam, len(d.TypeParams))
	for i, tp := range d.TypeParams {
		cli.TypeParams[i] = CLITypeParam{
			Name:        tp.Name,
			Ordinal:     tp.Ordinal,
			Constraints: tp.Constraints,
		}
	}

	cli.Annotations = make([]CLIAnnotation, len(d.Annotations))
	for i, a := range d.Annotations {
		cli.Annotations[i] = CLIAnnotation{
			Name:      a.Name,
			Arguments: a.Arguments,
		}
	}

	return cli
}
