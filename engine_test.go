package canopy

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { e.Close() })
	return e
}

// testFileHash computes the same SHA256 hex hash the engine uses.
func testFileHash(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func TestNew_CreatesStoreAndRuntime(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, t.TempDir())
	require.NoError(t, err)
	defer e.Close()

	require.NotNil(t, e.store)
	require.NotNil(t, e.runtime)
	require.NotNil(t, e.Store())

	// Verify the DB is usable (migration ran).
	_, err = e.Store().InsertFile(&store.File{
		Path: "/tmp/test.go", Language: "go", Hash: "abc", LastIndexed: time.Now(),
	})
	require.NoError(t, err)
}

func TestNew_InvalidPath(t *testing.T) {
	_, err := New("/nonexistent/dir/db.sqlite", t.TempDir())
	require.Error(t, err)
}

func TestClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, t.TempDir())
	require.NoError(t, err)
	require.NoError(t, e.Close())
}

func TestWithLanguages(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, t.TempDir(), WithLanguages("go", "python"))
	require.NoError(t, err)
	defer e.Close()

	assert.True(t, e.languages["go"])
	assert.True(t, e.languages["python"])
	assert.False(t, e.languages["rust"])
}

func TestQuery_ReturnsQueryBuilder(t *testing.T) {
	e := newTestEngine(t)
	q := e.Query()
	require.NotNil(t, q)
}

func TestIndexFiles_SkipsUnsupportedExtensions(t *testing.T) {
	e := newTestEngine(t)

	tmp := filepath.Join(t.TempDir(), "readme.txt")
	require.NoError(t, os.WriteFile(tmp, []byte("hello"), 0644))

	err := e.IndexFiles(context.Background(), []string{tmp})
	require.NoError(t, err)

	f, err := e.Store().FileByPath(tmp)
	require.NoError(t, err)
	assert.Nil(t, f)
}

func TestIndexFiles_SkipsFilteredLanguages(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, t.TempDir(), WithLanguages("python"))
	require.NoError(t, err)
	defer e.Close()

	tmp := filepath.Join(t.TempDir(), "main.go")
	require.NoError(t, os.WriteFile(tmp, []byte("package main"), 0644))

	err = e.IndexFiles(context.Background(), []string{tmp})
	require.NoError(t, err)

	f, err := e.Store().FileByPath(tmp)
	require.NoError(t, err)
	assert.Nil(t, f)
}

func TestIndexFiles_SkipsUnchangedFiles(t *testing.T) {
	e := newTestEngine(t)

	tmp := filepath.Join(t.TempDir(), "main.go")
	content := []byte("package main")
	require.NoError(t, os.WriteFile(tmp, content, 0644))

	// Pre-insert with the correct hash.
	hash := testFileHash(content)
	_, err := e.Store().InsertFile(&store.File{
		Path: tmp, Language: "go", Hash: hash, LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	// Should skip — no extraction script error.
	err = e.IndexFiles(context.Background(), []string{tmp})
	require.NoError(t, err)
}

func TestIndexFiles_ReindexesChangedFiles(t *testing.T) {
	e := newTestEngine(t)

	tmp := filepath.Join(t.TempDir(), "main.go")
	require.NoError(t, os.WriteFile(tmp, []byte("package main"), 0644))

	// Insert with a stale hash.
	_, err := e.Store().InsertFile(&store.File{
		Path: tmp, Language: "go", Hash: "oldhash", LastIndexed: time.Now(),
	})
	require.NoError(t, err)

	// Should detect change and try extraction → fail on missing script.
	err = e.IndexFiles(context.Background(), []string{tmp})
	require.Error(t, err)
	require.Contains(t, err.Error(), "extraction script")
}

func TestIndexFiles_InsertsNewFile(t *testing.T) {
	e := newTestEngine(t)

	tmp := filepath.Join(t.TempDir(), "main.go")
	require.NoError(t, os.WriteFile(tmp, []byte("package main"), 0644))

	// New file → should try extraction → fail on missing script.
	err := e.IndexFiles(context.Background(), []string{tmp})
	require.Error(t, err)
	require.Contains(t, err.Error(), "extraction script")
}

func TestIndexDirectory_DiscoversGoFiles(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "pkg")
	require.NoError(t, os.MkdirAll(sub, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "util.go"), []byte("package pkg"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "readme.txt"), []byte("docs"), 0644))

	e := newTestEngine(t)

	err := e.IndexDirectory(context.Background(), root)
	// Should fail on extraction for the first .go file found.
	require.Error(t, err)
	require.Contains(t, err.Error(), "extraction script")
}

func TestIndexDirectory_SkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()
	hidden := filepath.Join(root, ".git")
	require.NoError(t, os.MkdirAll(hidden, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hidden, "config.go"), []byte("package git"), 0644))

	e := newTestEngine(t)

	// No supported files outside hidden dirs → no error.
	err := e.IndexDirectory(context.Background(), root)
	require.NoError(t, err)
}

func TestIndexDirectory_SkipsExcludedDirs(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"vendor", "node_modules", "__pycache__"} {
		d := filepath.Join(root, dir)
		require.NoError(t, os.MkdirAll(d, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(d, "lib.go"), []byte("package lib"), 0644))
	}

	e := newTestEngine(t)

	// All Go files are in excluded dirs → no error.
	err := e.IndexDirectory(context.Background(), root)
	require.NoError(t, err)
}

func TestResolve_NoFiles(t *testing.T) {
	e := newTestEngine(t)

	// No files → no languages → no-op.
	err := e.Resolve(context.Background())
	require.NoError(t, err)
}

func TestDistinctLanguages(t *testing.T) {
	e := newTestEngine(t)

	// Insert files for two languages.
	for _, f := range []*store.File{
		{Path: "/a.go", Language: "go", Hash: "a", LastIndexed: time.Now()},
		{Path: "/b.go", Language: "go", Hash: "b", LastIndexed: time.Now()},
		{Path: "/c.py", Language: "python", Hash: "c", LastIndexed: time.Now()},
	} {
		_, err := e.Store().InsertFile(f)
		require.NoError(t, err)
	}

	langs, err := e.distinctLanguages()
	require.NoError(t, err)
	assert.Len(t, langs, 2)
	assert.Contains(t, langs, "go")
	assert.Contains(t, langs, "python")
}

// --- WithScriptsFS tests ---

func TestNew_WithScriptsFS_RunsExtraction(t *testing.T) {
	// Use os.DirFS to load actual scripts from the repo's scripts/ directory.
	// This proves WithScriptsFS works end-to-end with real extraction scripts.
	scriptsFS := os.DirFS("scripts")

	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, "", WithScriptsFS(scriptsFS))
	require.NoError(t, err)
	defer e.Close()

	// Write a simple Go file to index.
	goFile := filepath.Join(t.TempDir(), "main.go")
	require.NoError(t, os.WriteFile(goFile, []byte(`package main

func Hello() string {
	return "hello"
}
`), 0644))

	err = e.IndexFiles(context.Background(), []string{goFile})
	require.NoError(t, err)

	// Verify extraction produced symbols.
	f, err := e.Store().FileByPath(goFile)
	require.NoError(t, err)
	require.NotNil(t, f)

	syms, err := e.Store().SymbolsByFile(f.ID)
	require.NoError(t, err)
	require.NotEmpty(t, syms, "expected symbols from extraction")

	// Should have at least a package symbol and a function symbol.
	var kinds []string
	for _, sym := range syms {
		kinds = append(kinds, sym.Kind)
	}
	assert.Contains(t, kinds, "package")
	assert.Contains(t, kinds, "function")
}

func TestNew_WithScriptsFS_TakesPrecedenceOverScriptsDir(t *testing.T) {
	// Even if scriptsDir is set to a nonexistent path, WithScriptsFS
	// should take precedence and scripts should load from the FS.
	scriptsFS := os.DirFS("scripts")

	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, "/nonexistent/scripts/path", WithScriptsFS(scriptsFS))
	require.NoError(t, err)
	defer e.Close()

	goFile := filepath.Join(t.TempDir(), "main.go")
	require.NoError(t, os.WriteFile(goFile, []byte(`package main

func Add(a, b int) int { return a + b }
`), 0644))

	err = e.IndexFiles(context.Background(), []string{goFile})
	require.NoError(t, err)

	f, err := e.Store().FileByPath(goFile)
	require.NoError(t, err)
	require.NotNil(t, f)

	syms, err := e.Store().SymbolsByFile(f.ID)
	require.NoError(t, err)
	require.NotEmpty(t, syms)
}

func TestNew_WithoutScriptsFS_PreservesExistingBehavior(t *testing.T) {
	// Without WithScriptsFS, should use scriptsDir (disk-based loading).
	dbPath := filepath.Join(t.TempDir(), "test.db")
	e, err := New(dbPath, "scripts")
	require.NoError(t, err)
	defer e.Close()

	goFile := filepath.Join(t.TempDir(), "main.go")
	require.NoError(t, os.WriteFile(goFile, []byte(`package main

var x = 1
`), 0644))

	err = e.IndexFiles(context.Background(), []string{goFile})
	require.NoError(t, err)

	f, err := e.Store().FileByPath(goFile)
	require.NoError(t, err)
	require.NotNil(t, f)
}

func TestNewQueryBuilder(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.Migrate())

	qb := NewQueryBuilder(s)
	require.NotNil(t, qb)

	// Should be able to query (returns nil for nonexistent file, no error).
	sym, err := qb.SymbolAt("nonexistent.go", 0, 0)
	require.NoError(t, err)
	assert.Nil(t, sym)

	// Discovery queries should also work on an empty DB.
	result, err := qb.Symbols(SymbolFilter{}, Sort{}, Pagination{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalCount)
	assert.Empty(t, result.Items)
}
