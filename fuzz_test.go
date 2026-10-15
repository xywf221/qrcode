package qrcode

import (
	"bytes"
	"testing"
)

// FuzzEncode checks that arbitrary input never panics the encoder and that any
// successfully encoded symbol satisfies its structural invariants and can be
// rendered by every renderer.
func FuzzEncode(f *testing.F) {
	f.Add([]byte("HELLO WORLD"), uint8(Low), uint8(40))
	f.Add([]byte("1234567890"), uint8(High), uint8(5))
	f.Add([]byte("https://example.com/\x00\xff"), uint8(Quartile), uint8(40))
	f.Add([]byte(""), uint8(Medium), uint8(1))

	f.Fuzz(func(t *testing.T, data []byte, level, maxVer uint8) {
		lv := Level(level % 4)
		ver := int(maxVer%40) + 1

		q, err := New(data, WithLevel(lv), WithMaxVersion(ver))
		if err != nil {
			return // ErrTooLong is an acceptable, non-panicking outcome.
		}
		if got, want := q.Size(), 4*q.Version()+17; got != want {
			t.Fatalf("size = %d, want %d", got, want)
		}
		if len(q.Bitmap()) != q.Size()*q.Size() {
			t.Fatalf("bitmap length = %d, want %d", len(q.Bitmap()), q.Size()*q.Size())
		}
		if q.Version() < 1 || q.Version() > ver {
			t.Fatalf("version = %d, want in [1,%d]", q.Version(), ver)
		}

		png, err := q.PNG()
		if err != nil {
			t.Fatalf("PNG: %v", err)
		}
		if len(png) == 0 {
			t.Fatal("PNG returned no bytes")
		}
		if svg := q.SVG(); len(svg) == 0 || !bytes.Contains([]byte(svg), []byte("</svg>")) {
			t.Fatal("SVG not well-formed")
		}
		if term := q.Terminal(); len(term) == 0 {
			t.Fatal("Terminal returned no output")
		}

		d, err := decodeSymbol(q)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !bytes.Equal(d.payload, data) {
			t.Fatalf("roundtrip mismatch: got %q want %q", d.payload, data)
		}
	})
}

// FuzzPenalty checks the optimised penalty scorer against the naive reference
// implementation on arbitrary module matrices.
func FuzzPenalty(f *testing.F) {
	f.Add(uint8(21), []byte{})
	f.Add(uint8(25), bytes.Repeat([]byte{1, 0}, 8))
	f.Add(uint8(45), bytes.Repeat([]byte{1}, 45*45))

	f.Fuzz(func(t *testing.T, size uint8, fill []byte) {
		n := int(size%6)*8 + 21 // one of 21, 29, ..., 61
		mods := make([]byte, n*n)
		if len(fill) > 0 {
			for i := range mods {
				mods[i] = fill[i%len(fill)] & 1
			}
		}
		b := &builder{size: n, modules: mods}
		if got, want := b.penalty(), naivePenalty(mods, n); got != want {
			t.Fatalf("size=%d: penalty=%d naive=%d", n, got, want)
		}
	})
}
