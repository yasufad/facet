// Package colour holds the colour model for Facet: the Rgba and Hsla types,
// conversion between them, and the operations the renderer and the layers above
// need on a colour.
//
// The package owns only the model. A palette of named colours is a design
// decision and belongs above this layer; gradients, colour spaces beyond sRGB,
// and theming are out of scope here.
//
// Invariants:
//
//   - No dependencies, not even on geometry. The layering test enforces this.
//   - Components are stored as float32 in [0, 1], straight (non-premultiplied)
//     alpha. The renderer wants premultiplied components; Premultiply provides
//     that conversion explicitly rather than storing colours premultiplied, so
//     blending and interpolation stay accurate.
//   - The spelling is "colour" throughout: the package name and every
//     identifier, comment and doc string.
package colour
