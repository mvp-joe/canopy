package go_resolve_test

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

// cppTestEnv wraps the shared testEnv with C++-specific helpers.
type cppTestEnv struct {
	*testEnv
}

func newCppTestEnv(t *testing.T) *cppTestEnv {
	return &cppTestEnv{testEnv: newTestEnv(t)}
}

// extractCppSource writes C++ source to a temp .cpp file, inserts a file record,
// and runs the C++ extraction script. Returns the file ID.
func (e *cppTestEnv) extractCppSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	cppFile := filepath.Join(dir, filename)
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

// resolveCpp runs the C++ resolution script.
func (e *cppTestEnv) resolveCpp() {
	e.t.Helper()
	extras := map[string]any{
		"files_to_resolve": runtime.MakeFilesToResolveFn(e.store, nil),
	}
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "cpp.risor"), extras)
	require.NoError(e.t, err)
}

// --- Tests ---

func TestCppResolve_ClassMethodResolution(t *testing.T) {
	env := newCppTestEnv(t)
	env.extractCppSource(`class Server {
public:
    void start() {}
    void run() {
        start();
    }
};
`, "server.cpp")

	env.resolveCpp()

	// Find the "start" call reference (from inside run())
	refs, err := env.store.ReferencesByName("start")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to start")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected start method call to be resolved")
}

func TestCppResolve_NamespaceQualifiedCall(t *testing.T) {
	env := newCppTestEnv(t)

	// File 1: namespace with a function
	env.extractCppSource(`namespace utils {
    int add(int a, int b) {
        return a + b;
    }
}
`, "utils.cpp")

	// File 2: uses the function
	env.extractCppSource(`#include "utils.cpp"

int main() {
    add(1, 2);
    return 0;
}
`, "main.cpp")

	env.resolveCpp()

	refs, err := env.store.ReferencesByName("add")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to add")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected add call to be resolved via include")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "add", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestCppResolve_InheritanceDetection(t *testing.T) {
	env := newCppTestEnv(t)
	env.extractCppSource(`class Animal {
public:
    virtual void speak() {}
};

class Dog : public Animal {
public:
    void speak() override {}
    void fetch() {}
};
`, "animals.cpp")

	env.resolveCpp()

	// Find implementation record: Dog -> Animal
	animalSym := findSymbolByName(t, env.store, "Animal", "class")
	require.NotNil(t, animalSym, "expected Animal class symbol")

	impls, err := env.store.ImplementationsByInterface(animalSym.ID)
	require.NoError(t, err)
	require.Len(t, impls, 1, "expected 1 implementation of Animal")

	typeSym := findSymbolByID(t, env.store, impls[0].TypeSymbolID)
	assert.Equal(t, "Dog", typeSym.Name)
	assert.Equal(t, "explicit", impls[0].Kind)
}

func TestCppResolve_VirtualMethodResolution(t *testing.T) {
	env := newCppTestEnv(t)
	// Use out-of-class method definitions so the override methods appear as
	// symbols with parent_symbol_id (inline methods are only type_members).
	env.extractCppSource(`class Shape {
public:
    virtual double area() = 0;
};

class Circle : public Shape {
public:
    double area() override;
};

double Circle::area() { return 3.14; }
`, "shapes.cpp")

	env.resolveCpp()

	// Verify inheritance: Circle -> Shape
	shapeSym := findSymbolByName(t, env.store, "Shape", "class")
	require.NotNil(t, shapeSym)

	impls, err := env.store.ImplementationsByInterface(shapeSym.ID)
	require.NoError(t, err)
	require.Len(t, impls, 1, "expected 1 implementation of Shape")

	typeSym := findSymbolByID(t, env.store, impls[0].TypeSymbolID)
	assert.Equal(t, "Circle", typeSym.Name)

	// Verify virtual override extension binding
	bindings, err := env.store.ExtensionBindingsByType(shapeSym.ID)
	require.NoError(t, err)

	foundOverride := false
	for _, b := range bindings {
		if b.Kind == "override" {
			sym := findSymbolByID(t, env.store, b.MemberSymbolID)
			if sym.Name == "area" {
				foundOverride = true
				assert.Equal(t, "Shape", b.ExtendedTypeExpr)
			}
		}
	}
	assert.True(t, foundOverride, "expected virtual override extension binding for area")
}

func TestCppResolve_CallGraphEdges(t *testing.T) {
	env := newCppTestEnv(t)
	env.extractCppSource(`void helper() {}

int main() {
    helper();
    return 0;
}
`, "main.cpp")

	env.resolveCpp()

	mainSym := findSymbolByName(t, env.store, "main", "function")
	helperSym := findSymbolByName(t, env.store, "helper", "function")
	require.NotNil(t, mainSym)
	require.NotNil(t, helperSym)

	edges, err := env.store.CalleesByCaller(mainSym.ID)
	require.NoError(t, err)

	found := false
	for _, e := range edges {
		if e.CalleeSymbolID == helperSym.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "expected call edge from main to helper")
}

func TestCppResolve_UnresolvedReference(t *testing.T) {
	env := newCppTestEnv(t)
	env.extractCppSource(`int main() {
    nonexistent();
    return 0;
}
`, "main.cpp")

	env.resolveCpp()

	refs, err := env.store.ReferencesByName("nonexistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonexistent should not be resolved")
	}
}

func TestCppResolve_TemplateClassResolution(t *testing.T) {
	env := newCppTestEnv(t)
	env.extractCppSource(`template<typename T>
class Container {
public:
    void add(T item) {}
    T get() { return T(); }
};

int main() {
    Container<int> c;
    return 0;
}
`, "container.cpp")

	env.resolveCpp()

	// Verify the Container template class exists
	containerSym := findSymbolByName(t, env.store, "Container", "class")
	require.NotNil(t, containerSym, "expected Container class symbol")

	// type_annotation references to "Container" should resolve
	refs, err := env.store.ReferencesByName("Container")
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Container")

	resolved, err := env.store.ResolvedReferencesByRef(typeRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Container type reference to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Container", targetSym.Name)
	assert.Equal(t, "class", targetSym.Kind)
}

func TestCppResolve_OutOfClassMethodExtensionBinding(t *testing.T) {
	env := newCppTestEnv(t)
	env.extractCppSource(`class Server {
public:
    void start();
    void stop();
};

void Server::start() {}
void Server::stop() {}
`, "server.cpp")

	env.resolveCpp()

	serverSym := findSymbolByName(t, env.store, "Server", "class")
	require.NotNil(t, serverSym)

	bindings, err := env.store.ExtensionBindingsByType(serverSym.ID)
	require.NoError(t, err)

	// Should have extension bindings for the out-of-class method definitions
	methodNames := map[string]bool{}
	for _, b := range bindings {
		if b.Kind == "method" {
			sym := findSymbolByID(t, env.store, b.MemberSymbolID)
			methodNames[sym.Name] = true
			assert.Equal(t, "Server", b.ExtendedTypeExpr)
		}
	}
	assert.True(t, methodNames["start"], "expected extension binding for start")
	assert.True(t, methodNames["stop"], "expected extension binding for stop")
}

func TestCppResolve_MultipleInheritance(t *testing.T) {
	env := newCppTestEnv(t)
	env.extractCppSource(`class Flyable {
public:
    virtual void fly() {}
};

class Swimmable {
public:
    virtual void swim() {}
};

class Duck : public Flyable, public Swimmable {
public:
    void fly() override {}
    void swim() override {}
};
`, "duck.cpp")

	env.resolveCpp()

	flyableSym := findSymbolByName(t, env.store, "Flyable", "class")
	swimmableSym := findSymbolByName(t, env.store, "Swimmable", "class")
	require.NotNil(t, flyableSym)
	require.NotNil(t, swimmableSym)

	flyableImpls, err := env.store.ImplementationsByInterface(flyableSym.ID)
	require.NoError(t, err)
	require.Len(t, flyableImpls, 1)

	swimmableImpls, err := env.store.ImplementationsByInterface(swimmableSym.ID)
	require.NoError(t, err)
	require.Len(t, swimmableImpls, 1)

	// Both should point to Duck
	assert.Equal(t, flyableImpls[0].TypeSymbolID, swimmableImpls[0].TypeSymbolID)

	duckSym := findSymbolByID(t, env.store, flyableImpls[0].TypeSymbolID)
	assert.Equal(t, "Duck", duckSym.Name)
}
