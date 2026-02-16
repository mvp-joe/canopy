package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexFixture builds the binary and indexes a Go fixture, returning the binary
// path and fixture directory. The fixture is ready for query commands.
func indexFixture(t *testing.T) (bin, fixtureDir, dbPath string) {
	t.Helper()
	bin = buildBinary(t)
	fixtureDir = createGoFixture(t)
	dbPath = filepath.Join(fixtureDir, ".canopy", "index.db")

	cmd := exec.Command(bin, "index", fixtureDir)
	cmd.Dir = fixtureDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "index failed: %s", string(out))
	require.FileExists(t, dbPath)

	return bin, fixtureDir, dbPath
}

// runQuery executes a canopy query command and returns the parsed CLIResult.
func runQuery(t *testing.T, bin, fixtureDir string, args ...string) map[string]any {
	t.Helper()
	fullArgs := append([]string{"query"}, args...)
	cmd := exec.Command(bin, fullArgs...)
	cmd.Dir = fixtureDir
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	stdout, err := cmd.Output()
	// Allow non-zero exit for error cases, but we always expect JSON on stdout.
	if err != nil && len(stdout) == 0 {
		t.Fatalf("query command failed with no output: %v", err)
	}

	var result map[string]any
	require.NoError(t, json.Unmarshal(stdout, &result), "invalid JSON output: %s", string(stdout))
	return result
}

func TestQuery_SymbolAt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// The fixture has `func main()` starting at line 4 (0-based).
	// Line 4 col 5 should be inside the "main" function name.
	result := runQuery(t, bin, fixtureDir, "symbol-at", "main.go", "4", "5")

	assert.Equal(t, "symbol-at", result["command"])
	assert.NotNil(t, result["results"], "should find a symbol")
	assert.Empty(t, result["error"])

	results, ok := result["results"].(map[string]any)
	require.True(t, ok, "results should be a symbol object")
	assert.Equal(t, "main", results["name"])
	assert.NotZero(t, results["id"])
}

func TestQuery_SymbolAt_NoSymbol(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Position with no symbol (large line number).
	result := runQuery(t, bin, fixtureDir, "symbol-at", "main.go", "99999", "0")

	assert.Equal(t, "symbol-at", result["command"])
	assert.Nil(t, result["results"], "should return null for no symbol")
}

func TestQuery_Symbols_KindFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	result := runQuery(t, bin, fixtureDir, "symbols", "--kind", "function")

	assert.Equal(t, "symbols", result["command"])
	assert.NotNil(t, result["total_count"])
	assert.Empty(t, result["error"])

	results, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")
	assert.GreaterOrEqual(t, len(results), 2, "should have at least main and helper functions")

	// Every result should be a function.
	for _, r := range results {
		sym := r.(map[string]any)
		assert.Equal(t, "function", sym["kind"])
	}
}

func TestQuery_Search(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	result := runQuery(t, bin, fixtureDir, "search", "hel*")

	assert.Equal(t, "search", result["command"])
	assert.NotNil(t, result["total_count"])

	results, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")
	assert.GreaterOrEqual(t, len(results), 1, "should find 'helper'")

	found := false
	for _, r := range results {
		sym := r.(map[string]any)
		if sym["name"] == "helper" {
			found = true
		}
	}
	assert.True(t, found, "should find the 'helper' function")
}

func TestQuery_Files(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	result := runQuery(t, bin, fixtureDir, "files", "--language", "go")

	assert.Equal(t, "files", result["command"])
	assert.NotNil(t, result["total_count"])

	results, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")
	assert.Equal(t, 1, len(results), "should have 1 Go file")

	f := results[0].(map[string]any)
	assert.Equal(t, "go", f["language"])
	assert.Contains(t, f["path"], "main.go")
}

func TestQuery_Summary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	result := runQuery(t, bin, fixtureDir, "summary")

	assert.Equal(t, "summary", result["command"])
	assert.Empty(t, result["error"])

	summary, ok := result["results"].(map[string]any)
	require.True(t, ok, "results should be a summary object")
	assert.NotNil(t, summary["languages"])
	assert.NotNil(t, summary["package_count"])
}

func TestQuery_Deps(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	result := runQuery(t, bin, fixtureDir, "deps", "main.go")

	assert.Equal(t, "deps", result["command"])
	assert.Empty(t, result["error"])

	results, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")
	assert.GreaterOrEqual(t, len(results), 1, "should have at least the 'fmt' import")

	found := false
	for _, r := range results {
		imp := r.(map[string]any)
		if imp["source"] == "fmt" {
			found = true
		}
	}
	assert.True(t, found, "should find the 'fmt' import")
}

func TestQuery_Dependents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	result := runQuery(t, bin, fixtureDir, "dependents", "fmt")

	assert.Equal(t, "dependents", result["command"])
	assert.Empty(t, result["error"])

	results, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")
	assert.GreaterOrEqual(t, len(results), 1, "main.go imports fmt")
}

func TestQuery_SymbolFlag_References(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// First, get a symbol ID via symbol-at.
	symbolResult := runQuery(t, bin, fixtureDir, "symbol-at", "main.go", "4", "5")
	require.Equal(t, "symbol-at", symbolResult["command"])
	require.NotNil(t, symbolResult["results"])

	sym := symbolResult["results"].(map[string]any)
	// json.Unmarshal produces float64 for numbers.
	symbolID := int64(sym["id"].(float64))
	require.NotZero(t, symbolID)

	// Now use --symbol flag with references.
	refResult := runQuery(t, bin, fixtureDir, "references", "--symbol", formatInt64(symbolID))

	assert.Equal(t, "references", refResult["command"])
	assert.Empty(t, refResult["error"])
	// Results should be an array (may be empty if no references).
	_, ok := refResult["results"].([]any)
	assert.True(t, ok, "results should be an array")
}

func TestQuery_SymbolFlag_Callers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Get a symbol ID for 'main' function.
	symbolResult := runQuery(t, bin, fixtureDir, "symbol-at", "main.go", "4", "5")
	sym := symbolResult["results"].(map[string]any)
	symbolID := int64(sym["id"].(float64))

	// Query callers -- may be empty for main but should not error.
	result := runQuery(t, bin, fixtureDir, "callers", "--symbol", formatInt64(symbolID))

	assert.Equal(t, "callers", result["command"])
	assert.Empty(t, result["error"])
	_, ok := result["results"].([]any)
	assert.True(t, ok, "results should be an array")
}

func TestQuery_SymbolFlag_Callees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Get a symbol ID for 'main' function.
	symbolResult := runQuery(t, bin, fixtureDir, "symbol-at", "main.go", "4", "5")
	sym := symbolResult["results"].(map[string]any)
	symbolID := int64(sym["id"].(float64))

	// Query callees.
	result := runQuery(t, bin, fixtureDir, "callees", "--symbol", formatInt64(symbolID))

	assert.Equal(t, "callees", result["command"])
	assert.Empty(t, result["error"])
	_, ok := result["results"].([]any)
	assert.True(t, ok, "results should be an array")
}

func TestQuery_SymbolFlag_Implementations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Get a symbol ID. Implementations on a function will be empty but should not error.
	symbolResult := runQuery(t, bin, fixtureDir, "symbol-at", "main.go", "4", "5")
	sym := symbolResult["results"].(map[string]any)
	symbolID := int64(sym["id"].(float64))

	result := runQuery(t, bin, fixtureDir, "implementations", "--symbol", formatInt64(symbolID))

	assert.Equal(t, "implementations", result["command"])
	assert.Empty(t, result["error"])
	_, ok := result["results"].([]any)
	assert.True(t, ok, "results should be an array")
}

func TestQuery_ErrorCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)
	// Do NOT index -- so the DB won't exist.

	t.Run("no database", func(t *testing.T) {
		cmd := exec.Command(bin, "query", "symbols")
		cmd.Dir = fixture
		stdout, _ := cmd.Output()

		var result map[string]any
		if len(stdout) > 0 {
			require.NoError(t, json.Unmarshal(stdout, &result))
			assert.NotEmpty(t, result["error"], "should have an error about missing database")
		}
	})

	// Index for the remaining error tests.
	idxCmd := exec.Command(bin, "index", fixture)
	idxCmd.Dir = fixture
	out, err := idxCmd.CombinedOutput()
	require.NoError(t, err, "index failed: %s", string(out))

	t.Run("non-numeric line", func(t *testing.T) {
		cmd := exec.Command(bin, "query", "symbol-at", "main.go", "abc", "5")
		cmd.Dir = fixture
		stdout, _ := cmd.Output()

		var result map[string]any
		if len(stdout) > 0 {
			require.NoError(t, json.Unmarshal(stdout, &result))
			assert.Contains(t, result["error"], "invalid line")
		}
	})

	t.Run("references without args or symbol", func(t *testing.T) {
		cmd := exec.Command(bin, "query", "references")
		cmd.Dir = fixture
		stdout, _ := cmd.Output()

		var result map[string]any
		if len(stdout) > 0 {
			require.NoError(t, json.Unmarshal(stdout, &result))
			assert.NotEmpty(t, result["error"])
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		cmd := exec.Command(bin, "--format", "xml", "query", "symbols")
		cmd.Dir = fixture
		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf
		err := cmd.Run()
		require.Error(t, err, "should fail with invalid format")
		assert.Contains(t, stderrBuf.String(), "invalid format", "error should mention invalid format")
	})

	t.Run("callers with non-existent symbol", func(t *testing.T) {
		cmd := exec.Command(bin, "query", "callers", "--symbol", "999999")
		cmd.Dir = fixture
		stdout, _ := cmd.Output()

		var result map[string]any
		if len(stdout) > 0 {
			require.NoError(t, json.Unmarshal(stdout, &result))
			assert.Equal(t, "callers", result["command"])
			// Should return an empty array, not crash.
			results, ok := result["results"].([]any)
			if ok {
				assert.Empty(t, results, "non-existent symbol should return empty results")
			}
		}
	})

	t.Run("definition with no args", func(t *testing.T) {
		cmd := exec.Command(bin, "query", "definition")
		cmd.Dir = fixture
		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf
		err := cmd.Run()
		// Cobra enforces ExactArgs(3), so this should fail.
		require.Error(t, err, "definition without args should fail")
	})
}

func TestQuery_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Query with limit 1.
	result := runQuery(t, bin, fixtureDir, "symbols", "--limit", "1")

	assert.Equal(t, "symbols", result["command"])
	results, ok := result["results"].([]any)
	require.True(t, ok)
	assert.LessOrEqual(t, len(results), 1, "limit 1 should return at most 1 result")

	// total_count should still reflect the full count.
	tc := result["total_count"].(float64)
	assert.GreaterOrEqual(t, int(tc), 1)
}

func TestQuery_Definition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Query definition at a reference location (e.g., "fmt" in fmt.Println).
	// Line 5, col 1 should be within "fmt" in `fmt.Println("hello")`.
	result := runQuery(t, bin, fixtureDir, "definition", "main.go", "5", "1")

	assert.Equal(t, "definition", result["command"])
	assert.Empty(t, result["error"])
	// Results is an array of locations (may be empty if resolution didn't find it).
	_, ok := result["results"].([]any)
	assert.True(t, ok, "results should be an array")
}

// formatInt64 formats an int64 as a string for command-line args.
func formatInt64(n int64) string {
	return fmt.Sprintf("%d", n)
}

// runQueryRaw executes a canopy query and returns raw stdout/stderr strings.
func runQueryRaw(t *testing.T, bin, fixtureDir string, args ...string) (stdout, stderr string) {
	t.Helper()
	fullArgs := append([]string{"query"}, args...)
	cmd := exec.Command(bin, fullArgs...)
	cmd.Dir = fixtureDir
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	_ = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String()
}

func TestQuery_FormatText_SymbolAt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	stdout, _ := runQueryRaw(t, bin, fixtureDir, "--format", "text", "symbol-at", "main.go", "4", "5")

	// Should NOT be JSON.
	assert.False(t, strings.HasPrefix(strings.TrimSpace(stdout), "{"), "text format should not produce JSON")

	// Should contain tabular output with header and the symbol.
	assert.Contains(t, stdout, "ID")
	assert.Contains(t, stdout, "NAME")
	assert.Contains(t, stdout, "main")
}

func TestQuery_FormatText_Symbols(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	stdout, _ := runQueryRaw(t, bin, fixtureDir, "--format", "text", "symbols", "--kind", "function")

	// Should have column headers.
	assert.Contains(t, stdout, "ID")
	assert.Contains(t, stdout, "NAME")
	assert.Contains(t, stdout, "KIND")
	assert.Contains(t, stdout, "VISIBILITY")

	// Should list the functions.
	assert.Contains(t, stdout, "main")
	assert.Contains(t, stdout, "helper")
	assert.Contains(t, stdout, "function")
}

func TestQuery_FormatText_Summary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	stdout, _ := runQueryRaw(t, bin, fixtureDir, "--format", "text", "summary")

	assert.Contains(t, stdout, "Project Summary")
	assert.Contains(t, stdout, "Packages:")
	assert.Contains(t, stdout, "Languages:")
	assert.Contains(t, stdout, "go:")
}

func TestQuery_FormatJSON_ValidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	stdout, _ := runQueryRaw(t, bin, fixtureDir, "--format", "json", "symbols")

	// Should be valid JSON.
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), "explicit --format json should produce valid JSON")
	assert.Equal(t, "symbols", result["command"])
}

func TestQuery_FormatText_ErrorGoesToStderr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin := buildBinary(t)
	fixture := createGoFixture(t)
	// Do NOT index, so the DB won't exist.

	stdout, stderr := runQueryRaw(t, bin, fixture, "--format", "text", "symbols")

	// Error should be on stderr, not stdout.
	assert.Empty(t, stdout, "text format errors should not write to stdout")
	assert.Contains(t, stderr, "Error:", "text format errors should go to stderr")
}

func TestQuery_JSON_SymbolIDsInLocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Get a symbol ID for 'helper' function and query references.
	// References should return locations with symbol_id populated.
	symbolResult := runQuery(t, bin, fixtureDir, "symbol-at", "main.go", "8", "5")
	require.NotNil(t, symbolResult["results"])
	sym := symbolResult["results"].(map[string]any)
	symbolID := int64(sym["id"].(float64))

	result := runQuery(t, bin, fixtureDir, "references", "--symbol", formatInt64(symbolID))
	assert.Equal(t, "references", result["command"])

	refs, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")

	// Each location in the references result should have a symbol_id field.
	for i, ref := range refs {
		loc := ref.(map[string]any)
		assert.Contains(t, loc, "symbol_id", "location %d should have symbol_id for LLM chaining", i)
	}
}

func TestQuery_Packages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	result := runQuery(t, bin, fixtureDir, "packages")

	assert.Equal(t, "packages", result["command"])
	assert.NotNil(t, result["total_count"], "should have total_count")
	assert.Empty(t, result["error"])

	results, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")
	assert.GreaterOrEqual(t, len(results), 1, "fixture has 'package main' so at least one package expected")
}

func TestQuery_PackageSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// First get a package from "canopy query packages".
	pkgResult := runQuery(t, bin, fixtureDir, "packages")
	require.NotNil(t, pkgResult["results"])

	packages, ok := pkgResult["results"].([]any)
	require.True(t, ok, "packages results should be an array")
	require.GreaterOrEqual(t, len(packages), 1, "need at least one package")

	// Extract the symbol ID from the first package.
	firstPkg := packages[0].(map[string]any)
	pkgID := int64(firstPkg["id"].(float64))
	require.NotZero(t, pkgID)

	// Query package-summary using the symbol ID.
	result := runQuery(t, bin, fixtureDir, "package-summary", formatInt64(pkgID))

	assert.Equal(t, "package-summary", result["command"])
	assert.Empty(t, result["error"])
	require.NotNil(t, result["results"])

	summary, ok := result["results"].(map[string]any)
	require.True(t, ok, "results should be a summary object")
	assert.NotNil(t, summary["symbol"], "should have symbol info")
	assert.NotNil(t, summary["path"], "should have path")
	assert.NotNil(t, summary["file_count"], "should have file_count")
}

func TestQuery_EndToEndChaining(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir, _ := indexFixture(t)

	// Step 1: Search for symbols matching "hel*".
	searchResult := runQuery(t, bin, fixtureDir, "search", "hel*")
	assert.Equal(t, "search", searchResult["command"])
	assert.Empty(t, searchResult["error"])

	results, ok := searchResult["results"].([]any)
	require.True(t, ok, "search results should be an array")
	require.GreaterOrEqual(t, len(results), 1, "should find 'helper'")

	// Step 2: Extract a symbol ID from the search results.
	firstSym := results[0].(map[string]any)
	symbolID := int64(firstSym["id"].(float64))
	require.NotZero(t, symbolID)

	// Step 3: Query references for the extracted symbol ID.
	refResult := runQuery(t, bin, fixtureDir, "references", "--symbol", formatInt64(symbolID))
	assert.Equal(t, "references", refResult["command"])
	assert.Empty(t, refResult["error"])

	refResults, ok := refResult["results"].([]any)
	require.True(t, ok, "references results should be an array")

	// Each reference location should have file and position info.
	for _, ref := range refResults {
		loc := ref.(map[string]any)
		assert.NotEmpty(t, loc["file"], "location should have a file")
		assert.NotNil(t, loc["start_line"], "location should have start_line")
		assert.NotNil(t, loc["start_col"], "location should have start_col")
	}
}

// createMethodCallFixture creates a fixture with two Go files: types.go defines
// a struct with a method, and main.go calls that method via a receiver.
func createMethodCallFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	typesFile := filepath.Join(dir, "types.go")
	typesSrc := `package main

type Server struct {
	Name string
}

func (s *Server) Handle(req string) string {
	return "ok"
}

func NewServer(name string) *Server {
	return &Server{Name: name}
}
`
	require.NoError(t, os.WriteFile(typesFile, []byte(typesSrc), 0o644))

	mainFile := filepath.Join(dir, "main.go")
	mainSrc := `package main

import "fmt"

func main() {
	s := NewServer("test")
	result := s.Handle("hello")
	fmt.Println(result)
}
`
	require.NoError(t, os.WriteFile(mainFile, []byte(mainSrc), 0o644))
	return dir
}

// indexMethodCallFixture builds and indexes the method-call fixture.
func indexMethodCallFixture(t *testing.T) (bin, fixtureDir string) {
	t.Helper()
	bin = buildBinary(t)
	fixtureDir = createMethodCallFixture(t)

	cmd := exec.Command(bin, "index", fixtureDir)
	cmd.Dir = fixtureDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "index failed: %s", string(out))
	require.FileExists(t, filepath.Join(fixtureDir, ".canopy", "index.db"))

	return bin, fixtureDir
}

func TestQuery_Callers_MethodCall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	bin, fixtureDir := indexMethodCallFixture(t)

	// Find the Handle method symbol. In types.go:
	// line 6: func (s *Server) Handle(req string) string {
	// "Handle" starts at col 19 (0-based).
	symbolResult := runQuery(t, bin, fixtureDir, "symbol-at", "types.go", "6", "19")
	require.NotNil(t, symbolResult["results"], "should find Handle method")
	sym := symbolResult["results"].(map[string]any)
	assert.Equal(t, "Handle", sym["name"])
	symbolID := int64(sym["id"].(float64))

	// Query callers of Handle — main() calls s.Handle("hello").
	result := runQuery(t, bin, fixtureDir, "callers", "--symbol", formatInt64(symbolID))
	assert.Equal(t, "callers", result["command"])
	assert.Empty(t, result["error"])

	results, ok := result["results"].([]any)
	require.True(t, ok, "results should be an array")
	assert.GreaterOrEqual(t, len(results), 1, "main() calls Handle, should have at least 1 caller")

	// Verify one of the callers is the main function.
	found := false
	for _, r := range results {
		edge := r.(map[string]any)
		if edge["caller_name"] == "main" {
			found = true
		}
	}
	assert.True(t, found, "main should be listed as a caller of Handle")
}
