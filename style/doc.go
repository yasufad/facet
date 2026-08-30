// Package style defines the visual and layout properties, the refinement model
// that layers them, and the fluent builder that elements expose.
//
// # The Refinement Model
//
// Styles are layered without a cascade or a stylesheet: a base style is
// refined by optional state-dependent overrides (such as hover, focus, active,
// or custom variants), where each refinement overrides only the fields it
// explicitly sets.
//
// A critical design requirement of the refinement model is distinguishing
// between a property that is unset and one that is explicitly set to its zero
// value (for example, setting opacity to 0.0 must override a base opacity of
// 1.0, whereas an omitted opacity property must leave the base opacity intact).
//
// # Representation: Parallel Bitset
//
// Rather than using pointer fields (which allocate on the heap per property per
// element per frame on the framework's hottest path) or sentinel values (which
// are fragile and error-prone across boolean flags, colours and enum types),
// Refinement uses a parallel bitset (mask).
//
// Property values are stored in flat fields, and a 128-bit mask tracks which
// properties have been explicitly set. This provides:
//
//   - Zero heap allocations per frame during construction, merging and application.
//   - Nanosecond merge operations via bitwise operations.
//   - Instant early-exit for unrefined states (when mask is zero).
//   - Unambiguous distinction between unset fields and zero-value assignments.
//
// # Invariants
//
//   - No cascade and no stylesheet. Inheritance is explicit and confined to
//     text properties.
//   - Setting a property is a method call resolved at build time, not a lookup
//     later.
//   - Imports only geometry, colour, layout and text from Facet.
package style
