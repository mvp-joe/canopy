package canopy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jward/canopy/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Golden test format.
type goldenFile struct {
	// Extraction assertions
	Definitions    []goldenDef           `json:"definitions,omitempty"`
	Imports        []goldenImport        `json:"imports,omitempty"`
	TypeMembers    []goldenTypeMember    `json:"type_members,omitempty"`
	FunctionParams []goldenFunctionParam `json:"function_params,omitempty"`
	TypeParams     []goldenTypeParam     `json:"type_params,omitempty"`

	// Resolution assertions
	References      []goldenRef  `json:"references,omitempty"`
	Implementations []goldenImpl `json:"implementations,omitempty"`
	Calls           []goldenCall `json:"calls,omitempty"`
}

type goldenDef struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type goldenRef struct {
	From goldenLoc    `json:"from"`
	To   goldenTarget `json:"to"`
}

type goldenLoc struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}

type goldenTarget struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type goldenImpl struct {
	Type      string `json:"type"`
	Interface string `json:"interface"`
}

type goldenCall struct {
	Caller string `json:"caller"`
	Callee string `json:"callee"`
}

type goldenImport struct {
	File         string `json:"file"`
	Source       string `json:"source"`
	ImportedName string `json:"imported_name,omitempty"`
	LocalAlias   string `json:"local_alias,omitempty"`
	Kind         string `json:"kind,omitempty"`
}

type goldenTypeMember struct {
	Parent     string `json:"parent"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	TypeExpr   string `json:"type_expr,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}

type goldenFunctionParam struct {
	Parent     string `json:"parent"`
	ParentFile string `json:"parent_file,omitempty"`
	Name       string `json:"name"`
	Ordinal    int    `json:"ordinal"`
	TypeExpr   string `json:"type_expr,omitempty"`
	IsReceiver bool   `json:"is_receiver,omitempty"`
	IsReturn   bool   `json:"is_return,omitempty"`
}

type goldenTypeParam struct {
	Parent      string `json:"parent"`
	Name        string `json:"name"`
	Ordinal     int    `json:"ordinal"`
	Constraints string `json:"constraints,omitempty"`
}

// TestGolden walks testdata/{language}/ directories and runs golden tests
// for all languages that have testdata.
func TestGolden(t *testing.T) {
	langDirs, err := os.ReadDir("testdata")
	if err != nil {
		t.Skip("no testdata directory found")
	}

	for _, langDir := range langDirs {
		if !langDir.IsDir() {
			continue
		}
		lang := langDir.Name()
		langRoot := filepath.Join("testdata", lang)
		levels, err := os.ReadDir(langRoot)
		if err != nil {
			continue
		}

		for _, level := range levels {
			if !level.IsDir() {
				continue
			}
			testDir := filepath.Join(langRoot, level.Name())
			goldenPath := filepath.Join(testDir, "golden.json")
			srcDir := filepath.Join(testDir, "src")

			if _, err := os.Stat(goldenPath); err != nil {
				continue
			}
			if _, err := os.Stat(srcDir); err != nil {
				continue
			}

			t.Run(lang+"/"+level.Name(), func(t *testing.T) {
				runGoldenTest(t, lang, srcDir, goldenPath)
			})
		}
	}
}

func runGoldenTest(t *testing.T, lang, srcDir, goldenPath string) {
	t.Helper()

	// Read golden file.
	goldenData, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	var golden goldenFile
	require.NoError(t, json.Unmarshal(goldenData, &golden))

	// Find the scripts dir relative to the test.
	scriptsDir := filepath.Join("scripts")
	if _, err := os.Stat(scriptsDir); err != nil {
		// Try from working directory
		wd, _ := os.Getwd()
		scriptsDir = filepath.Join(wd, "scripts")
	}

	// Create engine with temp DB.
	dbPath := filepath.Join(t.TempDir(), "golden.db")
	engine, err := New(dbPath, scriptsDir, WithLanguages(lang))
	require.NoError(t, err)
	defer engine.Close()

	// Discover and index all source files in src/.
	srcEntries, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	var paths []string
	for _, e := range srcEntries {
		if !e.IsDir() {
			paths = append(paths, filepath.Join(srcDir, e.Name()))
		}
	}
	require.NoError(t, engine.IndexFiles(context.Background(), paths))

	// Run resolution if golden has tier-2 data.
	needsResolution := len(golden.References) > 0 || len(golden.Implementations) > 0 || len(golden.Calls) > 0
	if needsResolution {
		require.NoError(t, engine.Resolve(context.Background()))
	}

	// --- Verify extraction assertions ---
	if len(golden.Definitions) > 0 {
		t.Run("definitions", func(t *testing.T) {
			verifyDefinitions(t, engine, srcDir, golden.Definitions)
		})
	}
	if len(golden.Imports) > 0 {
		t.Run("imports", func(t *testing.T) {
			verifyImports(t, engine, srcDir, golden.Imports)
		})
	}
	if len(golden.TypeMembers) > 0 {
		t.Run("type_members", func(t *testing.T) {
			verifyTypeMembers(t, engine, golden.TypeMembers)
		})
	}
	if len(golden.FunctionParams) > 0 {
		t.Run("function_params", func(t *testing.T) {
			verifyFunctionParams(t, engine, golden.FunctionParams)
		})
	}
	if len(golden.TypeParams) > 0 {
		t.Run("type_params", func(t *testing.T) {
			verifyTypeParams(t, engine, golden.TypeParams)
		})
	}

	// --- Verify resolution assertions ---
	if len(golden.References) > 0 {
		t.Run("references", func(t *testing.T) {
			verifyReferences(t, engine, srcDir, golden.References)
		})
	}
	if len(golden.Implementations) > 0 {
		t.Run("implementations", func(t *testing.T) {
			verifyImplementations(t, engine, golden.Implementations)
		})
	}
	if len(golden.Calls) > 0 {
		t.Run("calls", func(t *testing.T) {
			verifyCalls(t, engine, golden.Calls)
		})
	}
}

func verifyDefinitions(t *testing.T, engine *Engine, srcDir string, expected []goldenDef) {
	t.Helper()
	s := engine.Store()

	// Build set of actual definitions: (name, kind, file_basename, line)
	type defKey struct {
		Name string
		Kind string
		File string
		Line int
	}
	actual := make(map[defKey]bool)

	rows, err := s.DB().Query(
		`SELECT s.name, s.kind, f.path, s.start_line
		 FROM symbols s JOIN files f ON f.id = s.file_id`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var name, kind, path string
		var line int
		require.NoError(t, rows.Scan(&name, &kind, &path, &line))
		actual[defKey{name, kind, filepath.Base(path), line}] = true
	}
	require.NoError(t, rows.Err())

	for _, exp := range expected {
		key := defKey{exp.Name, exp.Kind, exp.File, exp.Line}
		assert.True(t, actual[key], "missing definition: %+v", exp)
	}
}

func verifyReferences(t *testing.T, engine *Engine, srcDir string, expected []goldenRef) {
	t.Helper()
	s := engine.Store()

	for _, exp := range expected {
		// Find reference at the "from" position.
		fromFile := filepath.Join(srcDir, exp.From.File)
		locs, err := engine.Query().DefinitionAt(fromFile, exp.From.Line, exp.From.Col)
		require.NoError(t, err, "error resolving reference from %s:%d:%d", exp.From.File, exp.From.Line, exp.From.Col)

		found := false
		for _, loc := range locs {
			// Check if any resolution target matches the "to" spec.
			baseName := filepath.Base(loc.File)
			if baseName == exp.To.File && loc.StartLine == exp.To.Line {
				// Verify the symbol name matches.
				var name string
				err := s.DB().QueryRow(
					`SELECT name FROM symbols WHERE file_id = (SELECT id FROM files WHERE path = ?)
					 AND start_line = ?`, loc.File, loc.StartLine,
				).Scan(&name)
				if err == nil && name == exp.To.Name {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "reference from %s:%d:%d should resolve to %s in %s:%d (got %d locations)",
			exp.From.File, exp.From.Line, exp.From.Col, exp.To.Name, exp.To.File, exp.To.Line, len(locs))
	}
}

func verifyImplementations(t *testing.T, engine *Engine, expected []goldenImpl) {
	t.Helper()
	s := engine.Store()

	for _, exp := range expected {
		rows, err := s.DB().Query(
			`SELECT ts.name, is_.name FROM implementations i
			 JOIN symbols ts ON ts.id = i.type_symbol_id
			 JOIN symbols is_ ON is_.id = i.interface_symbol_id
			 WHERE ts.name = ? AND is_.name = ?`,
			exp.Type, exp.Interface)
		require.NoError(t, err)
		found := rows.Next()
		rows.Close()
		assert.True(t, found, "missing implementation: %s implements %s", exp.Type, exp.Interface)
	}
}

func verifyCalls(t *testing.T, engine *Engine, expected []goldenCall) {
	t.Helper()
	s := engine.Store()

	for _, exp := range expected {
		rows, err := s.DB().Query(
			`SELECT cr.name, ce.name FROM call_graph cg
			 JOIN symbols cr ON cr.id = cg.caller_symbol_id
			 JOIN symbols ce ON ce.id = cg.callee_symbol_id
			 WHERE cr.name = ? AND ce.name = ?`,
			exp.Caller, exp.Callee)
		require.NoError(t, err)
		found := rows.Next()
		rows.Close()
		assert.True(t, found, "missing call edge: %s -> %s", exp.Caller, exp.Callee)
	}
}

func verifyImports(t *testing.T, engine *Engine, srcDir string, expected []goldenImport) {
	t.Helper()
	s := engine.Store()

	for _, exp := range expected {
		filePath := filepath.Join(srcDir, exp.File)
		f, err := s.FileByPath(filePath)
		require.NoError(t, err, "file lookup: %s", exp.File)
		require.NotNil(t, f, "file not indexed: %s", exp.File)

		imports, err := s.ImportsByFile(f.ID)
		require.NoError(t, err)

		found := false
		for _, imp := range imports {
			if imp.Source != exp.Source {
				continue
			}
			if exp.ImportedName != "" {
				if imp.ImportedName == nil || *imp.ImportedName != exp.ImportedName {
					continue
				}
			}
			if exp.LocalAlias != "" {
				if imp.LocalAlias == nil || *imp.LocalAlias != exp.LocalAlias {
					continue
				}
			}
			if exp.Kind != "" && imp.Kind != exp.Kind {
				continue
			}
			found = true
			break
		}
		assert.True(t, found, "missing import in %s: source=%s imported_name=%s",
			exp.File, exp.Source, exp.ImportedName)
	}
}

func verifyTypeMembers(t *testing.T, engine *Engine, expected []goldenTypeMember) {
	t.Helper()
	s := engine.Store()

	for _, exp := range expected {
		syms := findParentSymbols(t, s, exp.Parent, "", "")
		require.NotEmpty(t, syms, "parent symbol not found: %s", exp.Parent)

		found := false
		for _, sym := range syms {
			members, err := s.TypeMembers(sym.ID)
			require.NoError(t, err)
			for _, m := range members {
				if m.Name != exp.Name || m.Kind != exp.Kind {
					continue
				}
				if exp.TypeExpr != "" && m.TypeExpr != exp.TypeExpr {
					continue
				}
				if exp.Visibility != "" && m.Visibility != exp.Visibility {
					continue
				}
				found = true
				break
			}
			if found {
				break
			}
		}
		assert.True(t, found, "missing type member: %s.%s (kind=%s)", exp.Parent, exp.Name, exp.Kind)
	}
}

func verifyFunctionParams(t *testing.T, engine *Engine, expected []goldenFunctionParam) {
	t.Helper()
	s := engine.Store()

	for _, exp := range expected {
		syms := findParentSymbols(t, s, exp.Parent, "", exp.ParentFile)
		require.NotEmpty(t, syms, "parent symbol not found: %s", exp.Parent)

		found := false
		for _, sym := range syms {
			params, err := s.FunctionParams(sym.ID)
			require.NoError(t, err)
			for _, p := range params {
				if p.Ordinal != exp.Ordinal {
					continue
				}
				if exp.Name != "" && p.Name != exp.Name {
					continue
				}
				if exp.TypeExpr != "" && p.TypeExpr != exp.TypeExpr {
					continue
				}
				if p.IsReceiver != exp.IsReceiver || p.IsReturn != exp.IsReturn {
					continue
				}
				found = true
				break
			}
			if found {
				break
			}
		}
		assert.True(t, found, "missing param: %s ordinal=%d name=%s type_expr=%s is_receiver=%v is_return=%v",
			exp.Parent, exp.Ordinal, exp.Name, exp.TypeExpr, exp.IsReceiver, exp.IsReturn)
	}
}

func verifyTypeParams(t *testing.T, engine *Engine, expected []goldenTypeParam) {
	t.Helper()
	s := engine.Store()

	for _, exp := range expected {
		syms := findParentSymbols(t, s, exp.Parent, "", "")
		require.NotEmpty(t, syms, "parent symbol not found: %s", exp.Parent)

		found := false
		for _, sym := range syms {
			tps, err := s.TypeParams(sym.ID)
			require.NoError(t, err)
			for _, tp := range tps {
				if tp.Name != exp.Name || tp.Ordinal != exp.Ordinal {
					continue
				}
				if exp.Constraints != "" && tp.Constraints != exp.Constraints {
					continue
				}
				found = true
				break
			}
			if found {
				break
			}
		}
		assert.True(t, found, "missing type param: %s.%s (ordinal=%d)", exp.Parent, exp.Name, exp.Ordinal)
	}
}

// findParentSymbols returns symbols matching name, optionally filtered by kind and file basename.
func findParentSymbols(t *testing.T, s *store.Store, name, kind, fileBasename string) []*store.Symbol {
	t.Helper()
	syms, err := s.SymbolsByName(name)
	require.NoError(t, err)

	if kind == "" && fileBasename == "" {
		return syms
	}

	var filtered []*store.Symbol
	for _, sym := range syms {
		if kind != "" && sym.Kind != kind {
			continue
		}
		if fileBasename != "" && sym.FileID != nil {
			var path string
			err := s.DB().QueryRow("SELECT path FROM files WHERE id = ?", *sym.FileID).Scan(&path)
			if err != nil || filepath.Base(path) != fileBasename {
				continue
			}
		}
		filtered = append(filtered, sym)
	}
	return filtered
}
