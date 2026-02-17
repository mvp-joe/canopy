package canopy

import (
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TypeHierarchy
// =============================================================================

func TestTypeHierarchy_InterfaceReturnsImplementedBy(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	ifaceID := insertSymbol(t, s, &fID, "Reader", "interface", "public", nil)
	typeAID := insertSymbol(t, s, &fID, "FileReader", "struct", "public", nil)
	typeBID := insertSymbol(t, s, &fID, "BufReader", "struct", "public", nil)

	_, err := s.InsertImplementation(&store.Implementation{
		TypeSymbolID: typeAID, InterfaceSymbolID: ifaceID, Kind: "interface_impl", FileID: &fID,
	})
	require.NoError(t, err)
	_, err = s.InsertImplementation(&store.Implementation{
		TypeSymbolID: typeBID, InterfaceSymbolID: ifaceID, Kind: "interface_impl", FileID: &fID,
	})
	require.NoError(t, err)

	h, err := q.TypeHierarchy(ifaceID)
	require.NoError(t, err)
	require.NotNil(t, h)

	assert.Equal(t, "Reader", h.Symbol.Name)
	assert.Len(t, h.ImplementedBy, 2)
	names := []string{h.ImplementedBy[0].Symbol.Name, h.ImplementedBy[1].Symbol.Name}
	assert.Contains(t, names, "FileReader")
	assert.Contains(t, names, "BufReader")
	assert.Equal(t, "interface_impl", h.ImplementedBy[0].Kind)
	assert.Empty(t, h.Implements)
	assert.Empty(t, h.Composes)
	assert.Empty(t, h.ComposedBy)
	assert.Empty(t, h.Extensions)
}

func TestTypeHierarchy_ConcreteTypeReturnsImplements(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	ifaceID := insertSymbol(t, s, &fID, "Writer", "interface", "public", nil)
	typeID := insertSymbol(t, s, &fID, "FileWriter", "struct", "public", nil)

	_, err := s.InsertImplementation(&store.Implementation{
		TypeSymbolID: typeID, InterfaceSymbolID: ifaceID, Kind: "interface_impl", FileID: &fID,
	})
	require.NoError(t, err)

	h, err := q.TypeHierarchy(typeID)
	require.NoError(t, err)
	require.NotNil(t, h)

	assert.Equal(t, "FileWriter", h.Symbol.Name)
	require.Len(t, h.Implements, 1)
	assert.Equal(t, "Writer", h.Implements[0].Symbol.Name)
	assert.Equal(t, "interface_impl", h.Implements[0].Kind)
	assert.Empty(t, h.ImplementedBy)
}

func TestTypeHierarchy_EmbeddingReturnsComposes(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	baseID := insertSymbol(t, s, &fID, "Base", "struct", "public", nil)
	childID := insertSymbol(t, s, &fID, "Child", "struct", "public", nil)

	_, err := s.InsertTypeComposition(&store.TypeComposition{
		CompositeSymbolID: childID, ComponentSymbolID: baseID, CompositionKind: "embedding",
	})
	require.NoError(t, err)

	h, err := q.TypeHierarchy(childID)
	require.NoError(t, err)
	require.NotNil(t, h)

	assert.Equal(t, "Child", h.Symbol.Name)
	require.Len(t, h.Composes, 1)
	assert.Equal(t, "Base", h.Composes[0].Symbol.Name)
	assert.Equal(t, "embedding", h.Composes[0].Kind)
	assert.Empty(t, h.ComposedBy)
}

func TestTypeHierarchy_BaseTypeReturnsComposedBy(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	baseID := insertSymbol(t, s, &fID, "Base", "struct", "public", nil)
	childID := insertSymbol(t, s, &fID, "Derived", "struct", "public", nil)

	_, err := s.InsertTypeComposition(&store.TypeComposition{
		CompositeSymbolID: childID, ComponentSymbolID: baseID, CompositionKind: "embedding",
	})
	require.NoError(t, err)

	h, err := q.TypeHierarchy(baseID)
	require.NoError(t, err)
	require.NotNil(t, h)

	assert.Equal(t, "Base", h.Symbol.Name)
	require.Len(t, h.ComposedBy, 1)
	assert.Equal(t, "Derived", h.ComposedBy[0].Symbol.Name)
	assert.Equal(t, "embedding", h.ComposedBy[0].Kind)
	assert.Empty(t, h.Composes)
}

func TestTypeHierarchy_ExtensionMethods(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.rs", "rust")

	typeID := insertSymbol(t, s, &fID, "MyStruct", "struct", "public", nil)
	methodID := insertSymbol(t, s, &fID, "do_thing", "method", "public", nil)

	_, err := s.InsertExtensionBinding(&store.ExtensionBinding{
		MemberSymbolID:       methodID,
		ExtendedTypeExpr:     "MyStruct",
		ExtendedTypeSymbolID: &typeID,
		Kind:                 "impl_method",
	})
	require.NoError(t, err)

	h, err := q.TypeHierarchy(typeID)
	require.NoError(t, err)
	require.NotNil(t, h)

	require.Len(t, h.Extensions, 1)
	assert.Equal(t, methodID, h.Extensions[0].MemberSymbolID)
	assert.Equal(t, "impl_method", h.Extensions[0].Kind)
}

func TestTypeHierarchy_NoRelationshipsReturnsEmptyHierarchy(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	typeID := insertSymbol(t, s, &fID, "Isolated", "struct", "public", nil)

	h, err := q.TypeHierarchy(typeID)
	require.NoError(t, err)
	require.NotNil(t, h)

	assert.Equal(t, "Isolated", h.Symbol.Name)
	assert.Empty(t, h.Implements)
	assert.Empty(t, h.ImplementedBy)
	assert.Empty(t, h.Composes)
	assert.Empty(t, h.ComposedBy)
	assert.Empty(t, h.Extensions)
}

func TestTypeHierarchy_FunctionSymbolReturnsEmptyHierarchy(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	funcID := insertSymbol(t, s, &fID, "DoWork", "function", "public", nil)

	h, err := q.TypeHierarchy(funcID)
	require.NoError(t, err)
	require.NotNil(t, h)

	assert.Equal(t, "DoWork", h.Symbol.Name)
	assert.Equal(t, "function", h.Symbol.Kind)
	assert.Empty(t, h.Implements)
	assert.Empty(t, h.ImplementedBy)
	assert.Empty(t, h.Composes)
	assert.Empty(t, h.ComposedBy)
	assert.Empty(t, h.Extensions)
}

func TestTypeHierarchy_NonExistentReturnsNil(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	h, err := q.TypeHierarchy(99999)
	require.NoError(t, err)
	assert.Nil(t, h)
}

// =============================================================================
// ImplementsInterfaces
// =============================================================================

func TestImplementsInterfaces_OneInterface(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	ifaceID := insertSymbol(t, s, &fID, "Reader", "interface", "public", nil)
	typeID := insertSymbol(t, s, &fID, "MyReader", "struct", "public", nil)

	_, err := s.InsertImplementation(&store.Implementation{
		TypeSymbolID: typeID, InterfaceSymbolID: ifaceID, Kind: "interface_impl", FileID: &fID,
	})
	require.NoError(t, err)

	locs, err := q.ImplementsInterfaces(typeID)
	require.NoError(t, err)
	require.Len(t, locs, 1)
	assert.Equal(t, "/test.go", locs[0].File)
}

func TestImplementsInterfaces_MultipleInterfaces(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	iface1 := insertSymbol(t, s, &fID, "Reader", "interface", "public", nil)
	iface2 := insertSymbol(t, s, &fID, "Closer", "interface", "public", nil)
	typeID := insertSymbol(t, s, &fID, "File", "struct", "public", nil)

	_, err := s.InsertImplementation(&store.Implementation{
		TypeSymbolID: typeID, InterfaceSymbolID: iface1, Kind: "interface_impl", FileID: &fID,
	})
	require.NoError(t, err)
	_, err = s.InsertImplementation(&store.Implementation{
		TypeSymbolID: typeID, InterfaceSymbolID: iface2, Kind: "interface_impl", FileID: &fID,
	})
	require.NoError(t, err)

	locs, err := q.ImplementsInterfaces(typeID)
	require.NoError(t, err)
	assert.Len(t, locs, 2)
}

func TestImplementsInterfaces_NoInterfaces(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	typeID := insertSymbol(t, s, &fID, "Plain", "struct", "public", nil)

	locs, err := q.ImplementsInterfaces(typeID)
	require.NoError(t, err)
	assert.Empty(t, locs)
	assert.NotNil(t, locs) // empty slice, not nil
}

// =============================================================================
// ExtensionMethods
// =============================================================================

func TestExtensionMethods_WithBindings(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.rs", "rust")

	typeID := insertSymbol(t, s, &fID, "Vec", "struct", "public", nil)
	m1 := insertSymbol(t, s, &fID, "push", "method", "public", nil)
	m2 := insertSymbol(t, s, &fID, "pop", "method", "public", nil)

	_, err := s.InsertExtensionBinding(&store.ExtensionBinding{
		MemberSymbolID: m1, ExtendedTypeExpr: "Vec", ExtendedTypeSymbolID: &typeID, Kind: "impl_method",
	})
	require.NoError(t, err)
	_, err = s.InsertExtensionBinding(&store.ExtensionBinding{
		MemberSymbolID: m2, ExtendedTypeExpr: "Vec", ExtendedTypeSymbolID: &typeID, Kind: "impl_method",
	})
	require.NoError(t, err)

	bindings, err := q.ExtensionMethods(typeID)
	require.NoError(t, err)
	assert.Len(t, bindings, 2)

	memberIDs := []int64{bindings[0].MemberSymbolID, bindings[1].MemberSymbolID}
	assert.Contains(t, memberIDs, m1)
	assert.Contains(t, memberIDs, m2)
}

func TestExtensionMethods_NoExtensions(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/test.go", "go")

	typeID := insertSymbol(t, s, &fID, "Plain", "struct", "public", nil)

	bindings, err := q.ExtensionMethods(typeID)
	require.NoError(t, err)
	assert.Empty(t, bindings)
	assert.NotNil(t, bindings)
}

// =============================================================================
// Reexports
// =============================================================================

func TestReexports_FileWithReexports(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/index.ts", "typescript")

	sym1 := insertSymbol(t, s, &fID, "Foo", "class", "public", nil)
	sym2 := insertSymbol(t, s, &fID, "Bar", "function", "public", nil)

	_, err := s.InsertReexport(&store.Reexport{
		FileID: fID, OriginalSymbolID: sym1, ExportedName: "Foo",
	})
	require.NoError(t, err)
	_, err = s.InsertReexport(&store.Reexport{
		FileID: fID, OriginalSymbolID: sym2, ExportedName: "Bar",
	})
	require.NoError(t, err)

	reexports, err := q.Reexports(fID)
	require.NoError(t, err)
	assert.Len(t, reexports, 2)

	exportedNames := []string{reexports[0].ExportedName, reexports[1].ExportedName}
	assert.Contains(t, exportedNames, "Foo")
	assert.Contains(t, exportedNames, "Bar")
}

func TestReexports_FileWithNoReexports(t *testing.T) {
	t.Parallel()
	q, s := newTestQueryBuilder(t)
	fID := insertFile(t, s, "/plain.ts", "typescript")

	reexports, err := q.Reexports(fID)
	require.NoError(t, err)
	assert.Empty(t, reexports)
	assert.NotNil(t, reexports)
	_ = fID // ensure file exists but has no reexports
}

func TestReexports_NonExistentFileID(t *testing.T) {
	t.Parallel()
	q, _ := newTestQueryBuilder(t)

	reexports, err := q.Reexports(99999)
	require.NoError(t, err)
	assert.Empty(t, reexports)
	assert.NotNil(t, reexports)
}
