//go:build windows

package d3d11

import (
	"errors"
	"strings"
	"testing"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/scene"
)

func TestShelfPacker(t *testing.T) {
	packer := newShelfPacker(100, 100)

	// First tile fits at (0, 0)
	x, y, ok := packer.allocate(30, 20)
	if !ok || x != 0 || y != 0 {
		t.Fatalf("expected (0, 0), got (%d, %d), ok=%v", x, y, ok)
	}

	// Second tile fits on same shelf at (30, 0)
	x, y, ok = packer.allocate(40, 25)
	if !ok || x != 30 || y != 0 {
		t.Fatalf("expected (30, 0), got (%d, %d), ok=%v", x, y, ok)
	}

	// Third tile doesn't fit on first shelf (30 + 40 + 50 > 100), starts new shelf at y=25
	x, y, ok = packer.allocate(50, 30)
	if !ok || x != 0 || y != 25 {
		t.Fatalf("expected (0, 25), got (%d, %d), ok=%v", x, y, ok)
	}

	// Oversized tile fails
	_, _, ok = packer.allocate(150, 10)
	if ok {
		t.Fatalf("expected oversized tile to fail")
	}

	// Zero or negative size fails
	_, _, ok = packer.allocate(0, 10)
	if ok {
		t.Fatalf("expected zero width to fail")
	}
	_, _, ok = packer.allocate(10, 0)
	if ok {
		t.Fatalf("expected zero height to fail")
	}

	// Reset resets cursor
	packer.reset()
	x, y, ok = packer.allocate(10, 10)
	if !ok || x != 0 || y != 0 {
		t.Fatalf("expected (0, 0) after reset, got (%d, %d)", x, y)
	}
}

// TestShelfPackerSessionOccupancy simulates a realistic glyph session — a
// mix of tile sizes in the range a proportional font produces at UI sizes —
// packed until several pages fill, so the page-growth decision in
// docs/audit.md item 4 has a number behind it instead of a guess. It drives
// the packer directly rather than atlasManager.upload, which needs a real
// D3D11 device and belongs with the facet_debug GPU tests.
func TestShelfPackerSessionOccupancy(t *testing.T) {
	sizes := []struct{ w, h int }{
		{8, 12}, {10, 14}, {12, 16}, {9, 13}, {14, 18},
		{16, 20}, {11, 15}, {20, 24}, {13, 17}, {24, 28},
	}
	const tileCount = 20000

	pages := []*shelfPacker{newShelfPacker(atlasPageWidth, atlasPageHeight)}
	current := pages[len(pages)-1]
	var handoffOccupancy []float64

	for i := 0; i < tileCount; i++ {
		sz := sizes[i%len(sizes)]
		if _, _, ok := current.allocate(sz.w, sz.h); ok {
			continue
		}

		handoffOccupancy = append(handoffOccupancy, current.occupancy())
		current = newShelfPacker(atlasPageWidth, atlasPageHeight)
		pages = append(pages, current)
		if _, _, ok := current.allocate(sz.w, sz.h); !ok {
			t.Fatalf("tile %dx%d does not fit an empty %dx%d page", sz.w, sz.h, atlasPageWidth, atlasPageHeight)
		}
	}

	var sum float64
	for _, o := range handoffOccupancy {
		sum += o
	}
	avg := 0.0
	if len(handoffOccupancy) > 0 {
		avg = sum / float64(len(handoffOccupancy))
	}

	t.Logf("session of %d tiles packed into %d pages; average occupancy of a page when the next one was started: %.1f%%", tileCount, len(pages), avg*100)
	t.Logf("final page occupancy: %.1f%%", current.occupancy()*100)
}

// TestAtlasTileGeneration exercises the encoding docs/audit.md item 3 asks
// for without a GPU device: a tile is valid against the page it names until
// that page's generation advances (what ClearAtlas does), then invalid.
func TestAtlasTileGeneration(t *testing.T) {
	page := &atlasPage{kind: scene.TextureMonochrome, packer: newShelfPacker(atlasPageWidth, atlasPageHeight)}
	m := &atlasManager{monoPages: []*atlasPage{page}, nextTileID: 1}

	tile := scene.AtlasTile{
		TextureID: scene.AtlasTextureID{Index: 0, Kind: scene.TextureMonochrome},
		TileID:    makeTileID(page.generation, m.nextTileID),
		Bounds:    geometry.NewBounds(geometry.NewPoint[geometry.DevicePixels](0, 0), geometry.NewSize[geometry.DevicePixels](4, 4)),
	}

	if !m.tileValid(tile) {
		t.Fatalf("tile should be valid against the page it was just minted from")
	}

	page.generation++ // what atlasManager.clear does to every page of a kind

	if m.tileValid(tile) {
		t.Fatalf("tile handed out before the page's generation advanced should be invalid after")
	}

	// A tile naming a page index that no longer (or never did) exist is
	// invalid rather than a panic or an out-of-range read.
	stray := tile
	stray.TextureID.Index = 7
	if m.tileValid(stray) {
		t.Fatalf("tile naming a nonexistent page should be invalid")
	}
}

func TestNewInvalidSurface(t *testing.T) {
	_, err := New(0, geometry.NewSize[geometry.DevicePixels](100, 100), render.Options{})
	if err == nil {
		t.Fatalf("expected error for surface handle 0, got nil")
	}
}

// TestCreateResourceErrorDistinctMessages pins the fix for the round 02
// defect: pipeline.go collapsed "failing HRESULT" and "S_OK with a null
// out-pointer" into one check and one message, so a call that succeeded
// but returned no object reported as "failed with hr=0x00000000" — a
// successful call reported as a failure. Two agents lost an hour each
// attributing that message. The two conditions must produce distinct
// messages: "failed with hr=0x..." for a rejected call, and "returned
// null object with hr=0x..." for resource exhaustion under contention.
//
// This is a break test: reverting createResourceError to the old
// single-message format makes the null-object assertion fail, because
// "create pixel shader: hr=0x00000000" does not contain "returned null
// object".
func TestCreateResourceErrorDistinctMessages(t *testing.T) {
	failErr := createResourceError("create pixel shader", 0x8007000e, false)
	nullErr := createResourceError("create pixel shader", 0, true)

	if failErr == nil {
		t.Fatal("failing HRESULT should produce an error")
	}
	if nullErr == nil {
		t.Fatal("null object with S_OK should produce an error")
	}

	if !strings.Contains(nullErr.Error(), "returned null object") {
		t.Fatalf("null object with S_OK should say \"returned null object\", got: %s", nullErr)
	}
	if !strings.Contains(failErr.Error(), "failed with") {
		t.Fatalf("failing HRESULT should say \"failed with\", got: %s", failErr)
	}

	if failErr.Error() == nullErr.Error() {
		t.Fatalf("two distinct conditions produced the same message:\n  fail: %s\n  null: %s", failErr, nullErr)
	}
}

// TestClassifyDeviceErrorNoAdapter pins the ErrNoAdapter sentinel: the two
// HRESULTs that mean "no usable adapter" (DXGI_ERROR_UNSUPPORTED and E_FAIL)
// must wrap render.ErrNoAdapter so the four downstream consumers — window,
// the readback tests, the integration test and element's joint test — can
// branch on it with errors.Is and skip rather than fail. A generic failure
// HRESULT must NOT wrap it, and a successful call must return nil. This is
// a break test: removing the %w wrapping or changing the HRESULT set makes
// the errors.Is assertions fail.
func TestClassifyDeviceErrorNoAdapter(t *testing.T) {
	for _, hr := range []uintptr{dxgiErrorUnsupported, eFail} {
		err := classifyDeviceError(hr, nil, nil)
		if err == nil {
			t.Fatalf("hr=0x%08x with nil device should be an error", uint32(hr))
		}
		if !errors.Is(err, render.ErrNoAdapter) {
			t.Fatalf("hr=0x%08x should wrap ErrNoAdapter, got: %v", uint32(hr), err)
		}
	}

	// A generic failure HRESULT must not be mistaken for "no adapter".
	genericErr := classifyDeviceError(0x8007000e, nil, nil)
	if genericErr == nil {
		t.Fatal("E_OUTOFMEMORY with nil device should be an error")
	}
	if errors.Is(genericErr, render.ErrNoAdapter) {
		t.Fatalf("E_OUTOFMEMORY should not wrap ErrNoAdapter, got: %v", genericErr)
	}

	// S_OK with non-nil device and context is success, not an error.
	dev := &comObject{}
	ctx := &comObject{}
	if err := classifyDeviceError(0, dev, ctx); err != nil {
		t.Fatalf("S_OK with non-nil device and context should not be an error, got: %v", err)
	}

	// S_OK with a nil device is a failure (resource exhaustion), but not
	// "no adapter" — it does not wrap the sentinel.
	nullErr := classifyDeviceError(0, nil, ctx)
	if nullErr == nil {
		t.Fatal("S_OK with nil device should be an error")
	}
	if errors.Is(nullErr, render.ErrNoAdapter) {
		t.Fatalf("S_OK with nil device should not wrap ErrNoAdapter, got: %v", nullErr)
	}
}
