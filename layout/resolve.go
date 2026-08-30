// Ported from Taffy src/util/resolve.rs (MIT).
//
// Taffy's MaybeResolve and ResolveOrZero traits turn context-dependent length
// types into absolute values. Go has no traits, so each implementation becomes a
// free function named for the type it resolves.
package layout

// clMaybeResolve resolves a compact length against an optional context. Lengths
// resolve to themselves; percentages resolve against the context (none when the
// context is undefined); auto, content and intrinsic sizing keywords resolve to
// none.
func clMaybeResolve(c compactLength, ctx optF32) optF32 {
	switch c.tag {
	case clLength:
		return some(c.val)
	case clPercent:
		if ctx.isSome() {
			return some(ctx.v * c.val)
		}
		return none()
	case clAuto, clContent:
		return none()
	default:
		if c.isSizingKeyword() {
			return none()
		}
		return none()
	}
}

// lpMaybeResolve resolves a LengthPercentage against an optional context.
func lpMaybeResolve(lp LengthPercentage, ctx optF32) optF32 {
	return clMaybeResolve(lp.cl, ctx)
}

// lpaMaybeResolve resolves a LengthPercentageAuto against an optional context.
func lpaMaybeResolve(lpa LengthPercentageAuto, ctx optF32) optF32 {
	return clMaybeResolve(lpa.cl, ctx)
}

// dimMaybeResolve resolves a Dimension against an optional context.
func dimMaybeResolve(d Dimension, ctx optF32) optF32 {
	return clMaybeResolve(d.cl, ctx)
}

// lpResolveOrZero resolves a LengthPercentage, falling back to 0.
func lpResolveOrZero(lp LengthPercentage, ctx optF32) float32 {
	if v := lpMaybeResolve(lp, ctx); v.isSome() {
		return v.v
	}
	return 0
}

// lpaResolveOrZero resolves a LengthPercentageAuto, falling back to 0.
func lpaResolveOrZero(lpa LengthPercentageAuto, ctx optF32) float32 {
	if v := lpaMaybeResolve(lpa, ctx); v.isSome() {
		return v.v
	}
	return 0
}

// dimResolveOrZero resolves a Dimension, falling back to 0.
func dimResolveOrZero(d Dimension, ctx optF32) float32 {
	if v := dimMaybeResolve(d, ctx); v.isSome() {
		return v.v
	}
	return 0
}

// sizeDimMaybeResolve resolves a Size[Dimension] component-wise against a
// Size[optF32] context.
func sizeDimMaybeResolve(s Size[Dimension], ctx Size[optF32]) Size[optF32] {
	return Size[optF32]{
		Width:  dimMaybeResolve(s.Width, ctx.Width),
		Height: dimMaybeResolve(s.Height, ctx.Height),
	}
}

// sizeDimResolveOrZero resolves a Size[Dimension] to a Size[float32], falling
// back to 0 per component.
func sizeDimResolveOrZero(s Size[Dimension], ctx Size[optF32]) Size[float32] {
	return Size[float32]{
		Width:  dimResolveOrZero(s.Width, ctx.Width),
		Height: dimResolveOrZero(s.Height, ctx.Height),
	}
}

// sizeLPAMaybeResolve resolves a Size[LengthPercentageAuto] component-wise.
func sizeLPAMaybeResolve(s Size[LengthPercentageAuto], ctx Size[optF32]) Size[optF32] {
	return Size[optF32]{
		Width:  lpaMaybeResolve(s.Width, ctx.Width),
		Height: lpaMaybeResolve(s.Height, ctx.Height),
	}
}

// sizeLPAResolveOrZero resolves a Size[LengthPercentageAuto] to a Size[float32].
func sizeLPAResolveOrZero(s Size[LengthPercentageAuto], ctx Size[optF32]) Size[float32] {
	return Size[float32]{
		Width:  lpaResolveOrZero(s.Width, ctx.Width),
		Height: lpaResolveOrZero(s.Height, ctx.Height),
	}
}

// rectLPAResolveOrZeroSize resolves a Rect[LengthPercentageAuto] against a
// Size[optF32] context (left/right against width, top/bottom against height).
func rectLPAResolveOrZeroSize(r Rect[LengthPercentageAuto], ctx Size[optF32]) Rect[float32] {
	return Rect[float32]{
		Left:   lpaResolveOrZero(r.Left, ctx.Width),
		Right:  lpaResolveOrZero(r.Right, ctx.Width),
		Top:    lpaResolveOrZero(r.Top, ctx.Height),
		Bottom: lpaResolveOrZero(r.Bottom, ctx.Height),
	}
}

// rectLPAResolveOrZeroOpt resolves a Rect[LengthPercentageAuto] against a single
// optional context applied to all sides.
func rectLPAResolveOrZeroOpt(r Rect[LengthPercentageAuto], ctx optF32) Rect[float32] {
	return Rect[float32]{
		Left:   lpaResolveOrZero(r.Left, ctx),
		Right:  lpaResolveOrZero(r.Right, ctx),
		Top:    lpaResolveOrZero(r.Top, ctx),
		Bottom: lpaResolveOrZero(r.Bottom, ctx),
	}
}

// rectLPAResolveOrZeroF32 resolves a Rect[LengthPercentageAuto] against a
// definite context applied to all sides.
func rectLPAResolveOrZeroF32(r Rect[LengthPercentageAuto], ctx float32) Rect[float32] {
	return rectLPAResolveOrZeroOpt(r, some(ctx))
}

// rectLPResolveOrZeroSize resolves a Rect[LengthPercentage] against a Size context.
func rectLPResolveOrZeroSize(r Rect[LengthPercentage], ctx Size[optF32]) Rect[float32] {
	return Rect[float32]{
		Left:   lpResolveOrZero(r.Left, ctx.Width),
		Right:  lpResolveOrZero(r.Right, ctx.Width),
		Top:    lpResolveOrZero(r.Top, ctx.Height),
		Bottom: lpResolveOrZero(r.Bottom, ctx.Height),
	}
}

// sizeLPMaybeResolve maybe-resolves a Size[LengthPercentage] against a Size context.
func sizeLPMaybeResolve(s Size[LengthPercentage], ctx Size[optF32]) Size[optF32] {
	return Size[optF32]{
		Width:  lpMaybeResolve(s.Width, ctx.Width),
		Height: lpMaybeResolve(s.Height, ctx.Height),
	}
}

// sizeLPResolveOrZeroSize resolves a Size[LengthPercentage] against a Size context.
func sizeLPResolveOrZeroSize(s Size[LengthPercentage], ctx Size[optF32]) Size[float32] {
	return Size[float32]{
		Width:  lpResolveOrZero(s.Width, ctx.Width),
		Height: lpResolveOrZero(s.Height, ctx.Height),
	}
}

// rectLPResolveOrZeroOpt resolves a Rect[LengthPercentage] against a single
// optional context.
func rectLPResolveOrZeroOpt(r Rect[LengthPercentage], ctx optF32) Rect[float32] {
	return Rect[float32]{
		Left:   lpResolveOrZero(r.Left, ctx),
		Right:  lpResolveOrZero(r.Right, ctx),
		Top:    lpResolveOrZero(r.Top, ctx),
		Bottom: lpResolveOrZero(r.Bottom, ctx),
	}
}

// rectLPResolveOrZeroF32 resolves a Rect[LengthPercentage] against a definite
// context.
func rectLPResolveOrZeroF32(r Rect[LengthPercentage], ctx float32) Rect[float32] {
	return rectLPResolveOrZeroOpt(r, some(ctx))
}

// rectDimResolveOrZeroSize resolves a Rect[Dimension] against a Size context.
func rectDimResolveOrZeroSize(r Rect[Dimension], ctx Size[optF32]) Rect[float32] {
	return Rect[float32]{
		Left:   dimResolveOrZero(r.Left, ctx.Width),
		Right:  dimResolveOrZero(r.Right, ctx.Width),
		Top:    dimResolveOrZero(r.Top, ctx.Height),
		Bottom: dimResolveOrZero(r.Bottom, ctx.Height),
	}
}

// rectDimResolveOrZeroOpt resolves a Rect[Dimension] against a single optional
// context.
func rectDimResolveOrZeroOpt(r Rect[Dimension], ctx optF32) Rect[float32] {
	return Rect[float32]{
		Left:   dimResolveOrZero(r.Left, ctx),
		Right:  dimResolveOrZero(r.Right, ctx),
		Top:    dimResolveOrZero(r.Top, ctx),
		Bottom: dimResolveOrZero(r.Bottom, ctx),
	}
}
