package qrcode

// All reference data in this file is transcribed from ISO/IEC 18004.
// The tables were cross-checked against multiple independent implementations.

// eccPerBlock[l][v] is the number of error-correction codewords per block for
// level l (L,M,Q,H) and version v. Index 0 is unused.
var eccPerBlock = [4][41]int{
	{0, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18, 20, 24, 26, 30, 22, 24, 28, 30, 28, 28, 28, 28, 30, 30, 26, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{0, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28},
	{0, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24, 28, 26, 24, 20, 30, 24, 28, 28, 26, 30, 28, 30, 30, 30, 30, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{0, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28, 24, 28, 22, 24, 24, 30, 28, 28, 26, 28, 30, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
}

// numBlocks[l][v] is the number of error-correction blocks for level l and
// version v. Index 0 is unused.
var numBlocks = [4][41]int{
	{0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 6, 7, 8, 8, 9, 9, 10, 12, 12, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 24, 25},
	{0, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49},
	{0, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8, 8, 10, 12, 16, 12, 17, 16, 18, 21, 20, 23, 23, 25, 27, 29, 34, 34, 35, 38, 40, 43, 45, 48, 51, 53, 56, 59, 62, 65, 68},
	{0, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8, 11, 11, 16, 16, 18, 16, 19, 21, 25, 25, 25, 34, 30, 32, 35, 37, 40, 42, 45, 48, 51, 54, 57, 60, 63, 66, 70, 74, 77, 81},
}

// ccBits[mode][range] is the number of bits in the character count indicator.
var ccBits = [4][3]int{
	{10, 12, 14}, // numeric
	{9, 11, 13},  // alphanumeric
	{8, 16, 16},  // byte
	{8, 10, 12},  // kanji
}

const alnumChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

var alnumValue [256]int8

func init() {
	for i := range alnumValue {
		alnumValue[i] = -1
	}
	for i := 0; i < len(alnumChars); i++ {
		alnumValue[alnumChars[i]] = int8(i)
	}
}

// verRange maps a version to its character-count indicator range index:
// 0 for v1-9, 1 for v10-26, 2 for v27-40.
func verRange(ver int) int { return (ver + 7) / 17 }

// numRawDataModules returns the number of data modules (excluding function and
// formatting patterns) for a version, before error correction. It is a multiple
// of 8 plus a remainder of 0-7 bits.
func numRawDataModules(ver int) int {
	result := (16*ver+128)*ver + 64
	if ver >= 2 {
		n := ver/7 + 2
		result -= (25*n-10)*n - 55
		if ver >= 7 {
			result -= 36
		}
	}
	return result
}

// dataCodewords returns the number of data codewords available at a version and
// level (total codewords minus error-correction codewords).
func dataCodewords(ver int, l Level) int {
	raw := numRawDataModules(ver) / 8
	return raw - eccPerBlock[l][ver]*numBlocks[l][ver]
}

func dataBits(ver int, l Level) int { return dataCodewords(ver, l) * 8 }

// totalCodewords returns the total number of codewords (data plus EC).
func totalCodewords(ver int) int { return numRawDataModules(ver) / 8 }

// alignPositions returns the alignment pattern centre coordinates for a version.
// Version 1 has none.
func alignPositions(ver int) []int {
	if ver == 1 {
		return nil
	}
	n := ver/7 + 2
	size := 4*ver + 17
	step := (ver*8 + n*3 + 5) / (n*4 - 4) * 2
	pos := make([]int, 0, n)
	for i := 0; i < n-1; i++ {
		pos = append(pos, size-7-i*step)
	}
	pos = append(pos, 6)
	for i, j := 0, len(pos)-1; i < j; i, j = i+1, j-1 {
		pos[i], pos[j] = pos[j], pos[i]
	}
	return pos
}

// formatInfo returns the 15-bit encoded format information for a level and mask.
// It is a BCH(15,5) code with generator polynomial 0x537, XORed with 0x5412.
func formatInfo(l Level, mask int) uint32 {
	data := l.formatBits()<<3 | uint32(mask)
	rem := data << 10
	for i := 14; i >= 10; i-- {
		if rem&(1<<uint(i)) != 0 {
			rem ^= 0x537 << uint(i-10)
		}
	}
	return (data << 10) | rem ^ 0x5412
}

// versionInfo returns the 18-bit encoded version information for versions 7-40.
// It is a BCH(18,6) code with generator polynomial 0x1F25.
func versionInfo(ver int) uint32 {
	rem := uint32(ver) << 12
	for i := 17; i >= 12; i-- {
		if rem&(1<<uint(i)) != 0 {
			rem ^= 0x1F25 << uint(i-12)
		}
	}
	return uint32(ver)<<12 | rem
}
