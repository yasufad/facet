// Package geometry holds the units, shapes and axis arithmetic that every
// other package in Facet builds on.
//
// It owns four units — Pixels, DevicePixels, ScaledPixels and Rems — and five
// shapes generic over them: Point, Size, Bounds, Edges and Corners. Axis and
// Anchor describe directions and reference points so that layout algorithms
// need not be written twice, once per axis.
//
// Invariants:
//
//   - No dependencies at all, including on colour. geometry and colour are
//     siblings, not a stack.
//   - Units are distinct named types so they cannot be mixed by accident.
//     Conversion between them is explicit and takes the scale factor or rem
//     size as an argument; there is no implicit widening.
//   - Shapes are generic over a numeric constraint so the same types serve
//     logical pixels, device pixels and layout units alike.
//   - Values, not pointers. Everything here is small and copied freely; no
//     method takes a receiver pointer to mutate in place.
//   - No length or dimension types. Length, Dimension and LengthPercentage
//     belong to layout, which defines its own as part of the Taffy port;
//     style converts down to them.
package geometry
