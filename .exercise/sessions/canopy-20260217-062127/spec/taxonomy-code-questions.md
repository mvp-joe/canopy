# Taxonomy of Codebase Questions

Exhaustive list of questions an LLM or human would ask about a codebase, organized by perspective/activity. Use this as input for adversarial testing of Canopy's query API.

---

## 1. Orientation — "What is this codebase?"

- What languages are used, how much of each?
- What are the main packages/modules?
- What are the most-referenced symbols? (architectural pillars)
- What's the public API surface of package X?
- What files are in this project?
- What's the high-level dependency graph between packages?
- What are the entry points? (main functions, handlers, exported APIs)
- What's the layering? (which packages depend on which)
- How many symbols/files/lines of code are there?
- What's the package structure / organization?

## 2. Navigation — "Where is...?"

- Where is function/class X defined?
- Where is symbol X used?
- What symbol is at this cursor position?
- Find symbols matching a pattern
- What file contains symbol X?
- Jump from usage to definition
- Jump from definition to all usages

## 3. Understanding a Specific Symbol — "What is this thing?"

- What kind of symbol is this? (function, class, interface...)
- What's the visibility? (public, private, protected)
- What modifiers does it have? (static, async, abstract...)
- What are the parameters of this function?
- What type does this variable/parameter have?
- What does this function return?
- What members does this class/struct have?
- What annotations/decorators are on this symbol?
- What generic type parameters does this have?
- What's the parent symbol? (enclosing class/module)
- What interfaces does this type implement?
- What's the scope hierarchy at this position?
- What's the full signature of this function?

## 4. Call Graph — "What calls what?"

- Who calls this function? (direct callers)
- What does this function call? (direct callees)
- What's the full call chain from A to B?
- What entry points eventually reach this function? (transitive callers)
- What leaf functions does this eventually call? (transitive callees)
- Is this function recursive? (direct or via cycle)
- What's the call depth / call tree?
- How many things call this? (fan-in)
- How many things does this call? (fan-out)

## 5. Dependencies & Imports

- What does this file import?
- What files import this package/module?
- What's the package-to-package dependency graph?
- Are there circular dependencies?
- What external (third-party) vs internal deps?
- What's the transitive dependency closure?
- What files would be affected if I change package X?
- What are the re-exports?

## 6. Type Hierarchy & Composition

- What types implement this interface?
- What interfaces does this concrete type implement?
- What's the full inheritance chain?
- What's embedded/mixed into this type?
- What traits are implemented for this type? (Rust)
- What extension methods exist for this type? (Go)
- What methods must I implement for this interface?

## 7. Refactoring — "What breaks if I change this?"

- What are all references to this symbol?
- Is this symbol used outside its file?
- Is this symbol used at all? (dead code detection)
- Can I safely reduce visibility? (only internal refs)
- What's the blast radius of changing this function's signature?
- What callers need updating if I change params?
- Are there duplicate/similar symbols? (potential consolidation)
- What would a rename affect?

## 8. Code Smells & Quality — "What's wrong here?"

- Unused symbols (0 references)
- God objects (too many members)
- God functions (too many callees)
- High fan-in functions (fragile hotspots)
- Overly large files
- Circular dependencies
- Public symbols that could be private
- Too many parameters on a function
- Deeply nested scopes
- Classes with too many methods
- Unused imports

## 9. Impact Analysis — "How risky is this change?"

- How widely used is this API symbol?
- What's the transitive closure of things that depend on this?
- If this function's behavior changes, what tests might fail?
- What's the coupling between these two packages?
- Is this a critical path? (high fan-in across transitive callers)
- What would be the blast radius of removing this symbol entirely?

## 10. Cross-Cutting Concerns — "Show me all the..."

- All error handlers / error types
- All constructors / factory functions
- All test functions
- All async functions
- All functions with a specific annotation/decorator
- All implementations of a given interface
- All symbols with a specific modifier (static, abstract, etc.)
- All exported symbols across the project

## 11. Comparative & Structural — "How does X compare to Y?"

- Which package is biggest?
- Symbol kind distribution across the project
- Which packages are most tightly coupled?
- How does the structure of module A compare to B?
- What's the symbol density per file?
- What are the hotspot files? (most referenced symbols)
- What's the ratio of public to private symbols per package?

## 12. Type Queries — "What type is this?"

- What type is this variable?
- What's the return type of this function?
- What fields does this struct have and their types?
- Is this type generic? What are its type params?
- What types are used in this function's signature?
- What's the receiver type of this method?

## 13. Scope & Visibility — "What can I see from here?"

- What's visible from this scope?
- What symbols are in this namespace?
- What's the scope hierarchy?
- Is this name shadowed in an inner scope?
- What symbols does this scope introduce?
