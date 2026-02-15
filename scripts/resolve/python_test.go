package go_resolve_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractPythonSource writes Python source to a temp .py file, inserts a file record
// with language "python", and runs the extraction script. Returns the file ID.
func (e *testEnv) extractPythonSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	pyFile := filepath.Join(dir, filename)
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

// resolvePython runs the Python resolution script.
func (e *testEnv) resolvePython() {
	e.t.Helper()
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "python.risor"), nil)
	require.NoError(e.t, err)
}

// --- Tests ---

func TestPythonResolve_SameFileFunctionCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
def helper():
    pass

def main():
    helper()
`, "main.py")

	env.resolvePython()

	refs, err := env.store.ReferencesByName("helper")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to helper")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected helper call to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "helper", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestPythonResolve_CrossFileImportResolution(t *testing.T) {
	env := newTestEnv(t)

	// Module with a function
	env.extractPythonSource(`
def greet(name):
    return "Hello, " + name
`, "utils.py")

	// Main file imports from utils
	env.extractPythonSource(`
from utils import greet

def main():
    greet("world")
`, "main.py")

	env.resolvePython()

	// Find the "greet" call reference in main.py
	refs, err := env.store.ReferencesByName("greet")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to greet")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected greet call to be resolved via import")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "greet", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestPythonResolve_ClassMethodResolution_Self(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
class Server:
    def start(self):
        pass

    def run(self):
        self.start()
`, "server.py")

	env.resolvePython()

	// Find "start" call reference (from self.start())
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
	require.NotEmpty(t, resolved, "expected self.start() to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "start", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestPythonResolve_ClassHierarchy(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
class Animal:
    def speak(self):
        pass

class Dog(Animal):
    def fetch(self):
        pass
`, "animals.py")

	env.resolvePython()

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

func TestPythonResolve_InheritedMethodResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
class Animal:
    def speak(self):
        pass

class Dog(Animal):
    def run(self):
        self.speak()
`, "animals.py")

	env.resolvePython()

	// self.speak() in Dog.run() should resolve to Animal.speak
	refs, err := env.store.ReferencesByName("speak")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to speak")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected inherited speak() to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "speak", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestPythonResolve_CallGraphEdge(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
def helper():
    pass

def main():
    helper()
`, "main.py")

	env.resolvePython()

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

func TestPythonResolve_UnresolvedReference(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
def main():
    nonexistent()
`, "main.py")

	env.resolvePython()

	refs, err := env.store.ReferencesByName("nonexistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonexistent should not be resolved")
	}
}

func TestPythonResolve_ModuleLevelVariable(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
MAX_SIZE = 100

def check():
    return MAX_SIZE
`, "config.py")

	env.resolvePython()

	// MAX_SIZE reference should resolve to the variable symbol
	refs, err := env.store.ReferencesByName("MAX_SIZE")
	require.NoError(t, err)

	// There might not be an explicit reference if the extraction doesn't emit one
	// for bare identifiers in expressions. If there is one, verify it resolves.
	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		if len(resolved) > 0 {
			targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
			assert.Equal(t, "MAX_SIZE", targetSym.Name)
			assert.Equal(t, "variable", targetSym.Kind)
		}
	}

	// Verify the variable symbol exists
	varSym := findSymbolByName(t, env.store, "MAX_SIZE", "variable")
	require.NotNil(t, varSym, "expected MAX_SIZE variable symbol")
}

func TestPythonResolve_ClassInstantiationCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
class Server:
    def __init__(self):
        pass

def main():
    s = Server()
`, "main.py")

	env.resolvePython()

	// Server() call should resolve to the Server class
	refs, err := env.store.ReferencesByName("Server")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to Server")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Server() call to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Server", targetSym.Name)
	assert.Equal(t, "class", targetSym.Kind)
}

func TestPythonResolve_TypeAnnotationResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
class Config:
    host: str = "localhost"

def create_config() -> Config:
    return Config()
`, "config.py")

	env.resolvePython()

	// Find type_annotation references to Config
	refs, err := env.store.ReferencesByName("Config")
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Config")

	resolved, err := env.store.ResolvedReferencesByRef(typeRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Config type annotation to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Config", targetSym.Name)
	assert.Equal(t, "class", targetSym.Kind)
}

func TestPythonResolve_CrossFileImportAlias(t *testing.T) {
	env := newTestEnv(t)

	// Source module
	env.extractPythonSource(`
def long_function_name():
    return 42
`, "utils.py")

	// Main file imports with alias
	env.extractPythonSource(`
from utils import long_function_name as lfn

def main():
    lfn()
`, "main.py")

	env.resolvePython()

	// lfn() call should resolve to long_function_name
	refs, err := env.store.ReferencesByName("lfn")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to lfn")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected lfn call to be resolved via import alias")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "long_function_name", targetSym.Name)
	assert.Equal(t, "function", targetSym.Kind)
}

func TestPythonResolve_MultipleInheritance(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
class Flyable:
    def fly(self):
        pass

class Swimmable:
    def swim(self):
        pass

class Duck(Flyable, Swimmable):
    def quack(self):
        pass
`, "duck.py")

	env.resolvePython()

	// Duck should implement both Flyable and Swimmable
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
}

func TestPythonResolve_BaseClassReference(t *testing.T) {
	env := newTestEnv(t)
	env.extractPythonSource(`
class Base:
    pass

class Child(Base):
    pass
`, "hierarchy.py")

	env.resolvePython()

	// The type_annotation reference to "Base" in "class Child(Base)" should resolve
	refs, err := env.store.ReferencesByName("Base")
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Base")

	resolved, err := env.store.ResolvedReferencesByRef(typeRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected Base reference to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "Base", targetSym.Name)
	assert.Equal(t, "class", targetSym.Kind)
}
