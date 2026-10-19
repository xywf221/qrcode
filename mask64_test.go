package qrcode

import (
	"math/rand"
	"testing"
)

// TestPenaltyMask64MatchesScalar pins the bit-parallel fast path to the scalar
// scorer: it must report the same penalty for every candidate mask and select
// the same winner. Random function patterns included, so masked and unmasked
// modules both occur.
func TestPenaltyMask64MatchesScalar(t *testing.T) {
	rng := rand.New(rand.NewSource(64))
	for ver := 3; ver <= 11; ver++ {
		for iter := 0; iter < 10; iter++ {
			b := newBuilder(ver, Medium)
			for i := range b.modules {
				b.modules[i] = byte(rng.Intn(2))
				if rng.Intn(4) == 0 {
					b.isFunc[i] = 1
				}
			}
			for m := 0; m < 8; m++ {
				b.drawFormatBits(m)
				if got, want := b.penaltyMask64(m), b.penaltyMask(m); got != want {
					t.Fatalf("v%d iter %d mask %d: penaltyMask64 = %d, penaltyMask = %d",
						ver, iter, m, got, want)
				}
			}
			if got, want := b.selectMask64(), b.selectMaskScalar(); got != want {
				t.Fatalf("v%d iter %d: selectMask64 = %d, selectMaskScalar = %d", ver, iter, got, want)
			}
		}
	}
}
