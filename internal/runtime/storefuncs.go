package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/risor-io/risor/object"

	"github.com/jward/canopy/internal/store"
)

// makeStoreInsertFunctions creates host functions that wrap Store insert
// methods. Risor scripts cannot construct Go struct pointers, so these
// functions accept Risor maps with primitive values and build the structs
// on the Go side.

func makeInsertSymbolFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_symbol", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_symbol", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_symbol: %v", err)
		}

		sym := &store.Symbol{
			Name:       getString(m, "name"),
			Kind:       getString(m, "kind"),
			Visibility: getString(m, "visibility"),
			StartLine:  getInt(m, "start_line"),
			StartCol:   getInt(m, "start_col"),
			EndLine:    getInt(m, "end_line"),
			EndCol:     getInt(m, "end_col"),
		}
		if v, ok := getOptionalInt64(m, "file_id"); ok {
			sym.FileID = &v
		}
		if v, ok := getOptionalInt64(m, "parent_symbol_id"); ok {
			sym.ParentSymbolID = &v
		}

		id, insertErr := s.InsertSymbol(sym)
		if insertErr != nil {
			return object.Errorf("insert_symbol: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertScopeFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_scope", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_scope", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_scope: %v", err)
		}

		scope := &store.Scope{
			FileID:    getInt64(m, "file_id"),
			Kind:      getString(m, "kind"),
			StartLine: getInt(m, "start_line"),
			StartCol:  getInt(m, "start_col"),
			EndLine:   getInt(m, "end_line"),
			EndCol:    getInt(m, "end_col"),
		}
		if v, ok := getOptionalInt64(m, "symbol_id"); ok {
			scope.SymbolID = &v
		}
		if v, ok := getOptionalInt64(m, "parent_scope_id"); ok {
			scope.ParentScopeID = &v
		}

		id, insertErr := s.InsertScope(scope)
		if insertErr != nil {
			return object.Errorf("insert_scope: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertReferenceFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_reference", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_reference", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_reference: %v", err)
		}

		ref := &store.Reference{
			FileID:    getInt64(m, "file_id"),
			Name:      getString(m, "name"),
			StartLine: getInt(m, "start_line"),
			StartCol:  getInt(m, "start_col"),
			EndLine:   getInt(m, "end_line"),
			EndCol:    getInt(m, "end_col"),
			Context:   getString(m, "context"),
		}
		if v, ok := getOptionalInt64(m, "scope_id"); ok {
			ref.ScopeID = &v
		}

		id, insertErr := s.InsertReference(ref)
		if insertErr != nil {
			return object.Errorf("insert_reference: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertImportFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_import", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_import", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_import: %v", err)
		}

		imp := &store.Import{
			FileID: getInt64(m, "file_id"),
			Source: getString(m, "source"),
			Kind:   getStringDefault(m, "kind", "module"),
			Scope:  getStringDefault(m, "scope", "file"),
		}
		if v := getString(m, "imported_name"); v != "" {
			imp.ImportedName = &v
		}
		if v := getString(m, "local_alias"); v != "" {
			imp.LocalAlias = &v
		}

		id, insertErr := s.InsertImport(imp)
		if insertErr != nil {
			return object.Errorf("insert_import: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertTypeMemberFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_type_member", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_type_member", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_type_member: %v", err)
		}

		tm := &store.TypeMember{
			SymbolID:   getInt64(m, "symbol_id"),
			Name:       getString(m, "name"),
			Kind:       getString(m, "kind"),
			TypeExpr:   getString(m, "type_expr"),
			Visibility: getString(m, "visibility"),
		}

		id, insertErr := s.InsertTypeMember(tm)
		if insertErr != nil {
			return object.Errorf("insert_type_member: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertFunctionParamFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_function_param", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_function_param", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_function_param: %v", err)
		}

		fp := &store.FunctionParam{
			SymbolID:   getInt64(m, "symbol_id"),
			Name:       getString(m, "name"),
			Ordinal:    getInt(m, "ordinal"),
			TypeExpr:   getString(m, "type_expr"),
			IsReceiver: getBool(m, "is_receiver"),
			IsReturn:   getBool(m, "is_return"),
			HasDefault: getBool(m, "has_default"),
			DefaultExpr: getString(m, "default_expr"),
		}

		id, insertErr := s.InsertFunctionParam(fp)
		if insertErr != nil {
			return object.Errorf("insert_function_param: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertTypeParamFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_type_param", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_type_param", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_type_param: %v", err)
		}

		tp := &store.TypeParam{
			SymbolID:    getInt64(m, "symbol_id"),
			Name:        getString(m, "name"),
			Ordinal:     getInt(m, "ordinal"),
			Variance:    getString(m, "variance"),
			ParamKind:   getStringDefault(m, "param_kind", "type"),
			Constraints: getString(m, "constraints"),
		}

		id, insertErr := s.InsertTypeParam(tp)
		if insertErr != nil {
			return object.Errorf("insert_type_param: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertAnnotationFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_annotation", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_annotation", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_annotation: %v", err)
		}

		ann := &store.Annotation{
			TargetSymbolID: getInt64(m, "target_symbol_id"),
			Name:           getString(m, "name"),
			Arguments:      getString(m, "arguments"),
			Line:           getInt(m, "line"),
			Col:            getInt(m, "col"),
		}
		if v, ok := getOptionalInt64(m, "file_id"); ok {
			ann.FileID = &v
		}
		if v, ok := getOptionalInt64(m, "resolved_symbol_id"); ok {
			ann.ResolvedSymbolID = &v
		}

		id, insertErr := s.InsertAnnotation(ann)
		if insertErr != nil {
			return object.Errorf("insert_annotation: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeUpdateAnnotationResolvedFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("update_annotation_resolved", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 2 {
			return object.NewArgsError("update_annotation_resolved", 2, len(args))
		}
		annID, ok := args[0].(*object.Int)
		if !ok {
			return object.Errorf("update_annotation_resolved: annotation_id must be int, got %s", args[0].Type())
		}
		symID, ok := args[1].(*object.Int)
		if !ok {
			return object.Errorf("update_annotation_resolved: resolved_symbol_id must be int, got %s", args[1].Type())
		}
		if err := s.UpdateAnnotationResolved(annID.Value(), symID.Value()); err != nil {
			return object.Errorf("update_annotation_resolved: %v", err)
		}
		return object.Nil
	})
}

// Helper to query symbols by name (needed by extraction scripts for
// parent lookups, e.g., linking methods to receiver types).
func makeSymbolsByNameFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("symbols_by_name", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("symbols_by_name", 1, len(args))
		}
		nameStr, ok := args[0].(*object.String)
		if !ok {
			return object.Errorf("symbols_by_name: expected string, got %s", args[0].Type())
		}

		syms, err := s.SymbolsByName(nameStr.Value())
		if err != nil {
			return object.Errorf("symbols_by_name: %v", err)
		}

		return symbolsToList(syms)
	})
}

// Helper to query symbols by file (needed for method-to-receiver linking).
func makeSymbolsByFileFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("symbols_by_file", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("symbols_by_file", 1, len(args))
		}
		fileID, err := toInt64(args[0])
		if err != nil {
			return object.Errorf("symbols_by_file: %v", err)
		}

		syms, queryErr := s.SymbolsByFile(fileID)
		if queryErr != nil {
			return object.Errorf("symbols_by_file: %v", queryErr)
		}

		return symbolsToList(syms)
	})
}

// --- Map extraction helpers ---

func extractMap(obj object.Object) (map[string]object.Object, error) {
	m, ok := obj.(*object.Map)
	if !ok {
		return nil, fmt.Errorf("expected map, got %s", obj.Type())
	}
	return m.Value(), nil
}

func getString(m map[string]object.Object, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	if s, ok := v.(*object.String); ok {
		return s.Value()
	}
	return ""
}

func getStringDefault(m map[string]object.Object, key, def string) string {
	v := getString(m, key)
	if v == "" {
		return def
	}
	return v
}

func getInt(m map[string]object.Object, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	if i, ok := v.(*object.Int); ok {
		return int(i.Value())
	}
	if f, ok := v.(*object.Float); ok {
		return int(f.Value())
	}
	return 0
}

func getInt64(m map[string]object.Object, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	if i, ok := v.(*object.Int); ok {
		return i.Value()
	}
	if f, ok := v.(*object.Float); ok {
		return int64(f.Value())
	}
	return 0
}

func getOptionalInt64(m map[string]object.Object, key string) (int64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	if v == nil || v.Type() == "nil" {
		return 0, false
	}
	if _, ok := v.(*object.NilType); ok {
		return 0, false
	}
	if i, ok := v.(*object.Int); ok {
		return i.Value(), true
	}
	if f, ok := v.(*object.Float); ok {
		return int64(f.Value()), true
	}
	return 0, false
}

func getBool(m map[string]object.Object, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	if b, ok := v.(*object.Bool); ok {
		return b.Value()
	}
	return false
}

func toInt64(obj object.Object) (int64, error) {
	if i, ok := obj.(*object.Int); ok {
		return i.Value(), nil
	}
	if f, ok := obj.(*object.Float); ok {
		return int64(f.Value()), nil
	}
	return 0, fmt.Errorf("expected int, got %s", obj.Type())
}

func toString(obj object.Object) (string, error) {
	if s, ok := obj.(*object.String); ok {
		return s.Value(), nil
	}
	return "", fmt.Errorf("expected string, got %s", obj.Type())
}

// --- Resolution insert bridge functions ---

func makeInsertResolvedReferenceFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_resolved_reference", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_resolved_reference", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_resolved_reference: %v", err)
		}

		rr := &store.ResolvedReference{
			ReferenceID:    getInt64(m, "reference_id"),
			TargetSymbolID: getInt64(m, "target_symbol_id"),
			Confidence:     getFloat(m, "confidence"),
			ResolutionKind: getString(m, "resolution_kind"),
		}
		if rr.Confidence == 0 {
			rr.Confidence = 1.0
		}

		id, insertErr := s.InsertResolvedReference(rr)
		if insertErr != nil {
			return object.Errorf("insert_resolved_reference: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertImplementationFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_implementation", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_implementation", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_implementation: %v", err)
		}

		impl := &store.Implementation{
			TypeSymbolID:      getInt64(m, "type_symbol_id"),
			InterfaceSymbolID: getInt64(m, "interface_symbol_id"),
			Kind:              getString(m, "kind"),
			DeclaringModule:   getString(m, "declaring_module"),
		}
		if v, ok := getOptionalInt64(m, "file_id"); ok {
			impl.FileID = &v
		}

		id, insertErr := s.InsertImplementation(impl)
		if insertErr != nil {
			return object.Errorf("insert_implementation: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertCallEdgeFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_call_edge", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_call_edge", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_call_edge: %v", err)
		}

		edge := &store.CallEdge{
			CallerSymbolID: getInt64(m, "caller_symbol_id"),
			CalleeSymbolID: getInt64(m, "callee_symbol_id"),
			Line:           getInt(m, "line"),
			Col:            getInt(m, "col"),
		}
		if v, ok := getOptionalInt64(m, "file_id"); ok {
			edge.FileID = &v
		}

		id, insertErr := s.InsertCallEdge(edge)
		if insertErr != nil {
			return object.Errorf("insert_call_edge: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

func makeInsertExtensionBindingFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("insert_extension_binding", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("insert_extension_binding", 1, len(args))
		}
		m, err := extractMap(args[0])
		if err != nil {
			return object.Errorf("insert_extension_binding: %v", err)
		}

		eb := &store.ExtensionBinding{
			MemberSymbolID:   getInt64(m, "member_symbol_id"),
			ExtendedTypeExpr: getString(m, "extended_type_expr"),
			Kind:             getStringDefault(m, "kind", "method"),
			Constraints:      getString(m, "constraints"),
			IsDefaultImpl:    getBool(m, "is_default_impl"),
		}
		if v, ok := getOptionalInt64(m, "extended_type_symbol_id"); ok {
			eb.ExtendedTypeSymbolID = &v
		}

		id, insertErr := s.InsertExtensionBinding(eb)
		if insertErr != nil {
			return object.Errorf("insert_extension_binding: %v", insertErr)
		}
		return object.NewInt(id)
	})
}

// --- Resolution query bridge functions ---

func makeReferencesByFileFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("references_by_file", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("references_by_file", 1, len(args))
		}
		fileID, err := toInt64(args[0])
		if err != nil {
			return object.Errorf("references_by_file: %v", err)
		}

		refs, queryErr := s.ReferencesByFile(fileID)
		if queryErr != nil {
			return object.Errorf("references_by_file: %v", queryErr)
		}

		var results []object.Object
		for _, r := range refs {
			m := map[string]object.Object{
				"id":         object.NewInt(r.ID),
				"name":       object.NewString(r.Name),
				"context":    object.NewString(r.Context),
				"start_line": object.NewInt(int64(r.StartLine)),
				"start_col":  object.NewInt(int64(r.StartCol)),
				"end_line":   object.NewInt(int64(r.EndLine)),
				"end_col":    object.NewInt(int64(r.EndCol)),
			}
			if r.ScopeID != nil {
				m["scope_id"] = object.NewInt(*r.ScopeID)
			}
			results = append(results, object.NewMap(m))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

func makeScopesByFileFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("scopes_by_file", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("scopes_by_file", 1, len(args))
		}
		fileID, err := toInt64(args[0])
		if err != nil {
			return object.Errorf("scopes_by_file: %v", err)
		}

		scopes, queryErr := s.ScopesByFile(fileID)
		if queryErr != nil {
			return object.Errorf("scopes_by_file: %v", queryErr)
		}

		var results []object.Object
		for _, sc := range scopes {
			m := map[string]object.Object{
				"id":         object.NewInt(sc.ID),
				"kind":       object.NewString(sc.Kind),
				"start_line": object.NewInt(int64(sc.StartLine)),
				"start_col":  object.NewInt(int64(sc.StartCol)),
				"end_line":   object.NewInt(int64(sc.EndLine)),
				"end_col":    object.NewInt(int64(sc.EndCol)),
			}
			if sc.SymbolID != nil {
				m["symbol_id"] = object.NewInt(*sc.SymbolID)
			}
			if sc.ParentScopeID != nil {
				m["parent_scope_id"] = object.NewInt(*sc.ParentScopeID)
			}
			results = append(results, object.NewMap(m))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

func makeImportsByFileFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("imports_by_file", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("imports_by_file", 1, len(args))
		}
		fileID, err := toInt64(args[0])
		if err != nil {
			return object.Errorf("imports_by_file: %v", err)
		}

		imports, queryErr := s.ImportsByFile(fileID)
		if queryErr != nil {
			return object.Errorf("imports_by_file: %v", queryErr)
		}

		var results []object.Object
		for _, imp := range imports {
			m := map[string]object.Object{
				"id":     object.NewInt(imp.ID),
				"source": object.NewString(imp.Source),
				"kind":   object.NewString(imp.Kind),
			}
			if imp.ImportedName != nil {
				m["imported_name"] = object.NewString(*imp.ImportedName)
			}
			if imp.LocalAlias != nil {
				m["local_alias"] = object.NewString(*imp.LocalAlias)
			}
			results = append(results, object.NewMap(m))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

func makeTypeMembersFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("type_members", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("type_members", 1, len(args))
		}
		symbolID, err := toInt64(args[0])
		if err != nil {
			return object.Errorf("type_members: %v", err)
		}

		members, queryErr := s.TypeMembers(symbolID)
		if queryErr != nil {
			return object.Errorf("type_members: %v", queryErr)
		}

		var results []object.Object
		for _, tm := range members {
			results = append(results, object.NewMap(map[string]object.Object{
				"id":         object.NewInt(tm.ID),
				"name":       object.NewString(tm.Name),
				"kind":       object.NewString(tm.Kind),
				"type_expr":  object.NewString(tm.TypeExpr),
				"visibility": object.NewString(tm.Visibility),
			}))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

func makeFilesByLanguageFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("files_by_language", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("files_by_language", 1, len(args))
		}
		lang, err := toString(args[0])
		if err != nil {
			return object.Errorf("files_by_language: %v", err)
		}

		files, queryErr := s.FilesByLanguage(lang)
		if queryErr != nil {
			return object.Errorf("files_by_language: %v", queryErr)
		}

		var results []object.Object
		for _, f := range files {
			results = append(results, object.NewMap(map[string]object.Object{
				"id":       object.NewInt(f.ID),
				"path":     object.NewString(f.Path),
				"language": object.NewString(f.Language),
			}))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

// MakeFilesToResolveFn creates a files_to_resolve function that filters
// FilesByLanguage results to only files in the blast radius. When
// blastFileIDs is nil, returns all files (full resolve). Exported so
// engine.go can pass it as an extra global without importing object.
func MakeFilesToResolveFn(s *store.Store, blastFileIDs map[int64]bool) any {
	return object.NewBuiltin("files_to_resolve", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("files_to_resolve", 1, len(args))
		}
		lang, err := toString(args[0])
		if err != nil {
			return object.Errorf("files_to_resolve: %v", err)
		}

		files, queryErr := s.FilesByLanguage(lang)
		if queryErr != nil {
			return object.Errorf("files_to_resolve: %v", queryErr)
		}

		var results []object.Object
		for _, f := range files {
			if blastFileIDs != nil && !blastFileIDs[f.ID] {
				continue
			}
			results = append(results, object.NewMap(map[string]object.Object{
				"id":       object.NewInt(f.ID),
				"path":     object.NewString(f.Path),
				"language": object.NewString(f.Language),
			}))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

func makeSymbolsByKindFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("symbols_by_kind", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("symbols_by_kind", 1, len(args))
		}
		kind, err := toString(args[0])
		if err != nil {
			return object.Errorf("symbols_by_kind: %v", err)
		}

		syms, queryErr := s.SymbolsByKind(kind)
		if queryErr != nil {
			return object.Errorf("symbols_by_kind: %v", queryErr)
		}

		return symbolsToList(syms)
	})
}

func makeScopeChainFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("scope_chain", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("scope_chain", 1, len(args))
		}
		scopeID, err := toInt64(args[0])
		if err != nil {
			return object.Errorf("scope_chain: %v", err)
		}

		chain, queryErr := s.ScopeChain(scopeID)
		if queryErr != nil {
			return object.Errorf("scope_chain: %v", queryErr)
		}

		var results []object.Object
		for _, sc := range chain {
			m := map[string]object.Object{
				"id":         object.NewInt(sc.ID),
				"kind":       object.NewString(sc.Kind),
				"start_line": object.NewInt(int64(sc.StartLine)),
				"start_col":  object.NewInt(int64(sc.StartCol)),
				"end_line":   object.NewInt(int64(sc.EndLine)),
				"end_col":    object.NewInt(int64(sc.EndCol)),
			}
			if sc.SymbolID != nil {
				m["symbol_id"] = object.NewInt(*sc.SymbolID)
			}
			if sc.ParentScopeID != nil {
				m["parent_scope_id"] = object.NewInt(*sc.ParentScopeID)
			}
			results = append(results, object.NewMap(m))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

func makeFunctionParamsFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("function_params", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("function_params", 1, len(args))
		}
		symbolID, err := toInt64(args[0])
		if err != nil {
			return object.Errorf("function_params: %v", err)
		}

		params, queryErr := s.FunctionParams(symbolID)
		if queryErr != nil {
			return object.Errorf("function_params: %v", queryErr)
		}

		var results []object.Object
		for _, fp := range params {
			results = append(results, object.NewMap(map[string]object.Object{
				"id":          object.NewInt(fp.ID),
				"symbol_id":   object.NewInt(fp.SymbolID),
				"name":        object.NewString(fp.Name),
				"ordinal":     object.NewInt(int64(fp.Ordinal)),
				"type_expr":   object.NewString(fp.TypeExpr),
				"is_receiver": object.NewBool(fp.IsReceiver),
				"is_return":   object.NewBool(fp.IsReturn),
			}))
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

// makeDBQueryFn creates a db_query bridge that executes arbitrary read-only SQL.
// Returns a list of maps (column name → value).
func makeDBQueryFn(s *store.Store) *object.Builtin {
	return object.NewBuiltin("db_query", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) < 1 {
			return object.Errorf("db_query: expected at least 1 argument (sql), got %d", len(args))
		}
		sqlStr, err := toString(args[0])
		if err != nil {
			return object.Errorf("db_query: %v", err)
		}

		// Only allow SELECT statements.
		trimmed := strings.TrimSpace(strings.ToUpper(sqlStr))
		if !strings.HasPrefix(trimmed, "SELECT") {
			return object.Errorf("db_query: only SELECT queries are allowed")
		}

		// Convert remaining args to query parameters.
		var queryArgs []any
		for _, arg := range args[1:] {
			switch v := arg.(type) {
			case *object.Int:
				queryArgs = append(queryArgs, v.Value())
			case *object.Float:
				queryArgs = append(queryArgs, v.Value())
			case *object.String:
				queryArgs = append(queryArgs, v.Value())
			case *object.Bool:
				queryArgs = append(queryArgs, v.Value())
			case *object.NilType:
				queryArgs = append(queryArgs, nil)
			default:
				queryArgs = append(queryArgs, fmt.Sprintf("%v", arg))
			}
		}

		rows, queryErr := s.DB().QueryContext(ctx, sqlStr, queryArgs...)
		if queryErr != nil {
			return object.Errorf("db_query: %v", queryErr)
		}
		defer rows.Close()

		cols, colErr := rows.Columns()
		if colErr != nil {
			return object.Errorf("db_query: columns: %v", colErr)
		}

		var results []object.Object
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return object.Errorf("db_query: scan: %v", err)
			}
			row := make(map[string]object.Object, len(cols))
			for i, col := range cols {
				row[col] = sqlValueToObject(values[i])
			}
			results = append(results, object.NewMap(row))
		}
		if err := rows.Err(); err != nil {
			return object.Errorf("db_query: rows: %v", err)
		}
		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

// sqlValueToObject converts a database value to a Risor object.
func sqlValueToObject(v any) object.Object {
	if v == nil {
		return object.Nil
	}
	switch val := v.(type) {
	case int64:
		return object.NewInt(val)
	case float64:
		return object.NewFloat(val)
	case string:
		return object.NewString(val)
	case bool:
		return object.NewBool(val)
	case []byte:
		return object.NewString(string(val))
	default:
		return object.NewString(fmt.Sprintf("%v", val))
	}
}

// symbolsToList converts a slice of store.Symbol to a Risor list of maps.
func symbolsToList(syms []*store.Symbol) object.Object {
	var results []object.Object
	for _, sym := range syms {
		m := map[string]object.Object{
			"id":         object.NewInt(sym.ID),
			"name":       object.NewString(sym.Name),
			"kind":       object.NewString(sym.Kind),
			"visibility": object.NewString(sym.Visibility),
			"start_line": object.NewInt(int64(sym.StartLine)),
			"start_col":  object.NewInt(int64(sym.StartCol)),
			"end_line":   object.NewInt(int64(sym.EndLine)),
			"end_col":    object.NewInt(int64(sym.EndCol)),
		}
		if sym.FileID != nil {
			m["file_id"] = object.NewInt(*sym.FileID)
		}
		if sym.ParentSymbolID != nil {
			m["parent_symbol_id"] = object.NewInt(*sym.ParentSymbolID)
		}
		results = append(results, object.NewMap(m))
	}
	if results == nil {
		results = []object.Object{}
	}
	return object.NewList(results)
}

func getFloat(m map[string]object.Object, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	if f, ok := v.(*object.Float); ok {
		return f.Value()
	}
	if i, ok := v.(*object.Int); ok {
		return float64(i.Value())
	}
	return 0
}
