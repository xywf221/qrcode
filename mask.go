package qrcode

// Mask penalty weights and the four scoring rules from the specification.
const (
	penaltyN1 = 3
	penaltyN2 = 3
	penaltyN3 = 40
	penaltyN4 = 10
)

// The finder-like patterns of rule 3, as 11-bit windows: the 1:1:3:1:1 core
// followed by four light modules, and preceded by four light modules.
const (
	penaltyPatternA = 0b10111010000
	penaltyPatternB = 0b00001011101
)

// penalty returns the total mask penalty score of the symbol as currently
// drawn; the encoder picks the mask with the lowest score.
func (b *builder) penalty() int { return b.penaltyOf(b.modules) }

// penaltyOf scores an arbitrary module matrix (row-major, 1 = dark) against the
// four ISO/IEC 18004 rules, as implemented by ZXing: rule 3 matches the literal
// 11-module patterns with the symbol border treated as light.
func (b *builder) penaltyOf(mods []byte) int {
	size := b.size
	var res int

	// Per-column state for the incremental column scans of rules 1 and 3, so a
	// single suite of row-major passes covers every rule. The scratch is reused
	// across the eight mask evaluations, so only the first call allocates.
	if len(b.penRun) != size {
		b.penRun = make([]int32, size)
		b.penPrev = make([]byte, size)
		b.penWin = make([]uint, size)
	} else {
		clear(b.penRun)
		clear(b.penPrev)
		clear(b.penWin)
	}
	colRun, colPrev, colWin := b.penRun, b.penPrev, b.penWin

	dark := 0
	for y := 0; y < size; y++ {
		base := y * size
		row := mods[base : base+size]

		res += linePenalty(row)

		for x := 0; x < size; x++ {
			v := row[x]
			dark += int(v)

			// Rule 1 down the column.
			if r := colRun[x]; r == 0 {
				colPrev[x], colRun[x] = v, 1
			} else if v == colPrev[x] {
				colRun[x] = r + 1
			} else {
				if r >= 5 {
					res += int(r) - 2
				}
				colPrev[x], colRun[x] = v, 1
			}

			// Rule 3 down the column; the leading light border is implicit, so
			// the window is complete once the seventh module has been fed.
			win := (colWin[x]<<1 | uint(v)) & 0x7ff
			colWin[x] = win
			if y >= 6 && (win == penaltyPatternA || win == penaltyPatternB) {
				res += penaltyN3
			}
		}

		// Rule 2 with the row below.
		if y+1 < size {
			nxt := mods[base+size : base+2*size]
			for x := 0; x+1 < size; x++ {
				c := row[x]
				if c == row[x+1] && c == nxt[x] && c == nxt[x+1] {
					res += penaltyN2
				}
			}
		}
	}

	// Close the column scans: the final run and the four-module light border.
	for x := 0; x < size; x++ {
		if r := colRun[x]; r >= 5 {
			res += int(r) - 2
		}
		win := colWin[x]
		for i := 0; i < 4; i++ {
			win = (win << 1) & 0x7ff
			if win == penaltyPatternA || win == penaltyPatternB {
				res += penaltyN3
			}
		}
	}

	// Rule 4: deviation of the dark-module balance from 50%, in 5% steps.
	total := size * size
	res += abs(dark*2-total) * 10 / total * penaltyN4
	return res
}

// linePenalty applies rule 1 (long runs) and rule 3 (finder-like patterns) to a
// line of module values (each 0 or 1) of length size.
func linePenalty(mods []byte) int {
	size := len(mods)
	var res int

	// Rule 1: runs of five or more same-coloured modules.
	run := 1
	prev := mods[0]
	for i := 1; i < size; i++ {
		c := mods[i]
		if c == prev {
			run++
		} else {
			if run >= 5 {
				res += run - 2
			}
			prev, run = c, 1
		}
	}
	if run >= 5 {
		res += run - 2
	}

	// Rule 3: finder-like patterns over the line with a four-module light border
	// on each side, matched as an eleven-bit sliding window. The leading border
	// is implicit (it contributes no set bits), so the window is complete once
	// the seventh module has been shifted in.
	var win uint
	for i := 0; i < size && i < 6; i++ {
		win = (win<<1 | uint(mods[i])) & 0x7ff
	}
	for i := 6; i < size; i++ {
		win = (win<<1 | uint(mods[i])) & 0x7ff
		if win == penaltyPatternA || win == penaltyPatternB {
			res += penaltyN3
		}
	}
	for i := 0; i < 4; i++ {
		win = (win << 1) & 0x7ff
		if win == penaltyPatternA || win == penaltyPatternB {
			res += penaltyN3
		}
	}
	return res
}
