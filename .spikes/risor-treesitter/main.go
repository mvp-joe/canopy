// Spike: Validate that Risor can call methods on go-tree-sitter CGO-backed objects.
//
// Goal: prove Risor scripts can receive *sitter.Tree, call .RootNode(), .Type(),
// .ChildCount(), .NamedChild(i), and run tree-sitter queries.
//
// Uses smacker/go-tree-sitter which bundles C sources properly for go modules.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/risor-io/risor"
	"github.com/risor-io/risor/object"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// mustProxy wraps object.NewProxy, returning an error object on failure.
func mustProxy(v any) object.Object {
	p, err := object.NewProxy(v)
	if err != nil {
		return object.Errorf("proxy error: %v", err)
	}
	return p
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Read the Risor script.
	scriptBytes, err := os.ReadFile("test.risor")
	if err != nil {
		return fmt.Errorf("reading script: %w", err)
	}

	// The Go source we'll parse with tree-sitter.
	goSource := []byte(`package main

import "fmt"

// Greet returns a greeting for the given name.
func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

// Add returns the sum of two integers.
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
`)

	// Parse with go-tree-sitter.
	lang := golang.GetLanguage()

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(ctx, nil, goSource)
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}
	defer tree.Close()

	// Build host functions for Risor.

	// parse() - returns the pre-parsed tree (we parse in Go, hand the result to Risor).
	parseFn := object.NewBuiltin("parse", func(ctx context.Context, args ...object.Object) object.Object {
		return mustProxy(tree)
	})

	// source() - returns the source bytes as a string so Risor can use it.
	sourceFn := object.NewBuiltin("source", func(ctx context.Context, args ...object.Object) object.Object {
		return object.NewString(string(goSource))
	})

	// source_bytes() - returns the source as []byte for methods that need it.
	sourceBytesFn := object.NewBuiltin("source_bytes", func(ctx context.Context, args ...object.Object) object.Object {
		return mustProxy(goSource)
	})

	// new_query(pattern) - creates a tree-sitter query for the Go language.
	newQueryFn := object.NewBuiltin("new_query", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("new_query", 1, len(args))
		}
		pattern, ok := args[0].(*object.String)
		if !ok {
			return object.Errorf("new_query: expected string, got %s", args[0].Type())
		}
		q, qErr := sitter.NewQuery([]byte(pattern.Value()), lang)
		if qErr != nil {
			return object.Errorf("new_query: %v", qErr)
		}
		return mustProxy(q)
	})

	// new_query_cursor() - creates a query cursor.
	newQueryCursorFn := object.NewBuiltin("new_query_cursor", func(ctx context.Context, args ...object.Object) object.Object {
		return mustProxy(sitter.NewQueryCursor())
	})

	// node_text(node) - returns the source text for a node.
	// Workaround: Risor can't convert string -> []byte for Content() calls.
	nodeTextFn := object.NewBuiltin("node_text", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 1 {
			return object.NewArgsError("node_text", 1, len(args))
		}
		nodeProxy, ok := args[0].(*object.Proxy)
		if !ok {
			return object.Errorf("node_text: expected proxy, got %s", args[0].Type())
		}
		node, ok := nodeProxy.Interface().(*sitter.Node)
		if !ok {
			return object.Errorf("node_text: expected *sitter.Node, got %T", nodeProxy.Interface())
		}
		return object.NewString(node.Content(goSource))
	})

	// exec_query(query, node, source) - executes a query and returns matches as a list of maps.
	// This wraps the cursor.Exec + NextMatch loop since Risor may struggle with
	// the iterator pattern on CGO objects.
	execQueryFn := object.NewBuiltin("exec_query", func(ctx context.Context, args ...object.Object) object.Object {
		if len(args) != 3 {
			return object.NewArgsError("exec_query", 3, len(args))
		}

		queryProxy, ok := args[0].(*object.Proxy)
		if !ok {
			return object.Errorf("exec_query: arg 0 must be a query proxy, got %s", args[0].Type())
		}
		query, ok := queryProxy.Interface().(*sitter.Query)
		if !ok {
			return object.Errorf("exec_query: arg 0 must be a *sitter.Query, got %T", queryProxy.Interface())
		}

		nodeProxy, ok := args[1].(*object.Proxy)
		if !ok {
			return object.Errorf("exec_query: arg 1 must be a node proxy, got %s", args[1].Type())
		}
		node, ok := nodeProxy.Interface().(*sitter.Node)
		if !ok {
			return object.Errorf("exec_query: arg 1 must be a *sitter.Node, got %T", nodeProxy.Interface())
		}

		srcStr, ok := args[2].(*object.String)
		if !ok {
			return object.Errorf("exec_query: arg 2 must be a string (source), got %s", args[2].Type())
		}
		src := []byte(srcStr.Value())

		cursor := sitter.NewQueryCursor()
		defer cursor.Close()
		cursor.Exec(query, node)

		var results []object.Object
		for {
			match, found := cursor.NextMatch()
			if !found {
				break
			}
			match = cursor.FilterPredicates(match, src)
			for _, capture := range match.Captures {
				m := map[string]object.Object{
					"text":  object.NewString(capture.Node.Content(src)),
					"type":  object.NewString(capture.Node.Type()),
					"start": object.NewInt(int64(capture.Node.StartPoint().Row + 1)),
					"end":   object.NewInt(int64(capture.Node.EndPoint().Row + 1)),
					"index": object.NewInt(int64(capture.Index)),
					"name":  object.NewString(query.CaptureNameForId(capture.Index)),
				}
				results = append(results, object.NewMap(m))
			}
		}

		return object.NewList(results)
	})

	// Run the Risor script with our host functions as globals.
	result, err := risor.Eval(ctx, string(scriptBytes),
		risor.WithGlobal("parse", parseFn),
		risor.WithGlobal("source", sourceFn),
		risor.WithGlobal("source_bytes", sourceBytesFn),
		risor.WithGlobal("new_query", newQueryFn),
		risor.WithGlobal("new_query_cursor", newQueryCursorFn),
		risor.WithGlobal("node_text", nodeTextFn),
		risor.WithGlobal("exec_query", execQueryFn),
	)
	if err != nil {
		return fmt.Errorf("risor eval: %w", err)
	}

	if result != nil && result.Type() != "nil" {
		fmt.Printf("Script result: %s\n", result.Inspect())
	}
	return nil
}
