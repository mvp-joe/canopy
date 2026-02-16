# Canopy Project Status

## Golden Test Coverage: Resolution Gaps

Most languages have extraction-only tests (definitions) but very few resolution tests (references, implementations, calls). The resolution scripts often already implement logic that has no golden test coverage. This document tracks what's tested and what's missing.

### Coverage Summary

| Language   | Levels | With Resolution | Biggest Gaps |
|------------|--------|-----------------|--------------|
| Go         | 8      | 3               | Scope shadowing, promoted methods, pkg-qualified refs |
| TypeScript | 8      | 3               | Namespace imports, default exports, re-export chains |
| Python     | 10     | 4               | Star imports, `self.x` tracking, closure resolution |
| Rust       | 8      | 4 (calls only)  | Trait dispatch, `pub use` re-exports, associated types |
| Java       | 8      | 1 (impl only)   | Zero references/calls tested anywhere |
| C          | 7      | 1               | Struct field access, enum refs, typedef chains |
| C++        | 7      | 3               | Virtual overrides, method calls, using declarations |
| PHP        | 7      | 2               | Namespace `use`, static/instance method calls |
| Ruby       | 7      | 2               | Instance method calls, inheritance lookup, mixins |

### Universal Gaps (all languages)

1. **Cross-file reference resolution** — nearly every language lacks thorough tests
2. **Call graph edges** — scripts implement call tracking but tests rarely verify it
3. **Implementation/inheritance tables** — barely tested anywhere
4. **Method dispatch through types** — `obj.method()` resolution is universally weak

---

### Go

**Existing levels:** 8 (level-01 through level-08)
**Levels with resolution tests:** 3 (level-03 imports, level-05 embedding, level-07 closures, level-08 multi-file interfaces)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 09 | method-value-dispatch | Pointer vs value receiver matching | Medium |
| 10 | type-assertion-flow | Type assertions and type switches | Medium |
| 11 | package-init-functions | Call graph includes init() | Quick win |
| 12 | package-qualified-refs | `pkg.Symbol` external resolution | Quick win |
| 13 | method-on-alias-types | Methods on type aliases | Quick win |
| 14 | embedded-promoted-methods | Embedded type method promotion | High complexity |
| 15 | interface-embedding | Interface-within-interface satisfaction | High complexity |
| 16 | interface-dispatch | Multi-implementation tracking | High complexity |
| 17 | receiver-type-shadowing | Scope chain inner scope priority | Medium |
| 18 | closure-capture-refs | Closure capture + variable resolution | High complexity |
| 19 | unexported-cross-package | Visibility filtering blocks unexported symbols | Quick win |
| 20 | nested-struct-field-resolution | Chained dot-notation field access | Medium |

---

### TypeScript

**Existing levels:** 8 (level-01 through level-08)
**Levels with resolution tests:** 3 (level-03 imports, level-06 generics, level-08 arrow-functions)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 09 | namespace-imports-reexports | `import * as`, re-export chains | High |
| 10 | default-mixed-imports | Default + named imports, mixed styles | High |
| 11 | class-inheritance-methods | Cross-file extends, inherited method resolution | High |
| 12 | interface-extension-composition | Interface extends, multiple implements | Medium |
| 13 | decorators | Decorator reference resolution | Medium |
| 14 | generics-constraints | Generic constraints with cross-file resolution | Medium |
| 15 | circular-references | Circular imports robustness | Low |
| 16 | dynamic-imports-conditional | Dynamic import(), type guards | Low |

---

### Python

**Existing levels:** 10 (level-01 through level-10)
**Levels with resolution tests:** 4 (level-03, level-08, level-09, level-10)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 11 | star-imports-with-all | `from X import *` with `__all__` | High |
| 12 | relative-imports | `from . import x`, `from ..sub import y` | High |
| 13 | transitive-imports | Import chain a→b→c resolution | Medium |
| 14 | instance-variables | `self.x` assignments and references | High |
| 15 | closure-resolution | Nested function captures outer scope vars | Medium |
| 16 | class-vs-instance-vars | `ClassName.x` vs `self.x` | Medium |
| 17 | mro-with-super | Diamond inheritance, `super()` calls | Medium |
| 18 | decorator-resolution | `@my_decorator` resolves to definition | Medium |
| 19 | import-alias-chains | `from a import X as Y; Y.attr` | Low |
| 20 | module-attribute-chains | `import util; util.sub.func()` | Low |
| 21 | exception-handler-vars | `except Exception as e:` binding | Low |
| 22 | comprehension-scopes | `[x for x in items]` scope isolation | Low |
| 23 | dunder-all-exports | `__all__` extraction and import filtering | Low |
| 24 | property-resolution | `@property` access resolves to definition | Low |
| 25 | classmethod-calls | `cls.method()` in class methods | Low |

---

### Rust

**Existing levels:** 8 (level-01 through level-08)
**Levels with resolution tests:** 4 (level-04, level-05, level-06, level-07 — calls only, no references)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 09 | cross-file-reexports | `pub use` re-export chains, module paths | High |
| 10 | trait-dispatch | Trait method dispatch, polymorphic calls | High |
| 11 | associated-types | Associated types, where clauses | Medium |
| 12 | lifetimes-generics | Lifetime params, generic trait bounds | Medium |
| 13 | wildcard-imports | `use crate::prelude::*` resolution | Medium |
| 14 | derive-macros | `#[derive(Clone)]` trait implications | Medium |
| 15 | pattern-destructuring | Enum variant refs, struct field destructuring | Medium |
| 16 | closures | Closure captures, Fn trait bounds | Low |
| 17 | impl-trait-dynamic | `dyn Trait`, trait object dispatch | Low |
| 18 | module-visibility | Multi-file modules, `pub(crate)`/`pub(super)` | Low |

---

### Java

**Existing levels:** 8 (level-01 through level-08)
**Levels with resolution tests:** 1 (level-06 has implementations only — zero references or calls tested)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 09 | imports-multi-package | Cross-file import resolution | High |
| 10 | method-override-calls | Polymorphic method resolution, call edges | High |
| 11 | constructor-chaining | `super()`/`this()` constructor calls | High |
| 12 | generic-resolution | Generic type instantiation references | Medium |
| 13 | anonymous-lambda | Anonymous inner classes, lambda references | Medium |
| 14 | enum-references | Enum constant and method resolution | Medium |
| 15 | static-access | Static field/method access, static imports | Medium |
| 16 | method-overloading | Overloaded method disambiguation | Low |
| 17 | nested-class-resolution | Nested class instantiation and references | Low |
| 18 | annotation-references | Annotation name resolution | Low |
| 19 | interface-extension | Interface extends, multiple implements | Low |
| 20 | field-access | Field resolution within and across classes | Low |

---

### C

**Existing levels:** 7 (level-01 through level-07)
**Levels with resolution tests:** 1 (level-03 multi-file has basic cross-file refs and calls)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 08 | struct-field-resolution | `p.x`, `r.br.y` field access | High |
| 09 | enum-value-refs | Enum constant references | High |
| 10 | typedef-chains | Typedef resolution to underlying types | High |
| 11 | function-pointer-callbacks | Indirect calls through function pointers | Medium |
| 12 | macro-expansion-calls | Macro invocation references and call edges | Medium |
| 13 | static-scope-isolation | Static functions don't leak across files | Medium |
| 14 | variable-refs | Global variable reference resolution | Medium |
| 15 | pointer-member-chains | `p->val`, `p->next` field access | Medium |
| 16 | union-types | Union type extraction and resolution | Low |
| 17 | extern-declarations | Cross-file extern linking | Low |
| 18 | type-casts | Type references in casts | Low |

---

### C++

**Existing levels:** 7 (level-01 through level-07)
**Levels with resolution tests:** 3 (level-03, level-04, level-05 — basic cross-file, template refs, enum refs)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 08 | virtual-override-resolution | Virtual dispatch, override resolution | High |
| 09 | method-field-calls | `obj.method()`, `ptr->field` resolution | High |
| 10 | using-declarations | `using namespace`, `using X::Y` | High |
| 11 | ctor-dtor-chains | Constructor delegation, destructor chains | Medium |
| 12 | operator-overloads | `u + v` → `operator+` resolution | Medium |
| 13 | cross-file-templates | Template instantiation across `#include` | Medium |
| 14 | static-nested | Static members, nested classes | Low |
| 15 | friend-functions | Friend access and resolution | Low |
| 16 | move-semantics | Rvalue refs, `std::move` | Low |

---

### PHP

**Existing levels:** 7 (level-01 through level-07)
**Levels with resolution tests:** 2 (level-03 multi-file, level-06 inheritance)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 08 | use-statements-imports | Namespace `use` resolution | High |
| 09 | static-method-calls | `Class::method()` resolution | High |
| 10 | instance-method-calls | `$this->method()` resolution | High |
| 11 | interfaces-implementations | Interface implementation detection | High |
| 12 | trait-method-resolution | Trait method resolution via `use` | Medium |
| 13 | anonymous-classes | Anonymous class instantiation | Medium |
| 14 | late-static-binding | `static::` vs `self::` | Medium |
| 15 | type-hints-modern | Union types, nullable, intersection | Medium |
| 16 | magic-methods | `__call`, `__get`, `__set` | Low |
| 17 | closures-callable | Arrow functions, callable params | Low |
| 18 | enum-with-methods | PHP 8.1 enums | Low |
| 19 | property-promotion | Constructor property promotion | Low |
| 20 | mixed-resolution | Full integration test | Low |

---

### Ruby

**Existing levels:** 7 (level-01 through level-07)
**Levels with resolution tests:** 2 (level-03 imports, level-07 multi-file)

**Suggested new tests:**

| Level | Name | Focus | Priority |
|-------|------|-------|----------|
| 08 | call-graph-instance-methods | Instance method call edges | High |
| 09 | inheritance-method-lookup | Superclass resolution, `super()` | High |
| 10 | mixin-resolution | Multiple `include`, MRO through mixins | High |
| 11 | extend-vs-include | `extend` (class methods) vs `include` (instance) | Medium |
| 12 | scope-resolution-operator | `::` constant resolution | Medium |
| 13 | attr-accessor-resolution | Generated getter/setter method resolution | Medium |
| 14 | open-class-reopening | Multiple `class` decls with same name | Low |
| 15 | dynamic-dispatch | `method_missing` limitations | Low |
| 16 | block-and-yield | Block parameters, yield call edges | Low |
| 17 | global-instance-variables | `$var`, `@var` resolution | Low |
| 18 | prepend-mixin-order | `prepend` MRO differences | Low |
| 19 | multi-file-inheritance-chain | Deep inheritance across files | Low |
| 20 | lambda-proc-closures | Lambda/Proc creation and captures | Low |
