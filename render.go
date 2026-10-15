package qrcode

import "image/color"

// renderConfig holds resolved options common to every renderer.
type renderConfig struct {
	scale      int
	quietZone  int
	invert     bool
	foreground color.Color
	background color.Color
}

// A RenderOption configures image and text rendering.
type RenderOption func(*renderConfig)

// WithScale sets the size of one module in pixels (PNG, SVG) or, for SVG only,
// the nominal unit. Values below 1 are clamped to 1. Default is 4.
func WithScale(px int) RenderOption { return func(c *renderConfig) { c.scale = px } }

// WithQuietZone sets the width of the light margin around the symbol, in
// modules. The default is 4, the minimum required by the specification.
func WithQuietZone(modules int) RenderOption { return func(c *renderConfig) { c.quietZone = modules } }

// WithColor sets the foreground (dark-module) and background colours.
func WithColor(fg, bg color.Color) RenderOption {
	return func(c *renderConfig) { c.foreground, c.background = fg, bg }
}

// WithInvert swaps dark and light modules. It is useful for terminals with a
// dark background when rendering with Terminal. The default is false.
func WithInvert(b bool) RenderOption { return func(c *renderConfig) { c.invert = b } }

func newRenderConfig(opts []RenderOption) renderConfig {
	c := renderConfig{
		scale:      4,
		quietZone:  4,
		foreground: color.Black,
		background: color.White,
	}
	for _, o := range opts {
		o(&c)
	}
	if c.scale < 1 {
		c.scale = 1
	}
	if c.quietZone < 0 {
		c.quietZone = 0
	}
	if c.foreground == nil {
		c.foreground = color.Black
	}
	if c.background == nil {
		c.background = color.White
	}
	return c
}

// extent returns the total number of modules per side including the quiet zone.
func (c renderConfig) extent(size int) int { return size + 2*c.quietZone }

// darkAt reports whether the module at (x, y) in extended coordinates (quiet
// zone included) is to be drawn in the foreground colour. Coordinates in the
// quiet zone are light; WithInvert swaps the result.
func (q *QR) darkAt(c renderConfig, x, y int) bool {
	x -= c.quietZone
	y -= c.quietZone
	dark := uint(x) < uint(q.size) && uint(y) < uint(q.size) && q.modules[y*q.size+x] != 0
	return dark != c.invert
}
