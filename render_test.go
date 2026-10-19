package qrcode

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// pngRefImage rasterises the symbol the way the reference (image/png) encoder
// does: a two-entry palette, foreground modules set to index 1, the quiet zone
// left as index 0. It is the independent oracle for TestPNGMatchesStdlib.
func pngRefImage(q *QR, rc renderConfig) *image.Paletted {
	n := rc.extent(q.size)
	px := n * rc.scale
	img := image.NewPaletted(image.Rect(0, 0, px, px), color.Palette{rc.background, rc.foreground})
	for y := 0; y < q.size; y++ {
		for x := 0; x < q.size; x++ {
			if q.Module(x, y) == rc.invert {
				continue
			}
			x0 := (x + rc.quietZone) * rc.scale
			y0 := (y + rc.quietZone) * rc.scale
			for dy := 0; dy < rc.scale; dy++ {
				for dx := 0; dx < rc.scale; dx++ {
					img.SetColorIndex(x0+dx, y0+dy, 1)
				}
			}
		}
	}
	return img
}

// TestPNGMatchesStdlib checks the hand-rolled encoder produces, pixel for pixel,
// the same image as the standard library for a range of options.
func TestPNGMatchesStdlib(t *testing.T) {
	q := mustEncode(t, "https://example.com/some/path?query=value&x=1")
	red := color.RGBA{R: 0xff, A: 0xff}
	for _, opts := range [][]RenderOption{
		nil,
		{WithScale(1), WithQuietZone(0)},
		{WithScale(8), WithInvert(true)},
		{WithColor(red, color.White), WithScale(2)},
		{WithColor(color.NRGBA{R: 0xff, A: 0x80}, color.NRGBA{G: 0xff, A: 0x40}), WithScale(2)},
	} {
		got, err := q.PNG(opts...)
		if err != nil {
			t.Fatal(err)
		}
		gi, err := png.Decode(bytes.NewReader(got))
		if err != nil {
			t.Fatalf("opts=%v: decode ours: %v", opts, err)
		}
		var ref bytes.Buffer
		if err := png.Encode(&ref, pngRefImage(q, newRenderConfig(opts))); err != nil {
			t.Fatal(err)
		}
		wi, err := png.Decode(&ref)
		if err != nil {
			t.Fatal(err)
		}
		if gi.Bounds() != wi.Bounds() {
			t.Fatalf("opts=%v: bounds %v vs %v", opts, gi.Bounds(), wi.Bounds())
		}
		b := gi.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				gr, gg, gb, ga := gi.At(x, y).RGBA()
				wr, wg, wb, wa := wi.At(x, y).RGBA()
				if gr != wr || gg != wg || gb != wb || ga != wa {
					t.Fatalf("opts=%v: pixel (%d,%d) = %v, want %v", opts, x, y, gi.At(x, y), wi.At(x, y))
				}
			}
		}
	}
}

// TestQuietZoneNotInverted locks the rule shared by PNG, SVG and Terminal: the
// quiet zone is always light, even when WithInvert swaps the modules.
func TestQuietZoneNotInverted(t *testing.T) {
	q := mustEncode(t, "hello")
	const qz = 3
	for _, inv := range []bool{false, true} {
		rc := newRenderConfig([]RenderOption{WithQuietZone(qz), WithInvert(inv)})
		n := rc.extent(q.size)
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				if x >= qz && x < qz+q.size && y >= qz && y < qz+q.size {
					continue // module area
				}
				if q.darkAt(rc, x, y) {
					t.Fatalf("invert=%v: quiet-zone cell (%d,%d) is foreground", inv, x, y)
				}
			}
		}
	}
}

func TestPNGRoundTrip(t *testing.T) {
	q := mustEncode(t, "hello world", WithLevel(Quartile))
	data, err := q.PNG(WithScale(3), WithQuietZone(2))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	extent := q.Size() + 2*2
	want := extent * 3
	if b := img.Bounds(); b.Dx() != want || b.Dy() != want {
		t.Fatalf("image size = %dx%d, want %dx%d", b.Dx(), b.Dy(), want, want)
	}
	// Compare every module against the decoded pixels.
	for y := 0; y < q.Size(); y++ {
		for x := 0; x < q.Size(); x++ {
			px := (x + 2) * 3
			py := (y + 2) * 3
			r, g, bl, _ := img.At(px, py).RGBA()
			dark := r < 0x8000 && g < 0x8000 && bl < 0x8000
			if dark != q.Module(x, y) {
				t.Fatalf("pixel for module (%d,%d) = %v, want %v", x, y, dark, q.Module(x, y))
			}
		}
	}
}

func TestPNGColor(t *testing.T) {
	q := mustEncode(t, "hello")
	red := color.RGBA{R: 0xff, A: 0xff}
	data, err := q.PNG(WithColor(red, color.White), WithScale(1), WithQuietZone(0))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	// Find any dark module in the drawn symbol and check it is red.
	found := false
	for y := 0; y < q.Size() && !found; y++ {
		for x := 0; x < q.Size(); x++ {
			if q.Module(x, y) {
				r, g, b, _ := img.At(x, y).RGBA()
				if r>>8 != 0xff || g>>8 != 0 || b>>8 != 0 {
					t.Fatalf("dark module not red: %d %d %d", r>>8, g>>8, b>>8)
				}
				found = true
				break
			}
		}
	}
}

func TestPNGAlphaColor(t *testing.T) {
	q := mustEncode(t, "hello")
	fg := color.NRGBA{R: 0xff, G: 0x11, A: 0x80}
	data, err := q.PNG(WithColor(fg, color.White), WithScale(1), WithQuietZone(0))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < q.Size(); y++ {
		for x := 0; x < q.Size(); x++ {
			if !q.Module(x, y) {
				continue
			}
			if got := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA); got != fg {
				t.Fatalf("dark module (%d,%d) = %+v, want %+v", x, y, got, fg)
			}
			return
		}
	}
	t.Fatal("no dark module found")
}

func TestSVGAlphaColor(t *testing.T) {
	q := mustEncode(t, "hello")
	svg := q.SVG(WithColor(color.NRGBA{R: 0xff, A: 0x80}, color.NRGBA{B: 0xff, A: 0}))
	if strings.Count(svg, "fill-opacity=") != 2 {
		t.Fatalf("expected two fill-opacity attributes, got %d", strings.Count(svg, "fill-opacity="))
	}
	if !strings.Contains(svg, `fill="#ff0000"`) || !strings.Contains(svg, `fill="#0000ff"`) {
		t.Fatalf("RGB values missing from %s", svg)
	}
	opaque := q.SVG(WithColor(color.RGBA{R: 0xff, A: 0xff}, color.White))
	if strings.Contains(opaque, "fill-opacity=") {
		t.Fatal("opaque colours emitted fill-opacity")
	}
}

func TestSVGWellFormed(t *testing.T) {
	q := mustEncode(t, "https://example.com")
	svg := q.SVG(WithScale(8), WithQuietZone(4))
	if !strings.HasPrefix(svg, "<svg ") {
		t.Fatalf("svg does not start with <svg: %.20q", svg)
	}
	dec := xml.NewDecoder(strings.NewReader(svg))
	for {
		_, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("svg not well-formed: %v", err)
		}
	}
}

func TestTerminalShape(t *testing.T) {
	q := mustEncode(t, "hello world")
	for _, qz := range []int{0, 4} {
		out := q.Terminal(WithQuietZone(qz))
		lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
		extent := q.Size() + 2*qz
		wantLines := (extent + 1) / 2
		if len(lines) != wantLines {
			t.Fatalf("qz=%d: got %d lines, want %d", qz, len(lines), wantLines)
		}
		for i, ln := range lines {
			if n := len([]rune(ln)); n != extent {
				t.Fatalf("qz=%d line %d width = %d, want %d", qz, i, n, extent)
			}
		}
	}
}
