package runtime

import (
	"path/filepath"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	ts "github.com/smacker/go-tree-sitter/typescript/typescript"
)

// extToLanguage maps file extensions to canonical language names.
var extToLanguage = map[string]string{
	".go":  "go",
	".ts":  "typescript",
	".tsx": "typescript",
	".js":  "javascript",
	".jsx": "javascript",
	".py":  "python",
	".rs":  "rust",
	".c":   "c",
	".h":   "c",
	".cpp": "cpp",
	".cc":  "cpp",
	".cxx": "cpp",
	".hpp": "cpp",
	".java": "java",
	".php":  "php",
	".rb":   "ruby",
}

// langToGrammar maps language names to tree-sitter Language objects.
// Lazily initialized on first call via sync.Once.
var (
	langToGrammar map[string]*sitter.Language
	grammarsOnce  sync.Once
)

func initGrammars() {
	grammarsOnce.Do(func() {
		langToGrammar = map[string]*sitter.Language{
			"go":         golang.GetLanguage(),
			"typescript": ts.GetLanguage(),
			"javascript": javascript.GetLanguage(),
			"python":     python.GetLanguage(),
			"rust":       rust.GetLanguage(),
			"c":          c.GetLanguage(),
			"cpp":        cpp.GetLanguage(),
			"java":       java.GetLanguage(),
			"php":        php.GetLanguage(),
			"ruby":       ruby.GetLanguage(),
		}
	})
}

// LanguageForFile returns the canonical language name for a file path based
// on its extension. Returns ("", false) if the extension is not recognized.
func LanguageForFile(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	lang, ok := extToLanguage[ext]
	return lang, ok
}

// ParserForLanguage returns the tree-sitter Language for a canonical language
// name. Returns (nil, false) if the language is not supported.
func ParserForLanguage(lang string) (*sitter.Language, bool) {
	initGrammars()
	l, ok := langToGrammar[lang]
	return l, ok
}
