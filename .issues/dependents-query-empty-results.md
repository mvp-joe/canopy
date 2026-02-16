# `query dependents` returns empty results

**Status: RESOLVED**

## Summary

`canopy query dependents <file>` returned no results even for heavily-imported files, while `package-summary` correctly showed dependents for the same package.

## Root cause

`Dependents()` in `query.go` did an exact match (`WHERE source = ?`). The `imports.source` column contains full module paths like `github.com/jward/canopy/internal/store`, but users pass partial paths like `internal/store`.

## Fix

Changed `Dependents()` to also do suffix matching: `WHERE source = ? OR source LIKE ?` with `"%/" + source`. Now `dependents internal/store` correctly finds all 29 files that import the store package.

Covered by `TestDependents_SuffixMatch` in `query_test.go`.
