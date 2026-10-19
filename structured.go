package qrcode

// Structured Append (mode 0x3) lets one logical message span up to 16 symbols.
// Every symbol carries a header identifying its position, the total number of
// symbols, and an 8-bit parity of the whole original message.

// saHeaderBits is the size of the Structured Append header: a 4-bit mode
// indicator, a 4-bit symbol position, a 4-bit (total-1) field, and an 8-bit
// parity byte.
const saHeaderBits = 20

// saHeader is the Structured Append header emitted ahead of a symbol's data.
type saHeader struct {
	index  int // zero-based symbol position
	total  int // number of symbols in the sequence
	parity int // XOR of every byte of the original message
}

func (h *saHeader) write(bb *bitBuffer) {
	bb.writeBits(0x3, 4)
	bb.writeBits(uint32(h.index), 4)
	bb.writeBits(uint32(h.total-1), 4)
	bb.writeBits(uint32(h.parity), 8)
}

// WithStructuredAppend marks the symbol as one part of a Structured Append
// sequence. index is the zero-based position, total the number of symbols
// (1 to 16), and parity the XOR of every byte of the original message. Prefer
// Structured, which computes the parity and splits the message for you.
func WithStructuredAppend(index, total int, parity byte) Option {
	return func(c *config) {
		c.saIndex, c.saTotal, c.saParity = index, total, int(parity)
		c.saSet = true
	}
}

// Structured encodes data across one or more QR symbols using Structured
// Append, allowing a single message to exceed the capacity of one symbol. All
// symbols share one version and error-correction level; the returned slice is
// in sequence order. The planner uses the smallest version that splits the
// message into at most 16 symbols and returns ErrTooManySymbols if none does.
// WithSegments and WithStructuredAppend are not supported and return
// ErrStructuredOption.
func Structured(data []byte, opts ...Option) ([]*QR, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	cfg.boostECL = false // keep every symbol at the same level

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	// Structured computes its own header and segmentation, so caller-supplied
	// segments or headers cannot be honoured.
	if cfg.explicitSeg || cfg.saSet {
		return nil, ErrStructuredOption
	}

	// Sixteen version-40 symbols bound the largest possible message. Compare a
	// mode-aware lower bound: a byte-based estimate would reject numeric and
	// alphanumeric messages, which encode in under eight bits per character.
	if structuredBitsLowerBound(data) > 16*int64(dataBits(40, cfg.level)) {
		return nil, ErrTooManySymbols
	}

	parity := byte(0)
	for _, b := range data {
		parity ^= b
	}

	for v := cfg.minVersion; v <= cfg.maxVersion; v++ {
		chunks := splitStructured(data, v, &cfg)
		if chunks == nil || len(chunks) > 16 {
			continue
		}
		out := make([]*QR, len(chunks))
		for i, ch := range chunks {
			sc := cfg
			sc.minVersion, sc.maxVersion = v, v
			sc.explicitSeg = true
			sc.segments = planSegments(ch, verRange(v))
			sc.saIndex, sc.saTotal, sc.saParity = i, len(chunks), int(parity)
			q, err := build(ch, &sc)
			if err != nil {
				return nil, err
			}
			out[i] = q
		}
		return out, nil
	}
	return nil, ErrTooManySymbols
}

// StructuredString is Structured for a UTF-8 string.
func StructuredString(text string, opts ...Option) ([]*QR, error) {
	return Structured([]byte(text), opts...)
}

// structuredBitsLowerBound returns a lower bound on the bits needed to encode
// data with automatic segmentation: three bits per digit, five per
// alphanumeric character, and eight per other byte. Per-segment overheads and
// group rounding only add bits, so a bound above the total capacity proves the
// message cannot fit.
func structuredBitsLowerBound(data []byte) int64 {
	var bits int64
	for _, c := range data {
		switch {
		case c >= '0' && c <= '9':
			bits += 10 / 3
		case alnumValue[c] >= 0:
			bits += 11 / 2
		default:
			bits += 8
		}
	}
	return bits
}

// splitStructured partitions data into the largest chunks that each fit one
// symbol at version v, or nil if even a single byte does not fit.
func splitStructured(data []byte, v int, cfg *config) [][]byte {
	avail := dataBits(v, cfg.level) - saHeaderBits
	if cfg.eci >= 0 {
		avail -= 4 + eciValueBits(cfg.eci)
	}
	if avail <= 0 {
		return nil
	}
	if len(data) == 0 {
		return [][]byte{nil}
	}

	vr := verRange(v)
	n := len(data)
	var chunks [][]byte
	for i := 0; i < n; {
		// Largest end position whose planned encoding fits the available bits.
		lo, hi, best := i+1, n, -1
		for lo <= hi {
			mid := int(uint(lo+hi) >> 1)
			if requiredBits(planSegments(data[i:mid], vr), v, -1) <= avail {
				best, lo = mid, mid+1
			} else {
				hi = mid - 1
			}
		}
		if best < 0 {
			return nil
		}
		chunks = append(chunks, data[i:best])
		i = best
		if len(chunks) > 16 {
			return nil // cannot fit within sixteen symbols at this version
		}
	}
	return chunks
}
