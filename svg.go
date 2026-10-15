package qrcode

import (
	"image/color"
	"io"
	"strconv"
	"strings"
)

// SVG renders the symbol as a standalone SVG document.
func (q *QR) SVG(opts ...RenderOption) string {
	rc := newRenderConfig(opts)
	n := rc.extent(q.size)
	px := n * rc.scale

	// One allocation up front; String() then hands the buffer back without a
	// copy. strconv.Itoa's temporaries do not escape past WriteString, so they
	// are stack-allocated and add no per-integer heap traffic.
	var b strings.Builder
	b.Grow(n * (n/2 + 1) * 8)
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 `)
	b.WriteString(strconv.Itoa(n))
	b.WriteByte(' ')
	b.WriteString(strconv.Itoa(n))
	b.WriteString(`" width="`)
	b.WriteString(strconv.Itoa(px))
	b.WriteString(`" height="`)
	b.WriteString(strconv.Itoa(px))
	b.WriteString(`" shape-rendering="crispEdges">`)

	b.WriteString(`<rect width="100%" height="100%" fill="`)
	writeHex(&b, rc.background)
	b.WriteString(`"/>`)

	b.WriteString(`<path fill="`)
	writeHex(&b, rc.foreground)
	b.WriteString(`" d="`)
	for y := 0; y < n; y++ {
		for x := 0; x < n; {
			if !q.darkAt(rc, x, y) {
				x++
				continue
			}
			start := x
			for x < n && q.darkAt(rc, x, y) {
				x++
			}
			b.WriteByte('M')
			b.WriteString(strconv.Itoa(start))
			b.WriteByte(' ')
			b.WriteString(strconv.Itoa(y))
			b.WriteByte('h')
			b.WriteString(strconv.Itoa(x - start))
			b.WriteString("v1h-")
			b.WriteString(strconv.Itoa(x - start))
			b.WriteByte('z')
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String()
}

// WriteSVG writes the symbol to w as an SVG document.
func (q *QR) WriteSVG(w io.Writer, opts ...RenderOption) error {
	_, err := io.WriteString(w, q.SVG(opts...))
	return err
}

func writeHex(b *strings.Builder, c color.Color) {
	r, g, bl, _ := c.RGBA()
	b.WriteByte('#')
	writeByteHex(b, uint8(r>>8))
	writeByteHex(b, uint8(g>>8))
	writeByteHex(b, uint8(bl>>8))
}

func writeByteHex(b *strings.Builder, v uint8) {
	const hexdigits = "0123456789abcdef"
	b.WriteByte(hexdigits[v>>4])
	b.WriteByte(hexdigits[v&0xf])
}
