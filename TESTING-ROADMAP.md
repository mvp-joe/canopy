# Testing Roadmap

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

---

## Gaps Found from Adversarial Exercise Sessions

Two adversarial exercise sessions (46 test scripts total) tested CLI commands and query API methods against real multi-language projects. The following resolution scenarios were exercised there but have **no golden test coverage**:

### Type Hierarchy Depth
Exercises tested `type-hierarchy` showing implements/implemented_by/composes/composed_by/extensions relationships. Golden tests only check the flat `implementations` table. Multi-level hierarchies (C++ inheritance chains, Java class hierarchies) are untested.

### Re-exports
Exercises tested `reexports` across TypeScript, Go, and JavaScript. No golden tests verify that re-exported symbols resolve correctly through re-export chains.

### Java Enums
Exercise session 1 specifically tested Java enum detection, enum methods, and symbol-at for enum constants. Current Java golden tests don't cover enums at all.

### Method Dispatch Through Types
Exercises tested `obj.method()` resolution via definition-at queries. Golden tests mostly verify function-level references, not method dispatch through receiver/instance types.

### Parameter Metadata
Exercises tested `symbol-detail` showing parameter ordinals, is_receiver, is_return flags. Golden tests have `function_params` assertions but don't verify receiver parameters on Go methods or return type parameters systematically.

---

## Suggested New Golden Tests

### Go

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 11 | method-value-dispatch | Pointer vs value receiver matching | Medium |
| 12 | type-assertion-flow | Type assertions and type switches | Medium |
| 13 | package-init-functions | Call graph includes init() | Quick win |
| 14 | package-qualified-refs | `pkg.Symbol` external resolution | Quick win |
| 15 | method-on-alias-types | Methods on type aliases | Quick win |
| 16 | embedded-promoted-methods | Embedded type method promotion | High complexity |
| 17 | interface-embedding | Interface-within-interface satisfaction | High complexity |
| 18 | interface-dispatch | Multi-implementation tracking | High complexity |
| 19 | receiver-type-shadowing | Scope chain inner scope priority | Medium |
| 20 | closure-capture-refs | Closure capture + variable resolution | High complexity |
| 21 | unexported-cross-package | Visibility filtering blocks unexported symbols | Quick win |
| 22 | nested-struct-field-resolution | Chained dot-notation field access | Medium |

---

### TypeScript

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 11 | namespace-imports-reexports | `import * as`, re-export chains | High |
| 12 | default-mixed-imports | Default + named imports, mixed styles | High |
| 13 | class-inheritance-methods | Cross-file extends, inherited method resolution | High |
| 14 | interface-extension-composition | Interface extends, multiple implements | Medium |
| 15 | decorators | Decorator reference resolution | Medium |
| 16 | generics-constraints | Generic constraints with cross-file resolution | Medium |
| 17 | circular-references | Circular imports robustness | Low |
| 18 | dynamic-imports-conditional | Dynamic import(), type guards | Low |

---

### Python

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 13 | star-imports-with-all | `from X import *` with `__all__` | High |
| 14 | relative-imports | `from . import x`, `from ..sub import y` | High |
| 15 | transitive-imports | Import chain a→b→c resolution | Medium |
| 16 | instance-variables | `self.x` assignments and references | High |
| 17 | closure-resolution | Nested function captures outer scope vars | Medium |
| 18 | class-vs-instance-vars | `ClassName.x` vs `self.x` | Medium |
| 19 | mro-with-super | Diamond inheritance, `super()` calls | Medium |
| 20 | decorator-resolution | `@my_decorator` resolves to definition | Medium |
| 21 | import-alias-chains | `from a import X as Y; Y.attr` | Low |
| 22 | module-attribute-chains | `import util; util.sub.func()` | Low |
| 23 | exception-handler-vars | `except Exception as e:` binding | Low |
| 24 | comprehension-scopes | `[x for x in items]` scope isolation | Low |
| 25 | dunder-all-exports | `__all__` extraction and import filtering | Low |
| 26 | property-resolution | `@property` access resolves to definition | Low |
| 27 | classmethod-calls | `cls.method()` in class methods | Low |

---

### Rust

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 11 | cross-file-reexports | `pub use` re-export chains, module paths | High |
| 12 | trait-dispatch | Trait method dispatch, polymorphic calls | High |
| 13 | associated-types | Associated types, where clauses | Medium |
| 14 | lifetimes-generics | Lifetime params, generic trait bounds | Medium |
| 15 | wildcard-imports | `use crate::prelude::*` resolution | Medium |
| 16 | derive-macros | `#[derive(Clone)]` trait implications | Medium |
| 17 | pattern-destructuring | Enum variant refs, struct field destructuring | Medium |
| 18 | closures | Closure captures, Fn trait bounds | Low |
| 19 | impl-trait-dynamic | `dyn Trait`, trait object dispatch | Low |
| 20 | module-visibility | Multi-file modules, `pub(crate)`/`pub(super)` | Low |

---

### Java

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 11 | imports-multi-package | Cross-file import resolution | High |
| 12 | method-override-calls | Polymorphic method resolution, call edges | High |
| 13 | constructor-chaining | `super()`/`this()` constructor calls | High |
| 14 | generic-resolution | Generic type instantiation references | Medium |
| 15 | anonymous-lambda | Anonymous inner classes, lambda references | Medium |
| 16 | enum-references | Enum constant and method resolution | Medium |
| 17 | static-access | Static field/method access, static imports | Medium |
| 18 | method-overloading | Overloaded method disambiguation | Low |
| 19 | nested-class-resolution | Nested class instantiation and references | Low |
| 20 | annotation-references | Annotation name resolution | Low |
| 21 | interface-extension | Interface extends, multiple implements | Low |
| 22 | field-access | Field resolution within and across classes | Low |

---

### C

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 10 | struct-field-resolution | `p.x`, `r.br.y` field access | High |
| 11 | enum-value-refs | Enum constant references | High |
| 12 | typedef-chains | Typedef resolution to underlying types | High |
| 13 | function-pointer-callbacks | Indirect calls through function pointers | Medium |
| 14 | macro-expansion-calls | Macro invocation references and call edges | Medium |
| 15 | static-scope-isolation | Static functions don't leak across files | Medium |
| 16 | variable-refs | Global variable reference resolution | Medium |
| 17 | pointer-member-chains | `p->val`, `p->next` field access | Medium |
| 18 | union-types | Union type extraction and resolution | Low |
| 19 | extern-declarations | Cross-file extern linking | Low |
| 20 | type-casts | Type references in casts | Low |

---

### C++

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 10 | virtual-override-resolution | Virtual dispatch, override resolution | High |
| 11 | method-field-calls | `obj.method()`, `ptr->field` resolution | High |
| 12 | using-declarations | `using namespace`, `using X::Y` | High |
| 13 | ctor-dtor-chains | Constructor delegation, destructor chains | Medium |
| 14 | operator-overloads | `u + v` → `operator+` resolution | Medium |
| 15 | cross-file-templates | Template instantiation across `#include` | Medium |
| 16 | static-nested | Static members, nested classes | Low |
| 17 | friend-functions | Friend access and resolution | Low |
| 18 | move-semantics | Rvalue refs, `std::move` | Low |

---

### PHP

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 10 | use-statements-imports | Namespace `use` resolution | High |
| 11 | static-method-calls | `Class::method()` resolution | High |
| 12 | instance-method-calls | `$this->method()` resolution | High |
| 13 | interfaces-implementations | Interface implementation detection | High |
| 14 | trait-method-resolution | Trait method resolution via `use` | Medium |
| 15 | anonymous-classes | Anonymous class instantiation | Medium |
| 16 | late-static-binding | `static::` vs `self::` | Medium |
| 17 | type-hints-modern | Union types, nullable, intersection | Medium |
| 18 | magic-methods | `__call`, `__get`, `__set` | Low |
| 19 | closures-callable | Arrow functions, callable params | Low |
| 20 | enum-with-methods | PHP 8.1 enums | Low |
| 21 | property-promotion | Constructor property promotion | Low |
| 22 | mixed-resolution | Full integration test | Low |

---

### Ruby

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 10 | call-graph-instance-methods | Instance method call edges | High |
| 11 | inheritance-method-lookup | Superclass resolution, `super()` | High |
| 12 | mixin-resolution | Multiple `include`, MRO through mixins | High |
| 13 | extend-vs-include | `extend` (class methods) vs `include` (instance) | Medium |
| 14 | scope-resolution-operator | `::` constant resolution | Medium |
| 15 | attr-accessor-resolution | Generated getter/setter method resolution | Medium |
| 16 | open-class-reopening | Multiple `class` decls with same name | Low |
| 17 | dynamic-dispatch | `method_missing` limitations | Low |
| 18 | block-and-yield | Block parameters, yield call edges | Low |
| 19 | global-instance-variables | `$var`, `@var` resolution | Low |
| 20 | prepend-mixin-order | `prepend` MRO differences | Low |
| 21 | multi-file-inheritance-chain | Deep inheritance across files | Low |
| 22 | lambda-proc-closures | Lambda/Proc creation and captures | Low |
