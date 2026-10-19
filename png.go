package qrcode

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image/color"
	"io"
	"sync"
)

var (
	pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

	// A PNG is always compressed with zlib; pooling the writer keeps the flate
	// compressor's buffers alive between renders instead of rebuilding (and
	// re-allocating roughly 1.3 MB) on every call, which is what makes the
	// standard image/png encoder slow for these tiny two-colour images.
	zlibPool = sync.Pool{New: func() any {
		zw, _ := zlib.NewWriterLevel(io.Discard, zlib.BestSpeed)
		return zw
	}}

	// pngRawPool reuses the uncompressed scanline buffer, the largest per-render
	// allocation (a few KB), across calls.
	pngRawPool = sync.Pool{New: func() any { return new([]byte) }}
)

// PNG encodes the symbol as a PNG image and returns the bytes. The image is a
// two-colour indexed image at one bit per pixel, so it stays compact.
func (q *QR) PNG(opts ...RenderOption) ([]byte, error) {
	return q.pngBytes(newRenderConfig(opts)), nil
}

// WritePNG writes the symbol to w as a PNG image. By default each module is
// rendered at 4 pixels with a 4-module quiet zone; use WithScale, WithQuietZone,
// WithColor and WithInvert to change this.
func (q *QR) WritePNG(w io.Writer, opts ...RenderOption) error {
	_, err := w.Write(q.pngBytes(newRenderConfig(opts)))
	return err
}

// pngBytes renders the symbol into a complete PNG document. It emits the four
// minimal indexed-PNG chunks by hand and deflates the scanlines with a pooled
// zlib writer.
func (q *QR) pngBytes(rc renderConfig) []byte {
	n := rc.extent(q.size)
	px := n * rc.scale
	bpr := (px + 7) / 8

	out := make([]byte, 0, 512)
	out = append(out, pngSignature...)

	var ihdr [13]byte
	binary.BigEndian.PutUint32(ihdr[0:], uint32(px))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(px))
	ihdr[8] = 1 // bit depth
	ihdr[9] = 3 // colour type: indexed
	out = appendChunk(out, "IHDR", ihdr[:])

	plte, trns := pngPalette(rc.background, rc.foreground)
	out = appendChunk(out, "PLTE", plte[:])
	if trns != nil {
		out = appendChunk(out, "tRNS", trns)
	}

	// Each scanline is a filter byte (0, None) followed by bpr packed bits.
	// Every module row is packed once and copied to its scale output rows, so
	// the pixel work scales with the module count, not the pixel count.
	rawp := pngRawPool.Get().(*[]byte)
	raw := *rawp
	if cap(raw) < px*(1+bpr) {
		raw = make([]byte, px*(1+bpr))
	} else {
		raw = raw[:px*(1+bpr)]
		clear(raw) // quiet-zone scanlines must stay zero
	}
	row := make([]byte, bpr)
	for my := 0; my < q.size; my++ {
		clear(row)
		mrow := q.modules[my*q.size:]
		for mx := 0; mx < q.size; mx++ {
			if (mrow[mx] != 0) == rc.invert {
				continue
			}
			base := (rc.quietZone + mx) * rc.scale
			for k := 0; k < rc.scale; k++ {
				bit := base + k
				row[bit>>3] |= 0x80 >> uint(bit&7)
			}
		}
		oy := (rc.quietZone + my) * rc.scale
		for dy := 0; dy < rc.scale; dy++ {
			off := (oy+dy)*(1+bpr) + 1
			copy(raw[off:off+bpr], row)
		}
	}
	out = appendChunk(out, "IDAT", deflate(raw))
	*rawp = raw
	pngRawPool.Put(rawp)
	out = appendChunk(out, "IEND", nil)
	return out
}

// appendChunk appends a PNG chunk: length, type, data, and the CRC-32 of the
// type and data.
func appendChunk(b []byte, typ string, data []byte) []byte {
	var head [8]byte
	binary.BigEndian.PutUint32(head[0:4], uint32(len(data)))
	copy(head[4:8], typ)
	b = append(b, head[:]...)
	b = append(b, data...)
	crc := crc32.Update(0, crc32.IEEETable, []byte(typ))
	crc = crc32.Update(crc, crc32.IEEETable, data)
	var tail [4]byte
	binary.BigEndian.PutUint32(tail[:], crc)
	return append(b, tail[:]...)
}

// deflate zlib-compresses raw using a pooled writer.
func deflate(raw []byte) []byte {
	var z bytes.Buffer
	z.Grow(len(raw)/2 + 16)
	zw := zlibPool.Get().(*zlib.Writer)
	zw.Reset(&z)
	_, _ = zw.Write(raw)
	_ = zw.Close()
	zw.Reset(io.Discard) // release the buffer before returning the writer
	zlibPool.Put(zw)
	return z.Bytes()
}

// pngPalette converts the background and foreground colours to indexed-PNG
// entries. PLTE holds straight (unassociated) RGB; when either colour is
// translucent, tRNS holds the alpha values, truncated after the last
// translucent entry as the standard library's encoder does.
func pngPalette(bg, fg color.Color) ([6]byte, []byte) {
	c0 := color.NRGBAModel.Convert(bg).(color.NRGBA)
	c1 := color.NRGBAModel.Convert(fg).(color.NRGBA)
	var plte [6]byte
	plte[0], plte[1], plte[2] = c0.R, c0.G, c0.B
	plte[3], plte[4], plte[5] = c1.R, c1.G, c1.B
	switch {
	case c1.A != 0xff:
		return plte, []byte{c0.A, c1.A}
	case c0.A != 0xff:
		return plte, []byte{c0.A}
	default:
		return plte, nil
	}
}
