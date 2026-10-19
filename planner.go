package qrcode

// Bit-cost model and dynamic program for choosing the minimal-bit segmentation
// of a byte string over the numeric, alphanumeric and byte modes.

const inf = 1 << 30

func modePeriod(m Mode) int {
	switch m {
	case modeNumeric:
		return 3
	case modeAlphanumeric:
		return 2
	default:
		return 1
	}
}

// modeDelta returns the additional bits incurred by growing a segment of mode m
// from length newLen-1 to newLen.
func modeDelta(m Mode, newLen int) int {
	switch m {
	case modeNumeric:
		if newLen%3 == 1 {
			return 4
		}
		return 3
	case modeAlphanumeric:
		if newLen%2 == 1 {
			return 6
		}
		return 5
	default:
		return 8
	}
}

func modeValid(m Mode, c byte) bool {
	switch m {
	case modeNumeric:
		return c >= '0' && c <= '9'
	case modeAlphanumeric:
		return alnumValue[c] >= 0
	default:
		return true
	}
}

// planSegments returns a minimal-bit segmentation of data for the character
// count widths of version range vr. It is an exact O(n) dynamic program: the
// state is (mode, run length modulo the mode's bit-group period).
func planSegments(data []byte, vr int) []segment {
	n := len(data)
	if n == 0 {
		return []segment{makeSeg(modeByte, nil, -1)}
	}

	var hdr, first [3]int
	for m := 0; m < 3; m++ {
		hdr[m] = 4 + ccBits[m][vr]
		first[m] = modeDelta(Mode(m), 1)
	}

	var cur [9]int
	for i := range cur {
		cur[i] = inf
	}
	bestState := make([]int8, n+1)
	bestState[0] = -1
	started := make([][9]bool, n+1)

	for i := 0; i < n; i++ {
		c := data[i]
		bv := inf
		for s := 0; s < 9; s++ {
			if cur[s] < bv {
				bv = cur[s]
			}
		}
		base := bv
		if i == 0 {
			base = 0
		}

		var next [9]int
		for s := range next {
			next[s] = inf
		}
		var stNext [9]bool

		for m := 0; m < 3; m++ {
			if !modeValid(Mode(m), c) {
				continue
			}
			p := modePeriod(Mode(m))
			if base < inf {
				ns := m*3 + 1%p
				if cost := base + hdr[m] + first[m]; cost < next[ns] {
					next[ns] = cost
					stNext[ns] = true
				}
			}
			for r := 0; r < p; r++ {
				s := m*3 + r
				if cur[s] >= inf {
					continue
				}
				ns := m*3 + (r+1)%p
				if cost := cur[s] + modeDelta(Mode(m), r+1); cost < next[ns] {
					next[ns] = cost
					stNext[ns] = false
				}
			}
		}

		cur = next
		started[i+1] = stNext
		mn, mi := inf, int8(-1)
		for s := 0; s < 9; s++ {
			if cur[s] < mn {
				mn, mi = cur[s], int8(s)
			}
		}
		bestState[i+1] = mi
	}

	// Recover the segmentation by following the back-pointers from the optimal
	// final state. A fresh-start state at position p marks a segment beginning
	// at char p-1 with mode s/3; otherwise the predecessor is the same mode one
	// run-length step back. Walking backwards records the boundaries in reverse.
	type boundary struct {
		start int
		mode  Mode
	}
	var rev []boundary
	s := bestState[n]
	for p := n; p > 0; {
		if started[p][s] {
			rev = append(rev, boundary{p - 1, Mode(int(s) / 3)})
			s = bestState[p-1]
			p--
		} else {
			m := int(s) / 3
			pm := modePeriod(Mode(m))
			s = int8(m*3 + (int(s)%3-1+pm)%pm)
			p--
		}
	}
	segs := make([]segment, 0, len(rev))
	for k := len(rev) - 1; k >= 0; k-- {
		st := rev[k].start
		end := n
		if k > 0 {
			end = rev[k-1].start
		}
		segs = append(segs, makeSeg(rev[k].mode, data[st:end], -1))
	}
	return segs
}

// layout chooses the smallest version that fits and returns the segmentation to
// use at that version.
func layout(data []byte, cfg *config) ([]segment, int, error) {
	if cfg.explicitSeg {
		if len(cfg.segments) == 0 {
			return nil, 0, ErrEmptySegments
		}
		for _, s := range cfg.segments {
			if err := validateSegment(s); err != nil {
				return nil, 0, err
			}
		}
		for v := cfg.minVersion; v <= cfg.maxVersion; v++ {
			if requiredBits(cfg.segments, v, cfg.eci)+cfg.saBits() <= dataBits(v, cfg.level) {
				return cfg.segments, v, nil
			}
		}
		return nil, 0, ErrTooLong
	}

	// No single symbol can hold more than 7089 characters (version 40-L,
	// numeric mode), so reject longer input before the segmentation DP runs.
	if len(data) > 7089 {
		return nil, 0, ErrTooLong
	}

	var plans [3][]segment
	var planned [3]bool
	for v := cfg.minVersion; v <= cfg.maxVersion; v++ {
		vr := verRange(v)
		if !planned[vr] {
			plans[vr] = planSegments(data, vr)
			planned[vr] = true
		}
		if requiredBits(plans[vr], v, cfg.eci)+cfg.saBits() <= dataBits(v, cfg.level) {
			return plans[vr], v, nil
		}
	}
	return nil, 0, ErrTooLong
}

// assembleCodewords encodes segs into the final interleaved data+EC codeword
// sequence ready for placement. A non-nil sa writes the Structured Append
// header ahead of the data stream.
func assembleCodewords(segs []segment, ver int, l Level, eci int, sa *saHeader) ([]byte, error) {
	capacity := dataCodewords(ver, l)
	var bb bitBuffer
	if sa != nil {
		sa.write(&bb)
	}
	if eci >= 0 {
		writeECI(&bb, eci)
	}
	for _, s := range segs {
		s.write(&bb, ver)
	}
	if rem := capacity*8 - bb.lenBits(); rem > 0 {
		n := rem
		if n > 4 {
			n = 4
		}
		bb.writeBits(0, n)
	}
	bb.padToByte()
	if bb.lenBits() > capacity*8 {
		return nil, ErrTooLong
	}
	bb.padToCap(capacity)
	return addECC(bb.buf, ver, l), nil
}

// addECC splits data codewords into blocks, appends Reed-Solomon codewords to
// each, and interleaves them into the final sequence.
func addECC(data []byte, ver int, l Level) []byte {
	raw := totalCodewords(ver)
	nb := numBlocks[l][ver]
	ecPer := eccPerBlock[l][ver]
	shortLen := raw / nb
	numShort := nb - raw%nb
	shortData := shortLen - ecPer

	blocksData := make([][]byte, nb)
	blocksEC := make([][]byte, nb)
	k := 0
	for j := 0; j < nb; j++ {
		dlen := shortData
		if j >= numShort {
			dlen++
		}
		blocksData[j] = data[k : k+dlen]
		k += dlen
		blocksEC[j] = rsEncode(blocksData[j], ecPer)
	}

	result := make([]byte, 0, raw)
	for i := 0; i <= shortData; i++ {
		for j := 0; j < nb; j++ {
			if i < len(blocksData[j]) {
				result = append(result, blocksData[j][i])
			}
		}
	}
	for i := 0; i < ecPer; i++ {
		for j := 0; j < nb; j++ {
			result = append(result, blocksEC[j][i])
		}
	}
	return result
}
