# Go resolver matches references by name without respecting scope

**Status: OPEN**

## Summary

The Go resolution script resolves references to symbols by name only, ignoring lexical scope. This causes local variables like `t`, `s`, `err` to be falsely resolved across function and file boundaries within a package.

## Example

In `project-cortex`, a local variable `t := &Type{}` at `internal/storage/graph_reader.go:127` (inside a `for rows.Next()` loop) accumulates 349 "external" references — all of which are actually `t *testing.T` parameters in other `_test.go` files in the same package.

## Impact

- **Inflated ref counts** for common short variable names (`t`, `s`, `err`, `ctx`)
- **Pollutes "top symbols by external references"** with noise — local variables appear more important than real cross-file API types
- **False call graph edges** if a local variable name collides with a function name

## Root cause

The Go resolution script (`scripts/resolve/go.risor`) resolves references by querying `symbols_by_name(name)` and matching within the same package. It does not check whether the reference's scope (function body, block) actually has visibility to the symbol's definition scope.

## Suggested fix

During resolution, filter candidate symbols by scope:
1. **Function parameters and local variables** (kind `variable` inside a function scope) should only resolve references within the same scope or nested child scopes
2. **Package-level symbols** (functions, types, constants, package-level vars) can resolve any reference within the package
3. As a simpler heuristic: skip resolving `variable` kind symbols across file boundaries entirely — a local `t` in file A should never resolve a `t` reference in file B
