package canopy

import "github.com/jward/canopy/internal/store"

// Public type aliases for internal store types used in the QueryBuilder API.
// These are Go type aliases (=) — identical to the internal types at compile
// time. External consumers use these names; no conversion is needed.

type Store = store.Store
type Symbol = store.Symbol
type File = store.File
type Scope = store.Scope
type CallEdge = store.CallEdge
type Import = store.Import
type FunctionParam = store.FunctionParam
type TypeMember = store.TypeMember
type TypeParam = store.TypeParam
type Annotation = store.Annotation
type ExtensionBinding = store.ExtensionBinding
type Reexport = store.Reexport
