package qrcode

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

// segBitCost is the bit length of a single segment of the given mode and
// character count at a version, or inf if the mode cannot hold the data.
func segBitCost(m Mode, n, vr int) int {
	b := 4 + ccBits[m][vr]
	switch m {
	case modeNumeric:
		b += n / 3 * 10
		switch n % 3 {
		case 1:
			b += 4
		case 2:
			b += 7
		}
	case modeAlphanumeric:
		b += n/2*11 + n%2*6
	case modeByte:
		b += n * 8
	default:
		return inf
	}
	return b
}

func rangeNumeric(d []byte) bool {
	for _, c := range d {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func rangeAlnum(d []byte) bool {
	for _, c := range d {
		if alnumValue[c] < 0 {
			return false
		}
	}
	return true
}

// bruteBits computes the minimal encoded bit length over all partitions with an
// O(n^2) dynamic program, the exact optimum the O(n) planner must match.
func bruteBits(data []byte, vr int) int {
	n := len(data)
	best := make([]int, n+1)
	for i := 1; i <= n; i++ {
		best[i] = inf
	}
	for i := 1; i <= n; i++ {
		for j := 0; j < i; j++ {
			if best[j] >= inf {
				continue
			}
			seg := data[j:i]
			c := segBitCost(modeByte, i-j, vr)
			if rangeNumeric(seg) {
				c = min(c, segBitCost(modeNumeric, i-j, vr))
			}
			if rangeAlnum(seg) {
				c = min(c, segBitCost(modeAlphanumeric, i-j, vr))
			}
			if best[j]+c < best[i] {
				best[i] = best[j] + c
			}
		}
	}
	return best[n]
}

func TestPlannerOptimal(t *testing.T) {
	// A representative version for each character-count width range.
	verOfRange := []int{1, 10, 27}
	rng := rand.New(rand.NewSource(1))
	alphabet := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij $%*+-./:")
	for iter := 0; iter < 3000; iter++ {
		n := 1 + rng.Intn(40)
		data := make([]byte, n)
		for i := range data {
			data[i] = alphabet[rng.Intn(len(alphabet))]
		}
		for vr := 0; vr < 3; vr++ {
			segs := planSegments(data, vr)
			got := 0
			for _, s := range segs {
				got += s.bits(verOfRange[vr])
			}
			if want := bruteBits(data, vr); got != want {
				t.Fatalf("data=%q vr=%d: planner=%d brute=%d segs=%s", data, vr, got, want, dumpSegs(segs))
			}
			checkCoverage(t, data, segs)
		}
	}
}

// checkCoverage verifies the segments reconstruct the input exactly, each with a
// mode valid for its bytes.
func checkCoverage(t *testing.T, data []byte, segs []segment) {
	t.Helper()
	var joined []byte
	for _, s := range segs {
		switch s.mode {
		case modeNumeric:
			if !rangeNumeric(s.data) {
				t.Fatalf("numeric segment has non-digits: %q", s.data)
			}
		case modeAlphanumeric:
			if !rangeAlnum(s.data) {
				t.Fatalf("alnum segment has non-alnum bytes: %q", s.data)
			}
		}
		joined = append(joined, s.data...)
	}
	if !bytes.Equal(joined, data) {
		t.Fatalf("segments %q do not reconstruct %q", joined, data)
	}
}

func dumpSegs(segs []segment) string {
	s := ""
	for i, seg := range segs {
		if i > 0 {
			s += "|"
		}
		s += fmt.Sprintf("m%d:%s", seg.mode, seg.data)
	}
	return s
}
