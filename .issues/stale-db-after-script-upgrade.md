# Stale DB after script upgrade shows empty resolution data

**Status: OPEN**

## Summary

When canopy resolution scripts are updated (e.g. implementation matching logic
added or improved), an incremental `canopy index` correctly detects no source
file changes and skips resolution entirely. This preserves old (possibly empty)
resolution data from the prior script version. The user must know to run
`canopy index --force` to pick up script changes.

This is technically correct behavior but surprising — a user upgrades canopy,
re-runs index, and wonders why new resolution features produce no data.

## Repro

```bash
# Build DB with current canopy:
canopy index ../project-cortex --force
# Verify implementations exist:
canopy query symbols --db ../project-cortex/.canopy/index.db --kind interface --format text
# Pick an interface, confirm it has implementors

# Now re-run without --force (simulating "scripts changed but files didn't"):
canopy index ../project-cortex
# Output: "resolve: 0s" — resolution was skipped
# All resolution data is unchanged (fine if scripts didn't change,
# but stale if they did)
```

## Possible fix

Track a hash of resolution script contents in the DB (e.g. a `metadata` table
with `scripts_hash`). On index, compute the current scripts hash. If it differs
from the stored hash, force a full re-resolve even if no source files changed.

This would be transparent to the user — upgrading canopy automatically
triggers re-resolution on the next index.
