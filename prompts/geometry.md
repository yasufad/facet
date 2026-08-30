# Assignment: geometry

The package is built, reviewed and committed — fifteen files, all checks clean.
The units, the five generic shapes, `Axis` and `Anchor` are all in place, and the
device-pixel snapping is right: adjacent rectangles stay adjacent across fractional
edges at fractional scale factors, because the conversion snaps both edges and
derives the size instead of rounding origin and size independently. That is the
subtle one, and getting it unprompted is why the rest of the stack can trust this
package.

One gap remains. It is small.

## Point has no device-pixel conversion

`Size` has `SizeToDevicePixels` and `DeviceSizeToPixels`. `Bounds` has its pair.
`Point` has only `ScalePoint` and `MapPoint`.

So anyone converting a bare point — a cursor position, a scroll offset, a glyph
origin — has to write the rounding rule themselves through `MapPoint`. That is the
mistake distinct unit types exist to prevent, and it is exactly where the snapping
subtlety lives: a caller who rounds a point differently from the way `Bounds` rounds
its origin will be a pixel out, intermittently, in a way that looks like a renderer
bug.

Add the pair, rounding identically to the existing conversions, and test that a
point converted on its own lands where the same point converted as part of a
`Bounds` origin lands.

## Done when

    go build -o bin/ ./...
    go test ./geometry/
    go test ./internal/layering
    go vet ./geometry/
    gofmt -l $(go list -f '{{.Dir}}' ./...)

One conventional commit, staged by path.

## Not worth changing

The four `Zero*` constants add nothing over `Pixels(0)`, since these are numeric
types with a usable zero value. Leave them; churning the API of a package three
others already depend on costs more than the tidiness is worth.

## One habit worth dropping

The report described `app` and `layout` as having pre-existing failures. Both passed
at the time, and had for a while. In a tree with several agents that state is
minutes old at best — re-run another package before reporting it broken, or the
reviewer chases something that is not there.
