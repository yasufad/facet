package scene

import (
	"math"

	"github.com/yasufad/facet/geometry"
)

// TransformationMatrix is a 2D affine transform applied to a sprite before
// sampling its atlas tile. Glyphs usually carry the identity matrix; it is here
// for sprites that need rotation or scale.
//
// RotationScale is stored row-major: [[a, b], [c, d]] maps (x, y) to
// (a*x + b*y + tx, c*x + d*y + ty). Translation is in scaled pixels.
type TransformationMatrix struct {
	RotationScale [2][2]float32
	Translation   [2]float32
}

// IdentityMatrix is the transform that leaves a sprite untouched.
var IdentityMatrix = TransformationMatrix{
	RotationScale: [2][2]float32{{1, 0}, {0, 1}},
}

// Translate returns m with its translation offset by p.
func (m TransformationMatrix) Translate(p geometry.Point[geometry.ScaledPixels]) TransformationMatrix {
	return m.compose(TransformationMatrix{
		RotationScale: [2][2]float32{{1, 0}, {0, 1}},
		Translation:   [2]float32{p.X.Float32(), p.Y.Float32()},
	})
}

// Rotate returns m composed with a clockwise rotation of angle radians about
// the origin.
func (m TransformationMatrix) Rotate(angle float32) TransformationMatrix {
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	return m.compose(TransformationMatrix{
		RotationScale: [2][2]float32{{c, -s}, {s, c}},
	})
}

// Scale returns m composed with a uniform-per-axis scale.
func (m TransformationMatrix) Scale(sx, sy float32) TransformationMatrix {
	return m.compose(TransformationMatrix{
		RotationScale: [2][2]float32{{sx, 0}, {0, sy}},
	})
}

// Compose returns the matrix that applies other first, then m. It is matrix
// multiplication: result = m × other.
func (m TransformationMatrix) Compose(other TransformationMatrix) TransformationMatrix {
	return m.compose(other)
}

func (m TransformationMatrix) compose(other TransformationMatrix) TransformationMatrix {
	if other == IdentityMatrix {
		return m
	}
	rs := m.RotationScale
	or := other.RotationScale
	return TransformationMatrix{
		RotationScale: [2][2]float32{
			{
				rs[0][0]*or[0][0] + rs[0][1]*or[1][0],
				rs[0][0]*or[0][1] + rs[0][1]*or[1][1],
			},
			{
				rs[1][0]*or[0][0] + rs[1][1]*or[1][0],
				rs[1][0]*or[0][1] + rs[1][1]*or[1][1],
			},
		},
		Translation: [2]float32{
			m.Translation[0] + rs[0][0]*other.Translation[0] + rs[0][1]*other.Translation[1],
			m.Translation[1] + rs[1][0]*other.Translation[0] + rs[1][1]*other.Translation[1],
		},
	}
}

// Apply transforms a point by m, mainly useful in tests.
func (m TransformationMatrix) Apply(p geometry.Point[geometry.Pixels]) geometry.Point[geometry.Pixels] {
	x, y := p.X.Float32(), p.Y.Float32()
	rs := m.RotationScale
	return geometry.Point[geometry.Pixels]{
		X: geometry.Pixels(m.Translation[0] + rs[0][0]*x + rs[0][1]*y),
		Y: geometry.Pixels(m.Translation[1] + rs[1][0]*x + rs[1][1]*y),
	}
}
