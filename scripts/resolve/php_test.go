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

// extractPHPSource writes PHP source to a temp .php file, inserts a file record
// with language "php", and runs the extraction script. Returns the file ID.
func (e *testEnv) extractPHPSource(src string, filename string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	phpFile := filepath.Join(dir, filename)
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

// resolvePHP runs the PHP resolution script.
func (e *testEnv) resolvePHP() {
	e.t.Helper()
	err := e.rt.RunScript(context.Background(), filepath.Join("resolve", "php.risor"), nil)
	require.NoError(e.t, err)
}

// --- Tests ---

func TestPHPResolve_SameFileFunctionCall(t *testing.T) {
	env := newTestEnv(t)
	env.extractPHPSource(`<?php
function helper() {}

function main() {
    helper();
}
`, "main.php")

	env.resolvePHP()

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

func TestPHPResolve_CrossFileUseResolution(t *testing.T) {
	env := newTestEnv(t)

	// File 1: defines a class
	env.extractPHPSource(`<?php
namespace App\Models;

class User {
    public function getName() {
        return "name";
    }
}
`, "user.php")

	// File 2: uses the class via use statement
	env.extractPHPSource(`<?php
namespace App\Controllers;

use App\Models\User;

function getUser() {
    $u = new User();
}
`, "controller.php")

	env.resolvePHP()

	// Find "User" type_annotation reference (from new User())
	refs, err := env.store.ReferencesByName("User")
	require.NoError(t, err)
	var typeRef *store.Reference
	for _, r := range refs {
		if r.Context == "type_annotation" {
			typeRef = r
			break
		}
	}
	require.NotNil(t, typeRef, "expected type_annotation reference to User")

	resolved, err := env.store.ResolvedReferencesByRef(typeRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected User reference to be resolved via use statement")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "User", targetSym.Name)
	assert.Equal(t, "class", targetSym.Kind)
}

func TestPHPResolve_ClassMethodResolution(t *testing.T) {
	env := newTestEnv(t)
	env.extractPHPSource(`<?php
class Server {
    public function start() {}

    public function run() {
        $this->start();
    }
}
`, "server.php")

	env.resolvePHP()

	// Find "start" call reference
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

func TestPHPResolve_ClassHierarchy_Extends(t *testing.T) {
	env := newTestEnv(t)
	env.extractPHPSource(`<?php
class Animal {
    public function speak() {}
}

class Dog extends Animal {
    public function fetch() {}
}
`, "animals.php")

	env.resolvePHP()

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

func TestPHPResolve_CallGraphEdge(t *testing.T) {
	env := newTestEnv(t)
	env.extractPHPSource(`<?php
function helper() {}

function main() {
    helper();
}
`, "main.php")

	env.resolvePHP()

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

func TestPHPResolve_UnresolvedReference(t *testing.T) {
	env := newTestEnv(t)
	env.extractPHPSource(`<?php
function main() {
    nonExistent();
}
`, "main.php")

	env.resolvePHP()

	refs, err := env.store.ReferencesByName("nonExistent")
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, r := range refs {
		resolved, err := env.store.ResolvedReferencesByRef(r.ID)
		require.NoError(t, err)
		assert.Empty(t, resolved, "nonExistent should not be resolved")
	}
}

func TestPHPResolve_TraitInclusion(t *testing.T) {
	env := newTestEnv(t)
	env.extractPHPSource(`<?php
trait Loggable {
    public function log() {}
}

class Service {
    use Loggable;

    public function run() {
        $this->log();
    }
}
`, "service.php")

	env.resolvePHP()

	// Trait usage should create an implementation record
	traitSym := findSymbolByName(t, env.store, "Loggable", "trait")
	require.NotNil(t, traitSym, "expected Loggable trait symbol")

	impls, err := env.store.ImplementationsByInterface(traitSym.ID)
	require.NoError(t, err)
	require.Len(t, impls, 1, "expected 1 implementation of Loggable trait")

	typeSym := findSymbolByID(t, env.store, impls[0].TypeSymbolID)
	assert.Equal(t, "Service", typeSym.Name)
	assert.Equal(t, "trait", impls[0].Kind)

	// The log() call inside Service.run() should resolve to Loggable.log
	refs, err := env.store.ReferencesByName("log")
	require.NoError(t, err)
	var callRef *store.Reference
	for _, r := range refs {
		if r.Context == "call" {
			callRef = r
			break
		}
	}
	require.NotNil(t, callRef, "expected call reference to log")

	resolved, err := env.store.ResolvedReferencesByRef(callRef.ID)
	require.NoError(t, err)
	require.NotEmpty(t, resolved, "expected log() to resolve via trait inclusion")

	targetSym := findSymbolByID(t, env.store, resolved[0].TargetSymbolID)
	assert.Equal(t, "log", targetSym.Name)
	assert.Equal(t, "method", targetSym.Kind)
}

func TestPHPResolve_ImplementsInterface(t *testing.T) {
	env := newTestEnv(t)
	env.extractPHPSource(`<?php
interface Cacheable {
    public function getKey();
}

class Product implements Cacheable {
    public function getKey() {
        return "product";
    }
}
`, "product.php")

	env.resolvePHP()

	cacheableSym := findSymbolByName(t, env.store, "Cacheable", "interface")
	require.NotNil(t, cacheableSym, "expected Cacheable interface symbol")

	impls, err := env.store.ImplementationsByInterface(cacheableSym.ID)
	require.NoError(t, err)
	require.Len(t, impls, 1, "expected 1 implementation of Cacheable")

	typeSym := findSymbolByID(t, env.store, impls[0].TypeSymbolID)
	assert.Equal(t, "Product", typeSym.Name)
	assert.Equal(t, "explicit", impls[0].Kind)
}
