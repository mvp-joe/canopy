package canopy

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"

	"github.com/jward/canopy/internal/store"
)

// QueryBuilder provides a query API over the Store.
type QueryBuilder struct {
	store *store.Store
}

// NewQueryBuilder creates a QueryBuilder from a Store.
// Used by the CLI for query commands that don't need the Engine.
func NewQueryBuilder(s *store.Store) *QueryBuilder {
	return &QueryBuilder{store: s}
}

// Location represents a source code position range.
// All line and column numbers are 0-based, matching the tree-sitter convention.
type Location struct {
	File      string
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

// SymbolAt returns the most specific (narrowest) symbol whose range contains the
// given file position. Line and col are 0-based (tree-sitter convention).
// Returns nil with no error if no symbol exists at that location.
func (q *QueryBuilder) SymbolAt(file string, line, col int) (*Symbol, error) {
	f, err := q.store.FileByPath(file)
	if err != nil {
		return nil, fmt.Errorf("symbol at: lookup file: %w", err)
	}
	if f == nil {
		return nil, nil
	}

	// Validate line/col against actual file content if available.
	// Without this, multi-line symbols match any column on their start line
	// because the SQL only checks start_col <= col (with no upper bound).
	if content, err := os.ReadFile(file); err == nil {
		fileLines := bytes.Split(content, []byte{'\n'})
		if line >= len(fileLines) {
			return nil, nil
		}
		lineContent := bytes.TrimRight(fileLines[line], "\r")
		if col >= len(lineContent) {
			return nil, nil
		}
	}

	// Find all symbols containing this position, ordered by span size (narrowest first).
	row := q.store.DB().QueryRow(
		`SELECT `+store.SymbolCols+` FROM symbols
		 WHERE file_id = ?
		   AND (start_line < ? OR (start_line = ? AND start_col <= ?))
		   AND (end_line > ? OR (end_line = ? AND end_col >= ?))
		 ORDER BY (end_line - start_line) ASC, (end_col - start_col) ASC
		 LIMIT 1`,
		f.ID,
		line, line, col,
		line, line, col,
	)

	sym, err := q.store.ScanSymbolRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("symbol at: scan: %w", err)
	}
	return sym, nil
}

// DefinitionAt finds the definition(s) of the symbol referenced at the given position.
// Line and col are 0-based (tree-sitter convention). It looks up references at
// (file, line, col), resolves them, and returns the target symbol locations.
func (q *QueryBuilder) DefinitionAt(file string, line, col int) ([]Location, error) {
	f, err := q.store.FileByPath(file)
	if err != nil {
		return nil, fmt.Errorf("definition at: lookup file: %w", err)
	}
	if f == nil {
		return nil, nil
	}

	// Find references at this position: the position must fall within the reference span.
	rows, err := q.store.DB().Query(
		`SELECT id FROM references_
		 WHERE file_id = ? AND start_line <= ? AND end_line >= ?
		   AND (start_line < ? OR (start_line = ? AND start_col <= ?))
		   AND (end_line > ? OR (end_line = ? AND end_col >= ?))`,
		f.ID, line, line,
		line, line, col,
		line, line, col,
	)
	if err != nil {
		return nil, fmt.Errorf("definition at: query references: %w", err)
	}
	defer rows.Close()

	var refIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("definition at: scan ref: %w", err)
		}
		refIDs = append(refIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("definition at: rows: %w", err)
	}

	var locations []Location
	for _, refID := range refIDs {
		resolved, err := q.store.ResolvedReferencesByRef(refID)
		if err != nil {
			return nil, fmt.Errorf("definition at: resolve ref %d: %w", refID, err)
		}
		for _, rr := range resolved {
			loc, err := q.symbolLocation(rr.TargetSymbolID)
			if err != nil {
				return nil, fmt.Errorf("definition at: symbol location: %w", err)
			}
			if loc != nil {
				locations = append(locations, *loc)
			}
		}
	}
	return locations, nil
}

// ReferencesTo finds all source locations that reference the given symbol.
func (q *QueryBuilder) ReferencesTo(symbolID int64) ([]Location, error) {
	resolved, err := q.store.ResolvedReferencesByTarget(symbolID)
	if err != nil {
		return nil, fmt.Errorf("references to: %w", err)
	}

	var locations []Location
	for _, rr := range resolved {
		loc, err := q.referenceLocation(rr.ReferenceID)
		if err != nil {
			return nil, fmt.Errorf("references to: ref location: %w", err)
		}
		if loc != nil {
			locations = append(locations, *loc)
		}
	}
	return locations, nil
}

// Implementations finds all types implementing the given interface/trait symbol.
func (q *QueryBuilder) Implementations(symbolID int64) ([]Location, error) {
	impls, err := q.store.ImplementationsByInterface(symbolID)
	if err != nil {
		return nil, fmt.Errorf("implementations: %w", err)
	}

	var locations []Location
	for _, impl := range impls {
		loc, err := q.symbolLocation(impl.TypeSymbolID)
		if err != nil {
			return nil, fmt.Errorf("implementations: symbol location: %w", err)
		}
		if loc != nil {
			locations = append(locations, *loc)
		}
	}
	return locations, nil
}

// Callers returns call graph edges where the given symbol is the callee.
func (q *QueryBuilder) Callers(symbolID int64) ([]*CallEdge, error) {
	return q.store.CallersByCallee(symbolID)
}

// Callees returns call graph edges where the given symbol is the caller.
func (q *QueryBuilder) Callees(symbolID int64) ([]*CallEdge, error) {
	return q.store.CalleesByCaller(symbolID)
}

// Dependencies returns all imports for the given file.
func (q *QueryBuilder) Dependencies(fileID int64) ([]*Import, error) {
	return q.store.ImportsByFile(fileID)
}

// Dependents returns all imports across all files that reference the given source.
// Matches both exact source strings and suffix matches (e.g. "util" matches
// "github.com/example/util").
func (q *QueryBuilder) Dependents(source string) ([]*Import, error) {
	rows, err := q.store.DB().Query(
		"SELECT id, file_id, source, imported_name, local_alias, kind, scope FROM imports WHERE source = ? OR source LIKE ?",
		source, "%/"+source,
	)
	if err != nil {
		return nil, fmt.Errorf("dependents: %w", err)
	}
	defer rows.Close()

	var imports []*Import
	for rows.Next() {
		imp := &Import{}
		if err := rows.Scan(&imp.ID, &imp.FileID, &imp.Source, &imp.ImportedName,
			&imp.LocalAlias, &imp.Kind, &imp.Scope); err != nil {
			return nil, fmt.Errorf("dependents: scan import: %w", err)
		}
		imports = append(imports, imp)
	}
	return imports, rows.Err()
}

// symbolLocation resolves a symbol ID to its file path and position.
func (q *QueryBuilder) symbolLocation(symbolID int64) (*Location, error) {
	sym, err := q.store.SymbolByID(symbolID)
	if err != nil {
		return nil, err
	}
	if sym == nil || sym.FileID == nil {
		return nil, nil
	}

	var path string
	err = q.store.DB().QueryRow("SELECT path FROM files WHERE id = ?", *sym.FileID).Scan(&path)
	if err != nil {
		return nil, err
	}

	return &Location{
		File:      path,
		StartLine: sym.StartLine,
		StartCol:  sym.StartCol,
		EndLine:   sym.EndLine,
		EndCol:    sym.EndCol,
	}, nil
}

// referenceLocation resolves a reference ID to its file path and position.
func (q *QueryBuilder) referenceLocation(referenceID int64) (*Location, error) {
	var fileID int64
	var startLine, startCol, endLine, endCol int
	err := q.store.DB().QueryRow(
		`SELECT file_id, start_line, start_col, end_line, end_col
		 FROM references_ WHERE id = ?`, referenceID,
	).Scan(&fileID, &startLine, &startCol, &endLine, &endCol)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var path string
	err = q.store.DB().QueryRow("SELECT path FROM files WHERE id = ?", fileID).Scan(&path)
	if err != nil {
		return nil, err
	}

	return &Location{
		File:      path,
		StartLine: startLine,
		StartCol:  startCol,
		EndLine:   endLine,
		EndCol:    endCol,
	}, nil
}
