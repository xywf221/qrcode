package qrcode

import (
	"image/color"
	"io"
	"strconv"
)

// SVG renders the symbol as a standalone SVG document.
func (q *QR) SVG(opts ...RenderOption) string {
	rc := newRenderConfig(opts)
	n := rc.extent(q.size)
	px := n * rc.scale

	var b []byte
	b = append(b, `<svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 `...)
	b = strconv.AppendInt(b, int64(n), 10)
	b = append(b, ' ')
	b = strconv.AppendInt(b, int64(n), 10)
	b = append(b, `" width="`...)
	b = strconv.AppendInt(b, int64(px), 10)
	b = append(b, `" height="`...)
	b = strconv.AppendInt(b, int64(px), 10)
	b = append(b, `" shape-rendering="crispEdges">`...)

	b = append(b, `<rect width="100%" height="100%" fill="`...)
	b = appendHex(b, rc.background)
	b = append(b, `"/>`...)

	b = append(b, `<path fill="`...)
	b = appendHex(b, rc.foreground)
	b = append(b, `" d="`...)
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
			b = append(b, 'M')
			b = strconv.AppendInt(b, int64(start), 10)
			b = append(b, ' ')
			b = strconv.AppendInt(b, int64(y), 10)
			b = append(b, 'h')
			b = strconv.AppendInt(b, int64(x-start), 10)
			b = append(b, 'v', '1', 'h', '-')
			b = strconv.AppendInt(b, int64(x-start), 10)
			b = append(b, 'z')
		}
	}
	b = append(b, `"/></svg>`...)
	return string(b)
}

// WriteSVG writes the symbol to w as an SVG document.
func (q *QR) WriteSVG(w io.Writer, opts ...RenderOption) error {
	_, err := io.WriteString(w, q.SVG(opts...))
	return err
}

func appendHex(b []byte, c color.Color) []byte {
	r, g, bl, _ := c.RGBA()
	b = append(b, '#')
	b = appendByteHex(b, uint8(r>>8))
	b = appendByteHex(b, uint8(g>>8))
	b = appendByteHex(b, uint8(bl>>8))
	return b
}

func appendByteHex(b []byte, v uint8) []byte {
	const hexdigits = "0123456789abcdef"
	return append(b, hexdigits[v>>4], hexdigits[v&0xf])
}
