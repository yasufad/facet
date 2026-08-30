//go:build windows

package d3d11

import (
	_ "embed"
)

//go:embed shaders/quad_vs.dxbc
var quadVSBytecode []byte

//go:embed shaders/quad_ps.dxbc
var quadPSBytecode []byte

//go:embed shaders/shadow_vs.dxbc
var shadowVSBytecode []byte

//go:embed shaders/shadow_ps.dxbc
var shadowPSBytecode []byte

//go:embed shaders/mono_sprite_vs.dxbc
var monoSpriteVSBytecode []byte

//go:embed shaders/mono_sprite_ps.dxbc
var monoSpritePSBytecode []byte

//go:embed shaders/poly_sprite_vs.dxbc
var polySpriteVSBytecode []byte

//go:embed shaders/poly_sprite_ps.dxbc
var polySpritePSBytecode []byte

//go:embed shaders/path_vs.dxbc
var pathVSBytecode []byte

//go:embed shaders/path_ps.dxbc
var pathPSBytecode []byte

//go:embed shaders/underline_vs.dxbc
var underlineVSBytecode []byte

//go:embed shaders/underline_ps.dxbc
var underlinePSBytecode []byte
