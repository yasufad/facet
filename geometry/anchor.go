package geometry

// Anchor identifies a reference point on a box: one of the four corners or one
// of the four edge midpoints. It positions elements and reads corner values.
type Anchor int

const (
	TopLeft Anchor = iota
	TopRight
	BottomLeft
	BottomRight
	TopCentre
	BottomCentre
	LeftCentre
	RightCentre
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
	case TopCentre:
		return BottomCentre
	case BottomCentre:
		return TopCentre
	case LeftCentre:
		return RightCentre
	case RightCentre:
		return LeftCentre
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
		case TopCentre:
			return BottomCentre
		case BottomCentre:
			return TopCentre
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
	case LeftCentre:
		return RightCentre
	case RightCentre:
		return LeftCentre
	}
	return a
}

// IsCentre reports whether a is one of the four edge-midpoint anchors.
func (a Anchor) IsCentre() bool {
	return a == TopCentre || a == BottomCentre || a == LeftCentre || a == RightCentre
}
