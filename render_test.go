package qrcode

import (
	"bytes"
	"encoding/xml"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

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
