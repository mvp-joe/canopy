package canopy

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jward/canopy/internal/store"
)

// SymbolDetail is a combined response that bundles a symbol with all of its
// structural metadata. One call replaces four separate Store lookups.
type SymbolDetail struct {
	Symbol      SymbolResult           // the symbol itself with ref counts
	Parameters  []*store.FunctionParam // function/method params, receiver, returns (empty for non-functions)
	Members     []*store.TypeMember    // struct fields, class methods, interface contracts (empty for non-types)
	TypeParams  []*store.TypeParam     // generic type parameters with constraints (empty if non-generic)
	Annotations []*store.Annotation    // decorators, annotations, attributes (empty if none)
}

// SymbolDetail returns a combined response with the symbol and all its
// structural metadata (parameters, members, type parameters, annotations).
// Returns nil with no error if the symbol ID does not exist.
func (q *QueryBuilder) SymbolDetail(symbolID int64) (*SymbolDetail, error) {
	sr, err := q.symbolResultByID(symbolID)
	if err != nil {
		return nil, fmt.Errorf("symbol detail: %w", err)
	}
	if sr == nil {
		return nil, nil
	}

	params, err := q.store.FunctionParams(symbolID)
	if err != nil {
		return nil, fmt.Errorf("symbol detail: function params: %w", err)
	}

	members, err := q.store.TypeMembers(symbolID)
	if err != nil {
		return nil, fmt.Errorf("symbol detail: type members: %w", err)
	}

	typeParams, err := q.store.TypeParams(symbolID)
	if err != nil {
		return nil, fmt.Errorf("symbol detail: type params: %w", err)
	}

	annotations, err := q.store.AnnotationsByTarget(symbolID)
	if err != nil {
		return nil, fmt.Errorf("symbol detail: annotations: %w", err)
	}

	if params == nil {
		params = []*store.FunctionParam{}
	}
	if members == nil {
		members = []*store.TypeMember{}
	}
	if typeParams == nil {
		typeParams = []*store.TypeParam{}
	}
	if annotations == nil {
		annotations = []*store.Annotation{}
	}

	return &SymbolDetail{
		Symbol:      *sr,
		Parameters:  params,
		Members:     members,
		TypeParams:  typeParams,
		Annotations: annotations,
	}, nil
}

// SymbolDetailAt is a position-based convenience that resolves the narrowest
// symbol at (file, line, col) and returns its SymbolDetail.
// Line and col are 0-based. Returns nil with no error if no symbol exists.
func (q *QueryBuilder) SymbolDetailAt(file string, line, col int) (*SymbolDetail, error) {
	sym, err := q.SymbolAt(file, line, col)
	if err != nil {
		return nil, fmt.Errorf("symbol detail at: %w", err)
	}
	if sym == nil {
		return nil, nil
	}
	return q.SymbolDetail(sym.ID)
}

// ScopeAt returns the scope chain at a position, ordered from innermost to
// outermost. Finds the narrowest scope containing (file, line, col), then
// walks parent_scope_id to the file scope.
// Line and col are 0-based. Returns nil slice, nil error if no scope contains
// the position or file is not indexed.
func (q *QueryBuilder) ScopeAt(file string, line, col int) ([]*store.Scope, error) {
	f, err := q.store.FileByPath(file)
	if err != nil {
		return nil, fmt.Errorf("scope at: lookup file: %w", err)
	}
	if f == nil {
		return nil, nil
	}

	innermost, err := q.store.ScopeAt(f.ID, line, col)
	if err != nil {
		return nil, fmt.Errorf("scope at: find innermost: %w", err)
	}
	if innermost == nil {
		return nil, nil
	}

	chain, err := q.store.ScopeChain(innermost.ID)
	if err != nil {
		return nil, fmt.Errorf("scope at: scope chain: %w", err)
	}
	return chain, nil
}

// symbolResultsByIDs loads multiple symbols as SymbolResults (with ref counts)
// in a single query. Returns a map from symbol ID to *SymbolResult. Missing
// IDs are simply absent from the map.
func (q *QueryBuilder) symbolResultsByIDs(ids []int64) (map[int64]*SymbolResult, error) {
	if len(ids) == 0 {
		return map[int64]*SymbolResult{}, nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := q.store.DB().Query(
		fmt.Sprintf(
			`SELECT %s, COALESCE(f.path, '') AS file_path,
				(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
				(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count
			 FROM symbols s
			 LEFT JOIN files f ON s.file_id = f.id
			 WHERE s.id IN (%s)`,
			prefixSymbolCols("s"),
			placeholders,
		),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int64]*SymbolResult, len(ids))
	for rows.Next() {
		sr, err := scanSymbolResult(rows)
		if err != nil {
			return nil, err
		}
		result[sr.ID] = &sr
	}
	return result, rows.Err()
}

// symbolResultByID loads a single symbol as a SymbolResult (with ref counts)
// by its ID. Returns nil with no error if not found.
func (q *QueryBuilder) symbolResultByID(symbolID int64) (*SymbolResult, error) {
	row := q.store.DB().QueryRow(
		fmt.Sprintf(
			`SELECT %s, COALESCE(f.path, '') AS file_path,
				(SELECT COUNT(*) FROM resolved_references rr WHERE rr.target_symbol_id = s.id) AS ref_count,
				(SELECT COUNT(*) FROM resolved_references rr JOIN references_ r ON r.id = rr.reference_id WHERE rr.target_symbol_id = s.id AND r.file_id != s.file_id) AS external_ref_count
			 FROM symbols s
			 LEFT JOIN files f ON s.file_id = f.id
			 WHERE s.id = ?`,
			prefixSymbolCols("s"),
		),
		symbolID,
	)
	sr, err := scanSymbolResult(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sr, nil
}
