// Package ui provides standard desktop GUI widgets built strictly from the
// public APIs of element, style, input, and app.
//
// # Widget Styling Philosophy
//
// Rather than duplicating Div's 129 fluent builder methods on every widget,
// widgets expose only what is intrinsic to being that widget (such as Label,
// OnClick, Disabled, TrackFocus), pseudo-state refinement hooks (Hover, Active,
// Focus), and general refinement via Refine. Callers customising layout or
// appearance apply refinements directly, keeping widget surfaces focused and
// stable as new style properties are added to the framework.
//
// # Lifecycle and State
//
// Widgets implement element.Element and are ephemeral values rebuilt each frame.
// Any state that must survive across frames belongs in an app.Entity[T].
package ui
