# Spike: Risor + go-tree-sitter CGO Interop

## Verdict: WORKS -- with one caveat around []byte arguments

Risor's proxy system can call methods on go-tree-sitter's CGO-backed objects
via reflection. Tree walking, node inspection, and query execution all work.

## What Works

| Capability | Status | Notes |
|---|---|---|
| `tree.RootNode()` | PASS | Returns proxied `*sitter.Node` |
| `node.Type()` | PASS | String return works seamlessly |
| `node.ChildCount()` | PASS | uint32 return auto-converts |
| `node.NamedChildCount()` | PASS | Same |
| `node.NamedChild(i)` | PASS | int arg converts, returns proxied Node |
| `node.ChildByFieldName("name")` | PASS | String arg converts fine |
| `node.String()` (S-expression) | PASS | |
| `node.StartPoint()` | PASS | Returns proxied Point struct |
| `cursor.Exec(query, node)` | PASS | Passing proxied CGO objects as args to other CGO objects works |
| `cursor.NextMatch()` | PASS | Multi-return (ptr, bool) comes back as a list |
| `tree_sitter.NewQuery(...)` | PASS | Via Go-side helper (free function, not a method) |
| `tree_sitter.NewQueryCursor()` | PASS | Via Go-side helper |

## What Doesn't Work (and Workarounds)

### `node.Content([]byte)` -- Risor can't convert string to []byte

Risor's proxy reflection layer fails when a Go method expects `[]byte` and
Risor passes a string (or a proxied `[]byte`). Error:

```
type error: failed to convert argument 1 in Content() call: type error: expected bytes (proxy given)
```

**Workaround:** Provide a Go-side `node_text(node)` helper function that calls
`node.Content(source)` with the captured `[]byte` source. This is clean and
arguably better design anyway -- the source bytes are a concern of the host, not
the script.

### Free functions (NewQuery, NewQueryCursor) need Go-side wrappers

Tree-sitter's query API uses free constructor functions, not methods on objects.
Risor can't call arbitrary Go package-level functions -- only methods on proxied
objects. This is expected and fine: wrap them as Risor builtins.

### Multi-return methods return as lists

`cursor.NextMatch()` returns `(*QueryMatch, bool)`. Risor wraps this as a list
`[proxy, bool]`. This works but is awkward to use from scripts. For the query
iterator pattern specifically, a Go-side `exec_query()` helper that runs the
full loop and returns structured results is cleaner.

## Architecture Implications

For production use, the right pattern is:

1. **Thin host API:** Provide ~5-6 Go-side builtin functions:
   - `parse(source, language)` -> proxied Tree
   - `node_text(node)` -> string (works around []byte limitation)
   - `query(pattern, node)` -> list of match maps (wraps cursor loop)
   - Possibly `node_children(node)` if iteration is awkward

2. **Let Risor call methods directly** for navigation:
   - `tree.RootNode()`, `node.Type()`, `node.ChildCount()`,
     `node.NamedChild(i)`, `node.ChildByFieldName(name)`, `node.Parent()`
   - These all work via proxy reflection with zero Go-side wrapping

3. **Keep []byte handling on the Go side.** Any method that takes or returns
   `[]byte` should be wrapped in a Go builtin.

## Dependencies

- `github.com/smacker/go-tree-sitter` (community bindings, bundles C sources properly)
- NOT `github.com/tree-sitter/go-tree-sitter` (official bindings have broken CGO includes when used as a Go module -- `#include "../../src/parser.c"` doesn't resolve)
- `github.com/risor-io/risor` v1.8.1

## Reproduction

```
cd .spikes/risor-treesitter
go build -o spike && ./spike
```
