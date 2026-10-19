package qrcode

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func mustEncode(t *testing.T, text string, opts ...Option) *QR {
	t.Helper()
	q, err := NewString(text, opts...)
	if err != nil {
		t.Fatalf("NewString(%q): %v", text, err)
	}
	return q
}

func TestSizeAndVersion(t *testing.T) {
	for v := 1; v <= 40; v++ {
		q, err := New([]byte("x"), WithVersion(v), WithLevel(Low), WithBoostECL(false))
		if err != nil {
			t.Fatalf("version %d: %v", v, err)
		}
		if q.Version() != v {
			t.Errorf("version = %d, want %d", q.Version(), v)
		}
		if want := 4*v + 17; q.Size() != want {
			t.Errorf("version %d: size = %d, want %d", v, q.Size(), want)
		}
		if len(q.Bitmap()) != q.Size()*q.Size() {
			t.Errorf("version %d: bitmap length %d", v, len(q.Bitmap()))
		}
	}
}

func checkFinder(t *testing.T, q *QR, cx, cy int) {
	t.Helper()
	for dy := -3; dy <= 3; dy++ {
		for dx := -3; dx <= 3; dx++ {
			d := max(abs(dx), abs(dy))
			want := d != 2
			if got := q.Module(cx+dx, cy+dy); got != want {
				t.Errorf("finder at (%d,%d) offset (%d,%d) = %v, want %v", cx, cy, dx, dy, got, want)
			}
		}
	}
}

func TestFunctionPatterns(t *testing.T) {
	q := mustEncode(t, "HELLO", WithLevel(Low), WithBoostECL(false))
	size := q.Size()

	checkFinder(t, q, 3, 3)
	checkFinder(t, q, size-4, 3)
	checkFinder(t, q, 3, size-4)

	// Separators around the top-left finder must be light.
	for i := 0; i <= 7; i++ {
		if q.Module(i, 7) || q.Module(7, i) {
			t.Errorf("separator at %d is dark", i)
		}
	}

	// Timing patterns alternate, dark on even coordinates.
	for i := 8; i < size-8; i++ {
		if got, want := q.Module(i, 6), i%2 == 0; got != want {
			t.Errorf("timing row col %d = %v, want %v", i, got, want)
		}
		if got, want := q.Module(6, i), i%2 == 0; got != want {
			t.Errorf("timing col row %d = %v, want %v", i, got, want)
		}
	}

	// The fixed dark module sits at (8, size-8).
	if !q.Module(8, size-8) {
		t.Error("fixed dark module is light")
	}
}

func TestModeSelection(t *testing.T) {
	tests := []struct {
		text string
		want int // expected version
	}{
		{"1234567890", 1},            // numeric
		{"HELLO WORLD", 1},           // alphanumeric
		{"hello, world", 1},          // byte
		{strings.Repeat("1", 40), 1}, // 40 digits fit version 1-L
	}
	for _, tt := range tests {
		q := mustEncode(t, tt.text, WithLevel(Low), WithBoostECL(false))
		if q.Version() != tt.want {
			t.Errorf("%q: version = %d, want %d", tt.text, q.Version(), tt.want)
		}
	}
}

func TestSegmentationPacksTighterThanByte(t *testing.T) {
	digits := strings.Repeat("1234567890", 12) // 120 digits
	numeric := mustEncode(t, digits, WithLevel(Low), WithBoostECL(false))
	byteMode := mustEncode(t, "", WithSegments(NewBytes([]byte(digits))), WithLevel(Low), WithBoostECL(false))
	if numeric.Version() >= byteMode.Version() {
		t.Errorf("numeric version %d not smaller than byte version %d", numeric.Version(), byteMode.Version())
	}
}

func TestForcedMask(t *testing.T) {
	for m := 0; m < 8; m++ {
		q := mustEncode(t, "hello world", WithMask(m))
		if q.Mask() != m {
			t.Errorf("mask = %d, want %d", q.Mask(), m)
		}
	}
}

func TestLevelBoost(t *testing.T) {
	// A short string at version 1 with a large capacity available: boosting
	// should not lower the level, and Low input should stay at most Low.
	q := mustEncode(t, "1", WithLevel(Low))
	if q.Level() < Low || q.Level() > High {
		t.Fatalf("level = %v", q.Level())
	}
}

func TestStructuredRejectsUnsupportedOptions(t *testing.T) {
	if _, err := Structured([]byte("x"), WithSegments(NewNumeric("1"))); !errors.Is(err, ErrStructuredOption) {
		t.Errorf("segments: err = %v, want ErrStructuredOption", err)
	}
	if _, err := Structured([]byte("x"), WithStructuredAppend(0, 1, 0)); !errors.Is(err, ErrStructuredOption) {
		t.Errorf("header: err = %v, want ErrStructuredOption", err)
	}
}

func TestExplicitSegments(t *testing.T) {
	q := mustEncode(t, "", WithSegments(NewNumeric("12345"), NewText("abc")))
	if q.Version() < 1 {
		t.Fatalf("version = %d", q.Version())
	}
}

func TestKanjiSegment(t *testing.T) {
	// 0x935F is "点" in Shift-JIS.
	q := mustEncode(t, "", WithSegments(NewKanji([]byte{0x93, 0x5F})), WithLevel(Low), WithBoostECL(false))
	if q.Version() != 1 {
		t.Errorf("kanji version = %d, want 1", q.Version())
	}
}

func TestErrors(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		opts []Option
		want error
	}{
		{"too long", []byte(strings.Repeat("x", 8000)), []Option{WithVersion(1)}, ErrTooLong},
		{"bad level", []byte("x"), []Option{WithLevel(Level(9))}, ErrBadLevel},
		{"bad mask", []byte("x"), []Option{WithMask(8)}, ErrBadMask},
		{"bad mask low", []byte("x"), []Option{WithMask(-2)}, ErrBadMask},
		{"bad version", []byte("x"), []Option{WithVersion(41)}, ErrBadVersion},
		{"min version low", []byte("x"), []Option{WithMinVersion(0)}, ErrBadVersion},
		{"max version high", []byte("x"), []Option{WithMaxVersion(41)}, ErrBadVersion},
		{"bad eci high", []byte("x"), []Option{WithECI(1000000)}, ErrBadECI},
		{"bad eci low", []byte("x"), []Option{WithECI(-2)}, ErrBadECI},
		{"empty segments", []byte("x"), []Option{WithSegments()}, ErrEmptySegments},
		{"invalid numeric", []byte("x"), []Option{WithSegments(NewNumeric("12a"))}, ErrInvalidSegment},
		{"invalid alnum", []byte("x"), []Option{WithSegments(NewAlphanumeric("ab"))}, ErrInvalidSegment},
		{"kanji odd length", []byte("x"), []Option{WithSegments(NewKanji([]byte{0x93}))}, ErrInvalidSegment},
		{"kanji out of range", []byte("x"), []Option{WithSegments(NewKanji([]byte{0xEE, 0xEF}))}, ErrInvalidSegment},
		{"kanji non-reversible low byte", []byte("x"), []Option{WithSegments(NewKanji([]byte{0x93, 0x30}))}, ErrInvalidSegment},
	}
	for _, tc := range cases {
		if _, err := New(tc.data, tc.opts...); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestMaxCapacityBoundary(t *testing.T) {
	// Version 40-L numeric capacity is exactly 7089 digits; one more cannot fit
	// in any symbol and must be rejected.
	max := strings.Repeat("1", 7089)
	q, err := NewString(max, WithLevel(Low))
	if err != nil {
		t.Fatalf("max capacity: %v", err)
	}
	if q.Version() != 40 {
		t.Fatalf("version = %d, want 40", q.Version())
	}
	if _, err := NewString(max+"1", WithLevel(Low)); !errors.Is(err, ErrTooLong) {
		t.Fatalf("over capacity: err = %v, want ErrTooLong", err)
	}
}

// TestConcurrentEncode exercises the documented concurrency guarantee; the race
// detector is the real check (it previously fired on the lazily built
// Reed-Solomon generator cache).
func TestConcurrentEncode(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 40; j++ {
				if _, err := NewString("https://example.com/path?q="+strings.Repeat("x", n), WithLevel(Quartile)); err != nil {
					t.Errorf("encode: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestBitmapAliases(t *testing.T) {
	q := mustEncode(t, "hello")
	b := q.Bitmap()
	if &b[0] != &q.modules[0] {
		t.Error("Bitmap does not alias the internal matrix")
	}
}

func TestValidUTF8(t *testing.T) {
	if !ValidUTF8([]byte("héllo")) {
		t.Error("valid UTF-8 reported invalid")
	}
	if ValidUTF8([]byte{0xff, 0xfe}) {
		t.Error("invalid UTF-8 reported valid")
	}
}

func TestECIAddsOverhead(t *testing.T) {
	plain := mustEncode(t, "hello world hello world", WithLevel(High), WithBoostECL(false))
	withECI := mustEncode(t, "hello world hello world", WithLevel(High), WithBoostECL(false), WithECI(26))
	if withECI.Version() < plain.Version() {
		t.Errorf("ECI version %d < plain version %d", withECI.Version(), plain.Version())
	}
}
