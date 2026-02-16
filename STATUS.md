# Canopy Project Status

## Golden Test Coverage

90 golden test levels across 10 languages. 50 levels include resolution data (references, implementations, or calls).

### Coverage Summary

| Language   | Levels | With Resolution | Resolution Levels | Key Types |
|------------|--------|-----------------|-------------------|-----------|
| Go         | 10     | 6               | 03, 05, 07, 08, 09, 10 | refs, impl, calls |
| TypeScript | 10     | 5               | 03, 06, 08, 09, 10 | refs, impl, calls |
| JavaScript | 2      | 2               | 01, 02 | refs |
| Python     | 12     | 6               | 03, 08, 09, 10, 11, 12 | refs, calls |
| Rust       | 10     | 7               | 04, 05, 06, 07, 08, 09, 10 | refs, calls |
| Java       | 10     | 3               | 06, 09, 10 | refs, impl |
| C          | 9      | 6               | 03, 04, 05, 06, 08, 09 | refs, calls |
| C++        | 9      | 7               | 03, 04, 05, 06, 07, 08, 09 | refs |
| PHP        | 9      | 4               | 03, 06, 08, 09 | refs, calls |
| Ruby       | 9      | 4               | 03, 07, 08, 09 | refs, calls |

### Remaining Gaps

1. **Call graph edges** — scripts implement call tracking but many languages only test calls in a few levels
2. **Implementation/inheritance tables** — only Go, TypeScript, and Java have implementation golden tests
3. **Method dispatch through types** — `obj.method()` resolution is largely untested via golden tests
4. **JavaScript coverage** — only scope-leak tests exist; no extraction-only levels for basic language features
5. **C++ missing calls/implementations** — 7 resolution levels but all references-only; no call graph or implementation tests

### What Each Level Tests

Every language has scope-leak tests (intrafile + crossfile) that verify reference resolution picks the correct scope. These were the most recent additions. Earlier levels vary by language:

- **Extraction-only levels** — basic declarations, classes/structs, enums, generics, nested structures
- **Import resolution levels** (level-03 for most languages) — cross-file reference and call tracking
- **Language-specific resolution** — embedding/implementations (Go), generics (TypeScript), method resolution (Python), module calls (Rust), abstract classes (Java), function pointers (C), templates/enums (C++), inheritance (PHP), multi-file classes (Ruby)
