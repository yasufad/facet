// Package style defines the visual and layout properties, the refinement model
// that layers them, and the mutation interface that elements build on.
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
//   - Fast merge operations via bitwise operations with per-word guards.
//   - Instant early-exit for unrefined states (when mask is zero).
//   - Unambiguous distinction between unset fields and zero-value assignments.
//
// # Mutators and Fluent Chaining
//
// Refinement exposes in-place mutators on *Refinement (such as SetOpacity,
// SetBackground, SetDisplay, SetFlexGrow) and an in-place MergeFrom method.
// It does not expose value-receiver fluent methods: returning Refinement by
// value on every call incurs struct-copy overhead on each property in the chain
// (~19 ns per call at 48 bytes, growing with struct size).
//
// In Facet, the fluent builder chain lives on the element (e.g. *Div in the
// element package), which is already addressable and mutates its internal
// refinement by pointer:
//
//	func (d *Div) Flex() *Div { d.style.SetDisplay(DisplayFlex); return d }
//
// This gives div().Flex().Bg(c) clean fluent reading with zero struct copies
// and zero allocations. Benchmark measurements:
//
//	control (48-byte copy)         2.13 ns
//	SetOpacity                     1.31 ns
//	SetBackground (colour.Rgba)    2.37 ns
//	sequence of 4 mutators         6.22 ns
//	MergeFrom                     20.45 ns
//
// # Colour Storage
//
// Background colour is stored directly as colour.Rgba in both Style and
// Refinement, matching scene.Quad. Storing Rgba avoids unnecessary conversions
// to and from Hsla on the hottest styling paths. SetBackgroundHsla converts at
// set time for callers working in HSL.
//
// # Compound Property Granularity
//
// Compound properties — inset, margin, padding, border widths and corner
// radii — use separate mask bits for each edge or corner (4 bits per compound
// property; 2 bits for size and gap). This per-edge granularity allows a
// refinement such as PaddingLeft(4) to override only the left padding while
// leaving top, right and bottom intact from earlier refinements.
//
// # Slice Immutability and Comparability
//
// Properties with slice values (such as box shadows) are immutable once set.
// Setting a slice property replaces the slice reference; code never appends
// to or mutates a slice stored in a Refinement.
//
// Refinement copies slice headers directly during MergeFrom and Refine. Because
// it contains slice fields, Refinement is non-comparable (cannot be used with ==
// or as map keys).
//
// # Style{} vs Default() Asymmetry
//
// Default() is the only valid constructor to obtain a Style. It initialises all
// framework defaults (such as DisplayFlex, Opacity 1.0, and transparent
// background). The zero-value Style{} has opacity 0 and uninitialised fields,
// rendering nothing.
//
// By contrast, Refinement{}'s zero value is a valid, meaningful empty
// refinement (mask == 0).
//
// # Invariants
//
//   - No cascade and no stylesheet. Inheritance is explicit and confined to
//     text properties.
//   - Setting a property is a method call resolved at build time, not a lookup
//     later.
//   - Imports only geometry, colour, layout and text from Facet.
package style
