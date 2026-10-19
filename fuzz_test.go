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

// FuzzExplicitSegments exercises the WithSegments path across all four segment
// modes: it builds one segment in a fuzzer-chosen mode, encodes it, and checks
// the decoded payload reproduces the input bytes exactly.
func FuzzExplicitSegments(f *testing.F) {
	f.Add(uint8(0), []byte("1234567890"))
	f.Add(uint8(1), []byte("HELLO $%*+-./:"))
	f.Add(uint8(2), []byte("bytes \x00\xff"))
	f.Add(uint8(3), []byte{0x93, 0x5F, 0x8E, 0x9A})

	f.Fuzz(func(t *testing.T, modeSel uint8, data []byte) {
		var seg Segment
		switch modeSel % 4 {
		case 0:
			seg = NewNumeric(string(data))
		case 1:
			seg = NewAlphanumeric(string(data))
		case 2:
			seg = NewBytes(data)
		default:
			seg = NewKanji(data)
		}
		q, err := New(nil, WithSegments(seg), WithLevel(Level((modeSel>>2)%4)))
		if err != nil {
			return // an invalid or oversized segment is an acceptable outcome.
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

// FuzzECI checks that any valid ECI designator round-trips, and that an
// out-of-range designator is rejected without panicking.
func FuzzECI(f *testing.F) {
	f.Add([]byte("café"), uint32(26), uint8(Quartile), uint8(40))
	f.Add([]byte("hello"), uint32(0), uint8(Low), uint8(5))
	f.Add([]byte("x"), uint32(16384), uint8(High), uint8(1))

	f.Fuzz(func(t *testing.T, data []byte, eci uint32, level, maxVer uint8) {
		designator := int(eci % 1000000)
		q, err := New(data, WithECI(designator), WithLevel(Level(level%4)), WithMaxVersion(int(maxVer%40)+1))
		if err != nil {
			return
		}
		d, err := decodeSymbol(q)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if d.eci != designator {
			t.Fatalf("eci = %d, want %d", d.eci, designator)
		}
		if !bytes.Equal(d.payload, data) {
			t.Fatalf("roundtrip mismatch: got %q want %q", d.payload, data)
		}
	})
}

// FuzzKanjiString checks that any string the Shift-JIS converter accepts
// round-trips through Kanji mode, and that everything else is rejected.
func FuzzKanjiString(f *testing.F) {
	f.Add("点字")
	f.Add("日本語テキスト")
	f.Add("A")

	f.Fuzz(func(t *testing.T, s string) {
		seg, err := NewKanjiString(s)
		if err != nil {
			return
		}
		q, err := NewString("", WithSegments(seg), WithLevel(Medium))
		if err != nil {
			return
		}
		d, err := decodeSymbol(q)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !bytes.Equal(d.payload, seg.s.data) {
			t.Fatalf("kanji roundtrip: got % x want % x", d.payload, seg.s.data)
		}
	})
}

// FuzzStructured checks that Structured Append splits, headers and reassembly
// are self-consistent for arbitrary messages.
func FuzzStructured(f *testing.F) {
	f.Add([]byte("HELLO"), uint8(Low), uint8(40))
	f.Add(bytes.Repeat([]byte("abcdef"), 50), uint8(Quartile), uint8(40))
	f.Add([]byte{}, uint8(Medium), uint8(10))

	f.Fuzz(func(t *testing.T, data []byte, level, maxVer uint8) {
		qs, err := Structured(data, WithLevel(Level(level%4)), WithMaxVersion(int(maxVer%40)+1))
		if err != nil {
			return
		}
		if len(qs) < 1 || len(qs) > 16 {
			t.Fatalf("symbol count = %d, want 1..16", len(qs))
		}
		var parity byte
		for _, b := range data {
			parity ^= b
		}
		var got []byte
		for i, q := range qs {
			d, err := decodeSymbol(q)
			if err != nil {
				t.Fatalf("decode symbol %d: %v", i, err)
			}
			if d.sa == nil || d.sa.index != i || d.sa.total != len(qs) || byte(d.sa.parity) != parity {
				t.Fatalf("symbol %d header = %+v, want index %d total %d parity %#x",
					i, d.sa, i, len(qs), parity)
			}
			got = append(got, d.payload...)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("reassembled %d bytes, want %d", len(got), len(data))
		}
	})
}

// FuzzMask checks that a forced mask is honoured and round-trips.
func FuzzMask(f *testing.F) {
	f.Add([]byte("hello world"), uint8(Quartile), uint8(0), uint8(40))
	f.Add([]byte("123"), uint8(Low), uint8(7), uint8(1))

	f.Fuzz(func(t *testing.T, data []byte, level, mask, maxVer uint8) {
		m := int(mask % 8)
		q, err := New(data, WithLevel(Level(level%4)), WithMask(m), WithMaxVersion(int(maxVer%40)+1))
		if err != nil {
			return
		}
		if q.Mask() != m {
			t.Fatalf("mask = %d, want %d", q.Mask(), m)
		}
		d, err := decodeSymbol(q)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if d.mask != m {
			t.Fatalf("decoded mask = %d, want %d", d.mask, m)
		}
		if !bytes.Equal(d.payload, data) {
			t.Fatalf("roundtrip mismatch: got %q want %q", d.payload, data)
		}
	})
}
