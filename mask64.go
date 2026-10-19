package qrcode

import "math/bits"

// Bit-parallel penalty scoring for symbols at most 64 modules wide (versions 3
// through 11), where a whole row or column fits in one uint64. The scalar
// scorer in mask.go is the reference for versions 1-2 and versions 12-40.

func reverseLine(x uint64, k int) uint64 { return bits.Reverse64(x) >> uint(64-k) }

// patternMatch returns a mask whose bit s is set when the eleven-bit window
// starting at s equals pat, where bit k of pat is the module at offset k.
//
// penaltyPatternA and penaltyPatternB are bit-reversals of each other. Packing
// the line with bit x = module x makes the window LSB-first, so matching a
// packed window against pat is equivalent to matching the original line against
// the reversed pattern; testing both patterns covers both orientations.
func patternMatch(line uint64, pat uint) uint64 {
	m := ^uint64(0)
	for k := 0; k < 11; k++ {
		if (pat>>uint(k))&1 != 0 {
			m &= line >> uint(k)
		} else {
			m &= ^(line >> uint(k))
		}
	}
	return m
}

// runPenalty64 applies rule 1 to one packed line. A run of length L >= 5
// contributes L-2, computed as the number of modules in long runs minus twice
// the number of long runs.
func runPenalty64(line uint64, n int) int {
	valid := ^uint64(0) >> uint(64-n)
	longRuns := func(m uint64) int {
		w5 := m & (m >> 1) & (m >> 2) & (m >> 3) & (m >> 4)
		if w5 == 0 {
			return 0
		}
		// Every module of a long run lies in some window of five same-coloured
		// modules; expanding each such window marks exactly the long runs.
		in := (w5 | w5<<1 | w5<<2 | w5<<3 | w5<<4) & valid
		starts := m & ^(m << 1)
		return bits.OnesCount64(in) - 2*bits.OnesCount64(starts&in)
	}
	return longRuns(line&valid) + longRuns(^line&valid)
}

// rule3Line counts rule-3 patterns along one packed line, including the four
// light border modules on each side.
func rule3Line(line uint64, n int) int {
	var res int
	if n >= 11 {
		// Windows fully inside the line, one bit per start position.
		m := patternMatch(line, penaltyPatternA) | patternMatch(line, penaltyPatternB)
		m &= uint64(1)<<uint(n-10) - 1
		res += bits.OnesCount64(m) * penaltyN3
	}
	// Leading border: a window starting at -d covers modules 0..10-d with the
	// oldest module at bit 10.
	for d := 1; d <= 4; d++ {
		k := 11 - d
		v := reverseLine(line&(uint64(1)<<uint(k)-1), k)
		if v == penaltyPatternA || v == penaltyPatternB {
			res += penaltyN3
		}
	}
	// Trailing border: a window starting at s has n-s real modules, oldest at
	// bit 10, and the rest light.
	for s := n - 10; s <= n-7; s++ {
		c := n - s
		v := reverseLine((line>>uint(s))&(uint64(1)<<uint(c)-1), c) << uint(11-c)
		if v == penaltyPatternA || v == penaltyPatternB {
			res += penaltyN3
		}
	}
	return res
}

// rule2Pair counts rule-2 blocks between two packed rows: two adjacent columns
// whose modules are equal vertically and horizontally.
func rule2Pair(a, b uint64, n int) int {
	valid := ^uint64(0) >> uint(64-n)
	eq := ^(a ^ b) & valid       // columns with an equal vertical pair
	h := ^(a ^ (a >> 1)) & valid // columns equal to the next column
	return bits.OnesCount64(eq&(eq>>1)&h) * penaltyN2
}

// penaltyMask64 returns the penalty for a candidate mask on a symbol of at most
// 64 modules. It applies the mask through maskMatrix, packs each row and column
// into a uint64, then scores the lines in parallel.
func (b *builder) penaltyMask64(mask int) int {
	n := b.size
	total := n * n
	if len(b.maskScratch) != total {
		b.maskScratch = make([]byte, total)
	}
	maskMatrix(b.maskScratch, b.modules, b.isFunc, n, mask)

	if len(b.bitRows) != n {
		b.bitRows = make([]uint64, n)
		b.bitCols = make([]uint64, n)
	} else {
		clear(b.bitRows)
		clear(b.bitCols)
	}
	rows, cols := b.bitRows, b.bitCols

	dark := 0
	for y := 0; y < n; y++ {
		base := y * n
		var row uint64
		for x := 0; x < n; x++ {
			v := uint64(b.maskScratch[base+x] & 1)
			row |= v << uint(x)
			cols[x] |= v << uint(y)
		}
		rows[y] = row
		dark += bits.OnesCount64(row)
	}

	var res int
	for y := 0; y < n; y++ {
		res += runPenalty64(rows[y], n) + rule3Line(rows[y], n)
		if y+1 < n {
			res += rule2Pair(rows[y], rows[y+1], n)
		}
	}
	for x := 0; x < n; x++ {
		res += runPenalty64(cols[x], n) + rule3Line(cols[x], n)
	}
	res += abs(dark*2-total) * 10 / total * penaltyN4
	return res
}

func (b *builder) selectMask64() int {
	best, bestPen := 0, maxInt
	for m := 0; m < 8; m++ {
		b.drawFormatBits(m)
		if p := b.penaltyMask64(m); p < bestPen {
			best, bestPen = m, p
		}
	}
	return best
}
