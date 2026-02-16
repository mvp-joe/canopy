package canopy

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jward/canopy/internal/store"
)

// --- Common Types ---

// Pagination controls offset+limit paging on list/search results.
type Pagination struct {
	Offset int // skip this many results (default 0)
	Limit  int // max results to return (default 50, max 500)
}

const (
	defaultLimit = 50
	maxLimit     = 500
)

// normalize returns a Pagination with defaults applied and bounds enforced.
func (p Pagination) normalize() Pagination {
	if p.Offset < 0 {
		p.Offset = 0
	}
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	return p
}

// SortField specifies how to order results.
type SortField string

const (
	SortByName     SortField = "name"
	SortByKind     SortField = "kind"
	SortByFile     SortField = "file"
	SortByRefCount         SortField = "ref_count"
	SortByExternalRefCount SortField = "external_ref_count"
)

// SortOrder specifies ascending or descending.
type SortOrder string

const (
	Asc  SortOrder = "asc"
	Desc SortOrder = "desc"
)

// Sort controls result ordering.
type Sort struct {
	Field SortField
	Order SortOrder
}

// SymbolResult extends Symbol with computed fields useful for discovery.
type SymbolResult struct {
	store.Symbol
	FilePath         string // resolved file path (empty for multi-file symbols)
	RefCount         int    // total resolved references targeting this symbol
	ExternalRefCount int    // refs from other files
	InternalRefCount int    // refs from the same file as the symbol definition
}

// PagedResult wraps a page of results with total count for pagination.
type PagedResult[T any] struct {
	Items      []T
	TotalCount int // total matching results (before pagination)
}

// SymbolFilter specifies which symbols to include.
type SymbolFilter struct {
	Kinds      []string // match any of these kinds
	Visibility *string  // exact match
	Modifiers  []string // symbol must have ALL of these modifiers
	FileID     *int64   // restrict to a single file
	ParentID   *int64   // restrict to direct children of this symbol
	PathPrefix *string  // restrict to symbols in files under this path
}

// --- Internal Helpers ---

// normalizePathPrefix ensures a path prefix ends with "/" for correct LIKE matching.
// "internal/store" -> "internal/store/" to prevent matching "internal/store_utils/".
func normalizePathPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	if !strings.HasSuffix(prefix, "/") {
		return prefix + "/"
	}
	return prefix
}

// symbolSortColumn returns the SQL ORDER BY expression for symbol queries.
// Falls back to "s.name" for unknown fields.
func symbolSortColumn(field SortField) string {
	switch field {
	case SortByName:
		return "s.name"
	case SortByKind:
		return "s.kind"
	case SortByFile:
		return "f.path"
	case SortByRefCount:
		return "ref_count"
	case SortByExternalRefCount:
		return "external_ref_count"
	default:
		return "s.name"
	}
}

// fileSortColumn returns the SQL ORDER BY expression for file queries.
// Falls back to "path" for inapplicable fields.
func fileSortColumn(field SortField) string {
	switch field {
	case SortByFile, SortByName:
		return "path"
	default:
		return "path"
	}
}

// sortDirection returns "ASC" or "DESC".
func sortDirection(order SortOrder) string {
	if order == Desc {
		return "DESC"
	}
	return "ASC"
}

// --- Enumeration Endpoints ---

// Symbols is the primary listing/filtering endpoint. All filter fields are optional.
func (q *QueryBuilder) Symbols(filter SymbolFilter, sort Sort, page Pagination) (*PagedResult[SymbolResult], error) {
	page = page.normalize()

	var where []string
	var args []any

	if len(filter.Kinds) > 0 {
		placeholders := strings.Repeat("?,", len(filter.Kinds)-1) + "?"
		where = append(where, "s.kind IN ("+placeholders+")")
		for _, k := range filter.Kinds {
			args = append(args, k)
		}
	}
	if filter.Visibility != nil {
		where = append(where, "s.visibility = ?")
		args = append(args, *filter.Visibility)
	}
	if filter.FileID != nil {
		where = append(where, "s.file_id = ?")
		args = append(args, *filter.FileID)
	}
	if filter.ParentID != nil {
		where = append(where, "s.parent_symbol_id = ?")
		args = append(args, *filter.ParentID)
	}
	if filter.PathPrefix != nil {
		prefix := normalizePathPrefix(*filter.PathPrefix)
		if prefix != "" {
			where = append(where, "f.path LIKE ? ESCAPE '\\'")
			args = append(args, escapeLike(prefix)+"%")
		}
	}
	// Modifier filtering using json_each
	for _, mod := range filter.Modifiers {
		where = append(where, "EXISTS (SELECT 1 FROM json_each(s.modifiers) WHERE json_each.value = ?)")
		args = append(args, mod)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count query
	countSQL := `SELECT COUNT(*) FROM symbols s LEFT JOIN files f ON s.file_id = f.id ` + whereClause
	var totalCount int
	if err := q.store.DB().QueryRow(countSQL, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("symbols: count: %w", err)
	}

	// Data query with ref count subquery
	orderCol := symbolSortColumn(sort.Field)
	orderDir := sortDirection(sort.Order)

	dataSQL := fmt.Sprintf(
		`SELECT %s, COALESCE(f.path, '') AS file_path,
			(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
			(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count
		 FROM symbols s
		 LEFT JOIN files f ON s.file_id = f.id
		 %s
		 ORDER BY %s %s
		 LIMIT ? OFFSET ?`,
		prefixSymbolCols("s"), whereClause, orderCol, orderDir,
	)
	dataArgs := append(append([]any{}, args...), page.Limit, page.Offset)

	rows, err := q.store.DB().Query(dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("symbols: query: %w", err)
	}
	defer rows.Close()

	var items []SymbolResult
	for rows.Next() {
		sr, err := scanSymbolResult(rows)
		if err != nil {
			return nil, fmt.Errorf("symbols: scan: %w", err)
		}
		items = append(items, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("symbols: rows: %w", err)
	}
	if items == nil {
		items = []SymbolResult{}
	}

	return &PagedResult[SymbolResult]{Items: items, TotalCount: totalCount}, nil
}

// Files is a convenience method for listing files.
func (q *QueryBuilder) Files(pathPrefix string, language string, sort Sort, page Pagination) (*PagedResult[store.File], error) {
	page = page.normalize()

	var where []string
	var args []any

	if pathPrefix != "" {
		prefix := normalizePathPrefix(pathPrefix)
		where = append(where, "path LIKE ? ESCAPE '\\'")
		args = append(args, escapeLike(prefix)+"%")
	}
	if language != "" {
		where = append(where, "language = ?")
		args = append(args, language)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count
	countSQL := "SELECT COUNT(*) FROM files " + whereClause
	var totalCount int
	if err := q.store.DB().QueryRow(countSQL, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("files: count: %w", err)
	}

	// Data
	orderCol := fileSortColumn(sort.Field)
	orderDir := sortDirection(sort.Order)

	dataSQL := fmt.Sprintf(
		`SELECT id, path, language, hash, last_indexed FROM files %s ORDER BY %s %s LIMIT ? OFFSET ?`,
		whereClause, orderCol, orderDir,
	)
	dataArgs := append(append([]any{}, args...), page.Limit, page.Offset)

	rows, err := q.store.DB().Query(dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("files: query: %w", err)
	}
	defer rows.Close()

	var items []store.File
	for rows.Next() {
		var f store.File
		if err := rows.Scan(&f.ID, &f.Path, &f.Language, &f.Hash, &f.LastIndexed); err != nil {
			return nil, fmt.Errorf("files: scan: %w", err)
		}
		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("files: rows: %w", err)
	}
	if items == nil {
		items = []store.File{}
	}

	return &PagedResult[store.File]{Items: items, TotalCount: totalCount}, nil
}

// Packages is a convenience method for listing packages, modules, and namespaces.
func (q *QueryBuilder) Packages(pathPrefix string, sort Sort, page Pagination) (*PagedResult[SymbolResult], error) {
	filter := SymbolFilter{
		Kinds: []string{"package", "module", "namespace"},
	}
	if pathPrefix != "" {
		filter.PathPrefix = &pathPrefix
	}
	return q.Symbols(filter, sort, page)
}

// --- Search ---

// SearchSymbols performs glob-style search on symbol names.
// '*' is the wildcard (mapped to SQL '%').
func (q *QueryBuilder) SearchSymbols(pattern string, filter SymbolFilter, sort Sort, page Pagination) (*PagedResult[SymbolResult], error) {
	page = page.normalize()

	var where []string
	var args []any

	// Pattern matching: escape literal % and _ first, then convert * to %
	if pattern != "" && pattern != "*" {
		likePattern := escapeLike(pattern)
		likePattern = strings.ReplaceAll(likePattern, "*", "%")
		where = append(where, "s.name LIKE ? ESCAPE '\\'")
		args = append(args, likePattern)
	}

	// Apply the same structured filters as Symbols
	if len(filter.Kinds) > 0 {
		placeholders := strings.Repeat("?,", len(filter.Kinds)-1) + "?"
		where = append(where, "s.kind IN ("+placeholders+")")
		for _, k := range filter.Kinds {
			args = append(args, k)
		}
	}
	if filter.Visibility != nil {
		where = append(where, "s.visibility = ?")
		args = append(args, *filter.Visibility)
	}
	if filter.FileID != nil {
		where = append(where, "s.file_id = ?")
		args = append(args, *filter.FileID)
	}
	if filter.ParentID != nil {
		where = append(where, "s.parent_symbol_id = ?")
		args = append(args, *filter.ParentID)
	}
	if filter.PathPrefix != nil {
		prefix := normalizePathPrefix(*filter.PathPrefix)
		if prefix != "" {
			where = append(where, "f.path LIKE ? ESCAPE '\\'")
			args = append(args, escapeLike(prefix)+"%")
		}
	}
	for _, mod := range filter.Modifiers {
		where = append(where, "EXISTS (SELECT 1 FROM json_each(s.modifiers) WHERE json_each.value = ?)")
		args = append(args, mod)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count
	countSQL := `SELECT COUNT(*) FROM symbols s LEFT JOIN files f ON s.file_id = f.id ` + whereClause
	var totalCount int
	if err := q.store.DB().QueryRow(countSQL, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("search symbols: count: %w", err)
	}

	// Data
	orderCol := symbolSortColumn(sort.Field)
	orderDir := sortDirection(sort.Order)

	dataSQL := fmt.Sprintf(
		`SELECT %s, COALESCE(f.path, '') AS file_path,
			(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
			(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count
		 FROM symbols s
		 LEFT JOIN files f ON s.file_id = f.id
		 %s
		 ORDER BY %s %s
		 LIMIT ? OFFSET ?`,
		prefixSymbolCols("s"), whereClause, orderCol, orderDir,
	)
	dataArgs := append(append([]any{}, args...), page.Limit, page.Offset)

	rows, err := q.store.DB().Query(dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("search symbols: query: %w", err)
	}
	defer rows.Close()

	var items []SymbolResult
	for rows.Next() {
		sr, err := scanSymbolResult(rows)
		if err != nil {
			return nil, fmt.Errorf("search symbols: scan: %w", err)
		}
		items = append(items, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search symbols: rows: %w", err)
	}
	if items == nil {
		items = []SymbolResult{}
	}

	return &PagedResult[SymbolResult]{Items: items, TotalCount: totalCount}, nil
}

// --- Digest Endpoints ---

// LanguageStats provides per-language breakdown for ProjectSummary.
type LanguageStats struct {
	Language    string
	FileCount   int
	SymbolCount int
	KindCounts  map[string]int
}

// ProjectSummary provides a high-level overview of the indexed codebase.
type ProjectSummary struct {
	Languages    []LanguageStats
	PackageCount int
	TopSymbols   []SymbolResult
}

// ProjectSummary returns a high-level overview of the entire indexed codebase.
func (q *QueryBuilder) ProjectSummary(topN int) (*ProjectSummary, error) {
	summary := &ProjectSummary{}

	// Language stats: file count per language
	langRows, err := q.store.DB().Query(
		`SELECT language, COUNT(*) FROM files GROUP BY language ORDER BY language`,
	)
	if err != nil {
		return nil, fmt.Errorf("project summary: languages: %w", err)
	}
	defer langRows.Close()

	var languages []LanguageStats
	for langRows.Next() {
		var ls LanguageStats
		if err := langRows.Scan(&ls.Language, &ls.FileCount); err != nil {
			return nil, fmt.Errorf("project summary: scan language: %w", err)
		}
		languages = append(languages, ls)
	}
	if err := langRows.Err(); err != nil {
		return nil, fmt.Errorf("project summary: language rows: %w", err)
	}

	// For each language, get symbol count and kind breakdown
	for i := range languages {
		lang := &languages[i]

		// Symbol count and kind counts
		kindRows, err := q.store.DB().Query(
			`SELECT s.kind, COUNT(*) FROM symbols s
			 JOIN files f ON s.file_id = f.id
			 WHERE f.language = ?
			 GROUP BY s.kind`,
			lang.Language,
		)
		if err != nil {
			return nil, fmt.Errorf("project summary: kind counts for %s: %w", lang.Language, err)
		}

		lang.KindCounts = make(map[string]int)
		for kindRows.Next() {
			var kind string
			var count int
			if err := kindRows.Scan(&kind, &count); err != nil {
				kindRows.Close()
				return nil, fmt.Errorf("project summary: scan kind: %w", err)
			}
			lang.KindCounts[kind] = count
			lang.SymbolCount += count
		}
		kindRows.Close()
		if err := kindRows.Err(); err != nil {
			return nil, fmt.Errorf("project summary: kind rows: %w", err)
		}
	}

	summary.Languages = languages
	if summary.Languages == nil {
		summary.Languages = []LanguageStats{}
	}

	// Package count
	err = q.store.DB().QueryRow(
		`SELECT COUNT(*) FROM symbols WHERE kind IN ('package', 'module', 'namespace')`,
	).Scan(&summary.PackageCount)
	if err != nil {
		return nil, fmt.Errorf("project summary: package count: %w", err)
	}

	// Top-N symbols by ref count
	if topN > 0 {
		topSQL := fmt.Sprintf(
			`SELECT %s, COALESCE(f.path, '') AS file_path,
				(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
				(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count
			 FROM symbols s
			 LEFT JOIN files f ON s.file_id = f.id
			 WHERE (SELECT COUNT(*) FROM resolved_references rr2 WHERE rr2.target_symbol_id = s.id) > 0
			 ORDER BY external_ref_count DESC
			 LIMIT ?`,
			prefixSymbolCols("s"),
		)
		topRows, err := q.store.DB().Query(topSQL, topN)
		if err != nil {
			return nil, fmt.Errorf("project summary: top symbols: %w", err)
		}
		defer topRows.Close()

		for topRows.Next() {
			sr, err := scanSymbolResult(topRows)
			if err != nil {
				return nil, fmt.Errorf("project summary: scan top symbol: %w", err)
			}
			summary.TopSymbols = append(summary.TopSymbols, sr)
		}
		if err := topRows.Err(); err != nil {
			return nil, fmt.Errorf("project summary: top rows: %w", err)
		}
	}
	if summary.TopSymbols == nil {
		summary.TopSymbols = []SymbolResult{}
	}

	return summary, nil
}

// PackageSummary provides a summary of a single package/module.
type PackageSummary struct {
	Symbol          SymbolResult
	Path            string
	FileCount       int
	ExportedSymbols []SymbolResult
	KindCounts      map[string]int
	Dependencies    []string
	Dependents      []string
}

// PackageSummary returns a summary of a single package/module.
// Accepts either a file path prefix or a symbol ID. If packageID is non-nil, it is used directly.
func (q *QueryBuilder) PackageSummary(packagePath string, packageID *int64) (*PackageSummary, error) {
	var symID int64

	if packageID != nil {
		symID = *packageID
	} else if packagePath != "" {
		// Resolve path to package symbol ID:
		// Find files under this path, then locate the package/module/namespace symbol in those files.
		prefix := normalizePathPrefix(packagePath)
		row := q.store.DB().QueryRow(
			`SELECT s.id FROM symbols s
			 JOIN files f ON s.file_id = f.id
			 WHERE f.path LIKE ? ESCAPE '\'
			   AND s.kind IN ('package', 'module', 'namespace')
			 LIMIT 1`,
			escapeLike(prefix)+"%",
		)
		if err := row.Scan(&symID); err == sql.ErrNoRows {
			return nil, fmt.Errorf("package summary: no package found at path %q", packagePath)
		} else if err != nil {
			return nil, fmt.Errorf("package summary: resolve path: %w", err)
		}
	} else {
		return nil, fmt.Errorf("package summary: either packagePath or packageID must be provided")
	}

	// Load the package symbol itself
	symRow := q.store.DB().QueryRow(
		fmt.Sprintf(
			`SELECT %s, COALESCE(f.path, '') AS file_path,
				(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
				(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count
			 FROM symbols s
			 LEFT JOIN files f ON s.file_id = f.id
			 WHERE s.id = ?`,
			prefixSymbolCols("s"),
		),
		symID,
	)
	pkgSymbol, err := scanSymbolResult(symRow)
	if err != nil {
		return nil, fmt.Errorf("package summary: load symbol: %w", err)
	}

	summary := &PackageSummary{
		Symbol: pkgSymbol,
		Path:   pkgSymbol.FilePath,
	}

	// Determine the package's file path prefix.
	// If the symbol has a file, use its directory as the prefix.
	var pathPrefix string
	if pkgSymbol.FilePath != "" {
		// Use the directory containing the file
		idx := strings.LastIndex(pkgSymbol.FilePath, "/")
		if idx >= 0 {
			pathPrefix = pkgSymbol.FilePath[:idx+1]
		}
	}
	if pathPrefix == "" && packagePath != "" {
		pathPrefix = normalizePathPrefix(packagePath)
	}

	// File count
	if pathPrefix != "" {
		err = q.store.DB().QueryRow(
			`SELECT COUNT(*) FROM files WHERE path LIKE ? ESCAPE '\'`,
			escapeLike(pathPrefix)+"%",
		).Scan(&summary.FileCount)
		if err != nil {
			return nil, fmt.Errorf("package summary: file count: %w", err)
		}
	}

	// Exported symbols (public visibility) within this package's files, sorted by ref count desc
	if pathPrefix != "" {
		expSQL := fmt.Sprintf(
			`SELECT %s, COALESCE(f.path, '') AS file_path,
				(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
				(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count
			 FROM symbols s
			 JOIN files f ON s.file_id = f.id
			 WHERE f.path LIKE ? ESCAPE '\'
			   AND s.visibility = 'public'
			   AND s.kind NOT IN ('package', 'module', 'namespace')
			 ORDER BY external_ref_count DESC`,
			prefixSymbolCols("s"),
		)
		expRows, err := q.store.DB().Query(expSQL, escapeLike(pathPrefix)+"%")
		if err != nil {
			return nil, fmt.Errorf("package summary: exported symbols: %w", err)
		}
		defer expRows.Close()

		for expRows.Next() {
			sr, err := scanSymbolResult(expRows)
			if err != nil {
				return nil, fmt.Errorf("package summary: scan exported: %w", err)
			}
			summary.ExportedSymbols = append(summary.ExportedSymbols, sr)
		}
		if err := expRows.Err(); err != nil {
			return nil, fmt.Errorf("package summary: exported rows: %w", err)
		}
	}
	if summary.ExportedSymbols == nil {
		summary.ExportedSymbols = []SymbolResult{}
	}

	// Kind counts for symbols in this package
	summary.KindCounts = make(map[string]int)
	if pathPrefix != "" {
		kindRows, err := q.store.DB().Query(
			`SELECT s.kind, COUNT(*) FROM symbols s
			 JOIN files f ON s.file_id = f.id
			 WHERE f.path LIKE ? ESCAPE '\'
			 GROUP BY s.kind`,
			escapeLike(pathPrefix)+"%",
		)
		if err != nil {
			return nil, fmt.Errorf("package summary: kind counts: %w", err)
		}
		defer kindRows.Close()

		for kindRows.Next() {
			var kind string
			var count int
			if err := kindRows.Scan(&kind, &count); err != nil {
				return nil, fmt.Errorf("package summary: scan kind: %w", err)
			}
			summary.KindCounts[kind] = count
		}
		if err := kindRows.Err(); err != nil {
			return nil, fmt.Errorf("package summary: kind rows: %w", err)
		}
	}

	// Dependencies: import sources from files in this package
	summary.Dependencies = []string{}
	if pathPrefix != "" {
		depRows, err := q.store.DB().Query(
			`SELECT DISTINCT i.source FROM imports i
			 JOIN files f ON i.file_id = f.id
			 WHERE f.path LIKE ? ESCAPE '\'
			 ORDER BY i.source`,
			escapeLike(pathPrefix)+"%",
		)
		if err != nil {
			return nil, fmt.Errorf("package summary: dependencies: %w", err)
		}
		defer depRows.Close()

		for depRows.Next() {
			var source string
			if err := depRows.Scan(&source); err != nil {
				return nil, fmt.Errorf("package summary: scan dep: %w", err)
			}
			summary.Dependencies = append(summary.Dependencies, source)
		}
		if err := depRows.Err(); err != nil {
			return nil, fmt.Errorf("package summary: dep rows: %w", err)
		}
	}

	// Dependents: packages that import this package
	// Use the package symbol name as the import source to find
	summary.Dependents = []string{}
	pkgName := pkgSymbol.Name
	if pkgName != "" {
		// Find files that import this package name (direct or as suffix)
		dentRows, err := q.store.DB().Query(
			`SELECT DISTINCT f.path FROM imports i
			 JOIN files f ON i.file_id = f.id
			 WHERE i.source = ? OR i.source LIKE ?
			 ORDER BY f.path`,
			pkgName, "%/"+pkgName,
		)
		if err != nil {
			return nil, fmt.Errorf("package summary: dependents: %w", err)
		}
		defer dentRows.Close()

		for dentRows.Next() {
			var path string
			if err := dentRows.Scan(&path); err != nil {
				return nil, fmt.Errorf("package summary: scan dependent: %w", err)
			}
			// Exclude files that are within this package itself
			if pathPrefix != "" && strings.HasPrefix(path, pathPrefix) {
				continue
			}
			summary.Dependents = append(summary.Dependents, path)
		}
		if err := dentRows.Err(); err != nil {
			return nil, fmt.Errorf("package summary: dependent rows: %w", err)
		}
	}

	return summary, nil
}

// --- Scan Helpers ---

// prefixSymbolCols returns the SymbolCols with a table prefix applied.
func prefixSymbolCols(prefix string) string {
	cols := []string{
		"id", "file_id", "name", "kind", "visibility", "modifiers", "signature_hash",
		"start_line", "start_col", "end_line", "end_col", "parent_symbol_id",
	}
	prefixed := make([]string, len(cols))
	for i, c := range cols {
		prefixed[i] = prefix + "." + c
	}
	return strings.Join(prefixed, ", ")
}

// scanSymbolResult scans a row into a SymbolResult.
// Expects columns: [SymbolCols..., file_path, ref_count].
type scanner interface {
	Scan(dest ...any) error
}

func scanSymbolResult(row scanner) (SymbolResult, error) {
	var sr SymbolResult
	var mods string
	err := row.Scan(
		&sr.ID, &sr.FileID, &sr.Name, &sr.Kind, &sr.Visibility, &mods,
		&sr.SignatureHash, &sr.StartLine, &sr.StartCol, &sr.EndLine, &sr.EndCol,
		&sr.ParentSymbolID,
		&sr.FilePath, &sr.RefCount, &sr.ExternalRefCount,
	)
	if err != nil {
		return sr, err
	}
	sr.Modifiers = store.UnmarshalModifiers(mods)
	sr.InternalRefCount = sr.RefCount - sr.ExternalRefCount
	return sr, nil
}

// escapeLike escapes SQL LIKE special characters (% and _) with backslash.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
