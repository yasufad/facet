package geometry

// Anchor identifies a reference point on a box: one of the four corners or one
// of the four edge midpoints. It positions elements and reads corner values.
type Anchor int

const (
	TopLeft Anchor = iota
	TopRight
	BottomLeft
	BottomRight
	TopCenter
	BottomCenter
	LeftCenter
	RightCenter
)

// Opposite returns the directly opposite anchor.
func (a Anchor) Opposite() Anchor {
	switch a {
	case TopLeft:
		return BottomRight
	case TopRight:
		return BottomLeft
	case BottomLeft:
		return TopRight
	case BottomRight:
		return TopLeft
	case TopCenter:
		return BottomCenter
	case BottomCenter:
		return TopCenter
	case LeftCenter:
		return RightCenter
	case RightCenter:
		return LeftCenter
	}
	return a
}

// OtherSideAlong returns the anchor across from a, moving along the given
// axis. Anchors that lie on that axis's perpendicular stay where they are.
func (a Anchor) OtherSideAlong(axis Axis) Anchor {
	if axis == Vertical {
		switch a {
		case TopLeft:
			return BottomLeft
		case TopRight:
			return BottomRight
		case BottomLeft:
			return TopLeft
		case BottomRight:
			return TopRight
		case TopCenter:
			return BottomCenter
		case BottomCenter:
			return TopCenter
		}
		return a
	}
	switch a {
	case TopLeft:
		return TopRight
	case TopRight:
		return TopLeft
	case BottomLeft:
		return BottomRight
	case BottomRight:
		return BottomLeft
	case LeftCenter:
		return RightCenter
	case RightCenter:
		return LeftCenter
	}
	return a
}

// IsCenter reports whether a is one of the four edge-midpoint anchors.
func (a Anchor) IsCenter() bool {
	return a == TopCenter || a == BottomCenter || a == LeftCenter || a == RightCenter
}
