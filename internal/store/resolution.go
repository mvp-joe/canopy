package store

import "fmt"

// --- ResolvedReference operations ---

func (s *Store) InsertResolvedReference(rr *ResolvedReference) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO resolved_references (reference_id, target_symbol_id, confidence, resolution_kind)
		 VALUES (?, ?, ?, ?)`,
		rr.ReferenceID, rr.TargetSymbolID, rr.Confidence, rr.ResolutionKind,
	)
	if err != nil {
		return 0, fmt.Errorf("insert resolved reference: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	rr.ID = id
	return id, nil
}

func (s *Store) queryResolvedRefs(query string, args ...any) ([]*ResolvedReference, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []*ResolvedReference
	for rows.Next() {
		rr := &ResolvedReference{}
		if err := rows.Scan(&rr.ID, &rr.ReferenceID, &rr.TargetSymbolID, &rr.Confidence, &rr.ResolutionKind); err != nil {
			return nil, fmt.Errorf("scan resolved reference: %w", err)
		}
		refs = append(refs, rr)
	}
	return refs, rows.Err()
}

const resolvedRefCols = `id, reference_id, target_symbol_id, confidence, resolution_kind`

func (s *Store) ResolvedReferencesByRef(referenceID int64) ([]*ResolvedReference, error) {
	return s.queryResolvedRefs(
		"SELECT "+resolvedRefCols+" FROM resolved_references WHERE reference_id = ?", referenceID,
	)
}

func (s *Store) ResolvedReferencesByTarget(symbolID int64) ([]*ResolvedReference, error) {
	return s.queryResolvedRefs(
		"SELECT "+resolvedRefCols+" FROM resolved_references WHERE target_symbol_id = ?", symbolID,
	)
}

// --- Implementation operations ---

func (s *Store) InsertImplementation(impl *Implementation) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO implementations (type_symbol_id, interface_symbol_id, kind, file_id, declaring_module)
		 VALUES (?, ?, ?, ?, ?)`,
		impl.TypeSymbolID, impl.InterfaceSymbolID, impl.Kind, impl.FileID, impl.DeclaringModule,
	)
	if err != nil {
		return 0, fmt.Errorf("insert implementation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	impl.ID = id
	return id, nil
}

func (s *Store) queryImplementations(query string, args ...any) ([]*Implementation, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var impls []*Implementation
	for rows.Next() {
		impl := &Implementation{}
		if err := rows.Scan(&impl.ID, &impl.TypeSymbolID, &impl.InterfaceSymbolID,
			&impl.Kind, &impl.FileID, &impl.DeclaringModule); err != nil {
			return nil, fmt.Errorf("scan implementation: %w", err)
		}
		impls = append(impls, impl)
	}
	return impls, rows.Err()
}

const implCols = `id, type_symbol_id, interface_symbol_id, kind, file_id, declaring_module`

func (s *Store) ImplementationsByType(typeSymbolID int64) ([]*Implementation, error) {
	return s.queryImplementations(
		"SELECT "+implCols+" FROM implementations WHERE type_symbol_id = ?", typeSymbolID,
	)
}

func (s *Store) ImplementationsByInterface(interfaceSymbolID int64) ([]*Implementation, error) {
	return s.queryImplementations(
		"SELECT "+implCols+" FROM implementations WHERE interface_symbol_id = ?", interfaceSymbolID,
	)
}

// --- CallEdge operations ---

func (s *Store) InsertCallEdge(edge *CallEdge) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO call_graph (caller_symbol_id, callee_symbol_id, file_id, line, col)
		 VALUES (?, ?, ?, ?, ?)`,
		edge.CallerSymbolID, edge.CalleeSymbolID, edge.FileID, edge.Line, edge.Col,
	)
	if err != nil {
		return 0, fmt.Errorf("insert call edge: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	edge.ID = id
	return id, nil
}

func (s *Store) queryCallEdges(query string, args ...any) ([]*CallEdge, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []*CallEdge
	for rows.Next() {
		e := &CallEdge{}
		if err := rows.Scan(&e.ID, &e.CallerSymbolID, &e.CalleeSymbolID, &e.FileID, &e.Line, &e.Col); err != nil {
			return nil, fmt.Errorf("scan call edge: %w", err)
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

const callEdgeCols = `id, caller_symbol_id, callee_symbol_id, file_id, line, col`

// AllCallEdges returns all call graph edges. Used for bulk-loading into
// in-memory adjacency maps for transitive traversal.
func (s *Store) AllCallEdges() ([]*CallEdge, error) {
	return s.queryCallEdges("SELECT " + callEdgeCols + " FROM call_graph")
}

func (s *Store) CallersByCallee(calleeSymbolID int64) ([]*CallEdge, error) {
	return s.queryCallEdges(
		"SELECT "+callEdgeCols+" FROM call_graph WHERE callee_symbol_id = ?", calleeSymbolID,
	)
}

func (s *Store) CalleesByCaller(callerSymbolID int64) ([]*CallEdge, error) {
	return s.queryCallEdges(
		"SELECT "+callEdgeCols+" FROM call_graph WHERE caller_symbol_id = ?", callerSymbolID,
	)
}

// --- Reexport operations ---

func (s *Store) InsertReexport(re *Reexport) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO reexports (file_id, original_symbol_id, exported_name)
		 VALUES (?, ?, ?)`,
		re.FileID, re.OriginalSymbolID, re.ExportedName,
	)
	if err != nil {
		return 0, fmt.Errorf("insert reexport: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	re.ID = id
	return id, nil
}

func (s *Store) ReexportsByFile(fileID int64) ([]*Reexport, error) {
	rows, err := s.db.Query(
		"SELECT id, file_id, original_symbol_id, exported_name FROM reexports WHERE file_id = ?",
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("reexports by file: %w", err)
	}
	defer rows.Close()
	var reexports []*Reexport
	for rows.Next() {
		re := &Reexport{}
		if err := rows.Scan(&re.ID, &re.FileID, &re.OriginalSymbolID, &re.ExportedName); err != nil {
			return nil, fmt.Errorf("scan reexport: %w", err)
		}
		reexports = append(reexports, re)
	}
	return reexports, rows.Err()
}

// --- ExtensionBinding operations ---

func (s *Store) InsertExtensionBinding(eb *ExtensionBinding) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO extension_bindings (member_symbol_id, extended_type_expr, extended_type_symbol_id, kind, constraints, is_default_impl)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		eb.MemberSymbolID, eb.ExtendedTypeExpr, eb.ExtendedTypeSymbolID,
		eb.Kind, eb.Constraints, eb.IsDefaultImpl,
	)
	if err != nil {
		return 0, fmt.Errorf("insert extension binding: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	eb.ID = id
	return id, nil
}

func (s *Store) ExtensionBindingsByType(typeSymbolID int64) ([]*ExtensionBinding, error) {
	rows, err := s.db.Query(
		`SELECT id, member_symbol_id, extended_type_expr, extended_type_symbol_id, kind, constraints, is_default_impl
		 FROM extension_bindings WHERE extended_type_symbol_id = ?`,
		typeSymbolID,
	)
	if err != nil {
		return nil, fmt.Errorf("extension bindings by type: %w", err)
	}
	defer rows.Close()
	var bindings []*ExtensionBinding
	for rows.Next() {
		eb := &ExtensionBinding{}
		if err := rows.Scan(&eb.ID, &eb.MemberSymbolID, &eb.ExtendedTypeExpr,
			&eb.ExtendedTypeSymbolID, &eb.Kind, &eb.Constraints, &eb.IsDefaultImpl); err != nil {
			return nil, fmt.Errorf("scan extension binding: %w", err)
		}
		bindings = append(bindings, eb)
	}
	return bindings, rows.Err()
}

// --- TypeComposition operations ---

func (s *Store) InsertTypeComposition(tc *TypeComposition) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO type_compositions (composite_symbol_id, component_symbol_id, composition_kind)
		 VALUES (?, ?, ?)`,
		tc.CompositeSymbolID, tc.ComponentSymbolID, tc.CompositionKind,
	)
	if err != nil {
		return 0, fmt.Errorf("insert type composition: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	tc.ID = id
	return id, nil
}

func (s *Store) queryTypeCompositions(query string, args ...any) ([]*TypeComposition, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var comps []*TypeComposition
	for rows.Next() {
		tc := &TypeComposition{}
		if err := rows.Scan(&tc.ID, &tc.CompositeSymbolID, &tc.ComponentSymbolID, &tc.CompositionKind); err != nil {
			return nil, fmt.Errorf("scan type composition: %w", err)
		}
		comps = append(comps, tc)
	}
	return comps, rows.Err()
}

const typeCompCols = `id, composite_symbol_id, component_symbol_id, composition_kind`

func (s *Store) TypeCompositions(compositeSymbolID int64) ([]*TypeComposition, error) {
	return s.queryTypeCompositions(
		"SELECT "+typeCompCols+" FROM type_compositions WHERE composite_symbol_id = ?", compositeSymbolID,
	)
}

func (s *Store) TypeComposedBy(componentSymbolID int64) ([]*TypeComposition, error) {
	return s.queryTypeCompositions(
		"SELECT "+typeCompCols+" FROM type_compositions WHERE component_symbol_id = ?", componentSymbolID,
	)
}
