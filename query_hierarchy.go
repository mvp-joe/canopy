package canopy

import (
	"fmt"

	"github.com/jward/canopy/internal/store"
)

// TypeRelation represents a relationship between two types in a hierarchy.
type TypeRelation struct {
	Symbol SymbolResult
	Kind   string // "inheritance", "interface_impl", "composition", "embedding", "implicit"
}

// TypeHierarchy is a complete hierarchy view for a single type, combining
// data from implementations, type_compositions, and extension_bindings tables.
type TypeHierarchy struct {
	Symbol        SymbolResult              // the queried type
	Implements    []*TypeRelation           // interfaces/traits this type implements
	ImplementedBy []*TypeRelation           // concrete types implementing this interface/trait
	Composes      []*TypeRelation           // parent types (inherited, embedded, composed)
	ComposedBy    []*TypeRelation           // child types that inherit/embed/compose this type
	Extensions    []*store.ExtensionBinding // extension methods, trait impls, default impls
}

// TypeHierarchy returns the full type hierarchy for a symbol: what it
// implements, what implements it, what it composes, what composes it,
// and its extension methods.
// Returns nil with no error if the symbol ID does not exist.
func (q *QueryBuilder) TypeHierarchy(symbolID int64) (*TypeHierarchy, error) {
	sr, err := q.symbolResultByID(symbolID)
	if err != nil {
		return nil, fmt.Errorf("type hierarchy: %w", err)
	}
	if sr == nil {
		return nil, nil
	}

	// ImplementedBy: concrete types that implement this interface
	implsByIface, err := q.store.ImplementationsByInterface(symbolID)
	if err != nil {
		return nil, fmt.Errorf("type hierarchy: implementations by interface: %w", err)
	}

	// Implements: interfaces this type satisfies
	implsByType, err := q.store.ImplementationsByType(symbolID)
	if err != nil {
		return nil, fmt.Errorf("type hierarchy: implementations by type: %w", err)
	}

	// Composes: parent types (embedded, inherited, composed)
	comps, err := q.store.TypeCompositions(symbolID)
	if err != nil {
		return nil, fmt.Errorf("type hierarchy: type compositions: %w", err)
	}

	// ComposedBy: child types that embed/inherit/compose this type
	composedByComps, err := q.store.TypeComposedBy(symbolID)
	if err != nil {
		return nil, fmt.Errorf("type hierarchy: type composed by: %w", err)
	}

	// Collect all needed symbol IDs and load them in one query.
	neededIDs := make([]int64, 0, len(implsByIface)+len(implsByType)+len(comps)+len(composedByComps))
	for _, impl := range implsByIface {
		neededIDs = append(neededIDs, impl.TypeSymbolID)
	}
	for _, impl := range implsByType {
		neededIDs = append(neededIDs, impl.InterfaceSymbolID)
	}
	for _, tc := range comps {
		neededIDs = append(neededIDs, tc.ComponentSymbolID)
	}
	for _, tc := range composedByComps {
		neededIDs = append(neededIDs, tc.CompositeSymbolID)
	}
	symbols, err := q.symbolResultsByIDs(neededIDs)
	if err != nil {
		return nil, fmt.Errorf("type hierarchy: batch symbol lookup: %w", err)
	}

	implementedBy := make([]*TypeRelation, 0, len(implsByIface))
	for _, impl := range implsByIface {
		if rel, ok := symbols[impl.TypeSymbolID]; ok {
			implementedBy = append(implementedBy, &TypeRelation{Symbol: *rel, Kind: impl.Kind})
		}
	}

	implements := make([]*TypeRelation, 0, len(implsByType))
	for _, impl := range implsByType {
		if rel, ok := symbols[impl.InterfaceSymbolID]; ok {
			implements = append(implements, &TypeRelation{Symbol: *rel, Kind: impl.Kind})
		}
	}

	composes := make([]*TypeRelation, 0, len(comps))
	for _, tc := range comps {
		if rel, ok := symbols[tc.ComponentSymbolID]; ok {
			composes = append(composes, &TypeRelation{Symbol: *rel, Kind: tc.CompositionKind})
		}
	}

	composedBy := make([]*TypeRelation, 0, len(composedByComps))
	for _, tc := range composedByComps {
		if rel, ok := symbols[tc.CompositeSymbolID]; ok {
			composedBy = append(composedBy, &TypeRelation{Symbol: *rel, Kind: tc.CompositionKind})
		}
	}

	// Extensions
	extensions, err := q.store.ExtensionBindingsByType(symbolID)
	if err != nil {
		return nil, fmt.Errorf("type hierarchy: extension bindings: %w", err)
	}
	if extensions == nil {
		extensions = []*store.ExtensionBinding{}
	}

	return &TypeHierarchy{
		Symbol:        *sr,
		Implements:    implements,
		ImplementedBy: implementedBy,
		Composes:      composes,
		ComposedBy:    composedBy,
		Extensions:    extensions,
	}, nil
}

// ImplementsInterfaces returns the interfaces/traits that a concrete type
// implements. Returns locations of the interface declarations.
func (q *QueryBuilder) ImplementsInterfaces(typeSymbolID int64) ([]Location, error) {
	impls, err := q.store.ImplementationsByType(typeSymbolID)
	if err != nil {
		return nil, fmt.Errorf("implements interfaces: %w", err)
	}

	var locations []Location
	for _, impl := range impls {
		loc, err := q.symbolLocation(impl.InterfaceSymbolID)
		if err != nil {
			return nil, fmt.Errorf("implements interfaces: symbol location: %w", err)
		}
		if loc != nil {
			locations = append(locations, *loc)
		}
	}
	if locations == nil {
		locations = []Location{}
	}
	return locations, nil
}

// ExtensionMethods returns extension bindings for a type.
func (q *QueryBuilder) ExtensionMethods(typeSymbolID int64) ([]*store.ExtensionBinding, error) {
	bindings, err := q.store.ExtensionBindingsByType(typeSymbolID)
	if err != nil {
		return nil, fmt.Errorf("extension methods: %w", err)
	}
	if bindings == nil {
		bindings = []*store.ExtensionBinding{}
	}
	return bindings, nil
}

// Reexports returns re-exported symbols from a file.
func (q *QueryBuilder) Reexports(fileID int64) ([]*store.Reexport, error) {
	reexports, err := q.store.ReexportsByFile(fileID)
	if err != nil {
		return nil, fmt.Errorf("reexports: %w", err)
	}
	if reexports == nil {
		reexports = []*store.Reexport{}
	}
	return reexports, nil
}
