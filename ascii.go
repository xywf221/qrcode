package qrcode

import (
	"io"
	"strings"
)

// Terminal renders the symbol as text, packing two module rows into each line
// with half-block characters so the result appears roughly square on a
// monospaced terminal. Dark modules are drawn as blocks; use WithInvert for
// terminals with a dark background.
func (q *QR) Terminal(opts ...RenderOption) string {
	rc := newRenderConfig(opts)
	n := rc.extent(q.size)
	var b strings.Builder
	b.Grow(n * (n/2 + 1) * 3)
	for y := 0; y < n; y += 2 {
		for x := 0; x < n; x++ {
			top := q.darkAt(rc, x, y)
			bot := y+1 < n && q.darkAt(rc, x, y+1)
			switch {
			case top && bot:
				b.WriteRune('█')
			case top:
				b.WriteRune('▀')
			case bot:
				b.WriteRune('▄')
			default:
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// WriteTerminal writes the symbol to w as terminal text.
func (q *QR) WriteTerminal(w io.Writer, opts ...RenderOption) error {
	_, err := io.WriteString(w, q.Terminal(opts...))
	return err
}
