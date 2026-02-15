package store

import "fmt"

// FilesReferencingSymbols returns file IDs that have resolved_references targeting any of the given symbols.
func (s *Store) FilesReferencingSymbols(symbolIDs []int64) ([]int64, error) {
	if len(symbolIDs) == 0 {
		return nil, nil
	}
	placeholders := placeholderList(len(symbolIDs))
	query := `SELECT DISTINCT r.file_id
		FROM resolved_references rr
		JOIN references_ r ON r.id = rr.reference_id
		WHERE rr.target_symbol_id IN (` + placeholders + `)`
	rows, err := s.db.Query(query, int64sToArgs(symbolIDs)...)
	if err != nil {
		return nil, fmt.Errorf("files referencing symbols: %w", err)
	}
	defer rows.Close()
	var fileIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan file id: %w", err)
		}
		fileIDs = append(fileIDs, id)
	}
	return fileIDs, rows.Err()
}

// FilesImportingSource returns file IDs that import the given module/package source.
func (s *Store) FilesImportingSource(source string) ([]int64, error) {
	rows, err := s.db.Query("SELECT DISTINCT file_id FROM imports WHERE source = ?", source)
	if err != nil {
		return nil, fmt.Errorf("files importing source: %w", err)
	}
	defer rows.Close()
	var fileIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan file id: %w", err)
		}
		fileIDs = append(fileIDs, id)
	}
	return fileIDs, rows.Err()
}

// DeleteResolutionDataForSymbols removes all resolution data targeting the given symbols:
// resolved_references, call_graph, implementations, extension_bindings, reexports, type_compositions.
func (s *Store) DeleteResolutionDataForSymbols(symbolIDs []int64) error {
	if len(symbolIDs) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	placeholders := placeholderList(len(symbolIDs))
	args := int64sToArgs(symbolIDs)

	queries := []struct {
		sql  string
		args []any
	}{
		{"DELETE FROM resolved_references WHERE target_symbol_id IN (" + placeholders + ")", args},
		{"DELETE FROM call_graph WHERE caller_symbol_id IN (" + placeholders + ") OR callee_symbol_id IN (" + placeholders + ")", repeatArgs(args, 2)},
		{"DELETE FROM implementations WHERE type_symbol_id IN (" + placeholders + ") OR interface_symbol_id IN (" + placeholders + ")", repeatArgs(args, 2)},
		{"DELETE FROM extension_bindings WHERE member_symbol_id IN (" + placeholders + ") OR extended_type_symbol_id IN (" + placeholders + ")", repeatArgs(args, 2)},
		{"DELETE FROM reexports WHERE original_symbol_id IN (" + placeholders + ")", args},
		{"DELETE FROM type_compositions WHERE composite_symbol_id IN (" + placeholders + ") OR component_symbol_id IN (" + placeholders + ")", repeatArgs(args, 2)},
	}

	for _, q := range queries {
		if _, err := tx.Exec(q.sql, q.args...); err != nil {
			return fmt.Errorf("delete resolution data for symbols: %w", err)
		}
	}

	return tx.Commit()
}

// DeleteResolutionDataForFiles removes all resolution data originating from the given files.
// This means: resolved_references whose reference comes from those files, call_graph/implementations
// with file_id in the set, reexports with file_id in the set, and any extension_bindings/type_compositions
// whose member/composite symbol belongs to those files.
func (s *Store) DeleteResolutionDataForFiles(fileIDs []int64) error {
	if len(fileIDs) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	placeholders := placeholderList(len(fileIDs))
	args := int64sToArgs(fileIDs)

	// Delete resolved_references whose reference originates from these files.
	if _, err := tx.Exec(
		`DELETE FROM resolved_references WHERE reference_id IN (
			SELECT id FROM references_ WHERE file_id IN (`+placeholders+`)
		)`, args...); err != nil {
		return fmt.Errorf("delete resolved refs for files: %w", err)
	}

	// Delete call_graph edges originating from these files.
	if _, err := tx.Exec("DELETE FROM call_graph WHERE file_id IN ("+placeholders+")", args...); err != nil {
		return fmt.Errorf("delete call graph for files: %w", err)
	}

	// Delete implementations originating from these files.
	if _, err := tx.Exec("DELETE FROM implementations WHERE file_id IN ("+placeholders+")", args...); err != nil {
		return fmt.Errorf("delete implementations for files: %w", err)
	}

	// Delete reexports from these files.
	if _, err := tx.Exec("DELETE FROM reexports WHERE file_id IN ("+placeholders+")", args...); err != nil {
		return fmt.Errorf("delete reexports for files: %w", err)
	}

	// Delete extension_bindings whose member symbol belongs to these files.
	if _, err := tx.Exec(
		`DELETE FROM extension_bindings WHERE member_symbol_id IN (
			SELECT id FROM symbols WHERE file_id IN (`+placeholders+`)
		)`, args...); err != nil {
		return fmt.Errorf("delete extension bindings for files: %w", err)
	}

	// Delete type_compositions whose composite symbol belongs to these files.
	if _, err := tx.Exec(
		`DELETE FROM type_compositions WHERE composite_symbol_id IN (
			SELECT id FROM symbols WHERE file_id IN (`+placeholders+`)
		)`, args...); err != nil {
		return fmt.Errorf("delete type compositions for files: %w", err)
	}

	return tx.Commit()
}
