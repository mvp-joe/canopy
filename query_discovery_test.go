package canopy

import (
	"fmt"
	"testing"
	"time"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test helpers ---

func insertFile(t *testing.T, s *store.Store, path, lang string) int64 {
	t.Helper()
	id, err := s.InsertFile(&store.File{
		Path: path, Language: lang, Hash: "h", LastIndexed: time.Now(),
	})
	require.NoError(t, err)
	return id
}

func insertSymbol(t *testing.T, s *store.Store, fileID *int64, name, kind, visibility string, modifiers []string) int64 {
	t.Helper()
	sym := &store.Symbol{
		FileID:     fileID,
		Name:       name,
		Kind:       kind,
		Visibility: visibility,
		Modifiers:  modifiers,
		StartLine:  0, StartCol: 0, EndLine: 9, EndCol: 0,
	}
	id, err := s.InsertSymbol(sym)
	require.NoError(t, err)
	return id
}

func insertSymbolWithParent(t *testing.T, s *store.Store, fileID *int64, name, kind, visibility string, parentID int64) int64 {
	t.Helper()
	sym := &store.Symbol{
		FileID:         fileID,
		Name:           name,
		Kind:           kind,
		Visibility:     visibility,
		ParentSymbolID: &parentID,
		StartLine:      0, StartCol: 0, EndLine: 9, EndCol: 0,
	}
	id, err := s.InsertSymbol(sym)
	require.NoError(t, err)
	return id
}

func insertResolvedRef(t *testing.T, s *store.Store, fileID, targetSymbolID int64) {
	t.Helper()
	refID, err := s.InsertReference(&store.Reference{
		FileID: fileID, Name: "ref", StartLine: 0, EndLine: 0, Context: "call",
	})
	require.NoError(t, err)
	_, err = s.InsertResolvedReference(&store.ResolvedReference{
		ReferenceID: refID, TargetSymbolID: targetSymbolID, Confidence: 1.0, ResolutionKind: "direct",
	})
	require.NoError(t, err)
}

func strPtr(s string) *string { return &s }
func i64Ptr(i int64) *int64   { return &i }

// =============================================================================
// Pagination normalization
// =============================================================================

func TestPagination_Normalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Pagination
		expected Pagination
	}{
		{"zero value uses defaults", Pagination{}, Pagination{Offset: 0, Limit: 50}},
		{"negative offset becomes 0", Pagination{Offset: -5, Limit: 10}, Pagination{Offset: 0, Limit: 10}},
		{"zero limit uses default", Pagination{Offset: 0, Limit: 0}, Pagination{Offset: 0, Limit: 50}},
		{"exceeding max limit capped", Pagination{Offset: 0, Limit: 1000}, Pagination{Offset: 0, Limit: 500}},
		{"valid values unchanged", Pagination{Offset: 10, Limit: 20}, Pagination{Offset: 10, Limit: 20}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.normalize()
			assert.Equal(t, tt.expected, got)
		})
	}
}

// =============================================================================
// escapeLike
// =============================================================================

func TestEscapeLike(t *testing.T) {
	t.Parallel()
	assert.Equal(t, `hello`, escapeLike("hello"))
	assert.Equal(t, `hello\%world`, escapeLike("hello%world"))
	assert.Equal(t, `hello\_world`, escapeLike("hello_world"))
	assert.Equal(t, `hello\\world`, escapeLike(`hello\world`))
	assert.Equal(t, `\%\_\\`, escapeLike(`%_\`))
}

// =============================================================================
// normalizePathPrefix
// =============================================================================

func TestNormalizePathPrefix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", normalizePathPrefix(""))
	assert.Equal(t, "internal/store/", normalizePathPrefix("internal/store"))
	assert.Equal(t, "internal/store/", normalizePathPrefix("internal/store/"))
}

// =============================================================================
// Symbols
// =============================================================================

func TestSymbols_EmptyDB(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	result, err := q.Symbols(SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalCount)
	assert.Empty(t, result.Items)
	assert.NotNil(t, result.Items) // should be empty slice, not nil
}

func TestSymbols_ReturnsAllWhenNoFilter(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "struct", "public", nil)

	result, err := q.Symbols(SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Items, 2)
}

func TestSymbols_FilterByKind(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "struct", "public", nil)
	insertSymbol(t, s, &fID, "Baz", "interface", "public", nil)

	// Single kind
	result, err := q.Symbols(SymbolFilter{Kinds: []string{"function"}}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Foo", result.Items[0].Name)

	// Multiple kinds
	result, err = q.Symbols(SymbolFilter{Kinds: []string{"interface", "struct"}}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSymbols_FilterByVisibility(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "bar", "function", "private", nil)

	result, err := q.Symbols(SymbolFilter{Visibility: strPtr("public")}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Foo", result.Items[0].Name)
}

func TestSymbols_FilterByModifiers(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", []string{"async", "static"})
	insertSymbol(t, s, &fID, "Bar", "function", "public", []string{"async"})
	insertSymbol(t, s, &fID, "Baz", "function", "public", nil)

	// Single modifier
	result, err := q.Symbols(SymbolFilter{Modifiers: []string{"async"}}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)

	// Multiple modifiers (AND)
	result, err = q.Symbols(SymbolFilter{Modifiers: []string{"async", "static"}}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Foo", result.Items[0].Name)
}

func TestSymbols_FilterByFileID(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "a.go", "go")
	fID2 := insertFile(t, s, "b.go", "go")
	insertSymbol(t, s, &fID1, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID2, "Bar", "function", "public", nil)

	result, err := q.Symbols(SymbolFilter{FileID: &fID1}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Foo", result.Items[0].Name)
}

func TestSymbols_FilterByParentID(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	parentID := insertSymbol(t, s, &fID, "MyClass", "class", "public", nil)
	insertSymbolWithParent(t, s, &fID, "method1", "method", "public", parentID)
	insertSymbolWithParent(t, s, &fID, "method2", "method", "public", parentID)
	insertSymbol(t, s, &fID, "Other", "function", "public", nil)

	result, err := q.Symbols(SymbolFilter{ParentID: &parentID}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSymbols_FilterByPathPrefix(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "internal/store/store.go", "go")
	fID2 := insertFile(t, s, "internal/runtime/runtime.go", "go")
	fID3 := insertFile(t, s, "internal/store_utils/file.go", "go")
	insertSymbol(t, s, &fID1, "Store", "struct", "public", nil)
	insertSymbol(t, s, &fID2, "Runtime", "struct", "public", nil)
	insertSymbol(t, s, &fID3, "Util", "function", "public", nil)

	// PathPrefix "internal/store" should match internal/store/ but NOT internal/store_utils/
	result, err := q.Symbols(SymbolFilter{PathPrefix: strPtr("internal/store")}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Store", result.Items[0].Name)
}

func TestSymbols_CombinedFilters(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "internal/store/store.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "struct", "public", nil)
	insertSymbol(t, s, &fID, "baz", "function", "private", nil)

	// Kind + visibility
	result, err := q.Symbols(SymbolFilter{
		Kinds:      []string{"function"},
		Visibility: strPtr("public"),
	}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Foo", result.Items[0].Name)

	// Kind + PathPrefix
	result, err = q.Symbols(SymbolFilter{
		Kinds:      []string{"function"},
		PathPrefix: strPtr("internal/store"),
	}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount) // Foo and baz
}

func TestSymbols_Pagination(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	for i := range 5 {
		insertSymbol(t, s, &fID, fmt.Sprintf("Sym%d", i), "function", "public", nil)
	}

	// First page
	result, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Offset: 0, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "Sym0", result.Items[0].Name)
	assert.Equal(t, "Sym1", result.Items[1].Name)

	// Second page
	result, err = q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Offset: 2, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "Sym2", result.Items[0].Name)
	assert.Equal(t, "Sym3", result.Items[1].Name)

	// Last page (partial)
	result, err = q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Offset: 4, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 1)
}

func TestSymbols_SortByName(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Charlie", "function", "public", nil)
	insertSymbol(t, s, &fID, "Alice", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bob", "function", "public", nil)

	result, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "Alice", result.Items[0].Name)
	assert.Equal(t, "Bob", result.Items[1].Name)
	assert.Equal(t, "Charlie", result.Items[2].Name)
}

func TestSymbols_SortByKind(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "struct", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "function", "public", nil)
	insertSymbol(t, s, &fID, "Baz", "interface", "public", nil)

	result, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByKind, Order: Asc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "function", result.Items[0].Kind)
	assert.Equal(t, "interface", result.Items[1].Kind)
	assert.Equal(t, "struct", result.Items[2].Kind)
}

func TestSymbols_SortByFile(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "b.go", "go")
	fID2 := insertFile(t, s, "a.go", "go")
	insertSymbol(t, s, &fID1, "Bsym", "function", "public", nil)
	insertSymbol(t, s, &fID2, "Asym", "function", "public", nil)

	result, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByFile, Order: Asc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "Asym", result.Items[0].Name)
	assert.Equal(t, "Bsym", result.Items[1].Name)
}

func TestSymbols_SortByRefCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "Popular", "function", "public", nil)
	insertSymbol(t, s, &fID, "Unpopular", "function", "public", nil)

	// Add 3 resolved references to sym1
	for range 3 {
		insertResolvedRef(t, s, fID, sym1)
	}

	result, err := q.Symbols(SymbolFilter{}, Sort{Field: SortByRefCount, Order: Desc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "Popular", result.Items[0].Name)
	assert.Equal(t, 3, result.Items[0].RefCount)
	assert.Equal(t, "Unpopular", result.Items[1].Name)
	assert.Equal(t, 0, result.Items[1].RefCount)
}

func TestSymbols_IncludesFilePath(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "internal/store/store.go", "go")
	insertSymbol(t, s, &fID, "Store", "struct", "public", nil)

	result, err := q.Symbols(SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "internal/store/store.go", result.Items[0].FilePath)
}

func TestSymbols_NilFileIDEmptyFilePath(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	// Symbol without a file
	insertSymbol(t, s, nil, "mypkg", "package", "public", nil)

	result, err := q.Symbols(SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "", result.Items[0].FilePath)
}

func TestSymbols_InvalidParentIDReturnsEmpty(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)

	nonexistent := int64(99999)
	result, err := q.Symbols(SymbolFilter{ParentID: &nonexistent}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalCount)
	assert.Empty(t, result.Items)
}

// =============================================================================
// Symbols — RefCount Filters
// =============================================================================

func intPtr(i int) *int { return &i }

func TestSymbols_RefCountMin_ExcludesZeroRefs(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "Referenced", "function", "public", nil)
	insertSymbol(t, s, &fID, "Unreferenced", "function", "public", nil)

	insertResolvedRef(t, s, fID, sym1)

	result, err := q.Symbols(SymbolFilter{RefCountMin: intPtr(1)}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Referenced", result.Items[0].Name)
}

func TestSymbols_RefCountMax_ReturnsOnlyZeroRefs(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "Referenced", "function", "public", nil)
	insertSymbol(t, s, &fID, "Unreferenced", "function", "public", nil)

	insertResolvedRef(t, s, fID, sym1)

	result, err := q.Symbols(SymbolFilter{RefCountMax: intPtr(0)}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Unreferenced", result.Items[0].Name)
}

func TestSymbols_RefCountRange(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "Low", "function", "public", nil)    // 1 ref
	sym2 := insertSymbol(t, s, &fID, "Mid", "function", "public", nil)    // 3 refs
	sym3 := insertSymbol(t, s, &fID, "High", "function", "public", nil)   // 6 refs
	insertSymbol(t, s, &fID, "Zero", "function", "public", nil)           // 0 refs

	insertResolvedRef(t, s, fID, sym1)
	for range 3 {
		insertResolvedRef(t, s, fID, sym2)
	}
	for range 6 {
		insertResolvedRef(t, s, fID, sym3)
	}

	result, err := q.Symbols(SymbolFilter{RefCountMin: intPtr(2), RefCountMax: intPtr(5)}, Sort{Field: SortByName, Order: Asc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Mid", result.Items[0].Name)
}

func TestSymbols_RefCountMin_NoMatch_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "function", "public", nil)

	result, err := q.Symbols(SymbolFilter{RefCountMin: intPtr(100)}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalCount)
	assert.Empty(t, result.Items)
	assert.NotNil(t, result.Items)
}

func TestSymbols_RefCount_TotalCountReflectsFilter(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "A", "function", "public", nil)
	sym2 := insertSymbol(t, s, &fID, "B", "function", "public", nil)
	insertSymbol(t, s, &fID, "C", "function", "public", nil) // 0 refs

	insertResolvedRef(t, s, fID, sym1)
	insertResolvedRef(t, s, fID, sym2)

	// Without filter: 3 total
	all, err := q.Symbols(SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 3, all.TotalCount)

	// With RefCountMin=1: TotalCount should be 2, not 3
	filtered, err := q.Symbols(SymbolFilter{RefCountMin: intPtr(1)}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, filtered.TotalCount)
	assert.Len(t, filtered.Items, 2)
}

// =============================================================================
// SearchSymbols — RefCount Filters
// =============================================================================

func TestSearchSymbols_RefCountMin_FiltersResults(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "GetUser", "function", "public", nil)
	insertSymbol(t, s, &fID, "GetAdmin", "function", "public", nil)

	for range 3 {
		insertResolvedRef(t, s, fID, sym1)
	}

	result, err := q.SearchSymbols("Get*", SymbolFilter{RefCountMin: intPtr(1)}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "GetUser", result.Items[0].Name)
}

// =============================================================================
// Files
// =============================================================================

func TestFiles_EmptyDB(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	result, err := q.Files("", "", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalCount)
	assert.NotNil(t, result.Items)
	assert.Empty(t, result.Items)
}

func TestFiles_NoFilter(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	insertFile(t, s, "a.go", "go")
	insertFile(t, s, "b.py", "python")

	result, err := q.Files("", "", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Items, 2)
}

func TestFiles_FilterByLanguage(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	insertFile(t, s, "a.go", "go")
	insertFile(t, s, "b.go", "go")
	insertFile(t, s, "c.py", "python")

	result, err := q.Files("", "go", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestFiles_FilterByPathPrefix(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	insertFile(t, s, "internal/store/store.go", "go")
	insertFile(t, s, "internal/store/helpers.go", "go")
	insertFile(t, s, "internal/runtime/runtime.go", "go")
	insertFile(t, s, "internal/store_utils/util.go", "go")

	result, err := q.Files("internal/store", "", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount) // store.go and helpers.go, NOT store_utils
}

func TestFiles_CombinedFilter(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	insertFile(t, s, "internal/store/store.go", "go")
	insertFile(t, s, "internal/store/store.py", "python")
	insertFile(t, s, "internal/runtime/runtime.go", "go")

	result, err := q.Files("internal/store", "go", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "internal/store/store.go", result.Items[0].Path)
}

func TestFiles_SortByPath(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	insertFile(t, s, "c.go", "go")
	insertFile(t, s, "a.go", "go")
	insertFile(t, s, "b.go", "go")

	result, err := q.Files("", "", Sort{Field: SortByFile, Order: Asc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "a.go", result.Items[0].Path)
	assert.Equal(t, "b.go", result.Items[1].Path)
	assert.Equal(t, "c.go", result.Items[2].Path)
}

func TestFiles_Pagination(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	for i := range 5 {
		insertFile(t, s, fmt.Sprintf("file%d.go", i), "go")
	}

	result, err := q.Files("", "", Sort{Field: SortByFile, Order: Asc}, Pagination{Offset: 0, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 2)
}

func TestFiles_InapplicableSortFallsBack(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	insertFile(t, s, "b.go", "go")
	insertFile(t, s, "a.go", "go")

	// SortByRefCount is inapplicable to Files, should fall back to path
	result, err := q.Files("", "", Sort{Field: SortByRefCount, Order: Asc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "a.go", result.Items[0].Path)
	assert.Equal(t, "b.go", result.Items[1].Path)
}

// =============================================================================
// Packages
// =============================================================================

func TestPackages_ReturnsPackageModuleNamespace(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "mypkg", "package", "public", nil)
	insertSymbol(t, s, &fID, "mymod", "module", "public", nil)
	insertSymbol(t, s, &fID, "myns", "namespace", "public", nil)
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "class", "public", nil)

	result, err := q.Packages("", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalCount)
	for _, item := range result.Items {
		assert.Contains(t, []string{"package", "module", "namespace"}, item.Kind)
	}
}

func TestPackages_FilterByPathPrefix(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "internal/store/store.go", "go")
	fID2 := insertFile(t, s, "internal/runtime/runtime.go", "go")
	insertSymbol(t, s, &fID1, "store", "package", "public", nil)
	insertSymbol(t, s, &fID2, "runtime", "package", "public", nil)

	result, err := q.Packages("internal/store", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "store", result.Items[0].Name)
}

func TestPackages_EmptyPathPrefixReturnsAll(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "pkg1", "package", "public", nil)
	insertSymbol(t, s, &fID, "pkg2", "package", "public", nil)

	result, err := q.Packages("", Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestPackages_SortByName(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "zeta", "package", "public", nil)
	insertSymbol(t, s, &fID, "alpha", "package", "public", nil)

	result, err := q.Packages("", Sort{Field: SortByName, Order: Asc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "alpha", result.Items[0].Name)
	assert.Equal(t, "zeta", result.Items[1].Name)
}

func TestPackages_SortByRefCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	pkg1 := insertSymbol(t, s, &fID, "popular", "package", "public", nil)
	insertSymbol(t, s, &fID, "unpopular", "package", "public", nil)

	for range 5 {
		insertResolvedRef(t, s, fID, pkg1)
	}

	result, err := q.Packages("", Sort{Field: SortByRefCount, Order: Desc}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, "popular", result.Items[0].Name)
	assert.Equal(t, 5, result.Items[0].RefCount)
}

// =============================================================================
// SearchSymbols
// =============================================================================

func TestSearchSymbols_PrefixMatch(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Animal", "class", "public", nil)
	insertSymbol(t, s, &fID, "Animation", "class", "public", nil)
	insertSymbol(t, s, &fID, "inanimate", "function", "private", nil)

	result, err := q.SearchSymbols("Anim*", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSearchSymbols_SuffixMatch(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Animal", "class", "public", nil)
	insertSymbol(t, s, &fID, "Animals", "class", "public", nil)

	result, err := q.SearchSymbols("*mal", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Animal", result.Items[0].Name)
}

func TestSearchSymbols_InfixWildcard(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Animal", "class", "public", nil)
	insertSymbol(t, s, &fID, "Animation", "class", "public", nil)

	result, err := q.SearchSymbols("Anim*l", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Animal", result.Items[0].Name)
}

func TestSearchSymbols_Contains(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "UserController", "class", "public", nil)
	insertSymbol(t, s, &fID, "ControllerBase", "class", "public", nil)
	insertSymbol(t, s, &fID, "Service", "class", "public", nil)

	result, err := q.SearchSymbols("*Controller*", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSearchSymbols_MultipleWildcards(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "GetCurrentUser", "function", "public", nil)
	insertSymbol(t, s, &fID, "GetUserByID", "function", "public", nil)
	insertSymbol(t, s, &fID, "SetUser", "function", "public", nil)

	result, err := q.SearchSymbols("Get*User*", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSearchSymbols_EmptyPatternReturnsAll(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "function", "public", nil)

	result, err := q.SearchSymbols("", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSearchSymbols_StarReturnsAll(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)

	result, err := q.SearchSymbols("*", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
}

func TestSearchSymbols_ExactMatch(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "FooBar", "function", "public", nil)

	result, err := q.SearchSymbols("Foo", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "Foo", result.Items[0].Name)
}

func TestSearchSymbols_CaseInsensitive(t *testing.T) {
	// SQLite LIKE is case-insensitive for ASCII by default, which is useful for symbol search.
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Animal", "class", "public", nil)
	insertSymbol(t, s, &fID, "animal", "function", "private", nil)

	result, err := q.SearchSymbols("animal", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount) // LIKE is case-insensitive
}

func TestSearchSymbols_EscapesSpecialChars(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "my_func", "function", "public", nil)
	insertSymbol(t, s, &fID, "myXfunc", "function", "public", nil) // _ should not match X

	result, err := q.SearchSymbols("my_func", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "my_func", result.Items[0].Name)
}

func TestSearchSymbols_CombinedWithFilter(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "GetUser", "function", "public", nil)
	insertSymbol(t, s, &fID, "GetAdmin", "function", "private", nil)

	result, err := q.SearchSymbols("Get*", SymbolFilter{Visibility: strPtr("public")}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "GetUser", result.Items[0].Name)
}

func TestSearchSymbols_CombinedWithPathPrefix(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "internal/store/store.go", "go")
	fID2 := insertFile(t, s, "internal/runtime/runtime.go", "go")
	insertSymbol(t, s, &fID1, "GetStore", "function", "public", nil)
	insertSymbol(t, s, &fID2, "GetRuntime", "function", "public", nil)

	result, err := q.SearchSymbols("Get*", SymbolFilter{PathPrefix: strPtr("internal/store")}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, "GetStore", result.Items[0].Name)
}

func TestSearchSymbols_Pagination(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	for i := range 5 {
		insertSymbol(t, s, &fID, fmt.Sprintf("Get%d", i), "function", "public", nil)
	}

	result, err := q.SearchSymbols("Get*", SymbolFilter{}, Sort{Field: SortByName, Order: Asc}, Pagination{Offset: 0, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCount)
	assert.Len(t, result.Items, 2)
}

// =============================================================================
// ProjectSummary
// =============================================================================

func TestProjectSummary_EmptyDB(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	summary, err := q.ProjectSummary(10)
	require.NoError(t, err)
	assert.NotNil(t, summary.Languages)
	assert.Empty(t, summary.Languages)
	assert.Equal(t, 0, summary.PackageCount)
	assert.NotNil(t, summary.TopSymbols)
	assert.Empty(t, summary.TopSymbols)
}

func TestProjectSummary_SingleLanguage(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "struct", "public", nil)
	insertSymbol(t, s, &fID, "mypkg", "package", "public", nil)

	summary, err := q.ProjectSummary(10)
	require.NoError(t, err)
	require.Len(t, summary.Languages, 1)
	assert.Equal(t, "go", summary.Languages[0].Language)
	assert.Equal(t, 1, summary.Languages[0].FileCount)
	assert.Equal(t, 3, summary.Languages[0].SymbolCount)
	assert.Equal(t, 1, summary.Languages[0].KindCounts["function"])
	assert.Equal(t, 1, summary.Languages[0].KindCounts["struct"])
	assert.Equal(t, 1, summary.PackageCount)
}

func TestProjectSummary_MultiLanguage(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "main.go", "go")
	fID2 := insertFile(t, s, "app.py", "python")
	insertSymbol(t, s, &fID1, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID2, "bar", "function", "public", nil)
	insertSymbol(t, s, &fID2, "Baz", "class", "public", nil)

	summary, err := q.ProjectSummary(10)
	require.NoError(t, err)
	assert.Len(t, summary.Languages, 2)

	// Find go stats
	var goStats *LanguageStats
	for i := range summary.Languages {
		if summary.Languages[i].Language == "go" {
			goStats = &summary.Languages[i]
		}
	}
	require.NotNil(t, goStats)
	assert.Equal(t, 1, goStats.FileCount)
	assert.Equal(t, 1, goStats.SymbolCount)
}

func TestProjectSummary_TopSymbols(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym1 := insertSymbol(t, s, &fID, "Popular", "function", "public", nil)
	sym2 := insertSymbol(t, s, &fID, "Medium", "function", "public", nil)
	insertSymbol(t, s, &fID, "Unpopular", "function", "public", nil) // no refs

	for range 5 {
		insertResolvedRef(t, s, fID, sym1)
	}
	for range 2 {
		insertResolvedRef(t, s, fID, sym2)
	}

	summary, err := q.ProjectSummary(2)
	require.NoError(t, err)
	require.Len(t, summary.TopSymbols, 2)
	assert.Equal(t, "Popular", summary.TopSymbols[0].Name)
	assert.Equal(t, 5, summary.TopSymbols[0].RefCount)
	assert.Equal(t, "Medium", summary.TopSymbols[1].Name)
	assert.Equal(t, 2, summary.TopSymbols[1].RefCount)
}

func TestProjectSummary_TopNZero(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)

	summary, err := q.ProjectSummary(0)
	require.NoError(t, err)
	assert.Empty(t, summary.TopSymbols)
	assert.Equal(t, 1, summary.Languages[0].SymbolCount) // still returns stats
}

func TestProjectSummary_TopNExceedsTotal(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	sym := insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertResolvedRef(t, s, fID, sym)

	summary, err := q.ProjectSummary(100)
	require.NoError(t, err)
	assert.Len(t, summary.TopSymbols, 1)
}

func TestProjectSummary_PackageCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	insertSymbol(t, s, &fID, "pkg1", "package", "public", nil)
	insertSymbol(t, s, &fID, "mod1", "module", "public", nil)
	insertSymbol(t, s, &fID, "ns1", "namespace", "public", nil)
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil) // not a package

	summary, err := q.ProjectSummary(0)
	require.NoError(t, err)
	assert.Equal(t, 3, summary.PackageCount)
}

// =============================================================================
// PackageSummary
// =============================================================================

func TestPackageSummary_ByPath(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "internal/store/store.go", "go")
	insertSymbol(t, s, &fID, "store", "package", "public", nil)
	insertSymbol(t, s, &fID, "Store", "struct", "public", nil)
	insertSymbol(t, s, &fID, "helper", "function", "private", nil)

	summary, err := q.PackageSummary("internal/store", nil)
	require.NoError(t, err)
	assert.Equal(t, "store", summary.Symbol.Name)
	assert.Equal(t, 1, summary.FileCount)
	// Only public non-package symbols
	assert.Len(t, summary.ExportedSymbols, 1)
	assert.Equal(t, "Store", summary.ExportedSymbols[0].Name)
}

func TestPackageSummary_ByID(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	pkgID := insertSymbol(t, s, &fID, "mypkg", "package", "public", nil)

	summary, err := q.PackageSummary("", i64Ptr(pkgID))
	require.NoError(t, err)
	assert.Equal(t, "mypkg", summary.Symbol.Name)
}

func TestPackageSummary_IDTakesPrecedence(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "internal/store/store.go", "go")
	fID2 := insertFile(t, s, "internal/runtime/runtime.go", "go")
	insertSymbol(t, s, &fID1, "store", "package", "public", nil)
	runtimePkgID := insertSymbol(t, s, &fID2, "runtime", "package", "public", nil)

	// packagePath points to store, but packageID points to runtime — ID wins
	summary, err := q.PackageSummary("internal/store", i64Ptr(runtimePkgID))
	require.NoError(t, err)
	assert.Equal(t, "runtime", summary.Symbol.Name)
}

func TestPackageSummary_NonexistentPath(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	_, err := q.PackageSummary("nonexistent/path", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no package found")
}

func TestPackageSummary_NeitherPathNorID(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	_, err := q.PackageSummary("", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either packagePath or packageID must be provided")
}

func TestPackageSummary_ExportedSortedByRefCount(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "internal/store/store.go", "go")
	otherFID := insertFile(t, s, "cmd/main.go", "go")
	insertSymbol(t, s, &fID, "store", "package", "public", nil)
	sym1 := insertSymbol(t, s, &fID, "Alpha", "function", "public", nil)
	sym2 := insertSymbol(t, s, &fID, "Beta", "function", "public", nil)

	// Use otherFID so refs count as external.
	for range 5 {
		insertResolvedRef(t, s, otherFID, sym2)
	}
	insertResolvedRef(t, s, otherFID, sym1)

	summary, err := q.PackageSummary("internal/store", nil)
	require.NoError(t, err)
	require.Len(t, summary.ExportedSymbols, 2)
	assert.Equal(t, "Beta", summary.ExportedSymbols[0].Name)  // 5 ext refs
	assert.Equal(t, "Alpha", summary.ExportedSymbols[1].Name) // 1 ext ref
}

func TestPackageSummary_KindCounts(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "internal/store/store.go", "go")
	insertSymbol(t, s, &fID, "store", "package", "public", nil)
	insertSymbol(t, s, &fID, "Foo", "function", "public", nil)
	insertSymbol(t, s, &fID, "Bar", "function", "public", nil)
	insertSymbol(t, s, &fID, "Baz", "struct", "public", nil)

	summary, err := q.PackageSummary("internal/store", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.KindCounts["function"])
	assert.Equal(t, 1, summary.KindCounts["struct"])
	assert.Equal(t, 1, summary.KindCounts["package"])
}

func TestPackageSummary_Dependencies(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "internal/store/store.go", "go")
	insertSymbol(t, s, &fID, "store", "package", "public", nil)
	_, err := s.InsertImport(&store.Import{FileID: fID, Source: "fmt", Kind: "module", Scope: "file"})
	require.NoError(t, err)
	_, err = s.InsertImport(&store.Import{FileID: fID, Source: "database/sql", Kind: "module", Scope: "file"})
	require.NoError(t, err)

	summary, err := q.PackageSummary("internal/store", nil)
	require.NoError(t, err)
	assert.Len(t, summary.Dependencies, 2)
	assert.Contains(t, summary.Dependencies, "fmt")
	assert.Contains(t, summary.Dependencies, "database/sql")
}

func TestPackageSummary_Dependents(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)

	// Package under test: internal/store
	storeFID := insertFile(t, s, "internal/store/store.go", "go")
	insertSymbol(t, s, &storeFID, "store", "package", "public", nil)

	// Another package that imports "store"
	mainFID := insertFile(t, s, "cmd/main.go", "go")
	insertSymbol(t, s, &mainFID, "main", "package", "public", nil)
	_, err := s.InsertImport(&store.Import{FileID: mainFID, Source: "store", Kind: "module", Scope: "file"})
	require.NoError(t, err)

	// A file within the same package should NOT appear as a dependent
	storeHelperFID := insertFile(t, s, "internal/store/helpers.go", "go")
	_, err = s.InsertImport(&store.Import{FileID: storeHelperFID, Source: "fmt", Kind: "module", Scope: "file"})
	require.NoError(t, err)

	summary, err := q.PackageSummary("internal/store", nil)
	require.NoError(t, err)
	assert.Len(t, summary.Dependents, 1)
	assert.Equal(t, "cmd/main.go", summary.Dependents[0])
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestIntegration_SearchThenReferencesTo(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	symID := insertSymbol(t, s, &fID, "MyFunc", "function", "public", nil)

	// Create a reference to MyFunc
	refID, err := s.InsertReference(&store.Reference{
		FileID: fID, Name: "MyFunc", StartLine: 10, StartCol: 2, EndLine: 10, EndCol: 8, Context: "call",
	})
	require.NoError(t, err)
	_, err = s.InsertResolvedReference(&store.ResolvedReference{
		ReferenceID: refID, TargetSymbolID: symID, Confidence: 1.0, ResolutionKind: "direct",
	})
	require.NoError(t, err)

	// Search for the symbol
	searchResult, err := q.SearchSymbols("MyFunc", SymbolFilter{}, Sort{}, Pagination{})
	require.NoError(t, err)
	require.Len(t, searchResult.Items, 1)

	// Use the symbol ID to find references
	locs, err := q.ReferencesTo(searchResult.Items[0].ID)
	require.NoError(t, err)
	assert.Len(t, locs, 1)
	assert.Equal(t, 10, locs[0].StartLine)
}

func TestIntegration_PackagesScopeSymbols(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID1 := insertFile(t, s, "internal/store/store.go", "go")
	fID2 := insertFile(t, s, "internal/runtime/runtime.go", "go")
	insertSymbol(t, s, &fID1, "store", "package", "public", nil)
	insertSymbol(t, s, &fID1, "Store", "struct", "public", nil)
	insertSymbol(t, s, &fID2, "runtime", "package", "public", nil)
	insertSymbol(t, s, &fID2, "Runtime", "struct", "public", nil)

	// List packages
	pkgs, err := q.Packages("internal/store", Sort{}, Pagination{})
	require.NoError(t, err)
	require.Len(t, pkgs.Items, 1)

	// Use the package's file path to scope a Symbols query
	result, err := q.Symbols(SymbolFilter{PathPrefix: strPtr("internal/store")}, Sort{}, Pagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount) // package + struct
	for _, item := range result.Items {
		assert.Equal(t, "internal/store/store.go", item.FilePath)
	}
}

func TestIntegration_ProjectSummaryTopToCallers(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "main.go", "go")
	callerID := insertSymbol(t, s, &fID, "main", "function", "public", nil)
	calleeID := insertSymbol(t, s, &fID, "helper", "function", "public", nil)

	// Create ref count and call edge
	for range 3 {
		insertResolvedRef(t, s, fID, calleeID)
	}
	_, err := s.InsertCallEdge(&store.CallEdge{
		CallerSymbolID: callerID, CalleeSymbolID: calleeID, FileID: &fID, Line: 5, Col: 2,
	})
	require.NoError(t, err)

	// Get top symbols
	summary, err := q.ProjectSummary(1)
	require.NoError(t, err)
	require.Len(t, summary.TopSymbols, 1)
	assert.Equal(t, "helper", summary.TopSymbols[0].Name)

	// Use the top symbol ID to find callers
	callers, err := q.Callers(summary.TopSymbols[0].ID)
	require.NoError(t, err)
	require.Len(t, callers, 1)
	assert.Equal(t, callerID, callers[0].CallerSymbolID)
}
