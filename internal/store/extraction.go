package store

import (
	"database/sql"
	"fmt"
)

// --- File operations ---

func (s *Store) InsertFile(f *File) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO files (path, language, hash, line_count, last_indexed) VALUES (?, ?, ?, ?, ?)",
		f.Path, f.Language, f.Hash, f.LineCount, f.LastIndexed,
	)
	if err != nil {
		return 0, fmt.Errorf("insert file: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	f.ID = id
	return id, nil
}

func (s *Store) FileByPath(path string) (*File, error) {
	f := &File{}
	err := s.db.QueryRow(
		"SELECT id, path, language, hash, line_count, last_indexed FROM files WHERE path = ?", path,
	).Scan(&f.ID, &f.Path, &f.Language, &f.Hash, &f.LineCount, &f.LastIndexed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("file by path: %w", err)
	}
	return f, nil
}

func (s *Store) FilesByLanguage(language string) ([]*File, error) {
	rows, err := s.db.Query(
		"SELECT id, path, language, hash, line_count, last_indexed FROM files WHERE language = ?", language,
	)
	if err != nil {
		return nil, fmt.Errorf("files by language: %w", err)
	}
	defer rows.Close()
	var files []*File
	for rows.Next() {
		f := &File{}
		if err := rows.Scan(&f.ID, &f.Path, &f.Language, &f.Hash, &f.LineCount, &f.LastIndexed); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// --- Symbol operations ---

func (s *Store) InsertSymbol(sym *Symbol) (int64, error) {
	mods := marshalModifiers(sym.Modifiers)
	res, err := s.db.Exec(
		`INSERT INTO symbols (file_id, name, kind, visibility, modifiers, signature_hash,
			start_line, start_col, end_line, end_col, parent_symbol_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sym.FileID, sym.Name, sym.Kind, sym.Visibility, mods, sym.SignatureHash,
		sym.StartLine, sym.StartCol, sym.EndLine, sym.EndCol, sym.ParentSymbolID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert symbol: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	sym.ID = id
	return id, nil
}

func (s *Store) scanSymbol(scanner interface{ Scan(...any) error }) (*Symbol, error) {
	sym := &Symbol{}
	var mods string
	err := scanner.Scan(
		&sym.ID, &sym.FileID, &sym.Name, &sym.Kind, &sym.Visibility, &mods,
		&sym.SignatureHash, &sym.StartLine, &sym.StartCol, &sym.EndLine, &sym.EndCol,
		&sym.ParentSymbolID,
	)
	if err != nil {
		return nil, err
	}
	sym.Modifiers = unmarshalModifiers(mods)
	return sym, nil
}

// ScanSymbolRow scans a single row into a Symbol. Exported for use by QueryBuilder.
func (s *Store) ScanSymbolRow(scanner interface{ Scan(...any) error }) (*Symbol, error) {
	return s.scanSymbol(scanner)
}

// SymbolCols is the column list for symbol queries, exported for use by QueryBuilder.
const SymbolCols = `id, file_id, name, kind, visibility, modifiers, signature_hash,
	start_line, start_col, end_line, end_col, parent_symbol_id`

func (s *Store) querySymbols(query string, args ...any) ([]*Symbol, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var symbols []*Symbol
	for rows.Next() {
		sym, err := s.scanSymbol(rows)
		if err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		symbols = append(symbols, sym)
	}
	return symbols, rows.Err()
}

func (s *Store) SymbolsByFile(fileID int64) ([]*Symbol, error) {
	return s.querySymbols("SELECT "+SymbolCols+" FROM symbols WHERE file_id = ?", fileID)
}

func (s *Store) SymbolsByName(name string) ([]*Symbol, error) {
	return s.querySymbols("SELECT "+SymbolCols+" FROM symbols WHERE name = ?", name)
}

func (s *Store) SymbolsByKind(kind string) ([]*Symbol, error) {
	return s.querySymbols("SELECT "+SymbolCols+" FROM symbols WHERE kind = ?", kind)
}

func (s *Store) SymbolChildren(symbolID int64) ([]*Symbol, error) {
	return s.querySymbols("SELECT "+SymbolCols+" FROM symbols WHERE parent_symbol_id = ?", symbolID)
}

// --- Scope operations ---

func (s *Store) InsertScope(scope *Scope) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO scopes (file_id, symbol_id, kind, start_line, start_col, end_line, end_col, parent_scope_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		scope.FileID, scope.SymbolID, scope.Kind,
		scope.StartLine, scope.StartCol, scope.EndLine, scope.EndCol, scope.ParentScopeID,
	)
	if err != nil {
		return 0, fmt.Errorf("insert scope: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	scope.ID = id
	return id, nil
}

func (s *Store) scanScope(scanner interface{ Scan(...any) error }) (*Scope, error) {
	sc := &Scope{}
	return sc, scanner.Scan(
		&sc.ID, &sc.FileID, &sc.SymbolID, &sc.Kind,
		&sc.StartLine, &sc.StartCol, &sc.EndLine, &sc.EndCol, &sc.ParentScopeID,
	)
}

const scopeCols = `id, file_id, symbol_id, kind, start_line, start_col, end_line, end_col, parent_scope_id`

func (s *Store) ScopesByFile(fileID int64) ([]*Scope, error) {
	rows, err := s.db.Query("SELECT "+scopeCols+" FROM scopes WHERE file_id = ?", fileID)
	if err != nil {
		return nil, fmt.Errorf("scopes by file: %w", err)
	}
	defer rows.Close()
	var scopes []*Scope
	for rows.Next() {
		sc, err := s.scanScope(rows)
		if err != nil {
			return nil, fmt.Errorf("scan scope: %w", err)
		}
		scopes = append(scopes, sc)
	}
	return scopes, rows.Err()
}

// ScopeChain walks up the parent_scope_id chain from scopeID to root.
func (s *Store) ScopeChain(scopeID int64) ([]*Scope, error) {
	var chain []*Scope
	currentID := &scopeID
	for currentID != nil {
		sc := &Scope{}
		err := s.db.QueryRow("SELECT "+scopeCols+" FROM scopes WHERE id = ?", *currentID).Scan(
			&sc.ID, &sc.FileID, &sc.SymbolID, &sc.Kind,
			&sc.StartLine, &sc.StartCol, &sc.EndLine, &sc.EndCol, &sc.ParentScopeID,
		)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("scope chain: %w", err)
		}
		chain = append(chain, sc)
		currentID = sc.ParentScopeID
	}
	return chain, nil
}

// --- Reference operations ---

func (s *Store) InsertReference(ref *Reference) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO references_ (file_id, scope_id, name, start_line, start_col, end_line, end_col, context)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ref.FileID, ref.ScopeID, ref.Name,
		ref.StartLine, ref.StartCol, ref.EndLine, ref.EndCol, ref.Context,
	)
	if err != nil {
		return 0, fmt.Errorf("insert reference: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	ref.ID = id
	return id, nil
}

func (s *Store) scanReference(scanner interface{ Scan(...any) error }) (*Reference, error) {
	r := &Reference{}
	return r, scanner.Scan(
		&r.ID, &r.FileID, &r.ScopeID, &r.Name,
		&r.StartLine, &r.StartCol, &r.EndLine, &r.EndCol, &r.Context,
	)
}

const refCols = `id, file_id, scope_id, name, start_line, start_col, end_line, end_col, context`

func (s *Store) queryReferences(query string, args ...any) ([]*Reference, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []*Reference
	for rows.Next() {
		r, err := s.scanReference(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reference: %w", err)
		}
		refs = append(refs, r)
	}
	return refs, rows.Err()
}

func (s *Store) ReferencesByFile(fileID int64) ([]*Reference, error) {
	return s.queryReferences("SELECT "+refCols+" FROM references_ WHERE file_id = ?", fileID)
}

func (s *Store) ReferencesByName(name string) ([]*Reference, error) {
	return s.queryReferences("SELECT "+refCols+" FROM references_ WHERE name = ?", name)
}

func (s *Store) ReferencesInScope(scopeID int64) ([]*Reference, error) {
	return s.queryReferences("SELECT "+refCols+" FROM references_ WHERE scope_id = ?", scopeID)
}

// --- Import operations ---

func (s *Store) InsertImport(imp *Import) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO imports (file_id, source, imported_name, local_alias, kind, scope)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		imp.FileID, imp.Source, imp.ImportedName, imp.LocalAlias, imp.Kind, imp.Scope,
	)
	if err != nil {
		return 0, fmt.Errorf("insert import: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	imp.ID = id
	return id, nil
}

func (s *Store) ImportsByFile(fileID int64) ([]*Import, error) {
	rows, err := s.db.Query(
		"SELECT id, file_id, source, imported_name, local_alias, kind, scope FROM imports WHERE file_id = ?",
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("imports by file: %w", err)
	}
	defer rows.Close()
	var imports []*Import
	for rows.Next() {
		imp := &Import{}
		if err := rows.Scan(&imp.ID, &imp.FileID, &imp.Source, &imp.ImportedName,
			&imp.LocalAlias, &imp.Kind, &imp.Scope); err != nil {
			return nil, fmt.Errorf("scan import: %w", err)
		}
		imports = append(imports, imp)
	}
	return imports, rows.Err()
}

// --- TypeMember operations ---

func (s *Store) InsertTypeMember(tm *TypeMember) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO type_members (symbol_id, name, kind, type_expr, visibility)
		 VALUES (?, ?, ?, ?, ?)`,
		tm.SymbolID, tm.Name, tm.Kind, tm.TypeExpr, tm.Visibility,
	)
	if err != nil {
		return 0, fmt.Errorf("insert type member: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	tm.ID = id
	return id, nil
}

func (s *Store) TypeMembers(symbolID int64) ([]*TypeMember, error) {
	rows, err := s.db.Query(
		"SELECT id, symbol_id, name, kind, type_expr, visibility FROM type_members WHERE symbol_id = ?",
		symbolID,
	)
	if err != nil {
		return nil, fmt.Errorf("type members: %w", err)
	}
	defer rows.Close()
	var members []*TypeMember
	for rows.Next() {
		tm := &TypeMember{}
		if err := rows.Scan(&tm.ID, &tm.SymbolID, &tm.Name, &tm.Kind, &tm.TypeExpr, &tm.Visibility); err != nil {
			return nil, fmt.Errorf("scan type member: %w", err)
		}
		members = append(members, tm)
	}
	return members, rows.Err()
}

// --- FunctionParam operations ---

func (s *Store) InsertFunctionParam(fp *FunctionParam) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO function_parameters (symbol_id, name, ordinal, type_expr, is_receiver, is_return, has_default, default_expr)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		fp.SymbolID, fp.Name, fp.Ordinal, fp.TypeExpr,
		fp.IsReceiver, fp.IsReturn, fp.HasDefault, fp.DefaultExpr,
	)
	if err != nil {
		return 0, fmt.Errorf("insert function param: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	fp.ID = id
	return id, nil
}

func (s *Store) FunctionParams(symbolID int64) ([]*FunctionParam, error) {
	rows, err := s.db.Query(
		`SELECT id, symbol_id, name, ordinal, type_expr, is_receiver, is_return, has_default, default_expr
		 FROM function_parameters WHERE symbol_id = ? ORDER BY ordinal`,
		symbolID,
	)
	if err != nil {
		return nil, fmt.Errorf("function params: %w", err)
	}
	defer rows.Close()
	var params []*FunctionParam
	for rows.Next() {
		fp := &FunctionParam{}
		if err := rows.Scan(&fp.ID, &fp.SymbolID, &fp.Name, &fp.Ordinal, &fp.TypeExpr,
			&fp.IsReceiver, &fp.IsReturn, &fp.HasDefault, &fp.DefaultExpr); err != nil {
			return nil, fmt.Errorf("scan function param: %w", err)
		}
		params = append(params, fp)
	}
	return params, rows.Err()
}

// --- TypeParam operations ---

func (s *Store) InsertTypeParam(tp *TypeParam) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO type_parameters (symbol_id, name, ordinal, variance, param_kind, constraints)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		tp.SymbolID, tp.Name, tp.Ordinal, tp.Variance, tp.ParamKind, tp.Constraints,
	)
	if err != nil {
		return 0, fmt.Errorf("insert type param: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	tp.ID = id
	return id, nil
}

func (s *Store) TypeParams(symbolID int64) ([]*TypeParam, error) {
	rows, err := s.db.Query(
		`SELECT id, symbol_id, name, ordinal, variance, param_kind, constraints
		 FROM type_parameters WHERE symbol_id = ? ORDER BY ordinal`,
		symbolID,
	)
	if err != nil {
		return nil, fmt.Errorf("type params: %w", err)
	}
	defer rows.Close()
	var params []*TypeParam
	for rows.Next() {
		tp := &TypeParam{}
		if err := rows.Scan(&tp.ID, &tp.SymbolID, &tp.Name, &tp.Ordinal,
			&tp.Variance, &tp.ParamKind, &tp.Constraints); err != nil {
			return nil, fmt.Errorf("scan type param: %w", err)
		}
		params = append(params, tp)
	}
	return params, rows.Err()
}

// --- Annotation operations ---

func (s *Store) InsertAnnotation(ann *Annotation) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO annotations (target_symbol_id, name, resolved_symbol_id, arguments, file_id, line, col)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ann.TargetSymbolID, ann.Name, ann.ResolvedSymbolID, ann.Arguments,
		ann.FileID, ann.Line, ann.Col,
	)
	if err != nil {
		return 0, fmt.Errorf("insert annotation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	ann.ID = id
	return id, nil
}

func (s *Store) AnnotationsByTarget(symbolID int64) ([]*Annotation, error) {
	rows, err := s.db.Query(
		`SELECT id, target_symbol_id, name, resolved_symbol_id, arguments, file_id, line, col
		 FROM annotations WHERE target_symbol_id = ?`,
		symbolID,
	)
	if err != nil {
		return nil, fmt.Errorf("annotations by target: %w", err)
	}
	defer rows.Close()
	var anns []*Annotation
	for rows.Next() {
		a := &Annotation{}
		if err := rows.Scan(&a.ID, &a.TargetSymbolID, &a.Name, &a.ResolvedSymbolID,
			&a.Arguments, &a.FileID, &a.Line, &a.Col); err != nil {
			return nil, fmt.Errorf("scan annotation: %w", err)
		}
		anns = append(anns, a)
	}
	return anns, rows.Err()
}

func (s *Store) UpdateAnnotationResolved(annotationID, resolvedSymbolID int64) error {
	_, err := s.db.Exec(
		`UPDATE annotations SET resolved_symbol_id = ? WHERE id = ?`,
		resolvedSymbolID, annotationID,
	)
	if err != nil {
		return fmt.Errorf("update annotation resolved: %w", err)
	}
	return nil
}

// --- SymbolFragment operations ---

func (s *Store) InsertSymbolFragment(frag *SymbolFragment) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO symbol_fragments (symbol_id, file_id, start_line, start_col, end_line, end_col, is_primary)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		frag.SymbolID, frag.FileID, frag.StartLine, frag.StartCol,
		frag.EndLine, frag.EndCol, frag.IsPrimary,
	)
	if err != nil {
		return 0, fmt.Errorf("insert symbol fragment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	frag.ID = id
	return id, nil
}

func (s *Store) SymbolFragments(symbolID int64) ([]*SymbolFragment, error) {
	rows, err := s.db.Query(
		`SELECT id, symbol_id, file_id, start_line, start_col, end_line, end_col, is_primary
		 FROM symbol_fragments WHERE symbol_id = ?`,
		symbolID,
	)
	if err != nil {
		return nil, fmt.Errorf("symbol fragments: %w", err)
	}
	defer rows.Close()
	var frags []*SymbolFragment
	for rows.Next() {
		f := &SymbolFragment{}
		if err := rows.Scan(&f.ID, &f.SymbolID, &f.FileID, &f.StartLine, &f.StartCol,
			&f.EndLine, &f.EndCol, &f.IsPrimary); err != nil {
			return nil, fmt.Errorf("scan symbol fragment: %w", err)
		}
		frags = append(frags, f)
	}
	return frags, rows.Err()
}
