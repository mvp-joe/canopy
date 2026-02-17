package canopy

import (
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SymbolDetail
// =============================================================================

func TestSymbolDetail_FunctionReturnsParams(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "MyFunc", Kind: "function", Visibility: "public",
		StartLine: 0, StartCol: 0, EndLine: 9, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertFunctionParam(&store.FunctionParam{
		SymbolID: symID, Name: "ctx", Ordinal: 0, TypeExpr: "context.Context",
	})
	require.NoError(t, err)
	_, err = s.InsertFunctionParam(&store.FunctionParam{
		SymbolID: symID, Name: "name", Ordinal: 1, TypeExpr: "string",
	})
	require.NoError(t, err)
	_, err = s.InsertFunctionParam(&store.FunctionParam{
		SymbolID: symID, Name: "", Ordinal: 2, TypeExpr: "error", IsReturn: true,
	})
	require.NoError(t, err)

	detail, err := q.SymbolDetail(symID)
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Equal(t, "MyFunc", detail.Symbol.Name)
	assert.Equal(t, "function", detail.Symbol.Kind)
	assert.Len(t, detail.Parameters, 3)
	assert.Equal(t, "ctx", detail.Parameters[0].Name)
	assert.Equal(t, "name", detail.Parameters[1].Name)
	assert.True(t, detail.Parameters[2].IsReturn)

	assert.Empty(t, detail.Members)
	assert.Empty(t, detail.TypeParams)
	assert.Empty(t, detail.Annotations)
}

func TestSymbolDetail_StructReturnsMembers(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "User", Kind: "struct", Visibility: "public",
		StartLine: 0, StartCol: 0, EndLine: 9, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertTypeMember(&store.TypeMember{
		SymbolID: symID, Name: "Name", Kind: "field", TypeExpr: "string", Visibility: "public",
	})
	require.NoError(t, err)
	_, err = s.InsertTypeMember(&store.TypeMember{
		SymbolID: symID, Name: "Age", Kind: "field", TypeExpr: "int", Visibility: "public",
	})
	require.NoError(t, err)

	detail, err := q.SymbolDetail(symID)
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Equal(t, "User", detail.Symbol.Name)
	assert.Len(t, detail.Members, 2)
	assert.Equal(t, "Name", detail.Members[0].Name)
	assert.Equal(t, "Age", detail.Members[1].Name)

	assert.Empty(t, detail.Parameters)
}

func TestSymbolDetail_AnnotatedSymbol(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.py", "python")

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "my_func", Kind: "function", Visibility: "public",
		StartLine: 2, StartCol: 0, EndLine: 9, EndCol: 0,
	})
	require.NoError(t, err)

	_, err = s.InsertAnnotation(&store.Annotation{
		TargetSymbolID: symID, Name: "staticmethod", FileID: &fID, Line: 1, Col: 0,
	})
	require.NoError(t, err)
	_, err = s.InsertAnnotation(&store.Annotation{
		TargetSymbolID: symID, Name: "deprecated", Arguments: `"use new_func"`, FileID: &fID, Line: 0, Col: 0,
	})
	require.NoError(t, err)

	detail, err := q.SymbolDetail(symID)
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Len(t, detail.Annotations, 2)
	names := []string{detail.Annotations[0].Name, detail.Annotations[1].Name}
	assert.Contains(t, names, "staticmethod")
	assert.Contains(t, names, "deprecated")
}

func TestSymbolDetail_PlainVariableReturnsEmpty(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "maxRetries", Kind: "variable", Visibility: "private",
		StartLine: 0, StartCol: 0, EndLine: 0, EndCol: 20,
	})
	require.NoError(t, err)

	detail, err := q.SymbolDetail(symID)
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Equal(t, "maxRetries", detail.Symbol.Name)
	assert.Empty(t, detail.Parameters)
	assert.Empty(t, detail.Members)
	assert.Empty(t, detail.TypeParams)
	assert.Empty(t, detail.Annotations)
}

func TestSymbolDetail_GenericTypeReturnsTypeParams(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "Container", Kind: "type", Visibility: "public",
		StartLine: 0, StartCol: 0, EndLine: 9, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertTypeParam(&store.TypeParam{
		SymbolID: symID, Name: "K", Ordinal: 0, ParamKind: "type", Constraints: "comparable",
	})
	require.NoError(t, err)
	_, err = s.InsertTypeParam(&store.TypeParam{
		SymbolID: symID, Name: "V", Ordinal: 1, ParamKind: "type", Constraints: "any",
	})
	require.NoError(t, err)

	detail, err := q.SymbolDetail(symID)
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Equal(t, "Container", detail.Symbol.Name)
	require.Len(t, detail.TypeParams, 2)
	assert.Equal(t, "K", detail.TypeParams[0].Name)
	assert.Equal(t, 0, detail.TypeParams[0].Ordinal)
	assert.Equal(t, "comparable", detail.TypeParams[0].Constraints)
	assert.Equal(t, "V", detail.TypeParams[1].Name)
	assert.Equal(t, 1, detail.TypeParams[1].Ordinal)
	assert.Equal(t, "any", detail.TypeParams[1].Constraints)

	assert.Empty(t, detail.Parameters)
	assert.Empty(t, detail.Members)
}

func TestSymbolDetail_NonExistentReturnsNil(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	detail, err := q.SymbolDetail(99999)
	require.NoError(t, err)
	assert.Nil(t, detail)
}

// =============================================================================
// SymbolDetailAt
// =============================================================================

func TestSymbolDetailAt_ValidPosition(t *testing.T) {
	// Not parallel: SymbolAt reads the file from disk via os.ReadFile.
	// Using a path that doesn't exist skips validation.
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test_detail_at.go", "go")

	symID, err := s.InsertSymbol(&store.Symbol{
		FileID: &fID, Name: "Process", Kind: "function", Visibility: "public",
		StartLine: 2, StartCol: 0, EndLine: 10, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertFunctionParam(&store.FunctionParam{
		SymbolID: symID, Name: "input", Ordinal: 0, TypeExpr: "string",
	})
	require.NoError(t, err)

	// Path doesn't exist on disk, so SymbolAt skips content validation.
	detail, err := q.SymbolDetailAt("/test_detail_at.go", 5, 0)
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Equal(t, "Process", detail.Symbol.Name)
	assert.Len(t, detail.Parameters, 1)
	assert.Equal(t, "input", detail.Parameters[0].Name)
}

func TestSymbolDetailAt_NoSymbolReturnsNil(t *testing.T) {
	q, s := newTestQueryBuilder(t)
	insertFile(t, s, "/test.go", "go")

	// File in DB but no symbols — position 50 is beyond any symbol.
	detail, err := q.SymbolDetailAt("/test.go", 50, 0)
	require.NoError(t, err)
	assert.Nil(t, detail)
}

func TestSymbolDetailAt_FileNotInDB(t *testing.T) {
	q, _ := newTestQueryBuilder(t)

	detail, err := q.SymbolDetailAt("/nonexistent.go", 0, 0)
	require.NoError(t, err)
	assert.Nil(t, detail)
}

// =============================================================================
// ScopeAt
// =============================================================================

func TestScopeAt_FileLevelPosition(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	_, err := s.InsertScope(&store.Scope{
		FileID: fID, Kind: "file",
		StartLine: 0, StartCol: 0, EndLine: 50, EndCol: 0,
	})
	require.NoError(t, err)

	chain, err := q.ScopeAt("/test.go", 25, 0)
	require.NoError(t, err)
	require.Len(t, chain, 1)
	assert.Equal(t, "file", chain[0].Kind)
}

func TestScopeAt_InsideFunction(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	fileScopeID, err := s.InsertScope(&store.Scope{
		FileID: fID, Kind: "file",
		StartLine: 0, StartCol: 0, EndLine: 50, EndCol: 0,
	})
	require.NoError(t, err)

	_, err = s.InsertScope(&store.Scope{
		FileID: fID, Kind: "function", ParentScopeID: &fileScopeID,
		StartLine: 5, StartCol: 0, EndLine: 15, EndCol: 1,
	})
	require.NoError(t, err)

	chain, err := q.ScopeAt("/test.go", 10, 5)
	require.NoError(t, err)
	require.Len(t, chain, 2)
	// Innermost first
	assert.Equal(t, "function", chain[0].Kind)
	assert.Equal(t, "file", chain[1].Kind)
}

func TestScopeAt_NestedBlock(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	fileScopeID, err := s.InsertScope(&store.Scope{
		FileID: fID, Kind: "file",
		StartLine: 0, StartCol: 0, EndLine: 50, EndCol: 0,
	})
	require.NoError(t, err)

	funcScopeID, err := s.InsertScope(&store.Scope{
		FileID: fID, Kind: "function", ParentScopeID: &fileScopeID,
		StartLine: 5, StartCol: 0, EndLine: 20, EndCol: 1,
	})
	require.NoError(t, err)

	_, err = s.InsertScope(&store.Scope{
		FileID: fID, Kind: "block", ParentScopeID: &funcScopeID,
		StartLine: 8, StartCol: 1, EndLine: 12, EndCol: 1,
	})
	require.NoError(t, err)

	chain, err := q.ScopeAt("/test.go", 10, 3)
	require.NoError(t, err)
	require.Len(t, chain, 3)
	assert.Equal(t, "block", chain[0].Kind)
	assert.Equal(t, "function", chain[1].Kind)
	assert.Equal(t, "file", chain[2].Kind)
}

func TestScopeAt_FileNotInDB(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	chain, err := q.ScopeAt("/nonexistent.go", 0, 0)
	require.NoError(t, err)
	assert.Nil(t, chain)
}

func TestScopeAt_NegativeLineColReturnsNilSlice(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	_, err := s.InsertScope(&store.Scope{
		FileID: fID, Kind: "file",
		StartLine: 0, StartCol: 0, EndLine: 50, EndCol: 0,
	})
	require.NoError(t, err)

	chain, err := q.ScopeAt("/test.go", -1, -1)
	require.NoError(t, err)
	assert.Nil(t, chain)
}

func TestScopeAt_PositionOutsideAnyScope(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	// Scope only covers lines 0-10
	_, err := s.InsertScope(&store.Scope{
		FileID: fID, Kind: "file",
		StartLine: 0, StartCol: 0, EndLine: 10, EndCol: 0,
	})
	require.NoError(t, err)

	// Position beyond the scope
	chain, err := q.ScopeAt("/test.go", 20, 0)
	require.NoError(t, err)
	assert.Nil(t, chain)
}
