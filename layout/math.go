// Ported from Taffy src/util/math.rs (MIT).
//
// Taffy's MaybeMath trait computes minima, maxima, clamps and sums where one
// operand may be undefined (None) or an indefinite available-space constraint.
// Each trait implementation becomes a free function here.
package layout

// --- optF32 vs optF32 -> optF32 ---

func optMaybeMin(a, b optF32) optF32 {
	if !a.isSome() {
		return none()
	}
	if !b.isSome() {
		return a
	}
	return some(f32Min(a.v, b.v))
}

func optMaybeMax(a, b optF32) optF32 {
	if !a.isSome() {
		return none()
	}
	if !b.isSome() {
		return a
	}
	return some(f32Max(a.v, b.v))
}

func optMaybeClamp(v, lo, hi optF32) optF32 {
	if !v.isSome() {
		return none()
	}
	x := v.v
	// Taffy applies min(max) then max(min), so min wins when min > max.
	if hi.isSome() {
		x = f32Min(x, hi.v)
	}
	if lo.isSome() {
		x = f32Max(x, lo.v)
	}
	return some(x)
}

func optMaybeAdd(a, b optF32) optF32 {
	if !a.isSome() || !b.isSome() {
		return none()
	}
	return some(a.v + b.v)
}

func optMaybeSub(a, b optF32) optF32 {
	if !a.isSome() || !b.isSome() {
		return none()
	}
	return some(a.v - b.v)
}

// --- optF32 vs float32 -> optF32 ---

func optMinF32(a optF32, b float32) optF32 {
	if !a.isSome() {
		return none()
	}
	return some(f32Min(a.v, b))
}

func optMaxF32(a optF32, b float32) optF32 {
	if !a.isSome() {
		return none()
	}
	return some(f32Max(a.v, b))
}

func optClampF32(a optF32, lo, hi float32) optF32 {
	if !a.isSome() {
		return none()
	}
	// Taffy: val.min(max).max(min) — min wins when min > max.
	return some(f32Max(f32Min(a.v, hi), lo))
}

func optAddF32(a optF32, b float32) optF32 {
	if !a.isSome() {
		return none()
	}
	return some(a.v + b)
}

func optSubF32(a optF32, b float32) optF32 {
	if !a.isSome() {
		return none()
	}
	return some(a.v - b)
}

// --- float32 vs optF32 -> float32 ---

func f32MaybeMin(a float32, b optF32) float32 {
	if b.isSome() {
		return f32Min(a, b.v)
	}
	return a
}

func f32MaybeMax(a float32, b optF32) float32 {
	if b.isSome() {
		return f32Max(a, b.v)
	}
	return a
}

func f32MaybeClamp(a float32, lo, hi optF32) float32 {
	// Taffy applies min(max) then max(min), so min wins when min > max.
	if hi.isSome() {
		a = f32Min(a, hi.v)
	}
	if lo.isSome() {
		a = f32Max(a, lo.v)
	}
	return a
}

func f32MaybeAdd(a float32, b optF32) float32 {
	if b.isSome() {
		return a + b.v
	}
	return a
}

func f32MaybeSub(a float32, b optF32) float32 {
	if b.isSome() {
		return a - b.v
	}
	return a
}

// --- availableSpace vs float32 -> availableSpace ---

func availMaybeMinF32(a availableSpace, b float32) availableSpace {
	switch a.kind {
	case availableDefinite:
		return definiteAvail(f32Min(a.val, b))
	default:
		return definiteAvail(b)
	}
}

func availMaybeMaxF32(a availableSpace, b float32) availableSpace {
	switch a.kind {
	case availableDefinite:
		return definiteAvail(f32Max(a.val, b))
	default:
		return a
	}
}

func availMaybeClampF32(a availableSpace, lo, hi float32) availableSpace {
	if a.kind == availableDefinite {
		// Taffy: val.min(max).max(min) — min wins when min > max.
		return definiteAvail(f32Max(f32Min(a.val, hi), lo))
	}
	return a
}

func availMaybeAddF32(a availableSpace, b float32) availableSpace {
	if a.kind == availableDefinite {
		return definiteAvail(a.val + b)
	}
	return a
}

func availMaybeSubF32(a availableSpace, b float32) availableSpace {
	if a.kind == availableDefinite {
		return definiteAvail(a.val - b)
	}
	return a
}

// --- availableSpace vs optF32 -> availableSpace ---

func availMaybeMinOpt(a availableSpace, b optF32) availableSpace {
	if a.kind == availableDefinite {
		if b.isSome() {
			return definiteAvail(f32Min(a.val, b.v))
		}
		return definiteAvail(a.val)
	}
	if b.isSome() {
		return definiteAvail(b.v)
	}
	return a
}

func availMaybeMaxOpt(a availableSpace, b optF32) availableSpace {
	if a.kind == availableDefinite {
		if b.isSome() {
			return definiteAvail(f32Max(a.val, b.v))
		}
		return definiteAvail(a.val)
	}
	return a
}

func availMaybeClampOpt(a availableSpace, lo, hi optF32) availableSpace {
	if a.kind == availableDefinite {
		v := a.val
		// Taffy: val.min(max).max(min) — min wins when min > max.
		if hi.isSome() {
			v = f32Min(v, hi.v)
		}
		if lo.isSome() {
			v = f32Max(v, lo.v)
		}
		return definiteAvail(v)
	}
	return a
}

func availMaybeAddOpt(a availableSpace, b optF32) availableSpace {
	if a.kind == availableDefinite && b.isSome() {
		return definiteAvail(a.val + b.v)
	}
	return a
}

func availMaybeSubOpt(a availableSpace, b optF32) availableSpace {
	if a.kind == availableDefinite && b.isSome() {
		return definiteAvail(a.val - b.v)
	}
	return a
}

// --- Size[optF32] component-wise ---

func sizeOptMaybeMin(a, b Size[optF32]) Size[optF32] {
	return Size[optF32]{Width: optMaybeMin(a.Width, b.Width), Height: optMaybeMin(a.Height, b.Height)}
}

func sizeOptMaybeMax(a, b Size[optF32]) Size[optF32] {
	return Size[optF32]{Width: optMaybeMax(a.Width, b.Width), Height: optMaybeMax(a.Height, b.Height)}
}

func sizeOptMaybeClamp(v, lo, hi Size[optF32]) Size[optF32] {
	return Size[optF32]{
		Width:  optMaybeClamp(v.Width, lo.Width, hi.Width),
		Height: optMaybeClamp(v.Height, lo.Height, hi.Height),
	}
}

func sizeOptMaybeAdd(a, b Size[optF32]) Size[optF32] {
	return Size[optF32]{Width: optMaybeAdd(a.Width, b.Width), Height: optMaybeAdd(a.Height, b.Height)}
}

func sizeOptMaybeSub(a, b Size[optF32]) Size[optF32] {
	return Size[optF32]{Width: optMaybeSub(a.Width, b.Width), Height: optMaybeSub(a.Height, b.Height)}
}

// optOr returns a when defined, else b.
func optOr(a, b optF32) optF32 {
	if a.isSome() {
		return a
	}
	return b
}

// unwrapOrFunc returns the value when defined, else the callback's result.
func (o optF32) unwrapOrFunc(f func() float32) float32 {
	if o.isSome() {
		return o.v
	}
	return f()
}
