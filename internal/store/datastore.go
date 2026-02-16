package store

// DataStore is the interface for extraction-phase data access. Both Store
// (direct SQLite) and BatchedStore (in-memory buffering for parallel
// extraction) implement this interface.
type DataStore interface {
	// Extraction inserts — each returns the assigned ID.
	InsertSymbol(sym *Symbol) (int64, error)
	InsertScope(scope *Scope) (int64, error)
	InsertReference(ref *Reference) (int64, error)
	InsertImport(imp *Import) (int64, error)
	InsertTypeMember(tm *TypeMember) (int64, error)
	InsertFunctionParam(fp *FunctionParam) (int64, error)
	InsertTypeParam(tp *TypeParam) (int64, error)
	InsertAnnotation(ann *Annotation) (int64, error)
	InsertSymbolFragment(frag *SymbolFragment) (int64, error)

	// Queries needed by extraction scripts for cross-file lookups.
	SymbolsByName(name string) ([]*Symbol, error)
	SymbolsByFile(fileID int64) ([]*Symbol, error)
}

// Compile-time check: *Store satisfies DataStore.
var _ DataStore = (*Store)(nil)
