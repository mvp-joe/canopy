# Position-based query commands missing `total_count` in JSON response

**Status: RESOLVED**

## Summary

Discovery/search query subcommands (`symbols`, `search`, `files`, `packages`,
`summary`, `package-summary`) included `total_count` in their JSON envelope.
Position-based query subcommands did not. Affected commands:

- `implementations`
- `callees`
- `callers`
- `references`
- `definition`
- `symbol-at`
- `deps`
- `dependents`

This made programmatic consumption inconsistent — any script checking
`response.get("total_count", 0)` would incorrectly report zero results for
these commands even when `results` contained data.

## Impact

This caused a false positive during testing — the Go call graph appeared empty
when it actually had data (see `go-call-graph-sparse-for-public-functions.md`).

## Fix

Added `TotalCount` to the `CLIResult` returned by all 8 affected commands in
`cmd/canopy/query.go` and `cmd/canopy/query_discovery.go`. Each uses
`len(results)` since these commands don't paginate.
