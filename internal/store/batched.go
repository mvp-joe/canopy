package store

import "sync"

// BatchedStore buffers extraction inserts in memory using fake (negative)
// IDs. It implements DataStore so extraction scripts can write to it
// without knowing whether they're hitting SQLite or an in-memory buffer.
//
// Thread safety: the mutex protects fake ID allocation and slice appends.
// Read queries (SymbolsByName, SymbolsByFile) are passed through to the
// underlying Store, which is safe for concurrent reads.
type BatchedStore struct {
	store *Store // for read passthrough
	mu    sync.Mutex

	// Buffered extraction data.
	Symbols         []Symbol
	Scopes          []Scope
	References      []Reference
	Imports         []Import
	TypeMembers     []TypeMember
	FunctionParams  []FunctionParam
	TypeParams      []TypeParam
	Annotations     []Annotation
	SymbolFragments []SymbolFragment

	nextFakeID int64 // starts at -1, decrements
}

// Compile-time check: *BatchedStore satisfies DataStore.
var _ DataStore = (*BatchedStore)(nil)

// NewBatchedStore creates a BatchedStore backed by the given Store for read queries.
func NewBatchedStore(s *Store) *BatchedStore {
	return &BatchedStore{
		store:      s,
		nextFakeID: -1,
	}
}

func (b *BatchedStore) allocFakeID() int64 {
	id := b.nextFakeID
	b.nextFakeID--
	return id
}

func (b *BatchedStore) InsertSymbol(sym *Symbol) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	sym.ID = fakeID
	b.Symbols = append(b.Symbols, *sym)
	return fakeID, nil
}

func (b *BatchedStore) InsertScope(scope *Scope) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	scope.ID = fakeID
	b.Scopes = append(b.Scopes, *scope)
	return fakeID, nil
}

func (b *BatchedStore) InsertReference(ref *Reference) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	ref.ID = fakeID
	b.References = append(b.References, *ref)
	return fakeID, nil
}

func (b *BatchedStore) InsertImport(imp *Import) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	imp.ID = fakeID
	b.Imports = append(b.Imports, *imp)
	return fakeID, nil
}

func (b *BatchedStore) InsertTypeMember(tm *TypeMember) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	tm.ID = fakeID
	b.TypeMembers = append(b.TypeMembers, *tm)
	return fakeID, nil
}

func (b *BatchedStore) InsertFunctionParam(fp *FunctionParam) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	fp.ID = fakeID
	b.FunctionParams = append(b.FunctionParams, *fp)
	return fakeID, nil
}

func (b *BatchedStore) InsertTypeParam(tp *TypeParam) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	tp.ID = fakeID
	b.TypeParams = append(b.TypeParams, *tp)
	return fakeID, nil
}

func (b *BatchedStore) InsertAnnotation(ann *Annotation) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	ann.ID = fakeID
	b.Annotations = append(b.Annotations, *ann)
	return fakeID, nil
}

func (b *BatchedStore) InsertSymbolFragment(frag *SymbolFragment) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fakeID := b.allocFakeID()
	frag.ID = fakeID
	b.SymbolFragments = append(b.SymbolFragments, *frag)
	return fakeID, nil
}

// SymbolsByName passes through to the underlying Store for cross-file lookups.
func (b *BatchedStore) SymbolsByName(name string) ([]*Symbol, error) {
	return b.store.SymbolsByName(name)
}

// SymbolsByFile returns symbols for a file, merging any buffered (not yet
// committed) symbols with those already in the database.
func (b *BatchedStore) SymbolsByFile(fileID int64) ([]*Symbol, error) {
	dbSyms, err := b.store.SymbolsByFile(fileID)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.Symbols {
		if b.Symbols[i].FileID != nil && *b.Symbols[i].FileID == fileID {
			dbSyms = append(dbSyms, &b.Symbols[i])
		}
	}
	return dbSyms, nil
}
