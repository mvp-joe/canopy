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

// extractPHPSource writes PHP source to a temp file, inserts a file record,
// and runs the PHP extraction script. Returns the file ID.
func (e *testEnv) extractPHPSource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	phpFile := filepath.Join(dir, "test.php")
	require.NoError(e.t, os.WriteFile(phpFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     phpFile,
		Language: "php",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": phpFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "php.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

func TestPHP_ClassWithPropertiesAndMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
class User {
    public string $name;
    protected int $age;
    private bool $active;

    public function getName(): string {
        return $this->name;
    }

    protected function getAge(): int {
        return $this->age;
    }
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
	require.NotNil(t, classSym, "expected class symbol")
	assert.Equal(t, "User", classSym.Name)
	assert.Equal(t, "public", classSym.Visibility)

	// Check methods
	var methods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}
	require.Len(t, methods, 2)

	methodVis := map[string]string{}
	for _, m := range methods {
		methodVis[m.Name] = m.Visibility
		assert.NotNil(t, m.ParentSymbolID, "method should have parent_symbol_id")
		assert.Equal(t, classSym.ID, *m.ParentSymbolID)
	}
	assert.Equal(t, "public", methodVis["getName"])
	assert.Equal(t, "protected", methodVis["getAge"])

	// Check properties as type_members
	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	propsByName := map[string]*store.TypeMember{}
	for _, m := range members {
		if m.Kind == "property" {
			propsByName[m.Name] = m
		}
	}
	require.Len(t, propsByName, 3)
	assert.Equal(t, "public", propsByName["name"].Visibility)
	assert.Equal(t, "string", propsByName["name"].TypeExpr)
	assert.Equal(t, "protected", propsByName["age"].Visibility)
	assert.Equal(t, "int", propsByName["age"].TypeExpr)
	assert.Equal(t, "private", propsByName["active"].Visibility)
	assert.Equal(t, "bool", propsByName["active"].TypeExpr)
}

func TestPHP_TraitWithMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
trait Loggable {
    public function log(string $message): void {
        echo $message;
    }

    protected function logError(string $error): void {
        echo $error;
    }
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var traitSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "trait" {
			traitSym = s
			break
		}
	}
	require.NotNil(t, traitSym, "expected trait symbol")
	assert.Equal(t, "Loggable", traitSym.Name)

	// Check methods
	var methods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}
	require.Len(t, methods, 2)

	methodMap := map[string]*store.Symbol{}
	for _, m := range methods {
		methodMap[m.Name] = m
		require.NotNil(t, m.ParentSymbolID)
		assert.Equal(t, traitSym.ID, *m.ParentSymbolID)
	}
	assert.Equal(t, "public", methodMap["log"].Visibility)
	assert.Equal(t, "protected", methodMap["logError"].Visibility)
}

func TestPHP_InterfaceWithMethodSignatures(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
interface Serializable {
    public function serialize(): string;
    public function deserialize(string $data): void;
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
	require.NotNil(t, ifaceSym, "expected interface symbol")
	assert.Equal(t, "Serializable", ifaceSym.Name)

	// Check methods
	var methods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}
	require.Len(t, methods, 2)

	names := map[string]bool{}
	for _, m := range methods {
		names[m.Name] = true
		require.NotNil(t, m.ParentSymbolID)
		assert.Equal(t, ifaceSym.ID, *m.ParentSymbolID)
	}
	assert.True(t, names["serialize"])
	assert.True(t, names["deserialize"])
}

func TestPHP_FunctionDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
function helper(int $x, string $name): int {
    return $x * 2;
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
	require.NotNil(t, fnSym, "expected function symbol")
	assert.Equal(t, "helper", fnSym.Name)
	assert.Equal(t, "public", fnSym.Visibility)

	// Check params
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
	assert.Equal(t, "x", regularParams[0].Name)
	assert.Equal(t, "int", regularParams[0].TypeExpr)
	assert.Equal(t, "name", regularParams[1].Name)
	assert.Equal(t, "string", regularParams[1].TypeExpr)

	// Return type
	require.Len(t, returnParams, 1)
	assert.Equal(t, "int", returnParams[0].TypeExpr)
}

func TestPHP_NamespaceDeclaration(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
namespace App\Models;
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
	require.NotNil(t, nsSym, "expected namespace symbol")
	assert.Equal(t, "App\\Models", nsSym.Name)
}

func TestPHP_UseStatementImports(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
use App\Models\User;
use App\Interfaces\Printable;
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	impByName := map[string]*store.Import{}
	for _, imp := range imports {
		if imp.ImportedName != nil {
			impByName[*imp.ImportedName] = imp
		}
	}

	userImp := impByName["User"]
	require.NotNil(t, userImp)
	assert.Equal(t, "App\\Models\\User", userImp.Source)
	assert.Equal(t, "type", userImp.Kind)

	printImp := impByName["Printable"]
	require.NotNil(t, printImp)
	assert.Equal(t, "App\\Interfaces\\Printable", printImp.Source)
}

func TestPHP_ScopeTree(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
namespace App;

class User {
    public function getName(): string {
        return $this->name;
    }
}

function helper(): void {}
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}

	assert.Equal(t, 1, kinds["file"], "expected 1 file scope")
	assert.Equal(t, 1, kinds["namespace"], "expected 1 namespace scope")
	assert.Equal(t, 1, kinds["class"], "expected 1 class scope")
	assert.GreaterOrEqual(t, kinds["function"], 2, "expected at least 2 function scopes (method + top-level func)")

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

	// Class scope should have file as parent
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
}

func TestPHP_References_FunctionCall(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
function helper() {}

helper();
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

func TestPHP_References_MethodCall(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
class User {
    public function getName(): string {
        return $this->name;
    }
}

$user = new User();
$name = $user->getName();
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var methodCallRef *store.Reference
	for _, r := range refs {
		if r.Name == "getName" && r.Context == "call" {
			methodCallRef = r
			break
		}
	}
	require.NotNil(t, methodCallRef, "expected call reference to getName")
}

func TestPHP_References_StaticCall(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
class Factory {
    public static function create(): void {}
}

Factory::create();
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var typeRef, callRef *store.Reference
	for _, r := range refs {
		if r.Name == "Factory" && r.Context == "type_annotation" {
			typeRef = r
		}
		if r.Name == "create" && r.Context == "call" {
			callRef = r
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Factory")
	require.NotNil(t, callRef, "expected call reference to create")
}

func TestPHP_ParametersWithTypeHints(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
function greet(string $name, int $age = 25): string {
    return $name;
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

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)

	var regularParams []*store.FunctionParam
	for _, p := range params {
		if !p.IsReturn {
			regularParams = append(regularParams, p)
		}
	}
	require.Len(t, regularParams, 2)
	assert.Equal(t, "name", regularParams[0].Name)
	assert.Equal(t, "string", regularParams[0].TypeExpr)
	assert.False(t, regularParams[0].HasDefault)

	assert.Equal(t, "age", regularParams[1].Name)
	assert.Equal(t, "int", regularParams[1].TypeExpr)
	assert.True(t, regularParams[1].HasDefault)
}

func TestPHP_Visibility(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
class Example {
    public function pubMethod(): void {}
    protected function protMethod(): void {}
    private function privMethod(): void {}
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	methodVis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "method" {
			methodVis[s.Name] = s.Visibility
		}
	}
	assert.Equal(t, "public", methodVis["pubMethod"])
	assert.Equal(t, "protected", methodVis["protMethod"])
	assert.Equal(t, "private", methodVis["privMethod"])
}

func TestPHP_Constants(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
const APP_VERSION = "1.0";

class Config {
    public const MAX_RETRIES = 3;
    protected const TIMEOUT = 30;
}
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	// Top-level constant
	var constSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "constant" {
			constSym = s
			break
		}
	}
	require.NotNil(t, constSym, "expected top-level constant")
	assert.Equal(t, "APP_VERSION", constSym.Name)

	// Class constants as type_members
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

	constMembers := map[string]*store.TypeMember{}
	for _, m := range members {
		if m.Kind == "constant" {
			constMembers[m.Name] = m
		}
	}
	require.Len(t, constMembers, 2)
	assert.Equal(t, "public", constMembers["MAX_RETRIES"].Visibility)
	assert.Equal(t, "protected", constMembers["TIMEOUT"].Visibility)
}

func TestPHP_TraitUse(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
trait Loggable {}

class User {
    use Loggable;
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

	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	var embedded *store.TypeMember
	for _, m := range members {
		if m.Kind == "embedded" {
			embedded = m
			break
		}
	}
	require.NotNil(t, embedded, "expected embedded trait member")
	assert.Equal(t, "Loggable", embedded.Name)
}

func TestPHP_ImplementsInterface(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
interface Printable {}

class Report implements Printable {
}
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var typeRef *store.Reference
	for _, r := range refs {
		if r.Name == "Printable" && r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to Printable")
}

func TestPHP_ComprehensiveFile(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractPHPSource(`<?php
namespace App\Models;

use App\Interfaces\Serializable;

const VERSION = "1.0";

interface Printable {
    public function print(): void;
}

trait Loggable {
    public function log(string $msg): void {}
}

class User implements Printable {
    use Loggable;

    public const STATUS_ACTIVE = 1;
    protected string $name;

    public function __construct(string $name) {
        $this->name = $name;
    }

    public function print(): void {
        echo $this->name;
    }

    public function getName(): string {
        return $this->name;
    }
}

function helper(): void {}
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["namespace"], "App\\Models")
	assert.Contains(t, kinds["constant"], "VERSION")
	assert.Contains(t, kinds["interface"], "Printable")
	assert.Contains(t, kinds["trait"], "Loggable")
	assert.Contains(t, kinds["class"], "User")
	assert.Contains(t, kinds["function"], "helper")
	assert.GreaterOrEqual(t, len(kinds["method"]), 4)

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 1)
	assert.Equal(t, "App\\Interfaces\\Serializable", imports[0].Source)

	// Verify scopes
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 5) // file + namespace + class + trait + interface + functions

	// Verify references
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)
}
