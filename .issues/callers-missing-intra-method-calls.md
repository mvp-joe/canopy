# `query callers` misses calls between methods in the same package

**Status: RESOLVED**

## Summary

`canopy query callers` did not find method calls like `e.IndexFiles(ctx, paths)` inside `IndexDirectory`.

## Root cause

Method calls like `s.Handle("hello")` are extracted as two references:
- `"s"` with `context: "call"` — resolves to a variable, not a function
- `"Handle"` with `context: "field_access"` — resolves correctly to the method

The call graph builder (phase f in resolution scripts) only processed `context == "call"` references, so `"Handle"` was skipped despite being correctly resolved.

This was **universal across all 10 language resolution scripts**.

## Fix

Updated all 10 resolution scripts (`scripts/resolve/{go,typescript,javascript,python,ruby,rust,java,php,c,cpp}.risor`) to:
1. Also process `context == "field_access"` in the call graph phase
2. Guard against struct field access by checking that the resolved target is a `"method"` or `"function"` before creating the call edge

`callers engine.go 175 9` now finds 28 callers of `IndexFiles`, including `IndexDirectory`.

Covered by `TestQuery_Callers_MethodCall` in `cmd/canopy/query_integration_test.go`.
