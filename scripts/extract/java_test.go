package go_extract_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractJavaSource writes Java source to a temp file, inserts a file record,
// and runs the extraction script. Returns the file ID.
func (e *testEnv) extractJavaSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	javaFile := filepath.Join(dir, "Test.java")
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

// ---------- Java Tests ----------

func TestJava_ClassWithFieldsAndMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Person {
    private String name;
    private int age;

    public String getName() {
        return name;
    }

    public void setAge(int age) {
        this.age = age;
    }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	var methods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}

	require.NotNil(t, classSym, "expected class symbol")
	assert.Equal(t, "Person", classSym.Name)
	assert.Equal(t, "public", classSym.Visibility)

	// Two methods
	require.Len(t, methods, 2)
	methodNames := map[string]string{}
	for _, m := range methods {
		methodNames[m.Name] = m.Visibility
		require.NotNil(t, m.ParentSymbolID, "method should have parent_symbol_id")
		assert.Equal(t, classSym.ID, *m.ParentSymbolID)
	}
	assert.Equal(t, "public", methodNames["getName"])
	assert.Equal(t, "public", methodNames["setAge"])

	// Fields as type_members
	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	fieldMembers := map[string]*store.TypeMember{}
	for _, m := range members {
		if m.Kind == "field" {
			fieldMembers[m.Name] = m
		}
	}
	require.Len(t, fieldMembers, 2)
	assert.Equal(t, "String", fieldMembers["name"].TypeExpr)
	assert.Equal(t, "private", fieldMembers["name"].Visibility)
	assert.Equal(t, "int", fieldMembers["age"].TypeExpr)
	assert.Equal(t, "private", fieldMembers["age"].Visibility)
}

func TestJava_InterfaceWithMethodSignatures(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public interface Repository {
    void save(String item);
    String findById(int id);
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var ifaceSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "interface" {
			ifaceSym = s
			break
		}
	}
	require.NotNil(t, ifaceSym)
	assert.Equal(t, "Repository", ifaceSym.Name)
	assert.Equal(t, "public", ifaceSym.Visibility)

	members, err := env.store.TypeMembers(ifaceSym.ID)
	require.NoError(t, err)

	methodMembers := map[string]*store.TypeMember{}
	for _, m := range members {
		if m.Kind == "method" {
			methodMembers[m.Name] = m
		}
	}
	require.Len(t, methodMembers, 2)
	assert.Equal(t, "void", methodMembers["save"].TypeExpr)
	assert.Equal(t, "String", methodMembers["findById"].TypeExpr)

	// Interface methods should be public by default
	var methods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}
	for _, m := range methods {
		assert.Equal(t, "public", m.Visibility, "interface method %s should be public", m.Name)
	}
}

func TestJava_Annotations(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Service {
    @Override
    public String toString() {
        return "service";
    }

    @Deprecated
    public void oldMethod() {}

    @SuppressWarnings("unchecked")
    public void warned() {}
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	for _, s := range syms {
		if s.Kind != "method" {
			continue
		}

		anns, err := env.store.AnnotationsByTarget(s.ID)
		require.NoError(t, err)

		switch s.Name {
		case "toString":
			require.Len(t, anns, 1)
			assert.Equal(t, "Override", anns[0].Name)
		case "oldMethod":
			require.Len(t, anns, 1)
			assert.Equal(t, "Deprecated", anns[0].Name)
		case "warned":
			require.Len(t, anns, 1)
			assert.Equal(t, "SuppressWarnings", anns[0].Name)
			assert.Contains(t, anns[0].Arguments, "unchecked")
		}
	}
}

func TestJava_PackageDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`package com.example.myapp;

public class App {}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var pkgSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "package" {
			pkgSym = s
			break
		}
	}
	require.NotNil(t, pkgSym)
	assert.Equal(t, "com.example.myapp", pkgSym.Name)
}

func TestJava_Imports(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`package com.example;

import java.util.List;
import java.util.*;
import static java.util.Collections.emptyList;

public class App {}
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 3)

	impBySource := map[string]*store.Import{}
	for _, imp := range imports {
		impBySource[imp.Source] = imp
	}

	// Regular import
	listImp := impBySource["java.util.List"]
	require.NotNil(t, listImp)
	require.NotNil(t, listImp.ImportedName)
	assert.Equal(t, "List", *listImp.ImportedName)
	assert.Equal(t, "module", listImp.Kind)

	// Wildcard import
	starImp := impBySource["java.util.*"]
	require.NotNil(t, starImp)
	require.NotNil(t, starImp.ImportedName)
	assert.Equal(t, "*", *starImp.ImportedName)
	assert.Equal(t, "module", starImp.Kind)

	// Static import
	staticImp := impBySource["java.util.Collections.emptyList"]
	require.NotNil(t, staticImp)
	require.NotNil(t, staticImp.ImportedName)
	assert.Equal(t, "emptyList", *staticImp.ImportedName)
	assert.Equal(t, "static", staticImp.Kind)
}

func TestJava_EnumWithConstants(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public enum Status {
    ACTIVE,
    INACTIVE,
    PENDING
}
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
	assert.Equal(t, "Status", enumSym.Name)
	assert.Equal(t, "public", enumSym.Visibility)

	members, err := env.store.TypeMembers(enumSym.ID)
	require.NoError(t, err)

	variants := map[string]*store.TypeMember{}
	for _, m := range members {
		if m.Kind == "variant" {
			variants[m.Name] = m
		}
	}
	require.Len(t, variants, 3)
	assert.NotNil(t, variants["ACTIVE"])
	assert.NotNil(t, variants["INACTIVE"])
	assert.NotNil(t, variants["PENDING"])

	// Variants should have the enum type as type_expr
	assert.Equal(t, "Status", variants["ACTIVE"].TypeExpr)
}

func TestJava_ScopeTree(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Processor {
    public void process(int x) {
        if (x > 0) {
            for (int i = 0; i < x; i++) {
            }
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
	assert.Equal(t, 1, kinds["class"], "expected 1 class scope")
	assert.Equal(t, 1, kinds["function"], "expected 1 function scope (method)")
	assert.GreaterOrEqual(t, kinds["block"], 2, "expected at least 2 block scopes (if + for)")

	// Verify nesting: file -> class -> function -> block
	var fileScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "file" {
			fileScope = s
			break
		}
	}
	require.NotNil(t, fileScope)
	assert.Nil(t, fileScope.ParentScopeID, "file scope should have no parent")

	var classScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "class" {
			classScope = s
			break
		}
	}
	require.NotNil(t, classScope)
	require.NotNil(t, classScope.ParentScopeID)
	assert.Equal(t, fileScope.ID, *classScope.ParentScopeID)

	var funcScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "function" {
			funcScope = s
			break
		}
	}
	require.NotNil(t, funcScope)
	require.NotNil(t, funcScope.ParentScopeID)
	assert.Equal(t, classScope.ID, *funcScope.ParentScopeID)
}

func TestJava_References(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class App {
    public void run() {
        System.out.println("hello");
        String s = new String("test");
    }
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	refsByContext := map[string][]string{}
	for _, r := range refs {
		refsByContext[r.Context] = append(refsByContext[r.Context], r.Name)
	}

	// Method call reference
	assert.Contains(t, refsByContext["call"], "println")
	// Object field access
	assert.Contains(t, refsByContext["field_access"], "out")
	// new String() is a call reference
	assert.Contains(t, refsByContext["call"], "String")
}

func TestJava_FunctionParams(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var addSym *store.Symbol
	for _, s := range syms {
		if s.Name == "add" && s.Kind == "method" {
			addSym = s
			break
		}
	}
	require.NotNil(t, addSym)

	params, err := env.store.FunctionParams(addSym.ID)
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

	// Return type
	require.Len(t, returnParams, 1)
	assert.Equal(t, "int", returnParams[0].TypeExpr)
	assert.True(t, returnParams[0].IsReturn)
}

func TestJava_GenericsWithBounds(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Container<T extends Comparable<T>> {
    private T value;

    public <U> void transform(U input) {}
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym, methodSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
		if s.Kind == "method" && s.Name == "transform" {
			methodSym = s
		}
	}

	// Class type parameter
	require.NotNil(t, classSym)
	classTPs, err := env.store.TypeParams(classSym.ID)
	require.NoError(t, err)
	require.Len(t, classTPs, 1)
	assert.Equal(t, "T", classTPs[0].Name)
	assert.Contains(t, classTPs[0].Constraints, "extends")
	assert.Equal(t, 0, classTPs[0].Ordinal)

	// Method type parameter
	require.NotNil(t, methodSym)
	methodTPs, err := env.store.TypeParams(methodSym.ID)
	require.NoError(t, err)
	require.Len(t, methodTPs, 1)
	assert.Equal(t, "U", methodTPs[0].Name)
}

func TestJava_AccessModifiers(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Access {
    public void pubMethod() {}
    private void privMethod() {}
    protected void protMethod() {}
    void defaultMethod() {}
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	vis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "method" {
			vis[s.Name] = s.Visibility
		}
	}

	assert.Equal(t, "public", vis["pubMethod"])
	assert.Equal(t, "private", vis["privMethod"])
	assert.Equal(t, "protected", vis["protMethod"])
	assert.Equal(t, "package", vis["defaultMethod"])
}

func TestJava_StaticAndFinalModifiers(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Constants {
    public static final int MAX_SIZE = 100;
    public static final String NAME = "test";
    private int count;
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	consts := map[string]*store.Symbol{}
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
		if s.Kind == "constant" {
			consts[s.Name] = s
		}
	}
	require.NotNil(t, classSym)

	// static final fields should be extracted as constants
	require.Len(t, consts, 2)
	assert.NotNil(t, consts["MAX_SIZE"])
	assert.NotNil(t, consts["NAME"])

	// Check type_members too
	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	memberByName := map[string]*store.TypeMember{}
	for _, m := range members {
		memberByName[m.Name] = m
	}

	assert.Equal(t, "constant", memberByName["MAX_SIZE"].Kind)
	assert.Equal(t, "int", memberByName["MAX_SIZE"].TypeExpr)
	assert.Equal(t, "field", memberByName["count"].Kind)
}

func TestJava_Constructor(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class User {
    private String name;

    public User(String name) {
        this.name = name;
    }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym, ctorSym *store.Symbol
	for _, s := range syms {
		switch s.Kind {
		case "class":
			classSym = s
		case "constructor":
			ctorSym = s
		}
	}

	require.NotNil(t, classSym)
	require.NotNil(t, ctorSym, "expected constructor symbol")
	assert.Equal(t, "User", ctorSym.Name)
	assert.Equal(t, "public", ctorSym.Visibility)

	// Constructor should be linked to the class
	require.NotNil(t, ctorSym.ParentSymbolID)
	assert.Equal(t, classSym.ID, *ctorSym.ParentSymbolID)

	// Constructor parameters
	params, err := env.store.FunctionParams(ctorSym.ID)
	require.NoError(t, err)
	require.Len(t, params, 1)
	assert.Equal(t, "name", params[0].Name)
	assert.Equal(t, "String", params[0].TypeExpr)
}

func TestJava_InnerClass(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Outer {
    public class Inner {
        private int value;
    }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var outerSym, innerSym *store.Symbol
	for _, s := range syms {
		if s.Kind != "class" {
			continue
		}
		if s.Name == "Outer" {
			outerSym = s
		}
		if s.Name == "Inner" {
			innerSym = s
		}
	}

	require.NotNil(t, outerSym)
	require.NotNil(t, innerSym)

	// Inner class should have outer as parent
	require.NotNil(t, innerSym.ParentSymbolID)
	assert.Equal(t, outerSym.ID, *innerSym.ParentSymbolID)
}

func TestJava_AbstractClassAndMethod(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public abstract class Shape {
    public abstract double area();
    public void describe() {}
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	methods := map[string]*store.Symbol{}
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
		if s.Kind == "method" {
			methods[s.Name] = s
		}
	}

	require.NotNil(t, classSym)
	assert.Equal(t, "Shape", classSym.Name)

	// Both methods should exist
	require.Len(t, methods, 2)
	assert.NotNil(t, methods["area"])
	assert.NotNil(t, methods["describe"])
}

func TestJava_TypeReferences(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Converter {
    public String convert(Integer input) {
        return input.toString();
    }
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	typeRefs := map[string]bool{}
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRefs[r.Name] = true
		}
	}

	// Should reference String and Integer as type annotations
	assert.True(t, typeRefs["String"], "expected type_annotation reference to String")
	assert.True(t, typeRefs["Integer"], "expected type_annotation reference to Integer")
}

func TestJava_ComprehensiveFile(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`package com.example;

import java.util.List;
import java.util.*;

@SuppressWarnings("unchecked")
public class MyService<T extends Comparable<T>> implements Runnable {
    public static final int VERSION = 1;
    private String name;

    public MyService(String name) {
        this.name = name;
    }

    @Override
    public void run() {
        System.out.println(name);
    }

    public T process(T input) {
        return input;
    }

    private static String helper() {
        return "help";
    }

    public enum Status {
        ACTIVE,
        INACTIVE
    }

    public interface Callback {
        void onComplete(String result);
    }

    public class Inner {
        int value;
    }
}
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["package"], "com.example")
	assert.Contains(t, kinds["class"], "MyService")
	assert.Contains(t, kinds["class"], "Inner")
	assert.Contains(t, kinds["constructor"], "MyService")
	assert.Contains(t, kinds["method"], "run")
	assert.Contains(t, kinds["method"], "process")
	assert.Contains(t, kinds["method"], "helper")
	assert.Contains(t, kinds["constant"], "VERSION")
	assert.Contains(t, kinds["enum"], "Status")
	assert.Contains(t, kinds["interface"], "Callback")

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	// Verify type parameters
	var serviceSym *store.Symbol
	for _, s := range syms {
		if s.Name == "MyService" && s.Kind == "class" {
			serviceSym = s
			break
		}
	}
	require.NotNil(t, serviceSym)
	tps, err := env.store.TypeParams(serviceSym.ID)
	require.NoError(t, err)
	require.Len(t, tps, 1)
	assert.Equal(t, "T", tps[0].Name)

	// Verify scope tree exists
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 4) // file + class + methods

	// Verify references exist
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)

	// Verify annotations
	var runSym *store.Symbol
	for _, s := range syms {
		if s.Name == "run" && s.Kind == "method" {
			runSym = s
			break
		}
	}
	require.NotNil(t, runSym)
	anns, err := env.store.AnnotationsByTarget(runSym.ID)
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, "Override", anns[0].Name)

	// Verify enum members
	var enumSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "enum" {
			enumSym = s
			break
		}
	}
	require.NotNil(t, enumSym)
	members, err := env.store.TypeMembers(enumSym.ID)
	require.NoError(t, err)
	variantCount := 0
	for _, m := range members {
		if m.Kind == "variant" {
			variantCount++
		}
	}
	assert.Equal(t, 2, variantCount)

	// Verify inner class is linked to outer
	var innerSym *store.Symbol
	for _, s := range syms {
		if s.Name == "Inner" {
			innerSym = s
			break
		}
	}
	require.NotNil(t, innerSym)
	require.NotNil(t, innerSym.ParentSymbolID)
	assert.Equal(t, serviceSym.ID, *innerSym.ParentSymbolID)
}

func TestJava_VoidMethodNoReturnParam(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Foo {
    public void doNothing() {}
}
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

	// Void method should have no return params
	for _, p := range params {
		assert.False(t, p.IsReturn, "void method should not have return param")
	}
}

func TestJava_MethodCallReferences(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Caller {
    public void execute() {
        helper();
        System.out.println("test");
    }

    private void helper() {}
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	callNames := map[string]bool{}
	for _, r := range refs {
		if r.Context == "call" {
			callNames[r.Name] = true
		}
	}

	assert.True(t, callNames["helper"], "expected call reference to helper")
	assert.True(t, callNames["println"], "expected call reference to println")
}

func TestJava_ClassAnnotation(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`@Deprecated
public class OldService {
    public void run() {}
}
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

	anns, err := env.store.AnnotationsByTarget(classSym.ID)
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, "Deprecated", anns[0].Name)
}

func TestJava_ImplementsInterface(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Worker implements Runnable {
    public void run() {}
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var runnableRef *store.Reference
	for _, r := range refs {
		if r.Name == "Runnable" && r.Context == "type_annotation" {
			runnableRef = r
			break
		}
	}
	require.NotNil(t, runnableRef, "expected type_annotation reference to Runnable")
}

func TestJava_ExtendsClass(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Child extends Parent {
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var parentRef *store.Reference
	for _, r := range refs {
		if r.Name == "Parent" && r.Context == "type_annotation" {
			parentRef = r
			break
		}
	}
	require.NotNil(t, parentRef, "expected type_annotation reference to Parent")
}

func TestJava_FieldAccess(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractJavaSource(`public class Logger {
    public void log() {
        int length = System.out.toString().length();
    }
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	fieldAccessNames := map[string]bool{}
	for _, r := range refs {
		if r.Context == "field_access" {
			fieldAccessNames[r.Name] = true
		}
	}

	assert.True(t, fieldAccessNames["out"], "expected field_access reference to 'out'")
}
