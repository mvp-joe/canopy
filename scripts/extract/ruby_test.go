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

// extractRubySource writes Ruby source to a temp file, inserts a file record,
// and runs the Ruby extraction script. Returns the file ID.
func (e *testEnv) extractRubySource(src string) int64 {
	e.t.Helper()

	dir := e.t.TempDir()
	rbFile := filepath.Join(dir, "test.rb")
	require.NoError(e.t, os.WriteFile(rbFile, []byte(src), 0644))

	fileID, err := e.store.InsertFile(&store.File{
		Path:     rbFile,
		Language: "ruby",
	})
	require.NoError(e.t, err)

	extras := map[string]any{
		"file_path": rbFile,
		"file_id":   fileID,
	}
	err = e.rt.RunScript(context.Background(), filepath.Join("extract", "ruby.risor"), extras)
	require.NoError(e.t, err)

	return fileID
}

func TestRuby_ClassWithMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class User
  def initialize(name)
    @name = name
  end

  def greet
    "Hello, #{@name}"
  end
end
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

	var methods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}
	require.Len(t, methods, 2)

	methodNames := map[string]bool{}
	for _, m := range methods {
		methodNames[m.Name] = true
		require.NotNil(t, m.ParentSymbolID)
		assert.Equal(t, classSym.ID, *m.ParentSymbolID)
	}
	assert.True(t, methodNames["initialize"])
	assert.True(t, methodNames["greet"])
}

func TestRuby_ModuleWithMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`module Serializable
  def serialize
    to_json
  end

  def deserialize(data)
    from_json(data)
  end
end
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var modSym *store.Symbol
	for _, s := range syms {
		if s.Kind == "module" {
			modSym = s
			break
		}
	}
	require.NotNil(t, modSym, "expected module symbol")
	assert.Equal(t, "Serializable", modSym.Name)

	var methods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}
	require.Len(t, methods, 2)

	for _, m := range methods {
		require.NotNil(t, m.ParentSymbolID)
		assert.Equal(t, modSym.ID, *m.ParentSymbolID)
	}
}

func TestRuby_IncludeExtendMixins(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`module Serializable; end
module ClassMethods; end
module Auditable; end

class User
  include Serializable
  extend ClassMethods
  prepend Auditable
end
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	for _, s := range syms {
		if s.Name == "User" && s.Kind == "class" {
			classSym = s
			break
		}
	}
	require.NotNil(t, classSym)

	members, err := env.store.TypeMembers(classSym.ID)
	require.NoError(t, err)

	embeddedNames := map[string]bool{}
	for _, m := range members {
		if m.Kind == "embedded" {
			embeddedNames[m.Name] = true
		}
	}
	require.Len(t, embeddedNames, 3)
	assert.True(t, embeddedNames["Serializable"])
	assert.True(t, embeddedNames["ClassMethods"])
	assert.True(t, embeddedNames["Auditable"])
}

func TestRuby_RequireImports(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`require 'json'
require 'net/http'
require_relative 'helpers'
`)
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 3)

	impBySource := map[string]*store.Import{}
	for _, imp := range imports {
		impBySource[imp.Source] = imp
	}

	jsonImp := impBySource["json"]
	require.NotNil(t, jsonImp)
	assert.Equal(t, "module", jsonImp.Kind)
	require.NotNil(t, jsonImp.ImportedName)
	assert.Equal(t, "json", *jsonImp.ImportedName)

	httpImp := impBySource["net/http"]
	require.NotNil(t, httpImp)
	assert.Equal(t, "module", httpImp.Kind)
	require.NotNil(t, httpImp.ImportedName)
	assert.Equal(t, "http", *httpImp.ImportedName)

	helpersImp := impBySource["helpers"]
	require.NotNil(t, helpersImp)
	assert.Equal(t, "relative", helpersImp.Kind)
}

func TestRuby_ScopeTree(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`module MyApp
  class User
    def greet
      "hello"
    end
  end
end

def helper
  42
end
`)
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, s := range scopes {
		kinds[s.Kind]++
	}

	assert.Equal(t, 1, kinds["file"], "expected 1 file scope")
	assert.Equal(t, 1, kinds["module"], "expected 1 module scope")
	assert.Equal(t, 1, kinds["class"], "expected 1 class scope")
	assert.GreaterOrEqual(t, kinds["function"], 2, "expected at least 2 function scopes (method + top-level func)")

	// Verify file scope has no parent
	var fileScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "file" {
			fileScope = s
			break
		}
	}
	require.NotNil(t, fileScope)
	assert.Nil(t, fileScope.ParentScopeID)

	// Module scope should have file as parent
	var modScope *store.Scope
	for _, s := range scopes {
		if s.Kind == "module" {
			modScope = s
			break
		}
	}
	require.NotNil(t, modScope)
	require.NotNil(t, modScope.ParentScopeID)
	assert.Equal(t, fileScope.ID, *modScope.ParentScopeID)
}

func TestRuby_References_MethodCall(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class User
  def greet
    "hello"
  end
end

user = User.new("Alice")
name = user.greet
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var newRef, greetRef, userTypeRef *store.Reference
	for _, r := range refs {
		if r.Name == "new" && r.Context == "call" {
			newRef = r
		}
		if r.Name == "greet" && r.Context == "call" {
			greetRef = r
		}
		if r.Name == "User" && r.Context == "type_annotation" {
			userTypeRef = r
		}
	}
	require.NotNil(t, newRef, "expected call reference to new")
	require.NotNil(t, greetRef, "expected call reference to greet")
	require.NotNil(t, userTypeRef, "expected type_annotation reference to User")
}

func TestRuby_FunctionParametersWithDefaults(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class User
  def initialize(name, email, age = 25)
    @name = name
  end
end
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var initSym *store.Symbol
	for _, s := range syms {
		if s.Name == "initialize" {
			initSym = s
			break
		}
	}
	require.NotNil(t, initSym)

	params, err := env.store.FunctionParams(initSym.ID)
	require.NoError(t, err)
	require.Len(t, params, 3)

	assert.Equal(t, "name", params[0].Name)
	assert.False(t, params[0].HasDefault)
	assert.Equal(t, "email", params[1].Name)
	assert.False(t, params[1].HasDefault)
	assert.Equal(t, "age", params[2].Name)
	assert.True(t, params[2].HasDefault)
	assert.Equal(t, "25", params[2].DefaultExpr)
}

func TestRuby_ClassMethods(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class Counter
  @@count = 0

  def self.count
    @@count
  end

  def self.reset
    @@count = 0
  end
end
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	var classSym *store.Symbol
	var classMethods []*store.Symbol
	for _, s := range syms {
		if s.Kind == "class" {
			classSym = s
		}
		if s.Kind == "method" {
			classMethods = append(classMethods, s)
		}
	}
	require.NotNil(t, classSym)
	require.Len(t, classMethods, 2)

	names := map[string]bool{}
	for _, m := range classMethods {
		names[m.Name] = true
		require.NotNil(t, m.ParentSymbolID)
		assert.Equal(t, classSym.ID, *m.ParentSymbolID)
	}
	assert.True(t, names["count"])
	assert.True(t, names["reset"])
}

func TestRuby_AttrAccessors(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class User
  attr_reader :name, :email
  attr_accessor :age
  attr_writer :password
end
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

	propNames := map[string]bool{}
	for _, m := range members {
		if m.Kind == "property" {
			propNames[m.Name] = true
			assert.Equal(t, "public", m.Visibility)
		}
	}
	require.Len(t, propNames, 4)
	assert.True(t, propNames["name"])
	assert.True(t, propNames["email"])
	assert.True(t, propNames["age"])
	assert.True(t, propNames["password"])
}

func TestRuby_Constants(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class Config
  VERSION = '1.0'
  MAX_RETRIES = 3
end
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

	constNames := map[string]bool{}
	for _, m := range members {
		if m.Kind == "constant" {
			constNames[m.Name] = true
		}
	}
	require.Len(t, constNames, 2)
	assert.True(t, constNames["VERSION"])
	assert.True(t, constNames["MAX_RETRIES"])
}

func TestRuby_Visibility(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class Example
  def pub_method
    "public"
  end

  protected

  def prot_method
    "protected"
  end

  private

  def priv_method
    "private"
  end
end
`)
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	methodVis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "method" {
			methodVis[s.Name] = s.Visibility
		}
	}
	assert.Equal(t, "public", methodVis["pub_method"])
	assert.Equal(t, "protected", methodVis["prot_method"])
	assert.Equal(t, "private", methodVis["priv_method"])
}

func TestRuby_SuperclassReference(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`class User; end

class Admin < User
end
`)
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)

	var superRef *store.Reference
	for _, r := range refs {
		if r.Name == "User" && r.Context == "type_annotation" {
			superRef = r
			break
		}
	}
	require.NotNil(t, superRef, "expected type_annotation reference to User superclass")
}

func TestRuby_TopLevelFunction(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`def helper(x, y)
  x + y
end
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

	params, err := env.store.FunctionParams(fnSym.ID)
	require.NoError(t, err)
	require.Len(t, params, 2)
	assert.Equal(t, "x", params[0].Name)
	assert.Equal(t, "y", params[1].Name)
}

func TestRuby_ComprehensiveFile(t *testing.T) {
	env := newTestEnv(t)
	fileID := env.extractRubySource(`require 'json'
require_relative 'helpers'

module Serializable
  def serialize
    to_json
  end
end

class User
  include Serializable

  attr_reader :name
  attr_accessor :age

  ROLE_ADMIN = 'admin'

  def initialize(name, age = 25)
    @name = name
    @age = age
  end

  def self.count
    0
  end

  def full_info
    "#{@name}"
  end

  private

  def secret
    42
  end
end

class Admin < User
  LEVEL = 'super'

  def admin?
    true
  end
end

def helper(x)
  x * 2
end
`)
	// Verify symbols
	syms, err := env.store.SymbolsByFile(fileID)
	require.NoError(t, err)

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	assert.Contains(t, kinds["module"], "Serializable")
	assert.Contains(t, kinds["class"], "User")
	assert.Contains(t, kinds["class"], "Admin")
	assert.Contains(t, kinds["function"], "helper")
	assert.GreaterOrEqual(t, len(kinds["method"]), 5)

	// Verify imports
	imports, err := env.store.ImportsByFile(fileID)
	require.NoError(t, err)
	require.Len(t, imports, 2)

	// Verify scopes
	scopes, err := env.store.ScopesByFile(fileID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(scopes), 5)

	// Verify references
	refs, err := env.store.ReferencesByFile(fileID)
	require.NoError(t, err)
	assert.Greater(t, len(refs), 0)

	// Verify User has Serializable embedded
	var userSym *store.Symbol
	for _, s := range syms {
		if s.Name == "User" && s.Kind == "class" {
			userSym = s
			break
		}
	}
	require.NotNil(t, userSym)
	members, err := env.store.TypeMembers(userSym.ID)
	require.NoError(t, err)

	var embedded *store.TypeMember
	for _, m := range members {
		if m.Kind == "embedded" {
			embedded = m
			break
		}
	}
	require.NotNil(t, embedded, "expected embedded Serializable")
	assert.Equal(t, "Serializable", embedded.Name)

	// Verify visibility tracking
	methodVis := map[string]string{}
	for _, s := range syms {
		if s.Kind == "method" {
			methodVis[s.Name] = s.Visibility
		}
	}
	assert.Equal(t, "private", methodVis["secret"])
}
