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

// extractPythonSource writes Python source to a temp file, inserts a file
// record, and runs the extraction script. Returns the file ID.
func (e *testEnv) extractPythonSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	pyFile := filepath.Join(dir, "test.py")
	require.NoError(e.t, os.WriteFile(pyFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     pyFile,
		Language: "python",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": pyFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "python.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// ---------- Tests ----------

func TestPythonExtract_SimpleFunctionDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def hello():
    pass
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
	assert.Equal(t, "hello", fn.Name)
	assert.Equal(t, "function", fn.Kind)
	assert.Equal(t, "public", fn.Visibility)
	assert.Equal(t, 1, fn.StartLine)
}

func TestPythonExtract_MultipleFunctions(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def foo():
    pass

def _bar():
    pass

def baz():
    pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var funcs []*store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			funcs = append(funcs, s)
		}
	}
	require.Len(t, funcs, 3)

	names := map[string]string{}
	for _, f := range funcs {
		names[f.Name] = f.Visibility
	}
	assert.Equal(t, "public", names["foo"])
	assert.Equal(t, "private", names["_bar"])
	assert.Equal(t, "public", names["baz"])
}

func TestPythonExtract_ClassWithMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`class Server:
    host = "localhost"

    def start(self):
        pass

    def stop(self):
        pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	var methods []*store.Symbol
	for _, s := range syms {
		switch s.Kind {
		case "class":
			classSym = s
		case "method":
			methods = append(methods, s)
		}
	}

	require.NotNil(t, classSym, "expected class symbol")
	assert.Equal(t, "Server", classSym.Name)
	assert.Equal(t, "public", classSym.Visibility)

	require.Len(t, methods, 2)
	for _, m := range methods {
		require.NotNil(t, m.ParentSymbolID, "method should have parent_symbol_id")
		assert.Equal(t, classSym.ID, *m.ParentSymbolID)
	}

	// Check class variable (type_member)
	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	var hostMember *store.TypeMember
	for _, m := range members {
		if m.Name == "host" {
			hostMember = m
		}
	}
	require.NotNil(t, hostMember, "expected class variable 'host'")
	assert.Equal(t, "field", hostMember.Kind)
}

func TestPythonExtract_DecoratorExtraction(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`class MyClass:
    @staticmethod
    def create():
        pass

    @classmethod
    def from_config(cls, config):
        pass

    @property
    def name(self):
        return self._name
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	methodsByName := map[string]*store.Symbol{}
	for _, s := range syms {
		if s.Kind == "static_method" || s.Kind == "class_method" || s.Kind == "property" {
			methodsByName[s.Name] = s
		}
	}

	// Static method
	createSym := methodsByName["create"]
	require.NotNil(t, createSym, "expected static_method 'create'")
	assert.Equal(t, "static_method", createSym.Kind)

	// Class method
	fromCfgSym := methodsByName["from_config"]
	require.NotNil(t, fromCfgSym, "expected class_method 'from_config'")
	assert.Equal(t, "class_method", fromCfgSym.Kind)

	// Property
	nameSym := methodsByName["name"]
	require.NotNil(t, nameSym, "expected property 'name'")
	assert.Equal(t, "property", nameSym.Kind)

	// Verify decorators are stored as annotations
	anns, err := env.store.AnnotationsByTarget(createSym.ID)
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, "staticmethod", anns[0].Name)

	anns2, err := env.store.AnnotationsByTarget(fromCfgSym.ID)
	require.NoError(t, err)
	require.Len(t, anns2, 1)
	assert.Equal(t, "classmethod", anns2[0].Name)

	anns3, err := env.store.AnnotationsByTarget(nameSym.ID)
	require.NoError(t, err)
	require.Len(t, anns3, 1)
	assert.Equal(t, "property", anns3[0].Name)
}

func TestPythonExtract_ImportStatements(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`import os
from os import path
from os import path as p
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 3)

	// import os
	var osImp *store.Import
	for _, imp := range imports {
		if imp.Source == "os" && imp.Kind == "module" {
			osImp = imp
			break
		}
	}
	require.NotNil(t, osImp, "expected 'import os'")
	require.NotNil(t, osImp.ImportedName)
	assert.Equal(t, "os", *osImp.ImportedName)

	// from os import path
	var pathImp *store.Import
	for _, imp := range imports {
		if imp.Kind == "name" && imp.ImportedName != nil && *imp.ImportedName == "path" && imp.LocalAlias == nil {
			pathImp = imp
			break
		}
	}
	require.NotNil(t, pathImp, "expected 'from os import path'")
	assert.Equal(t, "os", pathImp.Source)

	// from os import path as p
	var aliasImp *store.Import
	for _, imp := range imports {
		if imp.LocalAlias != nil && *imp.LocalAlias == "p" {
			aliasImp = imp
			break
		}
	}
	require.NotNil(t, aliasImp, "expected 'from os import path as p'")
	assert.Equal(t, "os", aliasImp.Source)
	require.NotNil(t, aliasImp.ImportedName)
	assert.Equal(t, "path", *aliasImp.ImportedName)
}

func TestPythonExtract_ModuleLevelVariable(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`MAX_RETRIES = 3
_default_timeout = 30
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vars := map[string]*store.Symbol{}
	for _, s := range syms {
		if s.Kind == "variable" {
			vars[s.Name] = s
		}
	}
	require.Len(t, vars, 2)
	assert.Equal(t, "public", vars["MAX_RETRIES"].Visibility)
	assert.Equal(t, "private", vars["_default_timeout"].Visibility)
}

func TestPythonExtract_NestedFunctions(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def outer():
    def inner():
        pass
    inner()
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var outer, inner *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			if s.Name == "outer" {
				outer = s
			} else if s.Name == "inner" {
				inner = s
			}
		}
	}
	require.NotNil(t, outer, "expected outer function")
	require.NotNil(t, inner, "expected inner function")
	// inner is a nested function — currently not linked as parent since
	// we don't track nested function parent_symbol_id (only class methods).
	// This is fine for now; it's still extracted.
}

func TestPythonExtract_ScopeTree(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def process(x):
    if x > 0:
        for i in range(x):
            pass
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}
	assert.Equal(t, 1, kinds["module"], "expected 1 module scope")
	assert.Equal(t, 1, kinds["function"], "expected 1 function scope")
	assert.GreaterOrEqual(t, kinds["block"], 2, "expected at least 2 block scopes (if + for)")

	// Module scope should have no parent
	var moduleScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "module" {
			moduleScope = s
			break
		}
	}
	require.NotNil(t, moduleScope)
	assert.Nil(t, moduleScope.ParentScopeID, "module scope should have no parent")

	// Function scope should have module as parent
	var funcScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "function" {
			funcScope = s
			break
		}
	}
	require.NotNil(t, funcScope)
	require.NotNil(t, funcScope.ParentScopeID)
	assert.Equal(t, moduleScope.ID, *funcScope.ParentScopeID)
}

func TestPythonExtract_References_Call(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def helper():
    pass

def main():
    helper()
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
	assert.Equal(t, "call", callRef.Context)
}

func TestPythonExtract_References_TypeAnnotation(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def greet(name: str) -> str:
    return name
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	typeRefs := map[string]bool{}
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRefs[r.Name] = true
		}
	}
	assert.True(t, typeRefs["str"], "expected type_annotation reference to str")
}

func TestPythonExtract_References_AttributeAccess(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`import os

def main():
    os.path.join("a", "b")
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	// os.path.join() should produce references
	refNames := map[string]string{}
	for _, r := range refs {
		refNames[r.Name] = r.Context
	}
	// "join" should be a call reference (attribute call)
	assert.Equal(t, "call", refNames["join"])
}

func TestPythonExtract_FunctionParamsWithTypeHints(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def add(a: int, b: int) -> int:
    return a + b
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fnSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fnSym = s
			break
		}
	}
	require.NotNil(t, fnSym)

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)

	var regularParams, returnParams []*store.FunctionParam
	for _, p := range params {
		if p.IsReturn {
			returnParams = append(returnParams, p)
		} else {
			regularParams = append(regularParams, p)
		}
	}

	require.Len(t, regularParams, 2)
	assert.Equal(t, "a", regularParams[0].Name)
	assert.Equal(t, "int", regularParams[0].TypeExpr)
	assert.Equal(t, 0, regularParams[0].Ordinal)
	assert.Equal(t, "b", regularParams[1].Name)
	assert.Equal(t, "int", regularParams[1].TypeExpr)
	assert.Equal(t, 1, regularParams[1].Ordinal)

	require.Len(t, returnParams, 1)
	assert.Equal(t, "int", returnParams[0].TypeExpr)
	assert.True(t, returnParams[0].IsReturn)
}

func TestPythonExtract_FunctionParamsWithDefaults(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def connect(host: str, port: int = 8080):
    pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fnSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fnSym = s
			break
		}
	}
	require.NotNil(t, fnSym)

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)

	var regularParams []*store.FunctionParam
	for _, p := range params {
		if !p.IsReturn {
			regularParams = append(regularParams, p)
		}
	}

	require.Len(t, regularParams, 2)

	// First param: host: str (no default)
	assert.Equal(t, "host", regularParams[0].Name)
	assert.Equal(t, "str", regularParams[0].TypeExpr)
	assert.False(t, regularParams[0].HasDefault)

	// Second param: port: int = 8080 (with default)
	assert.Equal(t, "port", regularParams[1].Name)
	assert.Equal(t, "int", regularParams[1].TypeExpr)
	assert.True(t, regularParams[1].HasDefault)
	assert.Equal(t, "8080", regularParams[1].DefaultExpr)
}

func TestPythonExtract_ClassInheritance(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`class Animal:
    pass

class Dog(Animal):
    def bark(self):
        pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var dogSym *store.Symbol
	for _, s := range syms {
		if s.Name == "Dog" {
			dogSym = s
			break
		}
	}
	require.NotNil(t, dogSym)

	members, err := env.store.TypeMembers(dogSym.ID)
	require.NoError(t, err)

	var baseMember *store.TypeMember
	for _, m := range members {
		if m.Kind == "base_class" {
			baseMember = m
			break
		}
	}
	require.NotNil(t, baseMember, "expected base_class member")
	assert.Equal(t, "Animal", baseMember.Name)

	// Verify the bark method is linked to Dog
	var barkSym *store.Symbol
	for _, s := range syms {
		if s.Name == "bark" {
			barkSym = s
			break
		}
	}
	require.NotNil(t, barkSym)
	require.NotNil(t, barkSym.ParentSymbolID)
	assert.Equal(t, dogSym.ID, *barkSym.ParentSymbolID)

	// Verify reference to Animal in class inheritance
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	var animalRef *store.Reference
	for _, r := range refs {
		if r.Name == "Animal" && r.Context == "type_annotation" {
			animalRef = r
			break
		}
	}
	require.NotNil(t, animalRef, "expected type_annotation reference to Animal")
}

func TestPythonExtract_PrivateVisibility(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def public_func():
    pass

def _private_func():
    pass

class PublicClass:
    pass

class _PrivateClass:
    pass

PUBLIC_VAR = 1
_private_var = 2
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vis := map[string]string{}
	for _, s := range syms {
		vis[s.Name] = s.Visibility
	}

	assert.Equal(t, "public", vis["public_func"])
	assert.Equal(t, "private", vis["_private_func"])
	assert.Equal(t, "public", vis["PublicClass"])
	assert.Equal(t, "private", vis["_PrivateClass"])
	assert.Equal(t, "public", vis["PUBLIC_VAR"])
	assert.Equal(t, "private", vis["_private_var"])
}

func TestPythonExtract_NestedClasses(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`class Outer:
    class Inner:
        def method(self):
            pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var outerSym, innerSym, methodSym *store.Symbol
	for _, s := range syms {
		switch {
		case s.Name == "Outer" && s.Kind == "class":
			outerSym = s
		case s.Name == "Inner" && s.Kind == "class":
			innerSym = s
		case s.Name == "method" && s.Kind == "method":
			methodSym = s
		}
	}

	require.NotNil(t, outerSym)
	require.NotNil(t, innerSym)
	require.NotNil(t, methodSym)

	// Inner should be child of Outer
	require.NotNil(t, innerSym.ParentSymbolID)
	assert.Equal(t, outerSym.ID, *innerSym.ParentSymbolID)

	// method should be child of Inner
	require.NotNil(t, methodSym.ParentSymbolID)
	assert.Equal(t, innerSym.ID, *methodSym.ParentSymbolID)
}

func TestPythonExtract_SelfParameter(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`class Foo:
    def bar(self, x: int):
        pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var methodSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methodSym = s
			break
		}
	}
	require.NotNil(t, methodSym)

	params, err := env.store.FunctionParams(methodSym.ID)
	require.NoError(t, err)

	var selfParam *store.FunctionParam
	var xParam *store.FunctionParam
	for _, p := range params {
		if p.Name == "self" {
			selfParam = p
		} else if p.Name == "x" {
			xParam = p
		}
	}
	require.NotNil(t, selfParam, "expected self parameter")
	assert.True(t, selfParam.IsReceiver)
	require.NotNil(t, xParam, "expected x parameter")
	assert.Equal(t, "int", xParam.TypeExpr)
	assert.False(t, xParam.IsReceiver)
}

func TestPythonExtract_ComprehensiveFile(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`import os
from typing import List

MAX_SIZE = 100

class Config:
    host = "localhost"
    port = 8080

    def __init__(self, host: str, port: int = 8080):
        self.host = host
        self.port = port

    @property
    def address(self) -> str:
        return self.host

    @staticmethod
    def default():
        return Config()

def create_config(host: str) -> Config:
    return Config(host)
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["variable"], "MAX_SIZE")
	assert.Contains(t, kinds["class"], "Config")
	assert.Contains(t, kinds["method"], "__init__")
	assert.Contains(t, kinds["property"], "address")
	assert.Contains(t, kinds["static_method"], "default")
	assert.Contains(t, kinds["function"], "create_config")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	// Verify scope tree exists
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 2) // module + functions/class

	// Verify references exist
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)
}

func TestPythonExtract_DecoratorOnTopLevelFunction(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def my_decorator(func):
    return func

@my_decorator
def hello():
    pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var helloSym *store.Symbol
	for _, s := range syms {
		if s.Name == "hello" {
			helloSym = s
			break
		}
	}
	require.NotNil(t, helloSym)
	assert.Equal(t, "function", helloSym.Kind)

	// Verify decorator annotation
	anns, err := env.store.AnnotationsByTarget(helloSym.ID)
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, "my_decorator", anns[0].Name)
}

func TestPythonExtract_StarAndKwargs(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPythonSource(`def variadic(*args, **kwargs):
    pass
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var fnSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" {
			fnSym = s
			break
		}
	}
	require.NotNil(t, fnSym)

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)
	require.Len(t, params, 2)

	paramNames := []string{}
	for _, p := range params {
		paramNames = append(paramNames, p.Name)
	}
	assert.Contains(t, paramNames, "*args")
	assert.Contains(t, paramNames, "**kwargs")
}

func TestPythonExtract_EndToEnd_ViaEngine(t *testing.T) {
	dir := t.TempDir()
	pyFile := filepath.Join(dir, "main.py")
	require.NoError(t, os.WriteFile(pyFile, []byte(`def hello():
    return "hello"
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
		Path:     pyFile,
		Language: "python",
	})
	require.NoError(t, err)

	extras := map[string]any{
		"file_path": pyFile,
		"file_id":   fileID,
	}
	err = rt.RunScript(context.Background(), filepath.Join("extract", "python.risor"), extras)
	require.NoError(t, err)

	syms, err := s.SymbolsByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(syms), 1) // at least the function
}
