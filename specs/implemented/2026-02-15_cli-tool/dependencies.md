# Dependencies

## New Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | latest | CLI framework. Provides command/subcommand structure, flag parsing, help generation, and argument validation. |

## Rationale

### Cobra over alternatives

**Cobra** is the standard Go CLI library. Alternatives considered:

- **`flag` (stdlib)**: Too primitive for nested subcommands (`canopy query references --symbol 42`). Would require manual subcommand dispatch, help generation, and argument validation.
- **`urfave/cli`**: Capable but less idiomatic for complex nested command trees. Cobra's per-command struct model maps cleanly to canopy's `index` vs `query` distinction with 16 query subcommands.
- **`kong`**: Struct-tag based, cleaner for simple CLIs. Cobra is better established for the nesting depth needed here (root > query > references) and has wider community familiarity.

Cobra adds `github.com/spf13/pflag` as a transitive dependency (POSIX-style flags). No other significant transitive dependencies.

### No new dependency for embed/FS support

The script embedding feature uses only standard library packages (`embed`, `io/fs`, `testing/fstest`) and Risor's existing `importer.FSImporter` (already available in `github.com/risor-io/risor v1.8.1`). No new dependencies needed.
