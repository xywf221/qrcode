package qrcode

import (
	"math/rand"
	"testing"
)

// naivePenalty is a straightforward, obviously-correct reading of the four
// penalty rules used only to validate the optimised penalty implementation.
func naivePenalty(mods []byte, size int) int {
	get := func(x, y int) bool {
		if x < 0 || y < 0 || x >= size || y >= size {
			return false
		}
		return mods[y*size+x] != 0
	}
	p1 := []bool{true, false, true, true, true, false, true, false, false, false, false}
	p2 := []bool{false, false, false, false, true, false, true, true, true, false, true}
	res := 0
	for dir := 0; dir < 2; dir++ {
		at := func(i, a int) bool {
			if dir == 0 {
				return get(i, a)
			}
			return get(a, i)
		}
		for a := 0; a < size; a++ {
			run, prev := 1, at(0, a)
			for i := 1; i < size; i++ {
				if v := at(i, a); v == prev {
					run++
				} else {
					if run >= 5 {
						res += run - 2
					}
					prev, run = v, 1
				}
			}
			if run >= 5 {
				res += run - 2
			}
			line := make([]bool, size+8)
			for i := 0; i < size; i++ {
				line[i+4] = at(i, a)
			}
			for i := 0; i+11 <= len(line); i++ {
				m1, m2 := true, true
				for j := 0; j < 11; j++ {
					if line[i+j] != p1[j] {
						m1 = false
					}
					if line[i+j] != p2[j] {
						m2 = false
					}
				}
				if m1 {
					res += 40
				}
				if m2 {
					res += 40
				}
			}
		}
	}
	for y := 0; y < size-1; y++ {
		for x := 0; x < size-1; x++ {
			c := get(x, y)
			if c == get(x+1, y) && c == get(x, y+1) && c == get(x+1, y+1) {
				res += 3
			}
		}
	}
	dark := 0
	for _, v := range mods {
		if v != 0 {
			dark++
		}
	}
	total := size * size
	res += abs(dark*2-total) * 10 / total * 10
	return res
}

// unmaskedBuilder builds a symbol through the normal pipeline but stops before
// mask selection, so tests can evaluate every candidate from the same state.
func unmaskedBuilder(t *testing.T, text string, level Level) *builder {
	t.Helper()
	cfg := defaultConfig()
	cfg.level = level
	cfg.boostECL = false
	segs, ver, err := layout([]byte(text), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	cw, err := assembleCodewords(segs, ver, level, -1, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := newBuilder(ver, level)
	b.drawFunctionPatterns()
	b.placeCodewords(cw)
	return b
}

// TestPenaltyMaskMatchesApply checks the masked-copy scorer against the
// straightforward "apply the mask, then score" path, and selectMask against a
// naive search over the same eight candidates.
func TestPenaltyMaskMatchesApply(t *testing.T) {
	base := unmaskedBuilder(t, "https://example.com/some/path?query=value&x=1", Quartile)

	for m := 0; m < 8; m++ {
		base.drawFormatBits(m)
		got := base.penaltyMask(m)

		naive := newBuilder(base.version, base.level)
		naive.drawFunctionPatterns()
		copy(naive.modules, base.modules)
		naive.applyMask(m)
		if want := naive.penalty(); got != want {
			t.Errorf("mask %d: penaltyMask = %d, apply+penalty = %d", m, got, want)
		}
	}

	// selectMask must return an argmin of the naive scores.
	best, bestPen := 0, maxInt
	for m := 0; m < 8; m++ {
		naive := newBuilder(base.version, base.level)
		naive.drawFunctionPatterns()
		copy(naive.modules, base.modules)
		naive.drawFormatBits(m)
		naive.applyMask(m)
		if p := naive.penalty(); p < bestPen {
			best, bestPen = m, p
		}
	}
	if got := base.selectMask(); got != best {
		t.Errorf("selectMask = %d, want %d", got, best)
	}
}

func TestPenaltyMatchesNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for _, size := range []int{21, 25, 45, 69, 97} {
		for iter := 0; iter < 50; iter++ {
			mods := make([]byte, size*size)
			for i := range mods {
				mods[i] = byte(rng.Intn(2))
			}
			b := &builder{size: size, modules: mods}
			if got, want := b.penalty(), naivePenalty(mods, size); got != want {
				t.Fatalf("size=%d iter=%d: penalty=%d naive=%d", size, iter, got, want)
			}
		}
	}
}
