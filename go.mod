module github.com/yasufad/facet

go 1.26.5

require (
	github.com/ebitengine/purego v0.9.1 // used by platform: Cocoa and objc_msgSend through purego on macOS, no cgo
	github.com/go-text/typesetting v0.3.4
	golang.org/x/image v0.23.0 // used by text: math/fixed is typesetting's fixed-point type, vector is the glyph rasteriser
)

require golang.org/x/sys v0.47.0

require golang.org/x/text v0.21.0 // indirect
