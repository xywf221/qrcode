package qrcode

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
)

// PNG encodes the symbol as a PNG image and returns the bytes. The image uses a
// two-entry palette so it stays compact.
func (q *QR) PNG(opts ...RenderOption) ([]byte, error) {
	var buf bytes.Buffer
	if err := q.WritePNG(&buf, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WritePNG writes the symbol to w as a PNG image. By default each module is
// rendered at 4 pixels with a 4-module quiet zone; use WithScale, WithQuietZone
// and WithColor to change this.
func (q *QR) WritePNG(w io.Writer, opts ...RenderOption) error {
	img := q.renderImage(newRenderConfig(opts))
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(w, img)
}

// renderImage rasterises the symbol into a two-colour paletted image.
func (q *QR) renderImage(rc renderConfig) *image.Paletted {
	n := rc.extent(q.size)
	px := n * rc.scale
	img := image.NewPaletted(image.Rect(0, 0, px, px), color.Palette{rc.background, rc.foreground})
	for y := 0; y < q.size; y++ {
		for x := 0; x < q.size; x++ {
			dark := q.modules[y*q.size+x] != 0
			if dark == rc.invert {
				continue
			}
			x0 := (x + rc.quietZone) * rc.scale
			y0 := (y + rc.quietZone) * rc.scale
			for dy := 0; dy < rc.scale; dy++ {
				row := img.Pix[(y0+dy)*img.Stride+x0:]
				for dx := 0; dx < rc.scale; dx++ {
					row[dx] = 1
				}
			}
		}
	}
	return img
}
