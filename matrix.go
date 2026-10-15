package qrcode

// builder assembles the module matrix: function patterns, codeword placement,
// masking and format/version information.
type builder struct {
	size    int
	version int
	level   Level
	modules []byte // row-major, 1 = dark
	isFunc  []byte // 1 = function/reserved module
}

func newBuilder(ver int, level Level) *builder {
	size := 4*ver + 17
	return &builder{
		size:    size,
		version: ver,
		level:   level,
		modules: make([]byte, size*size),
		isFunc:  make([]byte, size*size),
	}
}

func (b *builder) setFunc(x, y int, dark bool) {
	b.isFunc[y*b.size+x] = 1
	if dark {
		b.modules[y*b.size+x] = 1
	} else {
		b.modules[y*b.size+x] = 0
	}
}

func (b *builder) drawFunctionPatterns() {
	size := b.size
	for i := 0; i < size; i++ {
		b.setFunc(6, i, i%2 == 0)
		b.setFunc(i, 6, i%2 == 0)
	}
	b.drawFinder(3, 3)
	b.drawFinder(size-4, 3)
	b.drawFinder(3, size-4)

	pos := alignPositions(b.version)
	n := len(pos)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if (i == 0 && j == 0) || (i == 0 && j == n-1) || (i == n-1 && j == 0) {
				continue
			}
			b.drawAlignment(pos[i], pos[j])
		}
	}

	b.drawFormatBits(0)
	b.drawVersionBits()
}

func (b *builder) drawFinder(cx, cy int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			x, y := cx+dx, cy+dy
			if uint(x) >= uint(b.size) || uint(y) >= uint(b.size) {
				continue
			}
			d := abs(dx)
			if a := abs(dy); a > d {
				d = a
			}
			b.setFunc(x, y, d != 2 && d != 4)
		}
	}
}

func (b *builder) drawAlignment(cx, cy int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			d := abs(dx)
			if a := abs(dy); a > d {
				d = a
			}
			b.setFunc(cx+dx, cy+dy, d != 1)
		}
	}
}

func (b *builder) drawFormatBits(mask int) {
	size := b.size
	bits := formatInfo(b.level, mask)
	bit := func(i int) bool { return (bits>>uint(i))&1 != 0 }
	for i := 0; i <= 5; i++ {
		b.setFunc(8, i, bit(i))
	}
	b.setFunc(8, 7, bit(6))
	b.setFunc(8, 8, bit(7))
	b.setFunc(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		b.setFunc(14-i, 8, bit(i))
	}
	for i := 0; i < 8; i++ {
		b.setFunc(size-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		b.setFunc(8, size-15+i, bit(i))
	}
	b.setFunc(8, size-8, true) // fixed dark module
}

func (b *builder) drawVersionBits() {
	if b.version < 7 {
		return
	}
	bits := versionInfo(b.version)
	size := b.size
	for i := 0; i < 18; i++ {
		dark := (bits>>uint(i))&1 != 0
		a := size - 11 + i%3
		c := i / 3
		b.setFunc(a, c, dark)
		b.setFunc(c, a, dark)
	}
}

// placeCodewords fills the data region with the codeword bit stream, walking
// two-module-wide columns right to left in an up/down zigzag.
func (b *builder) placeCodewords(data []byte) {
	size := b.size
	total := len(data) * 8
	i := 0
	for right := size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < size; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				y := vert
				if (right+1)&2 == 0 {
					y = size - 1 - vert
				}
				idx := y*size + x
				if b.isFunc[idx] == 0 && i < total {
					b.modules[idx] = data[i>>3] >> uint(7-(i&7)) & 1
					i++
				}
			}
		}
	}
}

// applyMask XORs the data mask over all non-function modules. It is its own
// inverse, so calling it twice restores the original. The mask condition is
// selected once per row rather than per module.
//
//nolint:gocyclo // the eight data-mask formulas are enumerated inline for speed
func (b *builder) applyMask(mask int) {
	size := b.size
	mods, funcs := b.modules, b.isFunc
	for y := 0; y < size; y++ {
		base := y * size
		row, fr := mods[base:base+size], funcs[base:base+size]
		switch mask {
		case 0:
			for x := 0; x < size; x++ {
				if fr[x] == 0 && (x+y)&1 == 0 {
					row[x] ^= 1
				}
			}
		case 1:
			if y&1 == 0 {
				for x := 0; x < size; x++ {
					if fr[x] == 0 {
						row[x] ^= 1
					}
				}
			}
		case 2:
			for x := 0; x < size; x++ {
				if fr[x] == 0 && x%3 == 0 {
					row[x] ^= 1
				}
			}
		case 3:
			for x := 0; x < size; x++ {
				if fr[x] == 0 && (x+y)%3 == 0 {
					row[x] ^= 1
				}
			}
		case 4:
			for x := 0; x < size; x++ {
				if fr[x] == 0 && (y/2+x/3)&1 == 0 {
					row[x] ^= 1
				}
			}
		case 5:
			for x := 0; x < size; x++ {
				if fr[x] == 0 && x*y%2+x*y%3 == 0 {
					row[x] ^= 1
				}
			}
		case 6:
			for x := 0; x < size; x++ {
				if fr[x] == 0 && (x*y%2+x*y%3)&1 == 0 {
					row[x] ^= 1
				}
			}
		default:
			for x := 0; x < size; x++ {
				if fr[x] == 0 && ((x+y)&1+x*y%3)&1 == 0 {
					row[x] ^= 1
				}
			}
		}
	}
}

func (b *builder) selectMask() int {
	best, bestPen := 0, maxInt
	for m := 0; m < 8; m++ {
		b.applyMask(m)
		b.drawFormatBits(m)
		if p := b.penalty(); p < bestPen {
			best, bestPen = m, p
		}
		b.applyMask(m)
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
