# Go call graph resolution is sparse/empty for public functions

**Status: INVALID — caused by issue #1 (missing `total_count`)**

## Summary

~~Querying `callees` for the top 50 most-referenced public functions in
project-cortex returned zero call graph edges.~~

This was a false positive. The call graph is working correctly:
- `CreateSchema` has 11 callees
- `NewParser` has 8 callees
- Top functions have 64+ callers

The original test script used `rd.get("total_count", 0)` to check results,
but `callees`/`callers`/`implementations` JSON responses don't include
`total_count` (see `implementations-json-missing-total-count.md`). This made
it appear that all results were empty when they actually had data in `results`.

## Lesson

The missing `total_count` issue affects all position-based query commands, not
just `implementations`. Any programmatic consumer checking `total_count` will
get wrong results for: `callees`, `callers`, `implementations`, `references`,
`definition`, `symbol-at`.
