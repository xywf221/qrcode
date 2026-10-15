package qrcode

import (
	"bytes"
	"errors"
	"testing"
)

func TestKanjiStringConversion(t *testing.T) {
	seg, err := NewKanjiString("点字")
	if err != nil {
		t.Fatalf("NewKanjiString: %v", err)
	}
	want := []byte{0x93, 0x5F, 0x8E, 0x9A} // 点 = 0x935F, 字 = 0x8E9A
	if !bytes.Equal(seg.s.data, want) {
		t.Fatalf("shift-jis = % x, want % x", seg.s.data, want)
	}

	q := mustEncode(t, "", WithSegments(seg), WithLevel(Medium))
	d, err := decodeSymbol(q)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(d.payload, want) {
		t.Fatalf("roundtrip = % x, want % x", d.payload, want)
	}
}

func TestKanjiStringRejectsUnmappable(t *testing.T) {
	for _, s := range []string{"A", "abc", "点A", "😀", "café"} {
		if _, err := NewKanjiString(s); !errors.Is(err, ErrInvalidSegment) {
			t.Errorf("NewKanjiString(%q) err = %v, want ErrInvalidSegment", s, err)
		}
	}
}
