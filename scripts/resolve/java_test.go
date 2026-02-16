package go_resolve_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jward/canopy/internal/runtime"
	"github.com/jward/canopy/internal/store"
)

// extractJavaSource writes Java source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractJavaSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	javaFile := filepath.Join(dir, filename)
	require.NoError(e.t, os.WriteFile(javaFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     javaFile,
		Language: "java",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": javaFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "java.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

// resolveJava runs the Java resolution script.
func (e *testEnv) resolveJava() {
	e.t.Helper()
	extras := map[string]any{
		"files_to_resolve": runtime.MakeFilesToResolveFn(e.store, nil),
	}
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "java.risor"), extras)
	require.NoError(e.t, err)
}

// --- Tests ---

func TestJavaResolve_SameFileMethodCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public class App {
    public void run() {
        helper();
    }

    public void helper() {}
}
`, "App.java")

	env.resolveJava()

	// Find the "helper" call reference
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

	// Verify it resolved
	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected helper call to be resolved")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "helper", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestJavaResolve_CrossFileImportResolution(t *testing.T) {
	env := newTestEnv(t)

	// File 1: A utility class in com.example package
	env.extractJavaSource(`package com.example;

public class Util {
    public static String format(String s) {
        return s;
    }
}
`, "Util.java")

	// File 2: Main class imports and uses Util
	env.extractJavaSource(`package com.app;

import com.example.Util;

public class Main {
    public void run() {
        Util.format("hello");
    }
}
`, "Main.java")

	env.resolveJava()

	// The "Util" reference (type_annotation from import resolution) should resolve
	refs, err := env.store.ReferencesByName("Util")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	resolved := false
	for _, r := range refs {
		rr, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		if len(rr) > 0 {
			targetSym := findSymbolByID(t, env.store, rr[0].TargetSymbolID)
			if targetSym.Name == "Util" && targetSym.Kind == "class" {
				resolved = true
				assert.Equal(t, "import", rr[0].ResolutionKind)
				break
			}
		}
	}
	assert.True(t, resolved, "expected Util to be resolved via import")
}

func TestJavaResolve_ClassHierarchyExtends(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public class Animal {
    public void eat() {}
}
`, "Animal.java")

	env.extractJavaSource(`public class Dog extends Animal {
    public void bark() {}
}
`, "Dog.java")

	env.resolveJava()

	animalSym := findSymbolByName(t, env.store, "Animal", "class")
	dogSym := findSymbolByName(t, env.store, "Dog", "class")
	require.NotNil(t, animalSym)
	require.NotNil(t, dogSym)

	// Dog extends Animal — should create an implementation record with kind="extends"
	impls, err := env.store.ImplementationsByType(dogSym.ID)
	require.NoError(t, err)
	require.NotEmpty(t, impls, "expected extends relationship for Dog")

	found := false
	for _, impl := range impls {
		if impl.InterfaceSymbolID == animalSym.ID && impl.Kind == "extends" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected Dog extends Animal implementation record")
}

func TestJavaResolve_InterfaceImplementation(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public interface Runnable {
    void run();
}
`, "Runnable.java")

	env.extractJavaSource(`public class Worker implements Runnable {
    public void run() {}
}
`, "Worker.java")

	env.resolveJava()

	runnableSym := findSymbolByName(t, env.store, "Runnable", "interface")
	workerSym := findSymbolByName(t, env.store, "Worker", "class")
	require.NotNil(t, runnableSym)
	require.NotNil(t, workerSym)

	// Worker implements Runnable — should create an implementation record with kind="explicit"
	impls, err := env.store.ImplementationsByInterface(runnableSym.ID)
	require.NoError(t, err)
	require.NotEmpty(t, impls, "expected implementation of Runnable")

	found := false
	for _, impl := range impls {
		if impl.TypeSymbolID == workerSym.ID && impl.Kind == "explicit" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected Worker implements Runnable implementation record")
}

func TestJavaResolve_MethodResolutionQualifiedCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`package com.example;

public class Service {
    public void start() {}

    public void stop() {}
}
`, "Service.java")

	env.extractJavaSource(`package com.example;

public class Controller {
    public void init() {
        Service svc = new Service();
        svc.start();
    }
}
`, "Controller.java")

	env.resolveJava()

	// The "start" call reference should resolve to Service.start
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

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "start", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestJavaResolve_CallGraphEdges(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public class Processor {
    public void process() {
        helper();
    }

    private void helper() {}
}
`, "Processor.java")

	env.resolveJava()

	processSym := findSymbolByName(t, env.store, "process", "method")
	helperSym := findSymbolByName(t, env.store, "helper", "method")
	require.NotNil(t, processSym)
	require.NotNil(t, helperSym)

	edges, err := env.store.CalleesByCaller(processSym.ID)
	require.NoError(t, err)

	found := false
	for _, e := range edges {
		if e.CalleeSymbolID == helperSym.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "expected call edge from process to helper")
}

func TestJavaResolve_UnresolvedReference(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public class App {
    public void run() {
        nonExistent();
    }
}
`, "App.java")

	env.resolveJava()

	refs, err := env.store.ReferencesByName("nonExistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	// nonExistent should NOT be resolved
	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonExistent should not be resolved")
	}
}

func TestJavaResolve_ExtensionBindingsForClassMethods(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public class Server {
    public void start() {}
    public void stop() {}
}
`, "Server.java")

	env.resolveJava()

	serverSym := findSymbolByName(t, env.store, "Server", "class")
	require.NotNil(t, serverSym)

	bindings, err := env.store.ExtensionBindingsByType(serverSym.ID)
	require.NoError(t, err)
	require.Len(t, bindings, 2, "expected 2 extension bindings (start + stop)")

	names := map[string]bool{}
	for _, b := range bindings {
		sym := findSymbolByID(t, env.store, b.MemberSymbolID)
		names[sym.Name] = true
		assert.Equal(t, "method", b.Kind)
		assert.Equal(t, "Server", b.ExtendedTypeExpr)
	}
	assert.True(t, names["start"])
	assert.True(t, names["stop"])
}

func TestJavaResolve_ConstructorCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public class User {
    private String name;

    public User(String name) {
        this.name = name;
    }

    public static User create() {
        return new User("test");
    }
}
`, "User.java")

	env.resolveJava()

	// "new User()" creates a call reference to "User" which should resolve
	// to the User class or constructor
	refs, err := env.store.ReferencesByName("User")
	require.NoError(t, err)

	resolvedCount := 0
	for _, r := range refs {
		if r.Context == "call" {
			rr, err := env.store.ResolvedReferencesByRef(r.ID)
			require.NoError(t, err)
			if len(rr) > 0 {
				resolvedCount++
			}
		}
	}
	assert.Greater(t, resolvedCount, 0, "expected new User() to resolve")
}

func TestJavaResolve_SamePackageCrossFileResolution(t *testing.T) {
	env := newTestEnv(t)

	// Two files in the same package
	env.extractJavaSource(`package com.example;

public class Helper {
    public static void assist() {}
}
`, "Helper.java")

	env.extractJavaSource(`package com.example;

public class Main {
    public void run() {
        Helper.assist();
    }
}
`, "Main.java")

	env.resolveJava()

	// "Helper" reference should resolve to the Helper class
	refs, err := env.store.ReferencesByName("Helper")
	require.NoError(t, err)

	resolved := false
	for _, r := range refs {
		rr, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		if len(rr) > 0 {
			targetSym := findSymbolByID(t, env.store, rr[0].TargetSymbolID)
			if targetSym.Name == "Helper" && targetSym.Kind == "class" {
				resolved = true
				break
			}
		}
	}
	assert.True(t, resolved, "expected Helper to be resolved via same-package resolution")
}

func TestJavaResolve_MultipleInterfaceImplementation(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`public interface Readable {
    String read();
}
`, "Readable.java")

	env.extractJavaSource(`public interface Writable {
    void write(String data);
}
`, "Writable.java")

	env.extractJavaSource(`public class FileHandler implements Readable, Writable {
    public String read() { return ""; }
    public void write(String data) {}
}
`, "FileHandler.java")

	env.resolveJava()

	readableSym := findSymbolByName(t, env.store, "Readable", "interface")
	writableSym := findSymbolByName(t, env.store, "Writable", "interface")
	fileHandlerSym := findSymbolByName(t, env.store, "FileHandler", "class")
	require.NotNil(t, readableSym)
	require.NotNil(t, writableSym)
	require.NotNil(t, fileHandlerSym)

	readImpls, err := env.store.ImplementationsByInterface(readableSym.ID)
	require.NoError(t, err)
	require.NotEmpty(t, readImpls, "expected implementation of Readable")

	writeImpls, err := env.store.ImplementationsByInterface(writableSym.ID)
	require.NoError(t, err)
	require.NotEmpty(t, writeImpls, "expected implementation of Writable")

	// Both should point to FileHandler
	assert.Equal(t, fileHandlerSym.ID, readImpls[0].TypeSymbolID)
	assert.Equal(t, fileHandlerSym.ID, writeImpls[0].TypeSymbolID)
	assert.Equal(t, "explicit", readImpls[0].Kind)
	assert.Equal(t, "explicit", writeImpls[0].Kind)
}

func TestJavaResolve_TypeAnnotationResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractJavaSource(`package com.example;

public class Config {
    private String host;
}
`, "Config.java")

	env.extractJavaSource(`package com.example;

public class App {
    public Config getConfig() {
        return new Config();
    }
}
`, "App.java")

	env.resolveJava()

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
