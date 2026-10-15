package qrcode

import (
	"io"
	"strings"
	"testing"
)

var benchInputs = map[string]string{
	"numeric": strings.Repeat("1234567890", 20),
	"alnum":   strings.Repeat("HELLO WORLD 123 ", 20),
	"byte":    strings.Repeat("hello world ", 20),
	"url":     "https://example.com/some/path?query=value&x=1",
}

// benchURLs are realistic URLs, one short and one long, used by the end-to-end
// encode-and-render benchmark.
var benchURLs = []struct{ name, url string }{
	{"short", "https://example.com/some/path?query=value&x=1"},
	{"long", "https://www.example.com/a/b/c/d/e?foo=bar&baz=qux&id=1234567890&ref=homepage&utm_source=newsletter"},
}

func BenchmarkEncode(b *testing.B) {
	for name, in := range benchInputs {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				if _, err := NewString(in, WithLevel(Quartile)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEncodeLevels(b *testing.B) {
	in := benchInputs["url"]
	for _, lv := range []Level{Low, Medium, Quartile, High} {
		b.Run(lv.String(), func(b *testing.B) {
			for b.Loop() {
				if _, err := NewString(in, WithLevel(lv)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEncodeForcedMask(b *testing.B) {
	in := benchInputs["url"]
	for b.Loop() {
		if _, err := NewString(in, WithMask(0)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncodePNG measures the whole path a caller pays to turn a URL into a
// PNG: optimal segmentation, error correction, the eight-mask search, and PNG
// rendering, rather than the encode and render stages in isolation.
func BenchmarkEncodePNG(b *testing.B) {
	for _, u := range benchURLs {
		b.Run(u.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				q, err := NewString(u.url, WithLevel(Quartile))
				if err != nil {
					b.Fatal(err)
				}
				if _, err := q.PNG(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkRenderPNG(b *testing.B) {
	q, err := NewString(benchInputs["url"], WithLevel(Quartile))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := q.WritePNG(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderSVG(b *testing.B) {
	q, err := NewString(benchInputs["url"], WithLevel(Quartile))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = q.SVG()
	}
}

func BenchmarkRenderTerminal(b *testing.B) {
	q, err := NewString(benchInputs["url"], WithLevel(Quartile))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = q.Terminal()
	}
}
