# Public API Types

## Summary

Canopy's public `QueryBuilder` API currently returns types from `internal/store`, making the API unusable to external Go consumers who cannot import `internal/` packages. This spec adds Go type aliases in the `canopy` package for all 12 internal store types that appear in public method signatures, struct fields, or return values, then updates those signatures to use the aliases. The result is a fully importable public API with zero behavioral change.

## Goals

- External Go packages can import and use all types returned by `QueryBuilder` methods and `Engine.Store()`
- No behavioral changes, no conversion functions, no runtime cost (type aliases are compile-time only)
- The CLI (`cmd/canopy/`) continues to work without modification (aliases are the same type)

## Non-Goals

- Wrapping or hiding `store.Store` methods behind a new interface
- Adding new functionality to the query API
- Changing the internal `store` package in any way
- Schema changes

## Depends On

- 2026-02-17_query-api-v2

## Current Status

Complete

## Key Files

- [interface.md](interface.md) -- Type alias declarations and updated method signatures
- [implementation.md](implementation.md) -- Single-phase implementation plan
- [tests.md](tests.md) -- Compilation test specifications
