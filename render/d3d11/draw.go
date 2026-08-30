//go:build windows

package d3d11

import (
	"fmt"
	"unsafe"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/scene"
)

type dynamicBuffer struct {
	device   *comObject
	context  *comObject
	buffer   *comObject
	capacity int
}

func newDynamicBuffer(device, context *comObject) *dynamicBuffer {
	return &dynamicBuffer{
		device:  device,
		context: context,
	}
}

func (b *dynamicBuffer) ensureSize(size int) error {
	if b.capacity >= size && b.buffer != nil {
		return nil
	}

	newCap := size
	if newCap < 65536 {
		newCap = 65536
	}
	if newCap < b.capacity*2 {
		newCap = b.capacity * 2
	}

	if b.buffer != nil {
		b.buffer.Release()
		b.buffer = nil
	}

	desc := d3d11BufferDesc{
		ByteWidth:      uint32(newCap),
		Usage:          d3d11UsageDynamic,
		BindFlags:      d3d11BindVertexBuffer,
		CPUAccessFlags: d3d11CpuAccessWrite,
	}

	// ID3D11Device::CreateBuffer is vtbl index 3
	r1, _, _ := b.device.call(3, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&b.buffer)))
	if int32(r1) < 0 || b.buffer == nil {
		return fmt.Errorf("create dynamic buffer (%d bytes): hr=0x%08x", newCap, uint32(r1))
	}

	b.capacity = newCap
	return nil
}

func (b *dynamicBuffer) write(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if err := b.ensureSize(len(data)); err != nil {
		return err
	}

	var mapped d3d11MappedSubresource
	// ID3D11DeviceContext::Map is vtbl index 14
	r1, _, _ := b.context.call(14,
		uintptr(unsafe.Pointer(b.buffer)),
		0,
		uintptr(d3d11MapWriteDiscard),
		0,
		uintptr(unsafe.Pointer(&mapped)),
	)
	if int32(r1) < 0 || mapped.PData == nil {
		return fmt.Errorf("map buffer: hr=0x%08x", uint32(r1))
	}

	// Sound because mapped.PData points to the GPU-mapped memory of capacity >= len(data).
	copy(unsafe.Slice((*byte)(mapped.PData), len(data)), data)

	// ID3D11DeviceContext::Unmap is vtbl index 15
	b.context.call(15, uintptr(unsafe.Pointer(b.buffer)), 0)
	return nil
}

func (b *dynamicBuffer) release() {
	if b.buffer != nil {
		b.buffer.Release()
		b.buffer = nil
	}
	b.capacity = 0
}

func boundsToFloats(b geometry.Bounds[geometry.ScaledPixels]) [4]float32 {
	return [4]float32{
		b.Origin.X.Float32(),
		b.Origin.Y.Float32(),
		(b.Origin.X + b.Size.Width).Float32(),
		(b.Origin.Y + b.Size.Height).Float32(),
	}
}

func maskToFloats(m scene.ContentMask[geometry.ScaledPixels]) [4]float32 {
	if m.Bounds.IsEmpty() {
		return [4]float32{0, 0, 0, 0}
	}
	return boundsToFloats(m.Bounds)
}

func (r *d3d11Renderer) drawQuadBatch(quads []scene.Quad) error {
	if len(quads) == 0 {
		return nil
	}

	type quadInstance struct {
		bounds       [4]float32
		contentMask  [4]float32
		bgColor      [4]float32
		borderColor  [4]float32
		cornerRadii  [4]float32
		borderWidths [4]float32
		borderStyle  uint32
		pad0         uint32
		pad1         uint32
		pad2         uint32
	}

	data := make([]quadInstance, len(quads))
	for i, q := range quads {
		bg := q.Background.Premultiply()
		bc := q.BorderColour.Premultiply()
		data[i] = quadInstance{
			bounds:       boundsToFloats(q.Bounds),
			contentMask:  maskToFloats(q.ContentMask),
			bgColor:      [4]float32{bg.R, bg.G, bg.B, bg.A},
			borderColor:  [4]float32{bc.R, bc.G, bc.B, bc.A},
			cornerRadii:  [4]float32{q.CornerRadii.TopLeft.Float32(), q.CornerRadii.TopRight.Float32(), q.CornerRadii.BottomRight.Float32(), q.CornerRadii.BottomLeft.Float32()},
			borderWidths: [4]float32{q.BorderWidths.Top.Float32(), q.BorderWidths.Right.Float32(), q.BorderWidths.Bottom.Float32(), q.BorderWidths.Left.Float32()},
			borderStyle:  uint32(q.BorderStyle),
		}
	}

	// Sound because quadInstance contains only numeric fields with explicit alignment.
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*int(unsafe.Sizeof(quadInstance{})))
	if err := r.dynamicBuffer.write(byteSlice); err != nil {
		return err
	}

	// Bind quad pipeline
	r.setShader(&r.pipeline.quadShader)
	r.bindVertexBuffer(r.dynamicBuffer.buffer, r.pipeline.quadShader.stride)

	// DrawInstanced: 6 vertices per quad, len(quads) instances
	// ID3D11DeviceContext::DrawInstanced is vtbl index 21
	r.context.call(21, 6, uintptr(len(quads)), 0, 0)
	return nil
}

func (r *d3d11Renderer) drawShadowBatch(shadows []scene.Shadow) error {
	if len(shadows) == 0 {
		return nil
	}

	type shadowInstance struct {
		bounds     [4]float32
		radii      [4]float32
		mask       [4]float32
		color      [4]float32
		elemBounds [4]float32
		elemRadii  [4]float32
		blurRadius float32
		inset      uint32
		pad0       uint32
		pad1       uint32
	}

	data := make([]shadowInstance, len(shadows))
	for i, s := range shadows {
		c := s.Colour.Premultiply()
		inset := uint32(0)
		if s.Inset {
			inset = 1
		}
		data[i] = shadowInstance{
			bounds:     boundsToFloats(s.Bounds),
			radii:      [4]float32{s.CornerRadii.TopLeft.Float32(), s.CornerRadii.TopRight.Float32(), s.CornerRadii.BottomRight.Float32(), s.CornerRadii.BottomLeft.Float32()},
			mask:       maskToFloats(s.ContentMask),
			color:      [4]float32{c.R, c.G, c.B, c.A},
			elemBounds: boundsToFloats(s.ElementBounds),
			elemRadii:  [4]float32{s.ElementCornerRadii.TopLeft.Float32(), s.ElementCornerRadii.TopRight.Float32(), s.ElementCornerRadii.BottomRight.Float32(), s.ElementCornerRadii.BottomLeft.Float32()},
			blurRadius: s.BlurRadius.Float32(),
			inset:      inset,
		}
	}

	// Sound because shadowInstance is pure numerical data.
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*int(unsafe.Sizeof(shadowInstance{})))
	if err := r.dynamicBuffer.write(byteSlice); err != nil {
		return err
	}

	r.setShader(&r.pipeline.shadowShader)
	r.bindVertexBuffer(r.dynamicBuffer.buffer, r.pipeline.shadowShader.stride)
	// ID3D11DeviceContext::DrawInstanced is vtbl index 21
	r.context.call(21, 6, uintptr(len(shadows)), 0, 0)
	return nil
}

func (r *d3d11Renderer) drawMonoSpriteBatch(sprites []scene.MonochromeSprite, textureID scene.AtlasTextureID) error {
	if len(sprites) == 0 {
		return nil
	}

	srv := r.atlas.getSRV(textureID)
	if srv == nil {
		return nil
	}

	type monoInstance struct {
		bounds       [4]float32
		mask         [4]float32
		color        [4]float32
		tile         [4]float32
		transformMat [4]float32
		transformTx  [2]float32
		pad0         [2]float32
	}

	data := make([]monoInstance, len(sprites))
	for i, sp := range sprites {
		c := sp.Colour.Premultiply()
		tb := sp.Tile.Bounds
		data[i] = monoInstance{
			bounds:       boundsToFloats(sp.Bounds),
			mask:         maskToFloats(sp.ContentMask),
			color:        [4]float32{c.R, c.G, c.B, c.A},
			tile:         [4]float32{float32(tb.Origin.X), float32(tb.Origin.Y), float32(tb.Origin.X + tb.Size.Width), float32(tb.Origin.Y + tb.Size.Height)},
			transformMat: [4]float32{sp.Transformation.RotationScale[0][0], sp.Transformation.RotationScale[0][1], sp.Transformation.RotationScale[1][0], sp.Transformation.RotationScale[1][1]},
			transformTx:  [2]float32{sp.Transformation.Translation[0], sp.Transformation.Translation[1]},
		}
	}

	// Sound because monoInstance is pure numerical data.
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*int(unsafe.Sizeof(monoInstance{})))
	if err := r.dynamicBuffer.write(byteSlice); err != nil {
		return err
	}

	// PSSetShaderResources is vtbl index 8
	r.context.call(8, 0, 1, uintptr(unsafe.Pointer(&srv)))

	r.setShader(&r.pipeline.monoShader)
	r.bindVertexBuffer(r.dynamicBuffer.buffer, r.pipeline.monoShader.stride)
	// ID3D11DeviceContext::DrawInstanced is vtbl index 21
	r.context.call(21, 6, uintptr(len(sprites)), 0, 0)
	return nil
}

func (r *d3d11Renderer) drawPolySpriteBatch(sprites []scene.PolychromeSprite, textureID scene.AtlasTextureID) error {
	if len(sprites) == 0 {
		return nil
	}

	srv := r.atlas.getSRV(textureID)
	if srv == nil {
		return nil
	}

	type polyInstance struct {
		bounds    [4]float32
		mask      [4]float32
		tile      [4]float32
		radii     [4]float32
		opacity   float32
		grayscale uint32
		pad0      [2]float32
	}

	data := make([]polyInstance, len(sprites))
	for i, sp := range sprites {
		tb := sp.Tile.Bounds
		gray := uint32(0)
		if sp.Grayscale {
			gray = 1
		}
		data[i] = polyInstance{
			bounds:    boundsToFloats(sp.Bounds),
			mask:      maskToFloats(sp.ContentMask),
			tile:      [4]float32{float32(tb.Origin.X), float32(tb.Origin.Y), float32(tb.Origin.X + tb.Size.Width), float32(tb.Origin.Y + tb.Size.Height)},
			radii:     [4]float32{sp.CornerRadii.TopLeft.Float32(), sp.CornerRadii.TopRight.Float32(), sp.CornerRadii.BottomRight.Float32(), sp.CornerRadii.BottomLeft.Float32()},
			opacity:   sp.Opacity,
			grayscale: gray,
		}
	}

	// Sound because polyInstance is pure numerical data.
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*int(unsafe.Sizeof(polyInstance{})))
	if err := r.dynamicBuffer.write(byteSlice); err != nil {
		return err
	}

	// PSSetShaderResources is vtbl index 8
	r.context.call(8, 0, 1, uintptr(unsafe.Pointer(&srv)))

	r.setShader(&r.pipeline.polyShader)
	r.bindVertexBuffer(r.dynamicBuffer.buffer, r.pipeline.polyShader.stride)
	// ID3D11DeviceContext::DrawInstanced is vtbl index 21
	r.context.call(21, 6, uintptr(len(sprites)), 0, 0)
	return nil
}

func (r *d3d11Renderer) drawPathBatch(paths []scene.Path[geometry.ScaledPixels]) error {
	totalVerts := 0
	for _, p := range paths {
		totalVerts += len(p.Vertices)
	}
	if totalVerts == 0 {
		return nil
	}

	type pathVertex struct {
		pos   [2]float32
		st    [2]float32
		mask  [4]float32
		color [4]float32
	}

	data := make([]pathVertex, 0, totalVerts)
	for _, p := range paths {
		c := p.Colour.Premultiply()
		colorFloats := [4]float32{c.R, c.G, c.B, c.A}
		for _, v := range p.Vertices {
			mask := maskToFloats(v.ContentMask)
			if mask[0] == 0 && mask[1] == 0 && mask[2] == 0 && mask[3] == 0 {
				mask = maskToFloats(p.ContentMask)
			}
			data = append(data, pathVertex{
				pos:   [2]float32{v.XYPosition.X.Float32(), v.XYPosition.Y.Float32()},
				st:    [2]float32{v.STPosition.X, v.STPosition.Y},
				mask:  mask,
				color: colorFloats,
			})
		}
	}

	// Sound because pathVertex is pure numerical data.
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*int(unsafe.Sizeof(pathVertex{})))
	if err := r.dynamicBuffer.write(byteSlice); err != nil {
		return err
	}

	r.setShader(&r.pipeline.pathShader)
	r.bindVertexBuffer(r.dynamicBuffer.buffer, r.pipeline.pathShader.stride)

	// ID3D11DeviceContext::Draw is vtbl index 13
	r.context.call(13, uintptr(len(data)), 0)
	return nil
}

func (r *d3d11Renderer) drawUnderlineBatch(underlines []scene.Underline) error {
	if len(underlines) == 0 {
		return nil
	}

	type underlineInstance struct {
		bounds    [4]float32
		mask      [4]float32
		color     [4]float32
		thickness float32
		wavy      uint32
		pad0      [2]float32
	}

	data := make([]underlineInstance, len(underlines))
	for i, u := range underlines {
		c := u.Colour.Premultiply()
		wavy := uint32(0)
		if u.Wavy {
			wavy = 1
		}
		data[i] = underlineInstance{
			bounds:    boundsToFloats(u.Bounds),
			mask:      maskToFloats(u.ContentMask),
			color:     [4]float32{c.R, c.G, c.B, c.A},
			thickness: u.Thickness.Float32(),
			wavy:      wavy,
		}
	}

	// Sound because underlineInstance is pure numerical data.
	byteSlice := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*int(unsafe.Sizeof(underlineInstance{})))
	if err := r.dynamicBuffer.write(byteSlice); err != nil {
		return err
	}

	r.setShader(&r.pipeline.underlineShader)
	r.bindVertexBuffer(r.dynamicBuffer.buffer, r.pipeline.underlineShader.stride)
	// ID3D11DeviceContext::DrawInstanced is vtbl index 21
	r.context.call(21, 6, uintptr(len(underlines)), 0, 0)
	return nil
}
