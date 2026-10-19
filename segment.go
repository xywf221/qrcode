package qrcode

import "unicode/utf8"

// A Segment is an explicit, pre-typed chunk of data for WithSegments. Use the
// New* constructors to build one.
type Segment struct{ s segment }

// segment is the internal representation of one encoded run.
type segment struct {
	mode Mode
	data []byte
	cc   int // character count (kanji pairs count as one)
	eci  int
}

// NewNumeric returns a numeric-mode segment. The data must contain only digits.
func NewNumeric(s string) Segment {
	return Segment{segment{mode: modeNumeric, data: []byte(s), cc: len(s)}}
}

// NewAlphanumeric returns an alphanumeric-mode segment. The data must contain
// only the 45 alphanumeric characters (0-9, A-Z, space, $%*+-./:).
func NewAlphanumeric(s string) Segment {
	return Segment{segment{mode: modeAlphanumeric, data: []byte(s), cc: len(s)}}
}

// NewBytes returns a byte-mode segment holding raw bytes.
func NewBytes(b []byte) Segment {
	return Segment{segment{mode: modeByte, data: b, cc: len(b)}}
}

// NewText returns a byte-mode segment holding the UTF-8 bytes of s.
func NewText(s string) Segment {
	return Segment{segment{mode: modeByte, data: []byte(s), cc: len(s)}}
}

// NewKanji returns a Kanji-mode segment. The data must be the Shift-JIS
// encoding of the text, so its length must be even.
func NewKanji(sjis []byte) Segment {
	return Segment{segment{mode: modeKanji, data: sjis, cc: len(sjis) / 2}}
}

func validateSegment(s segment) error {
	switch s.mode {
	case modeNumeric:
		for _, c := range s.data {
			if c < '0' || c > '9' {
				return ErrInvalidSegment
			}
		}
	case modeAlphanumeric:
		for _, c := range s.data {
			if alnumValue[c] < 0 {
				return ErrInvalidSegment
			}
		}
	case modeKanji:
		if len(s.data)%2 != 0 {
			return ErrInvalidSegment
		}
		for i := 0; i+1 < len(s.data); i += 2 {
			if !validKanjiCode(uint16(s.data[i])<<8 | uint16(s.data[i+1])) {
				return ErrInvalidSegment
			}
		}
	}
	return nil
}

// validKanjiCode reports whether a two-byte Shift-JIS value is representable in
// QR Kanji mode: the JIS X 0208 ranges 0x8140–0x9FFC and 0xE040–0xEBBF whose low
// byte is at least 0x40. Codes outside those ranges — such as the JIS X 0212 and
// vendor extensions in 0xED40–0xEEFC that the Shift-JIS table also contains —
// have no 13-bit Kanji representation.
//
// The low-byte test is the subtle part. Kanji mode stores (high*0xC0 + low) of
// the value left after subtracting the range base (0x8140 or 0xC140). That only
// inverts when the remainder's low byte is below 0xC0, i.e. the code's low byte
// is at least 0x40. A code such as 0x9330 passes the range test but decodes back
// as 0x9370, so it must be rejected rather than silently corrupted.
func validKanjiCode(c uint16) bool {
	if c&0xFF < 0x40 {
		return false
	}
	return (c >= 0x8140 && c <= 0x9FFC) || (c >= 0xE040 && c <= 0xEBBF)
}

func eciValueBits(d int) int {
	switch {
	case d < 0:
		return 0
	case d < 128:
		return 8
	case d < 16384:
		return 16
	default:
		return 24
	}
}

// bits reports the number of bits segment s occupies at the given version,
// including its mode indicator and character count.
func (s segment) bits(ver int) int {
	if s.mode == modeECI {
		return 4 + eciValueBits(s.eci)
	}
	vr := verRange(ver)
	b := 4 + ccBits[s.mode][vr]
	switch s.mode {
	case modeNumeric:
		n := s.cc
		b += n / 3 * 10
		switch n % 3 {
		case 1:
			b += 4
		case 2:
			b += 7
		}
	case modeAlphanumeric:
		n := s.cc
		b += n/2*11 + n%2*6
	case modeByte:
		b += s.cc * 8
	case modeKanji:
		b += s.cc * 13
	}
	return b
}

func (s segment) write(bb *bitBuffer, ver int) {
	if s.mode == modeECI {
		writeECI(bb, s.eci)
		return
	}
	bb.writeBits(s.mode.indicator(), 4)
	bb.writeBits(uint32(s.cc), ccBits[s.mode][verRange(ver)])
	switch s.mode {
	case modeNumeric:
		writeNumeric(bb, s.data)
	case modeAlphanumeric:
		writeAlnum(bb, s.data)
	case modeByte:
		for _, c := range s.data {
			bb.writeBits(uint32(c), 8)
		}
	case modeKanji:
		writeKanji(bb, s.data)
	}
}

func writeECI(bb *bitBuffer, d int) {
	bb.writeBits(0x7, 4)
	switch {
	case d < 128:
		bb.writeBits(uint32(d), 8)
	case d < 16384:
		bb.writeBits(uint32(0x8000|d), 16)
	default:
		bb.writeBits(uint32(0xC00000|d), 24)
	}
}

func writeNumeric(bb *bitBuffer, d []byte) {
	n := len(d)
	i := 0
	for ; i+3 <= n; i += 3 {
		v := int(d[i]-'0')*100 + int(d[i+1]-'0')*10 + int(d[i+2]-'0')
		bb.writeBits(uint32(v), 10)
	}
	switch n - i {
	case 2:
		bb.writeBits(uint32(int(d[i]-'0')*10+int(d[i+1]-'0')), 7)
	case 1:
		bb.writeBits(uint32(d[i]-'0'), 4)
	}
}

func writeAlnum(bb *bitBuffer, d []byte) {
	n := len(d)
	i := 0
	for ; i+2 <= n; i += 2 {
		v := int(alnumValue[d[i]])*45 + int(alnumValue[d[i+1]])
		bb.writeBits(uint32(v), 11)
	}
	if i < n {
		bb.writeBits(uint32(alnumValue[d[i]]), 6)
	}
}

// writeKanji emits the Shift-JIS pairs in d as 13-bit Kanji values. Callers must
// have validated d with validateSegment, which guarantees every pair falls in one
// of the two representable ranges.
func writeKanji(bb *bitBuffer, d []byte) {
	for i := 0; i+1 < len(d); i += 2 {
		v := int(d[i])<<8 | int(d[i+1])
		switch {
		case v >= 0x8140 && v <= 0x9FFC:
			v -= 0x8140
		default: // 0xE040..0xEBBF
			v -= 0xC140
		}
		bb.writeBits(uint32((v>>8)*0xC0+(v&0xFF)), 13)
	}
}

// requiredBits returns the total encoded bit length of segs at a version,
// including an optional ECI designator.
func requiredBits(segs []segment, ver, eci int) int {
	total := 0
	for _, s := range segs {
		total += s.bits(ver)
	}
	if eci >= 0 {
		total += 4 + eciValueBits(eci)
	}
	return total
}

func makeSeg(m Mode, d []byte, eci int) segment {
	cc := len(d)
	if m == modeKanji {
		cc = len(d) / 2
	}
	return segment{mode: m, data: d, cc: cc, eci: eci}
}

// ValidUTF8 reports whether b is valid UTF-8. It is a helper for callers that
// want to decide on WithECI(26); it has no effect on encoding.
func ValidUTF8(b []byte) bool { return utf8.Valid(b) }
