package qrcode

// bitBuffer appends big-endian bit fields to a byte slice. Pending bits live in
// a uint64 accumulator so whole bytes are emitted as soon as they fill.
type bitBuffer struct {
	buf []byte
	acc uint64
	n   uint // pending bits in acc, always < 8 between calls
}

func (b *bitBuffer) writeBits(v uint32, n int) {
	if n <= 0 {
		return
	}
	b.acc = b.acc<<uint(n) | uint64(v&(1<<uint(n)-1))
	b.n += uint(n)
	for b.n >= 8 {
		b.n -= 8
		b.buf = append(b.buf, byte(b.acc>>b.n))
	}
	b.acc &= 1<<b.n - 1
}

func (b *bitBuffer) lenBits() int { return len(b.buf)*8 + int(b.n) }

// padToByte flushes pending bits by appending zero bits up to a byte boundary.
func (b *bitBuffer) padToByte() {
	if b.n != 0 {
		b.writeBits(0, int(8-b.n))
	}
}

var padCodewords = [2]byte{0xEC, 0x11}

// padToCap fills the buffer with the alternating pad codewords until it holds
// capacity data codewords. The buffer must be byte-aligned.
func (b *bitBuffer) padToCap(capacity int) {
	for i := 0; len(b.buf) < capacity; i++ {
		b.buf = append(b.buf, padCodewords[i&1])
	}
}
