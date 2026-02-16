package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchedStore_SymbolsByFile_ReturnsBufferedSymbols(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	// Insert a real file into the database (simulates Phase A of parallel extraction).
	f := insertTestFile(t, s, "/main.go", "go")

	// Create a BatchedStore (simulates what a worker goroutine uses).
	batch := NewBatchedStore(s)

	// Insert symbols into the batch (not committed to DB yet).
	id1, err := batch.InsertSymbol(&Symbol{FileID: &f.ID, Name: "Foo", Kind: "function"})
	require.NoError(t, err)
	assert.Negative(t, id1, "batched IDs should be negative")

	id2, err := batch.InsertSymbol(&Symbol{FileID: &f.ID, Name: "Bar", Kind: "struct"})
	require.NoError(t, err)
	assert.Negative(t, id2)

	// SymbolsByFile should return the buffered symbols even though
	// they haven't been committed to SQLite yet.
	syms, err := batch.SymbolsByFile(f.ID)
	require.NoError(t, err)
	require.Len(t, syms, 2)

	names := []string{syms[0].Name, syms[1].Name}
	assert.Contains(t, names, "Foo")
	assert.Contains(t, names, "Bar")

	// The returned symbols should have fake (negative) IDs.
	for _, sym := range syms {
		assert.Negative(t, sym.ID, "buffered symbols should have negative IDs")
	}
}

func TestBatchedStore_SymbolsByFile_MergesWithDatabase(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f := insertTestFile(t, s, "/main.go", "go")

	// Insert a symbol directly into the database (e.g., from a previous indexing run).
	insertTestSymbol(t, s, &f.ID, "Existing", "function")

	// Create a batch and insert a new symbol.
	batch := NewBatchedStore(s)
	_, err := batch.InsertSymbol(&Symbol{FileID: &f.ID, Name: "New", Kind: "struct"})
	require.NoError(t, err)

	// Should return both the DB symbol and the buffered symbol.
	syms, err := batch.SymbolsByFile(f.ID)
	require.NoError(t, err)
	require.Len(t, syms, 2)

	names := []string{syms[0].Name, syms[1].Name}
	assert.Contains(t, names, "Existing")
	assert.Contains(t, names, "New")
}

func TestBatchedStore_SymbolsByFile_DoesNotReturnOtherFiles(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	f1 := insertTestFile(t, s, "/a.go", "go")
	f2 := insertTestFile(t, s, "/b.go", "go")

	batch := NewBatchedStore(s)
	_, err := batch.InsertSymbol(&Symbol{FileID: &f1.ID, Name: "InFileA", Kind: "function"})
	require.NoError(t, err)
	_, err = batch.InsertSymbol(&Symbol{FileID: &f2.ID, Name: "InFileB", Kind: "function"})
	require.NoError(t, err)

	// Query for file A should only return file A's symbol.
	syms, err := batch.SymbolsByFile(f1.ID)
	require.NoError(t, err)
	require.Len(t, syms, 1)
	assert.Equal(t, "InFileA", syms[0].Name)
}
