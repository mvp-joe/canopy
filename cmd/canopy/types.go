package main

// CLIResult is the top-level JSON envelope for all query commands.
type CLIResult struct {
	Command    string `json:"command"`
	Results    any    `json:"results"`
	TotalCount *int   `json:"total_count,omitempty"`
	Error      string `json:"error,omitempty"`
}

// CLISymbol is a JSON-friendly symbol representation.
type CLISymbol struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	Kind             string   `json:"kind"`
	Visibility       string   `json:"visibility"`
	Modifiers        []string `json:"modifiers,omitempty"`
	File             string   `json:"file,omitempty"`
	StartLine        int      `json:"start_line"`
	StartCol         int      `json:"start_col"`
	EndLine          int      `json:"end_line"`
	EndCol           int      `json:"end_col"`
	RefCount         int      `json:"ref_count"`
	ExternalRefCount int      `json:"external_ref_count"`
	InternalRefCount int      `json:"internal_ref_count"`
}

// CLILocation extends Location with the symbol ID for chaining.
type CLILocation struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	StartCol  int    `json:"start_col"`
	EndLine   int    `json:"end_line"`
	EndCol    int    `json:"end_col"`
	SymbolID  *int64 `json:"symbol_id,omitempty"`
}

// CLICallEdge is a JSON-friendly call graph edge.
type CLICallEdge struct {
	CallerID   int64  `json:"caller_id"`
	CallerName string `json:"caller_name,omitempty"`
	CalleeID   int64  `json:"callee_id"`
	CalleeName string `json:"callee_name,omitempty"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line"`
	Col        int    `json:"col"`
}

// CLIImport is a JSON-friendly import representation.
type CLIImport struct {
	FileID       int64   `json:"file_id"`
	FilePath     string  `json:"file_path,omitempty"`
	Source       string  `json:"source"`
	ImportedName *string `json:"imported_name,omitempty"`
	LocalAlias   *string `json:"local_alias,omitempty"`
	Kind         string  `json:"kind"`
}

// CLIFile is a JSON-friendly file representation.
type CLIFile struct {
	ID        int64  `json:"id"`
	Path      string `json:"path"`
	Language  string `json:"language"`
	LineCount int    `json:"line_count"`
}

// CLILanguageStats is a JSON-friendly language stats representation.
type CLILanguageStats struct {
	Language    string         `json:"language"`
	FileCount   int            `json:"file_count"`
	LineCount   int            `json:"line_count"`
	SymbolCount int            `json:"symbol_count"`
	KindCounts  map[string]int `json:"kind_counts"`
}

// CLIProjectSummary is a JSON-friendly project summary.
type CLIProjectSummary struct {
	Languages    []CLILanguageStats `json:"languages"`
	PackageCount int                `json:"package_count"`
	TopSymbols   []CLISymbol        `json:"top_symbols"`
}

// CLISymbolDetail is a JSON-friendly symbol detail.
type CLISymbolDetail struct {
	Symbol      CLISymbol          `json:"symbol"`
	Parameters  []CLIFunctionParam `json:"parameters"`
	Members     []CLITypeMember    `json:"members"`
	TypeParams  []CLITypeParam     `json:"type_params"`
	Annotations []CLIAnnotation    `json:"annotations"`
}

// CLIFunctionParam is a JSON-friendly function parameter.
type CLIFunctionParam struct {
	Name       string `json:"name"`
	Ordinal    int    `json:"ordinal"`
	TypeExpr   string `json:"type_expr,omitempty"`
	IsReceiver bool   `json:"is_receiver,omitempty"`
	IsReturn   bool   `json:"is_return,omitempty"`
	HasDefault bool   `json:"has_default,omitempty"`
}

// CLITypeMember is a JSON-friendly type member.
type CLITypeMember struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	TypeExpr   string `json:"type_expr,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}

// CLITypeParam is a JSON-friendly type parameter.
type CLITypeParam struct {
	Name        string `json:"name"`
	Ordinal     int    `json:"ordinal"`
	Constraints string `json:"constraints,omitempty"`
}

// CLIAnnotation is a JSON-friendly annotation.
type CLIAnnotation struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

// CLIScope is a JSON-friendly scope.
type CLIScope struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	StartLine int    `json:"start_line"`
	StartCol  int    `json:"start_col"`
	EndLine   int    `json:"end_line"`
	EndCol    int    `json:"end_col"`
	SymbolID  *int64 `json:"symbol_id,omitempty"`
}

// CLITypeHierarchy is a JSON-friendly type hierarchy.
type CLITypeHierarchy struct {
	Symbol        CLISymbol             `json:"symbol"`
	Implements    []CLITypeRelation     `json:"implements"`
	ImplementedBy []CLITypeRelation     `json:"implemented_by"`
	Composes      []CLITypeRelation     `json:"composes"`
	ComposedBy    []CLITypeRelation     `json:"composed_by"`
	Extensions    []CLIExtensionBinding `json:"extensions"`
}

// CLITypeRelation is a JSON-friendly type relationship.
type CLITypeRelation struct {
	Symbol CLISymbol `json:"symbol"`
	Kind   string    `json:"kind"`
}

// CLIExtensionBinding is a JSON-friendly extension binding.
type CLIExtensionBinding struct {
	MemberSymbolID       int64  `json:"member_symbol_id"`
	ExtendedTypeExpr     string `json:"extended_type_expr"`
	ExtendedTypeSymbolID *int64 `json:"extended_type_symbol_id,omitempty"`
	Kind                 string `json:"kind"`
	Constraints          string `json:"constraints,omitempty"`
	IsDefaultImpl        bool   `json:"is_default_impl,omitempty"`
}

// CLIReexport is a JSON-friendly reexport.
type CLIReexport struct {
	FileID           int64  `json:"file_id"`
	OriginalSymbolID int64  `json:"original_symbol_id"`
	ExportedName     string `json:"exported_name"`
}

// CLIPackageSummary is a JSON-friendly package summary.
type CLIPackageSummary struct {
	Symbol          CLISymbol      `json:"symbol"`
	Path            string         `json:"path"`
	FileCount       int            `json:"file_count"`
	ExportedSymbols []CLISymbol    `json:"exported_symbols"`
	KindCounts      map[string]int `json:"kind_counts"`
	Dependencies    []string       `json:"dependencies"`
	Dependents      []string       `json:"dependents"`
}

// CLICallGraph is a JSON-friendly transitive call graph.
type CLICallGraph struct {
	Root     int64              `json:"root"`
	Nodes    []CLICallGraphNode `json:"nodes"`
	Edges    []CLICallGraphEdge `json:"edges"`
	MaxDepth int                `json:"max_depth"`
}

// CLICallGraphNode is a node in a transitive call graph.
type CLICallGraphNode struct {
	Symbol CLISymbol `json:"symbol"`
	Depth  int       `json:"depth"`
}

// CLICallGraphEdge is an edge in a transitive call graph.
type CLICallGraphEdge struct {
	CallerID int64  `json:"caller_id"`
	CalleeID int64  `json:"callee_id"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
}

// CLIDependencyGraph is a JSON-friendly package dependency graph.
type CLIDependencyGraph struct {
	Packages []CLIPackageNode    `json:"packages"`
	Edges    []CLIDependencyEdge `json:"edges"`
}

// CLIPackageNode is a package in the dependency graph.
type CLIPackageNode struct {
	Name      string `json:"name"`
	FileCount int    `json:"file_count"`
	LineCount int    `json:"line_count"`
}

// CLIDependencyEdge is a dependency between two packages.
type CLIDependencyEdge struct {
	FromPackage string `json:"from_package"`
	ToPackage   string `json:"to_package"`
	ImportCount int    `json:"import_count"`
}

// CLICycle is a circular dependency cycle of package names.
type CLICycle struct {
	Packages []string `json:"packages"`
}

// CLIHotspot is a heavily-referenced symbol with fan-in/fan-out metrics.
type CLIHotspot struct {
	Symbol      CLISymbol `json:"symbol"`
	CallerCount int       `json:"caller_count"`
	CalleeCount int       `json:"callee_count"`
}
