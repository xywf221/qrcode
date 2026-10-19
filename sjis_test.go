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
	// The first three map to Shift-JIS codes outside the Kanji ranges
	// (0xED40–0xEEFC) and must be rejected, not truncated.
	for _, s := range []string{"A", "abc", "点A", "😀", "café", "ⅰ", "髙", "﨑"} {
		if _, err := NewKanjiString(s); !errors.Is(err, ErrInvalidSegment) {
			t.Errorf("NewKanjiString(%q) err = %v, want ErrInvalidSegment", s, err)
		}
	}
}

// TestKanjiRejectsNonReversibleCode guards the low-byte check: 0x9330 lies in
// the 0x8140–0x9FFC range but its low byte (0x30) makes the 13-bit encoding carry,
// decoding back as 0x9370. It must be rejected, and its reversible neighbour must
// still round-trip.
func TestKanjiRejectsNonReversibleCode(t *testing.T) {
	if _, err := New([]byte("x"), WithSegments(NewKanji([]byte{0x93, 0x30}))); !errors.Is(err, ErrInvalidSegment) {
		t.Errorf("NewKanji(0x9330) err = %v, want ErrInvalidSegment", err)
	}
	q, err := NewString("", WithSegments(NewKanji([]byte{0x93, 0x70})), WithLevel(Medium))
	if err != nil {
		t.Fatalf("NewKanji(0x9370): %v", err)
	}
	d, err := decodeSymbol(q)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(d.payload, []byte{0x93, 0x70}) {
		t.Fatalf("roundtrip = % x, want 93 70", d.payload)
	}
}
