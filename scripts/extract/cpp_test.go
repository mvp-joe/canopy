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

// cppTestEnv wraps the test environment for C++ extraction tests.
type cppTestEnv struct {
	store *store.Store
	rt    *runtime.Runtime
	t     *testing.T
}

func newCppTestEnv(t *testing.T) *cppTestEnv {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	require.NoError(t, s.Migrate())

	modRoot := findModuleRoot(t)
	scriptsDir := filepath.Join(modRoot, "scripts")
	rt := runtime.NewRuntime(s, scriptsDir)

	t.Cleanup(func() { s.Close() })

	return &cppTestEnv{store: s, rt: rt, t: t}
}

// extractCppSource writes C++ source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *cppTestEnv) extractCppSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	cppFile := filepath.Join(dir, "test.cpp")
	require.NoError(e.t, os.WriteFile(cppFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     cppFile,
		Language: "cpp",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": cppFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "cpp.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// ---------- Tests ----------

func TestCppExtract_ClassWithMembers(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`class Animal {
public:
    int getAge() const;
    void setAge(int age);
private:
    int age_;
};
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
			break
		}
	}
	require.NotNil(t, classSym)
	assert.Equal(t, "Animal", classSym.Name)

	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3) // getAge, setAge, age_

	memberInfo := map[string]*store.TypeMember{}
	for _, m := range members {
		memberInfo[m.Name] = m
	}

	// Public methods
	require.Contains(t, memberInfo, "getAge")
	assert.Equal(t, "method", memberInfo["getAge"].Kind)
	assert.Equal(t, "public", memberInfo["getAge"].Visibility)

	require.Contains(t, memberInfo, "setAge")
	assert.Equal(t, "method", memberInfo["setAge"].Kind)
	assert.Equal(t, "public", memberInfo["setAge"].Visibility)

	// Private field
	require.Contains(t, memberInfo, "age_")
	assert.Equal(t, "field", memberInfo["age_"].Kind)
	assert.Equal(t, "private", memberInfo["age_"].Visibility)
}

func TestCppExtract_NamespaceDeclaration(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`namespace mylib {
    void helper();
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var nsSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "namespace" {
			nsSym = s
			break
		}
	}
	require.NotNil(t, nsSym)
	assert.Equal(t, "mylib", nsSym.Name)

	// Also check that the function inside is extracted
	var fnSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "function" && s.Name == "helper" {
			fnSym = s
			break
		}
	}
	require.NotNil(t, fnSym, "expected function declared in namespace")
}

func TestCppExtract_TemplateClass(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`template<typename T>
class Container {
    T value;
public:
    T get() const;
};
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
			break
		}
	}
	require.NotNil(t, classSym)
	assert.Equal(t, "Container", classSym.Name)

	// Check template type params
	tps, err := env.store.TypeParams(classSym.ID)
	require.NoError(t, err)
	require.Len(t, tps, 1)
	assert.Equal(t, "T", tps[0].Name)
	assert.Equal(t, 0, tps[0].Ordinal)
}

func TestCppExtract_TemplateFunction(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`template<typename T, typename U>
T convert(U input) {
    return static_cast<T>(input);
}
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
	assert.Equal(t, "convert", fnSym.Name)

	tps, err := env.store.TypeParams(fnSym.ID)
	require.NoError(t, err)
	require.Len(t, tps, 2)
	assert.Equal(t, "T", tps[0].Name)
	assert.Equal(t, "U", tps[1].Name)
}

func TestCppExtract_Inheritance(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`class Animal {
public:
    virtual void speak() const;
};

class Dog : public Animal {
public:
    void speak() const override;
};
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

	var embedded *store.TypeMember
	var method *store.TypeMember
	for _, m := range members {
		switch m.Kind {
		case "embedded":
			embedded = m
		case "method":
			method = m
		}
	}
	require.NotNil(t, embedded, "expected embedded base class")
	assert.Equal(t, "Animal", embedded.Name)

	require.NotNil(t, method, "expected method member")
	assert.Equal(t, "speak", method.Name)
}

func TestCppExtract_ConstructorDestructor(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`class Widget {
public:
    Widget(int size);
    virtual ~Widget();
private:
    int size_;
};
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
			break
		}
	}
	require.NotNil(t, classSym)

	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	memberNames := map[string]*store.TypeMember{}
	for _, m := range members {
		memberNames[m.Name] = m
	}

	// Constructor
	require.Contains(t, memberNames, "Widget")
	assert.Equal(t, "method", memberNames["Widget"].Kind)
	assert.Equal(t, "public", memberNames["Widget"].Visibility)

	// Destructor
	require.Contains(t, memberNames, "~Widget")
	assert.Equal(t, "method", memberNames["~Widget"].Kind)
	assert.Equal(t, "public", memberNames["~Widget"].Visibility)

	// Private field
	require.Contains(t, memberNames, "size_")
	assert.Equal(t, "field", memberNames["size_"].Kind)
	assert.Equal(t, "private", memberNames["size_"].Visibility)
}

func TestCppExtract_VirtualMethods(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`class Shape {
public:
    virtual double area() const = 0;
    virtual void draw() const;
    int id() const;
};
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
			break
		}
	}
	require.NotNil(t, classSym)

	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3)

	memberInfo := map[string]*store.TypeMember{}
	for _, m := range members {
		memberInfo[m.Name] = m
	}

	// Pure virtual method
	require.Contains(t, memberInfo, "area")
	assert.Contains(t, memberInfo["area"].TypeExpr, "virtual")
	assert.Contains(t, memberInfo["area"].TypeExpr, "= 0")

	// Virtual method
	require.Contains(t, memberInfo, "draw")
	assert.Contains(t, memberInfo["draw"].TypeExpr, "virtual")

	// Non-virtual method
	require.Contains(t, memberInfo, "id")
	assert.NotContains(t, memberInfo["id"].TypeExpr, "virtual")
}

func TestCppExtract_UsingDeclarations(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`using std::string;
using namespace std;
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	impByKind := map[string]*store.Import{}
	for _, imp := range imports {
		impByKind[imp.Kind] = imp
	}

	// using std::string -> symbol import
	require.Contains(t, impByKind, "symbol")
	assert.Equal(t, "std", impByKind["symbol"].Source)
	require.NotNil(t, impByKind["symbol"].ImportedName)
	assert.Equal(t, "string", *impByKind["symbol"].ImportedName)

	// using namespace std -> namespace import
	require.Contains(t, impByKind, "namespace")
	assert.Equal(t, "std", impByKind["namespace"].Source)
}

func TestCppExtract_ScopeTree(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`namespace myns {
    class Foo {
    public:
        void bar();
    };
}

void process(int x) {
    if (x > 0) {
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
	assert.GreaterOrEqual(t, kinds["namespace"], 1, "expected namespace scope")
	assert.GreaterOrEqual(t, kinds["class"], 1, "expected class scope")
	assert.GreaterOrEqual(t, kinds["function"], 1, "expected function scope")
	assert.GreaterOrEqual(t, kinds["block"], 1, "expected at least 1 block scope")
}

func TestCppExtract_IncludeDirectives(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`#include <iostream>
#include <vector>
#include "config.h"
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 3)

	sources := []string{}
	for _, imp := range imports {
		sources = append(sources, imp.Source)
		assert.Equal(t, "header", imp.Kind)
	}
	assert.Contains(t, sources, "iostream")
	assert.Contains(t, sources, "vector")
	assert.Contains(t, sources, "config.h")
}

func TestCppExtract_Struct(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`struct Point {
    double x;
    double y;
    double distance() const;
};
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var structSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "struct" {
			structSym = s
			break
		}
	}
	require.NotNil(t, structSym)
	assert.Equal(t, "Point", structSym.Name)

	members, err := env.store.TypeMembers(structSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3) // x, y, distance

	memberInfo := map[string]*store.TypeMember{}
	for _, m := range members {
		memberInfo[m.Name] = m
	}

	assert.Equal(t, "field", memberInfo["x"].Kind)
	assert.Equal(t, "public", memberInfo["x"].Visibility) // struct default
	assert.Equal(t, "field", memberInfo["y"].Kind)
	assert.Equal(t, "method", memberInfo["distance"].Kind)
	assert.Equal(t, "public", memberInfo["distance"].Visibility) // struct default
}

func TestCppExtract_EnumClass(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`enum class Color { Red, Green, Blue };
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var enumSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "enum" {
			enumSym = s
			break
		}
	}
	require.NotNil(t, enumSym)
	assert.Equal(t, "Color", enumSym.Name)

	members, err := env.store.TypeMembers(enumSym.ID)
	require.NoError(t, err)
	require.Len(t, members, 3)

	memberNames := []string{}
	for _, m := range members {
		memberNames = append(memberNames, m.Name)
		assert.Equal(t, "variant", m.Kind)
	}
	assert.Contains(t, memberNames, "Red")
	assert.Contains(t, memberNames, "Green")
	assert.Contains(t, memberNames, "Blue")
}

func TestCppExtract_ComprehensiveFile(t *testing.T) {
	env := newCppTestEnv(t)
	fileID := env.extractCppSource(`#include <iostream>
#include <string>

namespace app {

class Service {
public:
    Service(const std::string& name);
    virtual ~Service();
    virtual void start() = 0;
    std::string getName() const;
private:
    std::string name_;
};

template<typename T>
class Cache {
    T data;
public:
    T get() const;
};

enum class Status { Running, Stopped };

void helper() {
    std::cout << "hello" << std::endl;
}

}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["namespace"], "app")
	assert.Contains(t, kinds["class"], "Service")
	assert.Contains(t, kinds["class"], "Cache")
	assert.Contains(t, kinds["enum"], "Status")
	assert.Contains(t, kinds["function"], "helper")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	// Verify scope tree
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 3) // file + namespace + classes + function

	// Verify references exist
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)
}
