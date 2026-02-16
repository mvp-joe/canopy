package store

import (
	"database/sql"
	"fmt"
)

// CommitBatch inserts all buffered data from a BatchedStore into SQLite
// within a single transaction. Fake (negative) IDs are remapped to real
// (positive, AUTOINCREMENT) IDs, and all FK references within the batch
// are rewritten using the fakeToReal mapping.
//
// Insert order respects FK dependencies:
//  1. Symbols (depend on file_id only, which is already real)
//  2. Scopes (depend on file_id, symbol_id, parent_scope_id)
//  3. References (depend on file_id, scope_id)
//  4. Imports (depend on file_id only)
//  5. TypeMembers (depend on symbol_id)
//  6. FunctionParams (depend on symbol_id)
//  7. TypeParams (depend on symbol_id)
//  8. Annotations (depend on symbol_id, file_id)
//  9. SymbolFragments (depend on symbol_id, file_id)
func (s *Store) CommitBatch(batch *BatchedStore) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("commit batch: begin: %w", err)
	}
	defer tx.Rollback()

	fakeToReal := make(map[int64]int64)

	// 1. Symbols
	for _, sym := range batch.Symbols {
		// parent_symbol_id may be fake (intra-file) or real (cross-file)
		if sym.ParentSymbolID != nil && *sym.ParentSymbolID < 0 {
			realID := fakeToReal[*sym.ParentSymbolID]
			sym.ParentSymbolID = &realID
		}
		realID, err := insertSymbolTx(tx, &sym)
		if err != nil {
			return fmt.Errorf("commit batch: symbol %q: %w", sym.Name, err)
		}
		fakeToReal[sym.ID] = realID
	}

	// 2. Scopes
	for _, scope := range batch.Scopes {
		if scope.ParentScopeID != nil && *scope.ParentScopeID < 0 {
			realID := fakeToReal[*scope.ParentScopeID]
			scope.ParentScopeID = &realID
		}
		if scope.SymbolID != nil && *scope.SymbolID < 0 {
			realID := fakeToReal[*scope.SymbolID]
			scope.SymbolID = &realID
		}
		realID, err := insertScopeTx(tx, &scope)
		if err != nil {
			return fmt.Errorf("commit batch: scope: %w", err)
		}
		fakeToReal[scope.ID] = realID
	}

	// 3. References
	for _, ref := range batch.References {
		if ref.ScopeID != nil && *ref.ScopeID < 0 {
			realID := fakeToReal[*ref.ScopeID]
			ref.ScopeID = &realID
		}
		realID, err := insertReferenceTx(tx, &ref)
		if err != nil {
			return fmt.Errorf("commit batch: reference %q: %w", ref.Name, err)
		}
		fakeToReal[ref.ID] = realID
	}

	// 4. Imports
	for _, imp := range batch.Imports {
		realID, err := insertImportTx(tx, &imp)
		if err != nil {
			return fmt.Errorf("commit batch: import %q: %w", imp.Source, err)
		}
		fakeToReal[imp.ID] = realID
	}

	// 5. TypeMembers
	for _, tm := range batch.TypeMembers {
		if tm.SymbolID < 0 {
			realID, ok := fakeToReal[tm.SymbolID]
			if !ok {
				return fmt.Errorf("commit batch: type member %q has symbol_id=%d not in fakeToReal map (have %d symbols)", tm.Name, tm.SymbolID, len(batch.Symbols))
			}
			tm.SymbolID = realID
		}
		realID, err := insertTypeMemberTx(tx, &tm)
		if err != nil {
			return fmt.Errorf("commit batch: type member %q: %w", tm.Name, err)
		}
		fakeToReal[tm.ID] = realID
	}

	// 6. FunctionParams
	for _, fp := range batch.FunctionParams {
		if fp.SymbolID < 0 {
			fp.SymbolID = fakeToReal[fp.SymbolID]
		}
		realID, err := insertFunctionParamTx(tx, &fp)
		if err != nil {
			return fmt.Errorf("commit batch: function param %q: %w", fp.Name, err)
		}
		fakeToReal[fp.ID] = realID
	}

	// 7. TypeParams
	for _, tp := range batch.TypeParams {
		if tp.SymbolID < 0 {
			tp.SymbolID = fakeToReal[tp.SymbolID]
		}
		realID, err := insertTypeParamTx(tx, &tp)
		if err != nil {
			return fmt.Errorf("commit batch: type param %q: %w", tp.Name, err)
		}
		fakeToReal[tp.ID] = realID
	}

	// 8. Annotations
	for _, ann := range batch.Annotations {
		if ann.TargetSymbolID < 0 {
			ann.TargetSymbolID = fakeToReal[ann.TargetSymbolID]
		}
		if ann.ResolvedSymbolID != nil && *ann.ResolvedSymbolID < 0 {
			realID := fakeToReal[*ann.ResolvedSymbolID]
			ann.ResolvedSymbolID = &realID
		}
		if ann.FileID != nil && *ann.FileID < 0 {
			realID := fakeToReal[*ann.FileID]
			ann.FileID = &realID
		}
		realID, err := insertAnnotationTx(tx, &ann)
		if err != nil {
			return fmt.Errorf("commit batch: annotation %q: %w", ann.Name, err)
		}
		fakeToReal[ann.ID] = realID
	}

	// 9. SymbolFragments
	for _, sf := range batch.SymbolFragments {
		if sf.SymbolID < 0 {
			sf.SymbolID = fakeToReal[sf.SymbolID]
		}
		if sf.FileID < 0 {
			sf.FileID = fakeToReal[sf.FileID]
		}
		realID, err := insertSymbolFragmentTx(tx, &sf)
		if err != nil {
			return fmt.Errorf("commit batch: symbol fragment: %w", err)
		}
		fakeToReal[sf.ID] = realID
	}

	return tx.Commit()
}

// --- Transaction-scoped insert helpers ---
// These mirror the Store insert methods but accept *sql.Tx instead of using s.db.

func insertSymbolTx(tx *sql.Tx, sym *Symbol) (int64, error) {
	mods := marshalModifiers(sym.Modifiers)
	res, err := tx.Exec(
		`INSERT INTO symbols (file_id, name, kind, visibility, modifiers, signature_hash,
			start_line, start_col, end_line, end_col, parent_symbol_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sym.FileID, sym.Name, sym.Kind, sym.Visibility, mods, sym.SignatureHash,
		sym.StartLine, sym.StartCol, sym.EndLine, sym.EndCol, sym.ParentSymbolID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertScopeTx(tx *sql.Tx, scope *Scope) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO scopes (file_id, symbol_id, kind, start_line, start_col, end_line, end_col, parent_scope_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		scope.FileID, scope.SymbolID, scope.Kind,
		scope.StartLine, scope.StartCol, scope.EndLine, scope.EndCol, scope.ParentScopeID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertReferenceTx(tx *sql.Tx, ref *Reference) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO references_ (file_id, scope_id, name, start_line, start_col, end_line, end_col, context)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ref.FileID, ref.ScopeID, ref.Name,
		ref.StartLine, ref.StartCol, ref.EndLine, ref.EndCol, ref.Context,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertImportTx(tx *sql.Tx, imp *Import) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO imports (file_id, source, imported_name, local_alias, kind, scope)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		imp.FileID, imp.Source, imp.ImportedName, imp.LocalAlias, imp.Kind, imp.Scope,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertTypeMemberTx(tx *sql.Tx, tm *TypeMember) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO type_members (symbol_id, name, kind, type_expr, visibility)
		 VALUES (?, ?, ?, ?, ?)`,
		tm.SymbolID, tm.Name, tm.Kind, tm.TypeExpr, tm.Visibility,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertFunctionParamTx(tx *sql.Tx, fp *FunctionParam) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO function_parameters (symbol_id, name, ordinal, type_expr, is_receiver, is_return, has_default, default_expr)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		fp.SymbolID, fp.Name, fp.Ordinal, fp.TypeExpr,
		fp.IsReceiver, fp.IsReturn, fp.HasDefault, fp.DefaultExpr,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertTypeParamTx(tx *sql.Tx, tp *TypeParam) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO type_parameters (symbol_id, name, ordinal, variance, param_kind, constraints)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		tp.SymbolID, tp.Name, tp.Ordinal, tp.Variance, tp.ParamKind, tp.Constraints,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertAnnotationTx(tx *sql.Tx, ann *Annotation) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO annotations (target_symbol_id, name, resolved_symbol_id, arguments, file_id, line, col)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ann.TargetSymbolID, ann.Name, ann.ResolvedSymbolID, ann.Arguments,
		ann.FileID, ann.Line, ann.Col,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func insertSymbolFragmentTx(tx *sql.Tx, frag *SymbolFragment) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO symbol_fragments (symbol_id, file_id, start_line, start_col, end_line, end_col, is_primary)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		frag.SymbolID, frag.FileID, frag.StartLine, frag.StartCol,
		frag.EndLine, frag.EndCol, frag.IsPrimary,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
