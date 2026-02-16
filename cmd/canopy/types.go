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
	RefCount         int      `json:"ref_count,omitempty"`
	ExternalRefCount int      `json:"external_ref_count,omitempty"`
	InternalRefCount int      `json:"internal_ref_count,omitempty"`
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
	ID       int64  `json:"id"`
	Path     string `json:"path"`
	Language string `json:"language"`
}

// CLILanguageStats is a JSON-friendly language stats representation.
type CLILanguageStats struct {
	Language    string         `json:"language"`
	FileCount   int            `json:"file_count"`
	SymbolCount int            `json:"symbol_count"`
	KindCounts  map[string]int `json:"kind_counts"`
}

// CLIProjectSummary is a JSON-friendly project summary.
type CLIProjectSummary struct {
	Languages    []CLILanguageStats `json:"languages"`
	PackageCount int                `json:"package_count"`
	TopSymbols   []CLISymbol        `json:"top_symbols"`
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
