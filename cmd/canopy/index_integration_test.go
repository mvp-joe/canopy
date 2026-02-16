package main_test

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildBinary compiles the canopy binary and returns the path.
// The binary is placed in t.TempDir() so it's cleaned up automatically.
func buildBinary(t *testing.T) string {
	t.Helper()
	binName := "canopy"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	bin := filepath.Join(t.TempDir(), binName)
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(projectRoot(t), "cmd", "canopy")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", string(out))
	return bin
}

// projectRoot returns the root of the canopy project by walking up from
// the test file's directory to find go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "could not find project root")
		dir = parent
	}
}

// createGoFixture creates a temporary directory with a .git dir and a Go file.
// Returns the temp directory path.
func createGoFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .git directory so findRepoRoot works.
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	// Write a simple Go file.
	goFile := filepath.Join(dir, "main.go")
	src := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}

func helper() string {
	return "world"
}
`
	require.NoError(t, os.WriteFile(goFile, []byte(src), 0o644))
	return dir
}

// openDB opens the SQLite database at the given path for verification.
func openDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// fileCount returns the number of rows in the files table.
func fileCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM files").Scan(&count)
	require.NoError(t, err)
	return count
}

// fileCountForLanguage returns the number of files for a given language.
func fileCountForLanguage(t *testing.T, db *sql.DB, lang string) int {
	t.Helper()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM files WHERE language = ?", lang).Scan(&count)
	require.NoError(t, err)
	return count
}

// symbolCount returns the number of rows in the symbols table.
func symbolCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM symbols").Scan(&count)
	require.NoError(t, err)
	return count
}

func TestIndex_CreatesDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)

	cmd := exec.Command(bin, "index", fixture)
	cmd.Dir = fixture
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "index failed: %s", string(out))

	// Verify .canopy/index.db was created.
	dbPath := filepath.Join(fixture, ".canopy", "index.db")
	_, err = os.Stat(dbPath)
	require.NoError(t, err, ".canopy/index.db should exist")

	// Verify the database contains indexed data.
	db := openDB(t, dbPath)
	assert.Equal(t, 1, fileCount(t, db), "should have indexed 1 Go file")
	assert.Greater(t, symbolCount(t, db), 0, "should have extracted symbols")
}

func TestIndex_Force_ClearsAndReindexes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)
	dbPath := filepath.Join(fixture, ".canopy", "index.db")

	// First index.
	cmd := exec.Command(bin, "index", fixture)
	cmd.Dir = fixture
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "first index failed: %s", string(out))

	// Record initial file mod time.
	info1, err := os.Stat(dbPath)
	require.NoError(t, err)

	// Get initial symbol count.
	db1 := openDB(t, dbPath)
	initialSymbols := symbolCount(t, db1)
	db1.Close()

	// Add another Go file to the fixture.
	extraFile := filepath.Join(fixture, "extra.go")
	require.NoError(t, os.WriteFile(extraFile, []byte(`package main

func extra() int { return 42 }
`), 0o644))

	// Run with --force.
	cmd = exec.Command(bin, "index", "--force", fixture)
	cmd.Dir = fixture
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "force index failed: %s", string(out))

	// DB file should exist (recreated).
	info2, err := os.Stat(dbPath)
	require.NoError(t, err)

	// DB was recreated, so mod time should differ or size should change.
	// More reliable: check that we now have 2 files and more symbols.
	_ = info1
	_ = info2

	db2 := openDB(t, dbPath)
	assert.Equal(t, 2, fileCount(t, db2), "should have 2 files after force reindex")
	assert.Greater(t, symbolCount(t, db2), initialSymbols, "should have more symbols with extra file")
}

func TestIndex_LanguagesFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)

	// Add a Python file to the fixture.
	pyFile := filepath.Join(fixture, "script.py")
	require.NoError(t, os.WriteFile(pyFile, []byte(`def hello():
    print("hello")
`), 0o644))

	// Index with --languages=go (should skip Python).
	cmd := exec.Command(bin, "index", "--languages", "go", fixture)
	cmd.Dir = fixture
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "index with --languages failed: %s", string(out))

	dbPath := filepath.Join(fixture, ".canopy", "index.db")
	db := openDB(t, dbPath)
	assert.Equal(t, 1, fileCount(t, db), "should only have 1 file (Go)")
	assert.Equal(t, 1, fileCountForLanguage(t, db, "go"), "the file should be Go")
	assert.Equal(t, 0, fileCountForLanguage(t, db, "python"), "no Python files should be indexed")
}

func TestIndex_CustomDBPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)

	customDB := filepath.Join(t.TempDir(), "custom.db")

	cmd := exec.Command(bin, "index", "--db", customDB, fixture)
	cmd.Dir = fixture
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "index with --db failed: %s", string(out))

	// Custom DB should exist.
	_, err = os.Stat(customDB)
	require.NoError(t, err, "custom DB should exist at %s", customDB)

	// Default location should NOT exist.
	_, err = os.Stat(filepath.Join(fixture, ".canopy", "index.db"))
	assert.True(t, os.IsNotExist(err), ".canopy/index.db should not be created when --db is set")
}

func TestIndex_NonExistentDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)

	cmd := exec.Command(bin, "index", "/nonexistent/path/that/does/not/exist")
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "should fail for non-existent directory")
	assert.Contains(t, string(out), "not found", "error should mention 'not found'")
}

func TestIndex_StderrTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)

	cmd := exec.Command(bin, "index", fixture)
	cmd.Dir = fixture
	// Capture combined output (index only writes to stderr).
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "index failed: %s", string(out))

	output := string(out)
	assert.Contains(t, output, "Indexed")
	assert.Contains(t, output, "extract:")
	assert.Contains(t, output, "resolve:")
	assert.Contains(t, output, "Database:")
}

func TestIndex_ScriptsDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)

	// Locate the repo's scripts/ directory.
	scriptsDir := filepath.Join(projectRoot(t), "scripts")
	_, err := os.Stat(filepath.Join(scriptsDir, "extract", "go.risor"))
	require.NoError(t, err, "scripts/extract/go.risor should exist at repo root")

	// Run index with --scripts-dir pointing to disk scripts.
	cmd := exec.Command(bin, "index", "--scripts-dir", scriptsDir, fixture)
	cmd.Dir = fixture
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "index with --scripts-dir failed: %s", string(out))

	// Verify .canopy/index.db was created and contains symbols.
	dbPath := filepath.Join(fixture, ".canopy", "index.db")
	_, err = os.Stat(dbPath)
	require.NoError(t, err, ".canopy/index.db should exist")

	db := openDB(t, dbPath)
	assert.Equal(t, 1, fileCount(t, db), "should have indexed 1 Go file")
	assert.Greater(t, symbolCount(t, db), 0, "should have extracted symbols using disk scripts")
}

func TestIndex_IncrementalSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)
	dbPath := filepath.Join(fixture, ".canopy", "index.db")

	// First index.
	cmd := exec.Command(bin, "index", fixture)
	cmd.Dir = fixture
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "first index failed: %s", string(out))

	// Record the symbol count after first index.
	db1 := openDB(t, dbPath)
	firstSymbolCount := symbolCount(t, db1)
	firstFileCount := fileCount(t, db1)
	db1.Close()
	require.Greater(t, firstSymbolCount, 0, "first index should produce symbols")

	// Re-index without --force (no changes to files).
	cmd = exec.Command(bin, "index", fixture)
	cmd.Dir = fixture
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "second index failed: %s", string(out))

	// DB should still exist with equivalent content.
	db2 := openDB(t, dbPath)
	assert.Equal(t, firstFileCount, fileCount(t, db2), "file count should be the same after re-index")
	assert.Equal(t, firstSymbolCount, symbolCount(t, db2), "symbol count should be the same after re-index")
}
