package go_extract_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jward/canopy/internal/runtime"
	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractTSSource writes TypeScript source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractTSSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	tsFile := filepath.Join(dir, "test.ts")
	require.NoError(e.t, os.WriteFile(tsFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     tsFile,
		Language: "typescript",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": tsFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "typescript.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// ---------- TypeScript Tests ----------

func TestTS_FunctionDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`function greet(name: string): string {
  return "Hello, " + name;
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fn *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fn = s
		}
	}
	require.NotNil(t, fn, "expected function symbol")
	assert.Equal(t, "greet", fn.Name)
	assert.Equal(t, "function", fn.Kind)
	assert.Equal(t, 1, fn.StartLine)

	// Check parameters
	params, err := env.store.FunctionParams(fn.ID)
	require.NoError(t, err)

	var regularParams, returnParams []*store.FunctionParam
	for _, p := range params {
		if p.IsReturn {
			returnParams = append(returnParams, p)
		} else {
			regularParams = append(regularParams, p)
		}
	}
	require.Len(t, regularParams, 1)
	assert.Equal(t, "name", regularParams[0].Name)
	assert.Equal(t, "string", regularParams[0].TypeExpr)

	require.Len(t, returnParams, 1)
	assert.Equal(t, "string", returnParams[0].TypeExpr)
}

func TestTS_ArrowFunction(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`const multiply = (a: number, b: number): number => a * b;
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fn *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fn = s
		}
	}
	require.NotNil(t, fn, "expected function symbol for arrow function")
	assert.Equal(t, "multiply", fn.Name)

	params, err := env.store.FunctionParams(fn.ID)
	require.NoError(t, err)

	var regularParams []*store.FunctionParam
	for _, p := range params {
		if !p.IsReturn {
			regularParams = append(regularParams, p)
		}
	}
	require.Len(t, regularParams, 2)
	assert.Equal(t, "a", regularParams[0].Name)
	assert.Equal(t, "number", regularParams[0].TypeExpr)
	assert.Equal(t, "b", regularParams[1].Name)
}

func TestTS_ClassWithMembers(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`class Greeter {
  name: string;
  constructor(name: string) {
    this.name = name;
  }
  greet(): string {
    return "Hello, " + this.name;
  }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
	}
	require.NotNil(t, classSym, "expected class symbol")
	assert.Equal(t, "Greeter", classSym.Name)

	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	membersByName := map[string]*store.TypeMember{}
	for _, m := range members {
		membersByName[m.Name] = m
	}

	// Should have: name (property), constructor (method), greet (method)
	require.Contains(t, membersByName, "name")
	assert.Equal(t, "property", membersByName["name"].Kind)
	assert.Equal(t, "string", membersByName["name"].TypeExpr)

	require.Contains(t, membersByName, "constructor")
	assert.Equal(t, "method", membersByName["constructor"].Kind)

	require.Contains(t, membersByName, "greet")
	assert.Equal(t, "method", membersByName["greet"].Kind)

	// Methods should also be separate symbols linked to the class
	var methodSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "method" && s.Name == "greet" {
			methodSym = s
		}
	}
	require.NotNil(t, methodSym)
	require.NotNil(t, methodSym.ParentSymbolID)
	assert.Equal(t, classSym.ID, *methodSym.ParentSymbolID)
}

func TestTS_InterfaceWithMembers(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`interface Shape {
  area(): number;
  perimeter(): number;
  readonly name: string;
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var ifaceSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "interface" {
			ifaceSym = s
		}
	}
	require.NotNil(t, ifaceSym, "expected interface symbol")
	assert.Equal(t, "Shape", ifaceSym.Name)

	members, err := env.store.TypeMembers(ifaceSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3)

	membersByName := map[string]*store.TypeMember{}
	for _, m := range members {
		membersByName[m.Name] = m
	}

	assert.Equal(t, "method", membersByName["area"].Kind)
	assert.Equal(t, "method", membersByName["perimeter"].Kind)
	assert.Equal(t, "property", membersByName["name"].Kind)
	assert.Equal(t, "string", membersByName["name"].TypeExpr)
}

func TestTS_EnumWithVariants(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`enum Direction {
  Up = "UP",
  Down = "DOWN",
  Left = "LEFT",
  Right = "RIGHT",
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var enumSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "enum" {
			enumSym = s
		}
	}
	require.NotNil(t, enumSym, "expected enum symbol")
	assert.Equal(t, "Direction", enumSym.Name)

	members, err := env.store.TypeMembers(enumSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 4)

	names := []string{}
	for _, m := range members {
		assert.Equal(t, "variant", m.Kind)
		names = append(names, m.Name)
	}
	assert.Contains(t, names, "Up")
	assert.Contains(t, names, "Down")
	assert.Contains(t, names, "Left")
	assert.Contains(t, names, "Right")
}

func TestTS_TypeAlias(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`type StringOrNumber = string | number;
type Handler = (req: string) => void;
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var aliases []*store.Symbol
	for _, s := range syms {
		if s.Kind == "type_alias" {
			aliases = append(aliases, s)
		}
	}
	require.Len(t, aliases, 2)

	names := map[string]bool{}
	for _, a := range aliases {
		names[a.Name] = true
	}
	assert.True(t, names["StringOrNumber"])
	assert.True(t, names["Handler"])
}

func TestTS_NamedImports(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`import { foo, bar } from './module';
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	names := map[string]bool{}
	for _, imp := range imports {
		assert.Equal(t, "./module", imp.Source)
		assert.Equal(t, "named", imp.Kind)
		require.NotNil(t, imp.ImportedName)
		names[*imp.ImportedName] = true
	}
	assert.True(t, names["foo"])
	assert.True(t, names["bar"])
}

func TestTS_DefaultImport(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`import defaultExport from './default';
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 1)

	imp := imports[0]
	assert.Equal(t, "./default", imp.Source)
	assert.Equal(t, "default", imp.Kind)
	require.NotNil(t, imp.ImportedName)
	assert.Equal(t, "defaultExport", *imp.ImportedName)
}

func TestTS_NamespaceImport(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`import * as utils from './utils';
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 1)

	imp := imports[0]
	assert.Equal(t, "./utils", imp.Source)
	assert.Equal(t, "namespace", imp.Kind)
	require.NotNil(t, imp.ImportedName)
	assert.Equal(t, "*", *imp.ImportedName)
	require.NotNil(t, imp.LocalAlias)
	assert.Equal(t, "utils", *imp.LocalAlias)
}

func TestTS_ExportedFunction(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`export function hello(): void {}
function internal(): void {}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "function" {
			vis[s.Name] = s.Visibility
		}
	}
	assert.Equal(t, "public", vis["hello"])
	assert.Equal(t, "private", vis["internal"])
}

func TestTS_ExportDefaultClass(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`export default class Foo {}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
	}
	require.NotNil(t, classSym)
	assert.Equal(t, "Foo", classSym.Name)
	assert.Equal(t, "public", classSym.Visibility)
}

func TestTS_ReExport(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`export { qux } from './other';
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 1)

	imp := imports[0]
	assert.Equal(t, "./other", imp.Source)
	assert.Equal(t, "reexport", imp.Kind)
	require.NotNil(t, imp.ImportedName)
	assert.Equal(t, "qux", *imp.ImportedName)
}

func TestTS_Decorators(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`@Component({selector: 'app'})
class AppComponent {
  @Input() title: string = '';
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
	}
	require.NotNil(t, classSym)
	assert.Equal(t, "AppComponent", classSym.Name)

	// Check class-level decorator
	anns, err := env.store.AnnotationsByTarget(classSym.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(anns), 1)

	var componentAnn *store.Annotation
	for _, a := range anns {
		if a.Name == "Component" {
			componentAnn = a
		}
	}
	require.NotNil(t, componentAnn, "expected @Component annotation")
	assert.Contains(t, componentAnn.Arguments, "selector")
}

func TestTS_GenericsWithConstraints(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`function identity<T extends Comparable>(arg: T): T {
  return arg;
}

interface Container<T> {
  value: T;
  get(): T;
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fnSym, ifaceSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" && s.Name == "identity" {
			fnSym = s
		}
		if s.Kind == "interface" && s.Name == "Container" {
			ifaceSym = s
		}
	}

	// Function generic
	require.NotNil(t, fnSym)
	fnTPs, err := env.store.TypeParams(fnSym.ID)
	require.NoError(t, err)
	require.Len(t, fnTPs, 1)
	assert.Equal(t, "T", fnTPs[0].Name)
	assert.Equal(t, "Comparable", fnTPs[0].Constraints)
	assert.Equal(t, 0, fnTPs[0].Ordinal)

	// Interface generic (no constraint)
	require.NotNil(t, ifaceSym)
	ifaceTPs, err := env.store.TypeParams(ifaceSym.ID)
	require.NoError(t, err)
	require.Len(t, ifaceTPs, 1)
	assert.Equal(t, "T", ifaceTPs[0].Name)
	assert.Equal(t, "", ifaceTPs[0].Constraints) // no constraint
}

func TestTS_ScopeTree(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`function process(x: number): void {
  if (x > 0) {
    for (let i = 0; i < x; i++) {
    }
  }
}
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}
	assert.Equal(t, 1, kinds["file"], "expected 1 file scope")
	assert.Equal(t, 1, kinds["function"], "expected 1 function scope")
	assert.GreaterOrEqual(t, kinds["block"], 2, "expected at least 2 block scopes (if + for)")

	// Verify scope nesting
	var fileScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "file" {
			fileScope = s
			break
		}
	}
	require.NotNil(t, fileScope)
	assert.Nil(t, fileScope.ParentScopeID, "file scope should have no parent")

	var funcScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "function" {
			funcScope = s
			break
		}
	}
	require.NotNil(t, funcScope)
	require.NotNil(t, funcScope.ParentScopeID)
	assert.Equal(t, fileScope.ID, *funcScope.ParentScopeID)
}

func TestTS_References_Call(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`function helper(): void {}

function main(): void {
  helper();
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var callRef *store.Reference
	for _, r := range refs {
		if r.Name == "helper" && r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to helper")
}

func TestTS_References_TypeAnnotation(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`interface Foo {}

function create(): Foo {
  return {};
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Name == "Foo" && r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Foo")
}

func TestTS_References_FieldAccess(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`const x = console.log("hello");
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var fieldRef *store.Reference
	for _, r := range refs {
		if r.Name == "log" && r.Context == "field_access" {
			fieldRef = r
			break
		}
	}
	require.NotNil(t, fieldRef, "expected field_access reference to log")
}

func TestTS_References_NewExpression(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`class Foo {}
const x = new Foo();
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var newRef *store.Reference
	for _, r := range refs {
		if r.Name == "Foo" && r.Context == "call" {
			newRef = r
			break
		}
	}
	require.NotNil(t, newRef, "expected call reference from new Foo()")
}

func TestTS_VariableDeclarations(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`var x = 1;
let y = "hello";
const z = true;
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	symsByName := map[string]*store.Symbol{}
	for _, s := range syms {
		symsByName[s.Name] = s
	}

	require.Contains(t, symsByName, "x")
	assert.Equal(t, "variable", symsByName["x"].Kind)

	require.Contains(t, symsByName, "y")
	assert.Equal(t, "variable", symsByName["y"].Kind)

	require.Contains(t, symsByName, "z")
	assert.Equal(t, "constant", symsByName["z"].Kind)
}

func TestTS_ComprehensiveFile(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractTSSource(`import { EventEmitter } from 'events';

export interface Handler {
  handle(req: string): string;
}

export class Server implements Handler {
  private name: string;

  constructor(name: string) {
    this.name = name;
  }

  handle(req: string): string {
    return "handled: " + req;
  }
}

export function createServer(name: string): Server {
  return new Server(name);
}

const DEFAULT_PORT = 8080;

export type Config = {
  host: string;
  port: number;
};

enum Status {
  Active = "ACTIVE",
  Inactive = "INACTIVE",
}
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["interface"], "Handler")
	assert.Contains(t, kinds["class"], "Server")
	assert.Contains(t, kinds["function"], "createServer")
	assert.Contains(t, kinds["constant"], "DEFAULT_PORT")
	assert.Contains(t, kinds["type_alias"], "Config")
	assert.Contains(t, kinds["enum"], "Status")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(imports), 1)
	assert.Equal(t, "events", imports[0].Source)

	// Verify scopes
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 3) // file + function + class

	// Verify references
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)

	// Verify visibility: exported = public, non-exported = private
	vis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "method" {
			continue // methods inherit from class
		}
		vis[s.Name] = s.Visibility
	}
	assert.Equal(t, "public", vis["Handler"])
	assert.Equal(t, "public", vis["Server"])
	assert.Equal(t, "public", vis["createServer"])
	assert.Equal(t, "private", vis["DEFAULT_PORT"])
	assert.Equal(t, "public", vis["Config"])
	assert.Equal(t, "private", vis["Status"])
}

func TestTS_EndToEnd_ViaEngine(t *testing.T) {
	dir := t.TempDir()
	tsFile := filepath.Join(dir, "app.ts")
	require.NoError(t, os.WriteFile(tsFile, []byte(`
export function greet(name: string): string {
  return "Hello, " + name;
}
`), 0644))

	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())
	defer s.Close()

	rt := runtime.NewRuntime(s, scriptsDir)

	fileID, err := s.InsertFile(&store.File{
		Path:     tsFile,
		Language: "typescript",
	})
	require.NoError(t, err)

	extras := map[string]any{
		"file_path": tsFile,
		"file_id":   fileID,
	}
	err = rt.RunScript(context.Background(), filepath.Join("extract", "typescript.risor"), extras)
	require.NoError(t, err)

	syms, err := s.SymbolsByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(syms), 1) // at least the function
}
