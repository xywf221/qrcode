package qrcode

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

// This file contains a self-contained QR decoder used only by the tests. It
// re-derives the function-pattern map, format information, mask conditions,
// codeword reading, de-interleaving and Reed-Solomon syndromes from the
// specification, independently of the encoder's placement code. Encoding a
// value and decoding it back to the same bytes is strong evidence that a symbol
// is correctly formed and scannable.

type decoded struct {
	payload []byte
	sa      *saHeader
	eci     int
	level   Level
	mask    int
}

var errDecode = fmt.Errorf("qrcode: decode error")

func decodeSymbol(q *QR) (decoded, error) {
	size := q.Size()
	ver := (size - 17) / 4
	fm := decodeFuncMap(ver, size)

	level, mask, err := decodeFormat(q.modules, size)
	if err != nil {
		return decoded{}, err
	}
	cw := extractCodewords(q.modules, fm, mask)
	blocksData, blocksEC := splitBlocks(cw, ver, level)
	if err := verifyRS(blocksData, blocksEC, level); err != nil {
		return decoded{}, err
	}
	var data []byte
	for _, b := range blocksData {
		data = append(data, b...)
	}
	d, err := parsePayload(data, ver)
	d.level, d.mask = level, mask
	return d, err
}

// decodeFuncMap rebuilds the set of function/reserved modules from the spec.
func decodeFuncMap(ver, size int) []bool {
	fm := make([]bool, size*size)
	set := func(x, y int) { fm[y*size+x] = true }

	for i := 0; i < size; i++ {
		set(6, i)
		set(i, 6)
	}
	finder := func(cx, cy int) {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				if x, y := cx+dx, cy+dy; x >= 0 && y >= 0 && x < size && y < size {
					set(x, y)
				}
			}
		}
	}
	finder(3, 3)
	finder(size-4, 3)
	finder(3, size-4)

	pos := alignPositions(ver)
	n := len(pos)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if (i == 0 && j == 0) || (i == 0 && j == n-1) || (i == n-1 && j == 0) {
				continue
			}
			cx, cy := pos[i], pos[j]
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					set(cx+dx, cy+dy)
				}
			}
		}
	}

	for i := 0; i <= 5; i++ {
		set(8, i)
	}
	set(8, 7)
	set(8, 8)
	set(7, 8)
	for i := 9; i < 15; i++ {
		set(14-i, 8)
	}
	for i := 0; i < 8; i++ {
		set(size-1-i, 8)
	}
	for i := 8; i < 15; i++ {
		set(8, size-15+i)
	}
	set(8, size-8)

	if ver >= 7 {
		for i := 0; i < 18; i++ {
			a, c := size-11+i%3, i/3
			set(a, c)
			set(c, a)
		}
	}
	return fm
}

// decodeFormat reads both copies of the format information and matches them
// against the specification table.
func decodeFormat(mods []byte, size int) (Level, int, error) {
	at := func(x, y int) uint32 { return uint32(mods[y*size+x] & 1) }

	var first uint32
	for i := 0; i <= 5; i++ {
		first |= at(8, i) << uint(i)
	}
	first |= at(8, 7) << 6
	first |= at(8, 8) << 7
	first |= at(7, 8) << 8
	for i := 9; i < 15; i++ {
		first |= at(14-i, 8) << uint(i)
	}

	var second uint32
	for i := 0; i < 8; i++ {
		second |= at(size-1-i, 8) << uint(i)
	}
	for i := 8; i < 15; i++ {
		second |= at(8, size-15+i) << uint(i)
	}
	if first != second {
		return 0, 0, fmt.Errorf("%w: format copies disagree", errDecode)
	}

	for l := Low; l <= High; l++ {
		for mask := 0; mask < 8; mask++ {
			if formatInfo(l, mask) == first {
				return l, mask, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("%w: no matching format information", errDecode)
}

// maskCondition reimplements the eight data-mask conditions from the spec.
func maskCondition(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (y/2+x/3)%2 == 0
	case 5:
		return x*y%2+x*y%3 == 0
	case 6:
		return (x*y%2+x*y%3)%2 == 0
	default:
		return ((x+y)%2+x*y%3)%2 == 0
	}
}

// extractCodewords reads the data modules in the encoder's zigzag order,
// removing the mask, and packs them into codewords.
func extractCodewords(mods []byte, fm []bool, mask int) []byte {
	size := int(math.Round(math.Sqrt(float64(len(mods)))))
	ver := (size - 17) / 4
	totalBits := totalCodewords(ver) * 8

	bits := make([]byte, 0, totalBits)
	for right := size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < size; vert++ {
			for j := 0; j < 2; j++ {
				x, y := right-j, vert
				if (right+1)&2 == 0 {
					y = size - 1 - vert
				}
				idx := y*size + x
				if fm[idx] {
					continue
				}
				v := mods[idx] & 1
				if maskCondition(mask, x, y) {
					v ^= 1
				}
				bits = append(bits, v)
			}
		}
	}

	cw := make([]byte, totalBits/8)
	for i := 0; i < totalBits; i++ {
		cw[i/8] = cw[i/8]<<1 | bits[i]
	}
	return cw
}

// splitBlocks de-interleaves codewords into data and EC blocks for a level.
func splitBlocks(cw []byte, ver int, level Level) (data, ec [][]byte) {
	raw := totalCodewords(ver)
	nb := numBlocks[level][ver]
	ecPer := eccPerBlock[level][ver]
	shortLen := raw / nb
	numShort := nb - raw%nb
	shortData := shortLen - ecPer

	dlen := make([]int, nb)
	for j := range dlen {
		dlen[j] = shortData
		if j >= numShort {
			dlen[j]++
		}
	}
	data = make([][]byte, nb)
	for j := range data {
		data[j] = make([]byte, 0, dlen[j])
	}
	k := 0
	for i := 0; i <= shortData; i++ {
		for j := 0; j < nb; j++ {
			if i < dlen[j] {
				data[j] = append(data[j], cw[k])
				k++
			}
		}
	}
	ec = make([][]byte, nb)
	for j := range ec {
		ec[j] = make([]byte, ecPer)
	}
	for i := 0; i < ecPer; i++ {
		for j := 0; j < nb; j++ {
			ec[j][i] = cw[k]
			k++
		}
	}
	return data, ec
}

// verifyRS checks that every block's Reed-Solomon syndromes are zero, proving
// the error-correction codewords are consistent with the data.
func verifyRS(data, ec [][]byte, level Level) error {
	ecPer := len(ec[0])
	for j := range data {
		codeword := make([]byte, 0, len(data[j])+len(ec[j]))
		codeword = append(codeword, data[j]...)
		codeword = append(codeword, ec[j]...)
		for i := 0; i < ecPer; i++ {
			x := gfExp[i]
			var syn byte
			for _, c := range codeword {
				syn = gfMul(syn, x) ^ c
			}
			if syn != 0 {
				return fmt.Errorf("%w: block %d syndrome %d = %d", errDecode, j, i, syn)
			}
		}
	}
	return nil
}

type bitReader struct {
	b   []byte
	pos int
}

func (r *bitReader) read(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		byteIdx := r.pos >> 3
		if byteIdx >= len(r.b) {
			return v << uint(n-i)
		}
		bit := (r.b[byteIdx] >> uint(7-(r.pos&7))) & 1
		v = v<<1 | uint32(bit)
		r.pos++
	}
	return v
}

func (r *bitReader) remaining() int { return len(r.b)*8 - r.pos }

// parsePayload decodes the data bit stream into the original bytes, reading an
// optional Structured Append header and ECI designator first.
func parsePayload(data []byte, ver int) (decoded, error) {
	r := &bitReader{b: data}
	vr := verRange(ver)
	d := decoded{eci: -1}

	mode := r.read(4)
	if mode == 0x3 {
		idx := int(r.read(4))
		total := int(r.read(4)) + 1
		parity := int(r.read(8))
		if idx >= total {
			return d, fmt.Errorf("%w: bad structured-append header", errDecode)
		}
		d.sa = &saHeader{index: idx, total: total, parity: parity}
		mode = r.read(4)
	}
	if mode == 0x7 {
		v := r.read(8)
		switch {
		case v&0x80 == 0:
			d.eci = int(v)
		case v&0xC0 == 0x80:
			d.eci = int(v&0x3F)<<8 | int(r.read(8))
		default:
			d.eci = int(v&0x1F)<<16 | int(r.read(8))<<8 | int(r.read(8))
		}
		mode = r.read(4)
	}

	for {
		switch mode {
		case 0x0:
			return d, nil
		case 0x1: // numeric
			d.payload = append(d.payload, readNumeric(r, int(r.read(ccBits[0][vr])))...)
		case 0x2: // alphanumeric
			d.payload = append(d.payload, readAlnum(r, int(r.read(ccBits[1][vr])))...)
		case 0x4: // byte
			n := int(r.read(ccBits[2][vr]))
			for i := 0; i < n; i++ {
				d.payload = append(d.payload, byte(r.read(8)))
			}
		case 0x8: // kanji
			d.payload = append(d.payload, readKanji(r, int(r.read(ccBits[3][vr])))...)
		default:
			return d, fmt.Errorf("%w: unexpected mode %#x", errDecode, mode)
		}
		if r.remaining() < 4 {
			return d, nil
		}
		mode = r.read(4)
	}
}

func readNumeric(r *bitReader, cnt int) []byte {
	var out []byte
	for ; cnt >= 3; cnt -= 3 {
		v := r.read(10)
		out = append(out, byte('0'+v/100), byte('0'+v/10%10), byte('0'+v%10))
	}
	switch cnt {
	case 2:
		v := r.read(7)
		out = append(out, byte('0'+v/10), byte('0'+v%10))
	case 1:
		out = append(out, byte('0'+r.read(4)))
	}
	return out
}

func readAlnum(r *bitReader, cnt int) []byte {
	var out []byte
	for ; cnt >= 2; cnt -= 2 {
		v := r.read(11)
		out = append(out, alnumChars[v/45], alnumChars[v%45])
	}
	if cnt == 1 {
		out = append(out, alnumChars[r.read(6)])
	}
	return out
}

func readKanji(r *bitReader, cnt int) []byte {
	var out []byte
	for i := 0; i < cnt; i++ {
		v := int(r.read(13))
		sjis := (v/0xC0)<<8 | (v % 0xC0)
		if sjis+0x8140 <= 0x9FFC {
			sjis += 0x8140
		} else {
			sjis += 0xC140
		}
		out = append(out, byte(sjis>>8), byte(sjis))
	}
	return out
}

// --- Round-trip tests ---

func TestDecodeRoundtrip(t *testing.T) {
	inputs := []string{
		"1234567890",
		"HELLO WORLD 123 $%*+-./:",
		"hello, world",
		"https://example.com/some/path?query=value&x=1",
		"Ünïcödé — 日本語テキスト — ✓",
		strings.Repeat("0123456789", 40),
		strings.Repeat("ABC 123 ", 50),
		strings.Repeat("x", 500),
	}
	for _, in := range inputs {
		for _, lv := range []Level{Low, Medium, Quartile, High} {
			q, err := NewString(in, WithLevel(lv))
			if err != nil {
				t.Fatalf("encode %q: %v", in, err)
			}
			d, err := decodeSymbol(q)
			if err != nil {
				t.Fatalf("decode %q level %v: %v", in, lv, err)
			}
			if string(d.payload) != in {
				t.Fatalf("roundtrip mismatch:\n got %q\nwant %q", d.payload, in)
			}
			if d.level != q.Level() || d.mask != q.Mask() {
				t.Fatalf("metadata mismatch: got %v/%d want %v/%d", d.level, d.mask, q.Level(), q.Mask())
			}
		}
	}
}

func TestDecodeKanjiRoundtrip(t *testing.T) {
	sjis := []byte{0x93, 0x5F, 0x8A, 0xBF} // "点字"
	q := mustEncode(t, "", WithSegments(NewKanji(sjis)), WithLevel(Medium))
	d, err := decodeSymbol(q)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(d.payload, sjis) {
		t.Fatalf("kanji roundtrip: got % x want % x", d.payload, sjis)
	}
}

func TestDecodeECIRoundtrip(t *testing.T) {
	q, err := NewString("café", WithECI(26))
	if err != nil {
		t.Fatal(err)
	}
	d, err := decodeSymbol(q)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if d.eci != 26 {
		t.Fatalf("eci = %d, want 26", d.eci)
	}
	if string(d.payload) != "café" {
		t.Fatalf("payload = %q, want café", d.payload)
	}
}

func TestStructuredAppendRoundtrip(t *testing.T) {
	msg := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20)
	qs, err := StructuredString(msg, WithLevel(Quartile))
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) < 2 || len(qs) > 16 {
		t.Fatalf("symbol count = %d, want 2..16", len(qs))
	}

	var want byte
	for i := 0; i < len(msg); i++ {
		want ^= msg[i]
	}

	var got []byte
	ver := qs[0].Version()
	for i, q := range qs {
		if q.Version() != ver {
			t.Fatalf("symbol %d version %d, want %d", i, q.Version(), ver)
		}
		d, err := decodeSymbol(q)
		if err != nil {
			t.Fatalf("decode symbol %d: %v", i, err)
		}
		if d.sa == nil {
			t.Fatalf("symbol %d missing structured-append header", i)
		}
		if d.sa.index != i || d.sa.total != len(qs) || byte(d.sa.parity) != want {
			t.Fatalf("symbol %d header = %+v, want index %d total %d parity %#x",
				i, d.sa, i, len(qs), want)
		}
		got = append(got, d.payload...)
	}
	if string(got) != msg {
		t.Fatalf("reassembled message mismatch: got %d bytes want %d", len(got), len(msg))
	}
}

func TestStructuredAppendSmallMessage(t *testing.T) {
	qs, err := StructuredString("HELLO", WithLevel(Low))
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 1 {
		t.Fatalf("symbol count = %d, want 1", len(qs))
	}
	d, err := decodeSymbol(qs[0])
	if err != nil {
		t.Fatal(err)
	}
	if d.sa == nil || d.sa.index != 0 || d.sa.total != 1 {
		t.Fatalf("header = %+v, want index 0 total 1", d.sa)
	}
	if string(d.payload) != "HELLO" {
		t.Fatalf("payload = %q, want HELLO", d.payload)
	}
}

func TestStructuredAppendErrors(t *testing.T) {
	if _, err := New([]byte("x"), WithStructuredAppend(2, 2, 0)); !errors.Is(err, ErrStructuredAppend) {
		t.Errorf("index >= total: err = %v, want ErrStructuredAppend", err)
	}
	if _, err := New([]byte("x"), WithStructuredAppend(0, 17, 0)); !errors.Is(err, ErrStructuredAppend) {
		t.Errorf("total > 16: err = %v, want ErrStructuredAppend", err)
	}
	// A message far too large to fit in sixteen version-40 symbols.
	if _, err := Structured(make([]byte, 100000), WithLevel(High)); !errors.Is(err, ErrTooManySymbols) {
		t.Errorf("oversized: err = %v, want ErrTooManySymbols", err)
	}
}
