// Package layout is a port of Taffy's flexbox solver.
//
// This is a port, not a reimplementation: the algorithm and its edge cases come
// across intact from Taffy (https://github.com/DioxusLabs/taffy), and only the
// language changes. Taffy is MIT-licensed; every file in this package carries an
// attribution header naming the upstream file it came from, and Taffy's notice
// appears in the repository's NOTICE file.
//
// The package owns the flexbox algorithm and the tree plumbing it needs: node
// storage, the style input type, intrinsic-size caching, and the measure-function
// hook that lets a leaf report its own size. Grid is out of scope. The CSS Grid
// algorithm and its supporting types are deliberately absent.
//
// Taffy's own vocabulary comes across with the code: Dimension, LengthPercentage,
// LengthPercentageAuto, Size, Rect, Point and Line are defined here, not imported.
// They are what the algorithm and its test suite are written against, and
// substituting another package's types would make the port less faithful for no
// gain. The style package converts down into these types and the element package
// converts results back out; the vocabularies meet at that one boundary.
//
// Invariants: a behavioural difference from Taffy is a bug in this port, not a
// local improvement. The ordering of the sizing passes, auto margin resolution,
// min and max clamping and baseline alignment interact, and reordering them for
// readability breaks cases the test suite catches late. The structure stays
// recognisable against the original. Taffy's test suite is generated from browser
// behaviour and is ported alongside the code; it is the only real evidence the
// port is correct.
package layout
