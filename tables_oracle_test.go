package qrcode

import (
	"strconv"
	"testing"
)

// The tests in this file pin the spec tables in tables.go to values transcribed
// independently from ISO/IEC 18004 (Tables 1, 9, C.1, D.1 and E.1). The encoder
// can self-consistently use a wrong table for both writing and its own decoder
// (decode_test.go re-derives from the same primitives), so these oracles are the
// only check that catches a transcription error rather than an algorithm error.

func parseBits(t *testing.T, s string) uint32 {
	t.Helper()
	v, err := strconv.ParseUint(s, 2, 32)
	if err != nil {
		t.Fatalf("bad bit string %q: %v", s, err)
	}
	return uint32(v)
}

// TestFormatInfoOracle checks formatInfo against the 32 published 15-bit strings
// (ISO/IEC 18004 Table C.1).
func TestFormatInfoOracle(t *testing.T) {
	cases := []struct {
		level Level
		mask  int
		bits  string
	}{
		{Low, 0, "111011111000100"}, {Low, 1, "111001011110011"},
		{Low, 2, "111110110101010"}, {Low, 3, "111100010011101"},
		{Low, 4, "110011000101111"}, {Low, 5, "110001100011000"},
		{Low, 6, "110110001000001"}, {Low, 7, "110100101110110"},
		{Medium, 0, "101010000010010"}, {Medium, 1, "101000100100101"},
		{Medium, 2, "101111001111100"}, {Medium, 3, "101101101001011"},
		{Medium, 4, "100010111111001"}, {Medium, 5, "100000011001110"},
		{Medium, 6, "100111110010111"}, {Medium, 7, "100101010100000"},
		{Quartile, 0, "011010101011111"}, {Quartile, 1, "011000001101000"},
		{Quartile, 2, "011111100110001"}, {Quartile, 3, "011101000000110"},
		{Quartile, 4, "010010010110100"}, {Quartile, 5, "010000110000011"},
		{Quartile, 6, "010111011011010"}, {Quartile, 7, "010101111101101"},
		{High, 0, "001011010001001"}, {High, 1, "001001110111110"},
		{High, 2, "001110011100111"}, {High, 3, "001100111010000"},
		{High, 4, "000011101100010"}, {High, 5, "000001001010101"},
		{High, 6, "000110100001100"}, {High, 7, "000100000111011"},
	}
	for _, tc := range cases {
		want := parseBits(t, tc.bits)
		if got := formatInfo(tc.level, tc.mask); got != want {
			t.Errorf("formatInfo(%v, %d) = %015b, want %s", tc.level, tc.mask, got, tc.bits)
		}
	}
}

// TestVersionInfoOracle checks versionInfo against the published 18-bit strings
// (ISO/IEC 18004 Table D.1) for every version that carries version information.
func TestVersionInfoOracle(t *testing.T) {
	cases := map[int]string{
		7: "000111110010010100", 8: "001000010110111100",
		9: "001001101010011001", 10: "001010010011010011",
		11: "001011101111110110", 12: "001100011101100010",
		13: "001101100001000111", 14: "001110011000001101",
		15: "001111100100101000", 16: "010000101101111000",
		17: "010001010001011101", 18: "010010101000010111",
		19: "010011010100110010", 20: "010100100110100110",
		21: "010101011010000011", 22: "010110100011001001",
		23: "010111011111101100", 24: "011000111011000100",
		25: "011001000111100001", 26: "011010111110101011",
		27: "011011000010001110", 28: "011100110000011010",
		29: "011101001100111111", 30: "011110110101110101",
		31: "011111001001010000", 32: "100000100111010101",
		33: "100001011011110000", 34: "100010100010111010",
		35: "100011011110011111", 36: "100100101100001011",
		37: "100101010000101110", 38: "100110101001100100",
		39: "100111010101000001", 40: "101000110001101001",
	}
	for v := 7; v <= 40; v++ {
		want := parseBits(t, cases[v])
		if got := versionInfo(v); got != want {
			t.Errorf("versionInfo(%d) = %018b, want %s", v, got, cases[v])
		}
	}
}

// TestAlignPositionsOracle checks alignPositions against the published centre
// coordinates (ISO/IEC 18004 Table E.1) for all 40 versions.
func TestAlignPositionsOracle(t *testing.T) {
	cases := map[int][]int{
		1: nil,
		2: {6, 18}, 3: {6, 22}, 4: {6, 26}, 5: {6, 30}, 6: {6, 34},
		7: {6, 22, 38}, 8: {6, 24, 42}, 9: {6, 26, 46}, 10: {6, 28, 50},
		11: {6, 30, 54}, 12: {6, 32, 58}, 13: {6, 34, 62},
		14: {6, 26, 46, 66}, 15: {6, 26, 48, 70}, 16: {6, 26, 50, 74},
		17: {6, 30, 54, 78}, 18: {6, 30, 56, 82}, 19: {6, 30, 58, 86},
		20: {6, 34, 62, 90},
		21: {6, 28, 50, 72, 94}, 22: {6, 26, 50, 74, 98}, 23: {6, 30, 54, 78, 102},
		24: {6, 28, 54, 80, 106}, 25: {6, 32, 58, 84, 110}, 26: {6, 30, 58, 86, 114},
		27: {6, 34, 62, 90, 118},
		28: {6, 26, 50, 74, 98, 122}, 29: {6, 30, 54, 78, 102, 126}, 30: {6, 26, 52, 78, 104, 130},
		31: {6, 30, 56, 82, 108, 134}, 32: {6, 34, 60, 86, 112, 138}, 33: {6, 30, 58, 86, 114, 142},
		34: {6, 34, 62, 90, 118, 146},
		35: {6, 30, 54, 78, 102, 126, 150}, 36: {6, 24, 50, 76, 102, 128, 154},
		37: {6, 28, 54, 80, 106, 132, 158}, 38: {6, 32, 58, 84, 110, 136, 162},
		39: {6, 26, 54, 82, 110, 138, 166}, 40: {6, 30, 58, 86, 114, 142, 170},
	}
	for v := 1; v <= 40; v++ {
		got := alignPositions(v)
		want := cases[v]
		if len(got) != len(want) {
			t.Errorf("alignPositions(%d) = %v, want %v", v, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("alignPositions(%d) = %v, want %v", v, got, want)
				break
			}
		}
	}
}

// TestTotalCodewordsOracle checks the geometry formula against the published
// total-codeword counts (ISO/IEC 18004 Table 1).
func TestTotalCodewordsOracle(t *testing.T) {
	want := []int{
		26, 44, 70, 100, 134, 172, 196, 242, 292, 346,
		404, 466, 532, 581, 655, 733, 815, 901, 991, 1085,
		1156, 1258, 1364, 1474, 1588, 1706, 1828, 1921, 2051, 2185,
		2323, 2465, 2611, 2761, 2876, 3034, 3196, 3362, 3532, 3706,
	}
	for v := 1; v <= 40; v++ {
		if got := totalCodewords(v); got != want[v-1] {
			t.Errorf("totalCodewords(%d) = %d, want %d", v, got, want[v-1])
		}
		if got := numRawDataModules(v) / 8; got != want[v-1] {
			t.Errorf("numRawDataModules(%d)/8 = %d, want %d", v, got, want[v-1])
		}
	}
}

// TestDataCodewordsOracle checks the block tables against published data-codeword
// counts (ISO/IEC 18004 Table 9) for versions 1-10 and 40 across all levels.
func TestDataCodewordsOracle(t *testing.T) {
	type row struct {
		ver        int
		l, m, q, h int
	}
	rows := []row{
		{1, 19, 16, 13, 9},
		{2, 34, 28, 22, 16},
		{3, 55, 44, 34, 26},
		{4, 80, 64, 48, 36},
		{5, 108, 86, 62, 46},
		{6, 136, 108, 76, 60},
		{7, 156, 124, 88, 66},
		{8, 194, 154, 110, 86},
		{9, 232, 182, 132, 100},
		{10, 274, 216, 154, 122},
		{40, 2956, 2334, 1666, 1276},
	}
	for _, r := range rows {
		for _, tc := range []struct {
			lvl  Level
			want int
		}{{Low, r.l}, {Medium, r.m}, {Quartile, r.q}, {High, r.h}} {
			if got := dataCodewords(r.ver, tc.lvl); got != tc.want {
				t.Errorf("dataCodewords(%d, %v) = %d, want %d", r.ver, tc.lvl, got, tc.want)
			}
		}
	}
}

// TestBlockAccountingInvariant verifies that for every version and level the
// data codewords plus the error-correction codewords account for exactly the
// total codewords. A typo in eccPerBlock or numBlocks breaks this unless the
// other table is mistyped to compensate.
func TestBlockAccountingInvariant(t *testing.T) {
	for v := 1; v <= 40; v++ {
		total := totalCodewords(v)
		for l := Low; l <= High; l++ {
			ec := eccPerBlock[l][v] * numBlocks[l][v]
			if ec+dataCodewords(v, l) != total {
				t.Errorf("v%d level %v: %d ec + %d data != %d total",
					v, l, ec, dataCodewords(v, l), total)
			}
		}
	}
}

// TestECCBlockKnownValues pins a few block-structure entries (ISO/IEC 18004
// Table 9) so a consistent-but-wrong pair of tables cannot slip through.
func TestECCBlockKnownValues(t *testing.T) {
	cases := []struct {
		lvl      Level
		ver      int
		blocks   int
		ecPerBlk int
	}{
		{Low, 1, 1, 7},
		{High, 1, 1, 17},
		{Quartile, 5, 4, 18},
		{High, 7, 5, 26},
		{High, 10, 8, 28},
		{Low, 40, 25, 30},
		{High, 40, 81, 30},
	}
	for _, tc := range cases {
		if got := numBlocks[tc.lvl][tc.ver]; got != tc.blocks {
			t.Errorf("numBlocks[%v][%d] = %d, want %d", tc.lvl, tc.ver, got, tc.blocks)
		}
		if got := eccPerBlock[tc.lvl][tc.ver]; got != tc.ecPerBlk {
			t.Errorf("eccPerBlock[%v][%d] = %d, want %d", tc.lvl, tc.ver, got, tc.ecPerBlk)
		}
	}
}

// TestCCBitsOracle pins the character-count indicator widths (ISO/IEC 18004
// Table 3) and the version-range split that selects them.
func TestCCBitsOracle(t *testing.T) {
	want := [4][3]int{
		{10, 12, 14}, {9, 11, 13}, {8, 16, 16}, {8, 10, 12},
	}
	if ccBits != want {
		t.Errorf("ccBits = %v, want %v", ccBits, want)
	}
	for _, tc := range []struct{ ver, rng int }{
		{1, 0}, {9, 0}, {10, 1}, {26, 1}, {27, 2}, {40, 2},
	} {
		if got := verRange(tc.ver); got != tc.rng {
			t.Errorf("verRange(%d) = %d, want %d", tc.ver, got, tc.rng)
		}
	}
}
