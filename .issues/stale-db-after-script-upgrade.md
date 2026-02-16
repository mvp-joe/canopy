# Stale DB after script upgrade shows empty resolution data

**Status: RESOLVED**

## Summary

When canopy scripts changed (new binary with updated extract/resolve scripts),
an incremental `canopy index` would skip re-processing because no source files
changed. This left stale extraction and resolution data from the prior script
version.

## Fix

Added automatic script change detection:

1. **Metadata table** (`internal/store/store.go`): New `metadata` key-value
   table with `GetMetadata`/`SetMetadata` methods.

2. **Scripts hash** (`engine.go`): `scriptsHash()` walks all embedded `.risor`
   files (extract + resolve + lib), sorts by path, and SHA-256 hashes their
   concatenated contents. `ScriptsChanged()` compares this against the stored
   `scripts_hash` in the DB.

3. **CLI auto-rebuild** (`cmd/canopy/main.go`): Before indexing, checks
   `engine.ScriptsChanged()`. If true, deletes the DB and recreates the engine
   (same as `--force`), printing "Scripts changed, rebuilding database".

4. **Hash persistence**: After successful resolution, `storeScriptsHash()`
   saves the current hash to the metadata table.

Works with both embedded FS (`go:embed`) and `--scripts-dir` disk scripts.
