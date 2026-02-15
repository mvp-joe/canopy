package store

import "time"

// Extraction domain types

type File struct {
	ID          int64
	Path        string
	Language    string
	Hash        string
	LastIndexed time.Time
}

type Symbol struct {
	ID             int64
	FileID         *int64
	Name           string
	Kind           string
	Visibility     string
	Modifiers      []string
	SignatureHash  string
	StartLine      int
	StartCol       int
	EndLine        int
	EndCol         int
	ParentSymbolID *int64
}

type SymbolFragment struct {
	ID        int64
	SymbolID  int64
	FileID    int64
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	IsPrimary bool
}

type Scope struct {
	ID            int64
	FileID        int64
	SymbolID      *int64
	Kind          string
	StartLine     int
	StartCol      int
	EndLine       int
	EndCol        int
	ParentScopeID *int64
}

type Reference struct {
	ID        int64
	FileID    int64
	ScopeID   *int64
	Name      string
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	Context   string
}

type Import struct {
	ID           int64
	FileID       int64
	Source       string
	ImportedName *string
	LocalAlias   *string
	Kind         string
	Scope        string
}

type TypeMember struct {
	ID         int64
	SymbolID   int64
	Name       string
	Kind       string
	TypeExpr   string
	Visibility string
}

type FunctionParam struct {
	ID          int64
	SymbolID    int64
	Name        string
	Ordinal     int
	TypeExpr    string
	IsReceiver  bool
	IsReturn    bool
	HasDefault  bool
	DefaultExpr string
}

type TypeParam struct {
	ID          int64
	SymbolID    int64
	Name        string
	Ordinal     int
	Variance    string
	ParamKind   string
	Constraints string
}

type Annotation struct {
	ID               int64
	TargetSymbolID   int64
	Name             string
	ResolvedSymbolID *int64
	Arguments        string
	FileID           *int64
	Line             int
	Col              int
}

// Resolution domain types

type ResolvedReference struct {
	ID             int64
	ReferenceID    int64
	TargetSymbolID int64
	Confidence     float64
	ResolutionKind string
}

type Implementation struct {
	ID                int64
	TypeSymbolID      int64
	InterfaceSymbolID int64
	Kind              string
	FileID            *int64
	DeclaringModule   string
}

type CallEdge struct {
	ID             int64
	CallerSymbolID int64
	CalleeSymbolID int64
	FileID         *int64
	Line           int
	Col            int
}

type Reexport struct {
	ID               int64
	FileID           int64
	OriginalSymbolID int64
	ExportedName     string
}

type ExtensionBinding struct {
	ID                   int64
	MemberSymbolID       int64
	ExtendedTypeExpr     string
	ExtendedTypeSymbolID *int64
	Kind                 string
	Constraints          string
	IsDefaultImpl        bool
}

type TypeComposition struct {
	ID                int64
	CompositeSymbolID int64
	ComponentSymbolID int64
	CompositionKind   string
}
