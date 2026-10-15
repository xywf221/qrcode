package qrcode

// Reed-Solomon over GF(2^8) with the QR primitive polynomial
// x^8 + x^4 + x^3 + x^2 + 1 (0x11D) and generator element 0x02.

var (
	gfExp [512]byte // gfExp[i] = 2^i, doubled to avoid a modulo in gfMul
	gfLog [256]byte // gfLog[x] = i such that 2^i = x, defined for x != 0
)

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = byte(x)
		gfLog[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11D
		}
	}
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[int(gfLog[a])+int(gfLog[b])]
}

// genTerm is a non-zero coefficient of a generator polynomial.
type genTerm struct {
	idx int  // position in the coefficient slice, highest degree first
	log byte // gfLog of the coefficient
}

var genCache [31][]genTerm

// generatorTerms returns the generator polynomial of degree deg as its non-zero
// coefficients, cached. g(x) = (x - a^0)(x - a^1)...(x - a^(deg-1)); the leading
// coefficient (always 1) is omitted.
func generatorTerms(deg int) []genTerm {
	if deg < 1 || deg > 30 {
		panic("qrcode: invalid reed-solomon degree")
	}
	if genCache[deg] != nil {
		return genCache[deg]
	}
	// Build polynomial coefficients, highest degree first, including leading 1.
	// Each step multiplies by (x + a^i), represented as p*x + p*a^i.
	poly := []byte{1}
	for i := 0; i < deg; i++ {
		r := gfExp[i]
		next := make([]byte, len(poly)+1)
		for k := range next {
			if k < len(poly) {
				next[k] ^= poly[k]
			}
			if k > 0 {
				next[k] ^= gfMul(poly[k-1], r)
			}
		}
		poly = next
	}
	// poly[0] is the leading 1; store the remaining coefficients.
	terms := make([]genTerm, 0, deg)
	for i := 1; i < len(poly); i++ {
		if poly[i] != 0 {
			terms = append(terms, genTerm{idx: i - 1, log: gfLog[poly[i]]})
		}
	}
	genCache[deg] = terms
	return terms
}

// rsEncode returns the ecLen error-correction codewords for data.
func rsEncode(data []byte, ecLen int) []byte {
	terms := generatorTerms(ecLen)
	rem := make([]byte, ecLen)
	for _, b := range data {
		factor := b ^ rem[0]
		copy(rem, rem[1:])
		rem[ecLen-1] = 0
		if factor != 0 {
			lf := gfLog[factor]
			for _, t := range terms {
				rem[t.idx] ^= gfExp[int(t.log)+int(lf)]
			}
		}
	}
	return rem
}
