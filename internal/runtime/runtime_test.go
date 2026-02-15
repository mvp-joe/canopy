package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

const goTestSource = `package main

import "fmt"

func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

func Add(a, b int) int {
	return a + b
}

type Server struct {
	Host string
	Port int
}

func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
`

// parseGoSource is a test helper that parses Go source using tree-sitter
// directly and registers it in a Runtime's source store.
func parseGoSource(t *testing.T, src string) (*sitter.Tree, *Runtime) {
	t.Helper()

	rt := NewRuntime(nil, "")

	lang, ok := ParserForLanguage("go")
	if !ok {
		t.Fatal("go language not found")
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(src))
	if err != nil {
		t.Fatalf("tree-sitter parse: %v", err)
	}

	rt.sources.store(tree, []byte(src), lang)

	return tree, rt
}

// --- Language detection tests ---

func TestLanguageForFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{"main.go", "go", true},
		{"app.ts", "typescript", true},
		{"app.tsx", "typescript", true},
		{"app.js", "javascript", true},
		{"app.jsx", "javascript", true},
		{"script.py", "python", true},
		{"lib.rs", "rust", true},
		{"main.c", "c", true},
		{"util.h", "c", true},
		{"main.cpp", "cpp", true},
		{"main.cc", "cpp", true},
		{"main.cxx", "cpp", true},
		{"util.hpp", "cpp", true},
		{"App.java", "java", true},
		{"index.php", "php", true},
		{"app.rb", "ruby", true},
		{"file.txt", "", false},
		{"Makefile", "", false},
		{"path/to/file.GO", "go", true}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got, ok := LanguageForFile(tt.path)
			if ok != tt.ok {
				t.Errorf("LanguageForFile(%q) ok = %v, want %v", tt.path, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("LanguageForFile(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestParserForLanguage(t *testing.T) {
	t.Parallel()

	supported := []string{"go", "typescript", "javascript", "python", "rust", "c", "cpp", "java", "php", "ruby"}
	for _, lang := range supported {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()
			l, ok := ParserForLanguage(lang)
			if !ok {
				t.Errorf("ParserForLanguage(%q) not found", lang)
			}
			if l == nil {
				t.Errorf("ParserForLanguage(%q) returned nil", lang)
			}
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()
		_, ok := ParserForLanguage("cobol")
		if ok {
			t.Error("ParserForLanguage(\"cobol\") should return false")
		}
	})
}

// --- parse host function tests ---

func TestParse_ReturnsTree(t *testing.T) {
	tree, _ := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("RootNode() returned nil")
	}
	if root.Type() != "source_file" {
		t.Errorf("root node Type() = %q, want %q", root.Type(), "source_file")
	}
}

func TestParse_GoRootNodeType(t *testing.T) {
	tree, _ := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	if root.Type() != "source_file" {
		t.Errorf("root node Type() = %q, want %q", root.Type(), "source_file")
	}
}

func TestParse_InvalidSourceStillReturnsTree(t *testing.T) {
	tree, _ := parseGoSource(t, "this is not valid go code }{}{")
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("RootNode() returned nil for invalid source")
	}
	if !root.HasError() {
		t.Log("expected tree to contain errors for invalid source")
	}
}

// --- Proxied node method tests ---

func TestNode_NamedChild(t *testing.T) {
	tree, _ := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	child := root.NamedChild(0)
	if child == nil {
		t.Fatal("NamedChild(0) returned nil")
	}
	if child.Type() != "package_clause" {
		t.Errorf("first named child Type() = %q, want %q", child.Type(), "package_clause")
	}
}

func TestNode_ChildByFieldName(t *testing.T) {
	tree, _ := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	var funcDecl *sitter.Node
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "function_declaration" {
			funcDecl = child
			break
		}
	}
	if funcDecl == nil {
		t.Fatal("no function_declaration found")
	}

	nameNode := funcDecl.ChildByFieldName("name")
	if nameNode == nil {
		t.Fatal("ChildByFieldName(\"name\") returned nil")
	}
	if nameNode.Type() != "identifier" {
		t.Errorf("name node Type() = %q, want %q", nameNode.Type(), "identifier")
	}
}

func TestNode_ChildCount(t *testing.T) {
	tree, _ := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	if root.ChildCount() == 0 {
		t.Error("root ChildCount() = 0, expected > 0")
	}
}

func TestNode_Parent(t *testing.T) {
	tree, _ := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	child := root.NamedChild(0)
	if child == nil {
		t.Fatal("NamedChild(0) returned nil")
	}
	parent := child.Parent()
	if parent == nil {
		t.Fatal("Parent() returned nil")
	}
	if parent.Type() != "source_file" {
		t.Errorf("parent Type() = %q, want %q", parent.Type(), "source_file")
	}
}

func TestNode_StartPoint(t *testing.T) {
	tree, _ := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	sp := root.StartPoint()
	if sp.Row != 0 || sp.Column != 0 {
		t.Errorf("root StartPoint() = (%d,%d), want (0,0)", sp.Row, sp.Column)
	}
}

// --- node_text tests (via sourceStore) ---

func TestNodeText_FunctionName(t *testing.T) {
	tree, rt := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	var funcDecl *sitter.Node
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "function_declaration" {
			funcDecl = child
			break
		}
	}
	if funcDecl == nil {
		t.Fatal("no function_declaration found")
	}

	nameNode := funcDecl.ChildByFieldName("name")
	if nameNode == nil {
		t.Fatal("no name node")
	}

	src, ok := rt.sources.sourceForNode(nameNode)
	if !ok {
		t.Fatal("source not found")
	}
	text := nameNode.Content(src)
	if text != "Greet" {
		t.Errorf("node_text for function name = %q, want %q", text, "Greet")
	}
}

func TestNodeText_FullFunctionDeclaration(t *testing.T) {
	tree, rt := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	var funcDecl *sitter.Node
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "function_declaration" {
			funcDecl = child
			break
		}
	}
	if funcDecl == nil {
		t.Fatal("no function_declaration found")
	}

	src, ok := rt.sources.sourceForNode(funcDecl)
	if !ok {
		t.Fatal("source not found")
	}
	text := funcDecl.Content(src)
	expected := `func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}`
	if text != expected {
		t.Errorf("node_text for function decl:\ngot:  %q\nwant: %q", text, expected)
	}
}

func TestNodeText_RootNodeReturnsFullSource(t *testing.T) {
	src := `package main

func f() {}
`
	tree, rt := parseGoSource(t, src)
	defer tree.Close()

	root := tree.RootNode()
	srcBytes, ok := rt.sources.sourceForNode(root)
	if !ok {
		t.Fatal("source not found")
	}

	text := root.Content(srcBytes)
	if text != src {
		t.Errorf("root content doesn't match source")
	}
}

// --- query tests (via tree-sitter directly) ---

func TestQuery_FunctionDeclarations(t *testing.T) {
	tree, rt := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	lang, _ := rt.sources.languageForNode(root)
	srcBytes, _ := rt.sources.sourceForNode(root)

	pattern := "(function_declaration name: (identifier) @name)"
	q, err := sitter.NewQuery([]byte(pattern), lang)
	if err != nil {
		t.Fatalf("NewQuery: %v", err)
	}
	defer q.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	cursor.Exec(q, root)

	var names []string
	for {
		match, found := cursor.NextMatch()
		if !found {
			break
		}
		match = cursor.FilterPredicates(match, srcBytes)
		for _, capture := range match.Captures {
			names = append(names, capture.Node.Content(srcBytes))
		}
	}

	expected := []string{"Greet", "Add"}
	if len(names) != len(expected) {
		t.Fatalf("got %d function names %v, want %d %v", len(names), names, len(expected), expected)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("function[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestQuery_NoMatches(t *testing.T) {
	src := `package main

var x = 1
`
	tree, rt := parseGoSource(t, src)
	defer tree.Close()

	root := tree.RootNode()
	lang, _ := rt.sources.languageForNode(root)
	srcBytes, _ := rt.sources.sourceForNode(root)

	pattern := "(function_declaration name: (identifier) @name)"
	q, err := sitter.NewQuery([]byte(pattern), lang)
	if err != nil {
		t.Fatalf("NewQuery: %v", err)
	}
	defer q.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	cursor.Exec(q, root)

	var count int
	for {
		match, found := cursor.NextMatch()
		if !found {
			break
		}
		match = cursor.FilterPredicates(match, srcBytes)
		count += len(match.Captures)
	}
	if count != 0 {
		t.Errorf("expected 0 matches, got %d", count)
	}
}

func TestQuery_InvalidPattern(t *testing.T) {
	tree, rt := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	lang, _ := rt.sources.languageForNode(root)
	_ = root

	_, err := sitter.NewQuery([]byte("(not_a_real_node_type @x)"), lang)
	if err == nil {
		t.Error("expected error for invalid query pattern, got nil")
	}
}

func TestQuery_CaptureNamesAsKeys(t *testing.T) {
	tree, rt := parseGoSource(t, goTestSource)
	defer tree.Close()

	root := tree.RootNode()
	lang, _ := rt.sources.languageForNode(root)
	srcBytes, _ := rt.sources.sourceForNode(root)

	pattern := "(function_declaration name: (identifier) @name) @func"
	q, err := sitter.NewQuery([]byte(pattern), lang)
	if err != nil {
		t.Fatalf("NewQuery: %v", err)
	}
	defer q.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	cursor.Exec(q, root)

	match, found := cursor.NextMatch()
	if !found {
		t.Fatal("expected at least one match")
	}
	match = cursor.FilterPredicates(match, srcBytes)

	captureNames := make(map[string]bool)
	for _, capture := range match.Captures {
		name := q.CaptureNameForId(capture.Index)
		captureNames[name] = true
	}
	if !captureNames["name"] {
		t.Error("expected capture named 'name'")
	}
	if !captureNames["func"] {
		t.Error("expected capture named 'func'")
	}
}

// --- Risor integration tests (via RunSource) ---

func TestRunSource_ParseAndNodeText(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "test.go")
	if err := os.WriteFile(goFile, []byte(goTestSource), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	rt := NewRuntime(nil, "")
	ctx := context.Background()

	script := `
path := test_file
tree := parse(path, "go")
root := tree.RootNode()

assert(root.Type() == "source_file", "expected source_file")

names := []
count := int(root.NamedChildCount())
for i := 0; i < count; i++ {
    child := root.NamedChild(i)
    if child.Type() == "function_declaration" {
        name_node := child.ChildByFieldName("name")
        names.append(node_text(name_node))
    }
}

assert(len(names) == 2, 'expected 2 functions, got {len(names)}')
assert(names[0] == "Greet", 'expected Greet, got {names[0]}')
assert(names[1] == "Add", 'expected Add, got {names[1]}')
`

	err := rt.RunSource(ctx, script, map[string]any{
		"test_file": goFile,
	})
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}
}

func TestRunSource_QueryHostFunction(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "test.go")
	if err := os.WriteFile(goFile, []byte(goTestSource), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	rt := NewRuntime(nil, "")
	ctx := context.Background()

	script := `
path := test_file
tree := parse(path, "go")
root := tree.RootNode()

matches := query("(function_declaration name: (identifier) @name)", root)
assert(len(matches) == 2, 'expected 2 matches, got {len(matches)}')

first := matches[0]
name_node := first["name"]
text := node_text(name_node)
assert(text == "Greet", 'expected Greet, got {text}')

second := matches[1]
name_node2 := second["name"]
text2 := node_text(name_node2)
assert(text2 == "Add", 'expected Add, got {text2}')
`

	err := rt.RunSource(ctx, script, map[string]any{
		"test_file": goFile,
	})
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}
}

func TestRunSource_QueryNoMatches(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "test.go")
	src := `package main

var x = 1
`
	if err := os.WriteFile(goFile, []byte(src), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	rt := NewRuntime(nil, "")
	ctx := context.Background()

	script := `
path := test_file
tree := parse(path, "go")
root := tree.RootNode()

matches := query("(function_declaration name: (identifier) @name)", root)
assert(len(matches) == 0, 'expected 0 matches, got {len(matches)}')
`

	err := rt.RunSource(ctx, script, map[string]any{
		"test_file": goFile,
	})
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}
}

func TestRunSource_QueryInvalidPattern(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "test.go")
	if err := os.WriteFile(goFile, []byte(goTestSource), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	rt := NewRuntime(nil, "")
	ctx := context.Background()

	script := `
path := test_file
tree := parse(path, "go")
root := tree.RootNode()

query("(not_a_real_node_type @x)", root)
`

	err := rt.RunSource(ctx, script, map[string]any{
		"test_file": goFile,
	})
	if err == nil {
		t.Fatal("expected error for invalid query pattern, got nil")
	}
}

func TestRunSource_MethodDeclarationQuery(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "test.go")
	if err := os.WriteFile(goFile, []byte(goTestSource), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	rt := NewRuntime(nil, "")
	ctx := context.Background()

	script := `
path := test_file
tree := parse(path, "go")
root := tree.RootNode()

matches := query("(method_declaration name: (field_identifier) @name)", root)
assert(len(matches) == 1, 'expected 1 method match, got {len(matches)}')

name_node := matches[0]["name"]
text := node_text(name_node)
assert(text == "Address", 'expected Address, got {text}')
`

	err := rt.RunSource(ctx, script, map[string]any{
		"test_file": goFile,
	})
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}
}

func TestRunSource_NodeTraversal(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "test.go")
	if err := os.WriteFile(goFile, []byte(goTestSource), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	rt := NewRuntime(nil, "")
	ctx := context.Background()

	script := `
path := test_file
tree := parse(path, "go")
root := tree.RootNode()

assert(root.ChildCount() > 0, "root should have children")

first := root.NamedChild(0)
parent := first.Parent()
assert(parent.Type() == "source_file", "parent should be source_file")

sp := root.StartPoint()
assert(int(sp.Row) == 0, 'expected row 0, got {int(sp.Row)}')
assert(int(sp.Column) == 0, 'expected col 0, got {int(sp.Column)}')
`

	err := rt.RunSource(ctx, script, map[string]any{
		"test_file": goFile,
	})
	if err != nil {
		t.Fatalf("RunSource: %v", err)
	}
}

func TestRunScript_LoadsFile(t *testing.T) {
	dir := t.TempDir()

	scriptPath := filepath.Join(dir, "test.risor")
	if err := os.WriteFile(scriptPath, []byte(`result := 1 + 1`), 0644); err != nil {
		t.Fatalf("writing script: %v", err)
	}

	rt := NewRuntime(nil, dir)
	ctx := context.Background()

	err := rt.RunScript(ctx, "test.risor", nil)
	if err != nil {
		t.Fatalf("RunScript: %v", err)
	}
}

func TestRunScript_MissingFile(t *testing.T) {
	rt := NewRuntime(nil, t.TempDir())
	ctx := context.Background()

	err := rt.RunScript(ctx, "nonexistent.risor", nil)
	if err == nil {
		t.Fatal("expected error for missing script, got nil")
	}
}

func TestLoadScript(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.risor")
	content := `x := 42`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing: %v", err)
	}

	rt := NewRuntime(nil, dir)
	got, err := rt.LoadScript(path)
	if err != nil {
		t.Fatalf("LoadScript: %v", err)
	}
	if got != content {
		t.Errorf("LoadScript = %q, want %q", got, content)
	}
}

func TestExtractionScriptPath(t *testing.T) {
	t.Parallel()
	got := ExtractionScriptPath("go")
	if got != filepath.Join("extract", "go.risor") {
		t.Errorf("ExtractionScriptPath(\"go\") = %q", got)
	}
}

func TestResolutionScriptPath(t *testing.T) {
	t.Parallel()
	got := ResolutionScriptPath("go")
	if got != filepath.Join("resolve", "go.risor") {
		t.Errorf("ResolutionScriptPath(\"go\") = %q", got)
	}
}
