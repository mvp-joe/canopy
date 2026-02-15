package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/risor-io/risor/object"
	sitter "github.com/smacker/go-tree-sitter"
)

// sourceStore tracks source bytes and language for each parsed tree.
// node_text and query need to recover source/language from a Node, but
// smacker/go-tree-sitter doesn't expose Node.Tree(). We store mappings
// keyed by root node pointer (obtained via tree.RootNode() at parse time
// and by walking up Parent() at lookup time).
type sourceStore struct {
	mu      sync.RWMutex
	sources map[uintptr][]byte          // root node ptr → source bytes
	langs   map[uintptr]*sitter.Language // root node ptr → language
}

func newSourceStore() *sourceStore {
	return &sourceStore{
		sources: make(map[uintptr][]byte),
		langs:   make(map[uintptr]*sitter.Language),
	}
}

func (s *sourceStore) store(tree *sitter.Tree, src []byte, lang *sitter.Language) {
	root := tree.RootNode()
	key := uintptr(unsafe.Pointer(root))
	s.mu.Lock()
	s.sources[key] = src
	s.langs[key] = lang
	s.mu.Unlock()
}

// rootOf walks a node up to its root via Parent().
func rootOf(node *sitter.Node) *sitter.Node {
	for node.Parent() != nil {
		node = node.Parent()
	}
	return node
}

func (s *sourceStore) sourceForNode(node *sitter.Node) ([]byte, bool) {
	key := uintptr(unsafe.Pointer(rootOf(node)))
	s.mu.RLock()
	src, ok := s.sources[key]
	s.mu.RUnlock()
	return src, ok
}

func (s *sourceStore) languageForNode(node *sitter.Node) (*sitter.Language, bool) {
	key := uintptr(unsafe.Pointer(rootOf(node)))
	s.mu.RLock()
	lang, ok := s.langs[key]
	s.mu.RUnlock()
	return lang, ok
}

// makeParseFn creates the "parse" host function.
//
// parse(path, language) → *sitter.Tree
func makeParseFn(ss *sourceStore) *object.Builtin {
	return object.NewBuiltin("parse", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 2 {
			return object.NewArgsError("parse", 2, len(args))
		}

		pathStr, ok := args[0].(*object.String)
		if !ok {
			return object.Errorf("parse: path must be a string, got %s", args[0].Type())
		}

		langStr, ok := args[1].(*object.String)
		if !ok {
			return object.Errorf("parse: language must be a string, got %s", args[1].Type())
		}

		src, err := os.ReadFile(pathStr.Value())
		if err != nil {
			return object.Errorf("parse: reading %s: %v", pathStr.Value(), err)
		}

		return parseSource(ctx, ss, src, langStr.Value())
	})
}

// makeParseSrcFn creates "parse_src" — accepts source string directly (for testing).
//
// parse_src(source, language) → *sitter.Tree
func makeParseSrcFn(ss *sourceStore) *object.Builtin {
	return object.NewBuiltin("parse_src", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 2 {
			return object.NewArgsError("parse_src", 2, len(args))
		}

		srcStr, ok := args[0].(*object.String)
		if !ok {
			return object.Errorf("parse_src: source must be a string, got %s", args[0].Type())
		}

		langStr, ok := args[1].(*object.String)
		if !ok {
			return object.Errorf("parse_src: language must be a string, got %s", args[1].Type())
		}

		return parseSource(ctx, ss, []byte(srcStr.Value()), langStr.Value())
	})
}

// parseSource is the shared implementation for parse and parse_src.
func parseSource(ctx context.Context, ss *sourceStore, src []byte, langName string) object.Object {
	lang, found := ParserForLanguage(langName)
	if !found {
		return object.Errorf("parse: unsupported language %q", langName)
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(ctx, nil, src)
	if err != nil {
		return object.Errorf("parse: tree-sitter parse failed: %v", err)
	}

	ss.store(tree, src, lang)

	proxy, err := object.NewProxy(tree)
	if err != nil {
		return object.Errorf("parse: proxy error: %v", err)
	}
	return proxy
}

// makeNodeTextFn creates the "node_text" host function.
//
// node_text(node) → string
//
// Exists because Risor's proxy system cannot convert strings to []byte
// for node.Content([]byte).
func makeNodeTextFn(ss *sourceStore) *object.Builtin {
	return object.NewBuiltin("node_text", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("node_text", 1, len(args))
		}

		proxy, ok := args[0].(*object.Proxy)
		if !ok {
			return object.Errorf("node_text: expected proxy (Node), got %s", args[0].Type())
		}

		node, ok := proxy.Interface().(*sitter.Node)
		if !ok {
			return object.Errorf("node_text: expected *sitter.Node, got %T", proxy.Interface())
		}

		src, found := ss.sourceForNode(node)
		if !found {
			return object.Errorf("node_text: no source found for node's tree")
		}

		return object.NewString(node.Content(src))
	})
}

// makeQueryFn creates the "query" host function.
//
// query(pattern, node) → []map[string]any
//
// Each map has capture names as keys and proxied Nodes as values.
func makeQueryFn(ss *sourceStore) *object.Builtin {
	return object.NewBuiltin("query", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 2 {
			return object.NewArgsError("query", 2, len(args))
		}

		patternStr, ok := args[0].(*object.String)
		if !ok {
			return object.Errorf("query: pattern must be a string, got %s", args[0].Type())
		}

		nodeProxy, ok := args[1].(*object.Proxy)
		if !ok {
			return object.Errorf("query: node must be a proxy (Node), got %s", args[1].Type())
		}

		node, ok := nodeProxy.Interface().(*sitter.Node)
		if !ok {
			return object.Errorf("query: expected *sitter.Node, got %T", nodeProxy.Interface())
		}

		lang, found := ss.languageForNode(node)
		if !found {
			return object.Errorf("query: no language found for node's tree")
		}

		src, found := ss.sourceForNode(node)
		if !found {
			return object.Errorf("query: no source found for node's tree")
		}

		q, err := sitter.NewQuery([]byte(patternStr.Value()), lang)
		if err != nil {
			return object.Errorf("query: invalid pattern: %v", err)
		}
		defer q.Close()

		cursor := sitter.NewQueryCursor()
		defer cursor.Close()
		cursor.Exec(q, node)

		var results []object.Object
		for {
			match, ok := cursor.NextMatch()
			if !ok {
				break
			}
			match = cursor.FilterPredicates(match, src)

			matchMap := make(map[string]object.Object)
			for _, capture := range match.Captures {
				name := q.CaptureNameForId(capture.Index)
				nodeP, err := object.NewProxy(capture.Node)
				if err != nil {
					return object.Errorf("query: proxy error for capture %q: %v", name, err)
				}
				matchMap[name] = nodeP
			}
			results = append(results, object.NewMap(matchMap))
		}

		if results == nil {
			results = []object.Object{}
		}
		return object.NewList(results)
	})
}

// makeNodeChildFn creates "node_child" — safe wrapper for ChildByFieldName
// that returns Risor nil instead of a proxied Go nil pointer.
//
// node_child(node, fieldName) → Node or nil
func makeNodeChildFn() *object.Builtin {
	return object.NewBuiltin("node_child", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 2 {
			return object.NewArgsError("node_child", 2, len(args))
		}

		proxy, ok := args[0].(*object.Proxy)
		if !ok {
			return object.Errorf("node_child: expected proxy (Node), got %s", args[0].Type())
		}

		node, ok := proxy.Interface().(*sitter.Node)
		if !ok {
			return object.Errorf("node_child: expected *sitter.Node, got %T", proxy.Interface())
		}

		fieldStr, ok := args[1].(*object.String)
		if !ok {
			return object.Errorf("node_child: field must be a string, got %s", args[1].Type())
		}

		child := node.ChildByFieldName(fieldStr.Value())
		if child == nil {
			return object.Nil
		}

		p, err := object.NewProxy(child)
		if err != nil {
			return object.Errorf("node_child: proxy error: %v", err)
		}
		return p
	})
}

// logObject provides log.info/warn/error methods for Risor scripts.
type logObject struct {
	prefix string
}

func (l *logObject) Info(msg string) {
	fmt.Printf("[%s] INFO: %s\n", l.prefix, msg)
}

func (l *logObject) Warn(msg string) {
	fmt.Printf("[%s] WARN: %s\n", l.prefix, msg)
}

func (l *logObject) Error(msg string) {
	fmt.Printf("[%s] ERROR: %s\n", l.prefix, msg)
}
