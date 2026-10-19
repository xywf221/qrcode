// Package qrcode generates QR codes from arbitrary data.
//
// It implements the QR Code model as defined by ISO/IEC 18004, supporting
// versions 1 through 40, all four error-correction levels (L, M, Q, H),
// numeric/alphanumeric/byte/Kanji segment modes, ECI, and automatic optimal
// segmentation and mask selection.
//
// The zero-dependency core produces an immutable *QR, whose module matrix can
// be inspected directly or rendered to PNG, SVG or terminal text.
//
//	q, err := qrcode.NewString("https://example.com", qrcode.WithLevel(qrcode.Quartile))
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := q.WritePNG(w); err != nil {
//		log.Fatal(err)
//	}
package qrcode

import (
	"errors"
	"math"
)

// A Level is a QR error-correction level. Recovering more errors costs data
// capacity, so Low holds the most data and High the least.
type Level uint8

const (
	Low      Level = iota // L, recovers ~7% of codewords
	Medium                // M, recovers ~15%
	Quartile              // Q, recovers ~25%
	High                  // H, recovers ~30%
)

// String returns the single-letter designator ("L", "M", "Q", "H").
func (l Level) String() string {
	switch l {
	case Low:
		return "L"
	case Medium:
		return "M"
	case Quartile:
		return "Q"
	case High:
		return "H"
	}
	return "?"
}

// formatBits returns the 2-bit level field used in the format information.
// The mapping is L=01, M=00, Q=11, H=10.
func (l Level) formatBits() uint32 {
	switch l {
	case Low:
		return 1
	case Medium:
		return 0
	case Quartile:
		return 3
	default:
		return 2
	}
}

// A Mode identifies how a segment's data is encoded.
type Mode uint8

const (
	modeNumeric Mode = iota
	modeAlphanumeric
	modeByte
	modeKanji
	modeECI
)

// indicator returns the 4-bit mode indicator.
func (m Mode) indicator() uint32 {
	switch m {
	case modeNumeric:
		return 0x1
	case modeAlphanumeric:
		return 0x2
	case modeByte:
		return 0x4
	case modeKanji:
		return 0x8
	case modeECI:
		return 0x7
	}
	return 0
}

// Errors reported by New and its wrappers.
var (
	ErrTooLong        = errors.New("qrcode: data too long to encode")
	ErrBadVersion     = errors.New("qrcode: version out of range [1,40]")
	ErrBadLevel       = errors.New("qrcode: invalid error-correction level")
	ErrBadMask        = errors.New("qrcode: mask out of range [0,7]")
	ErrBadECI         = errors.New("qrcode: ECI designator out of range [0,999999]")
	ErrInvalidSegment = errors.New("qrcode: segment data is not valid for its mode")
	ErrEmptySegments  = errors.New("qrcode: explicit segments must not be empty")

	ErrStructuredAppend = errors.New("qrcode: invalid structured-append header")
	ErrTooManySymbols   = errors.New("qrcode: data needs more than 16 structured-append symbols")
)

// A QR is an encoded, immutable QR symbol. All of its methods are safe for
// concurrent use.
type QR struct {
	version int
	level   Level
	mask    int
	size    int
	modules []byte // row-major, size*size, non-zero means dark
}

// Size reports the symbol width in modules (4*version + 17).
func (q *QR) Size() int { return q.size }

// Version reports the symbol version, 1 through 40.
func (q *QR) Version() int { return q.version }

// Level reports the effective error-correction level, which may be higher than
// requested when ECL boosting found room at the chosen version.
func (q *QR) Level() Level { return q.level }

// Mask reports the data mask pattern in use, 0 through 7.
func (q *QR) Mask() int { return q.mask }

// Module reports whether the module at column x, row y is dark. Coordinates
// outside the symbol return false.
func (q *QR) Module(x, y int) bool {
	if uint(x) >= uint(q.size) || uint(y) >= uint(q.size) {
		return false
	}
	return q.modules[y*q.size+x] != 0
}

// Bitmap returns the module matrix in row-major order as size*size bytes, each
// 1 (dark) or 0 (light). The returned slice aliases the receiver and must not
// be modified; copy it if the caller needs to mutate it.
func (q *QR) Bitmap() []byte { return q.modules }

// config holds the resolved options for a single encoding.
type config struct {
	level       Level
	minVersion  int
	maxVersion  int
	mask        int
	boostECL    bool
	eci         int
	segments    []segment
	explicitSeg bool

	// Structured Append header fields; saTotal == 0 means "no header".
	saIndex  int
	saTotal  int
	saParity int
}

func defaultConfig() config {
	return config{
		level:      Medium,
		minVersion: 1,
		maxVersion: 40,
		mask:       -1,
		boostECL:   true,
		eci:        -1,
	}
}

// saHeader returns the Structured Append header to emit, or nil for a plain
// symbol.
func (c *config) saHeader() *saHeader {
	if c.saTotal <= 0 {
		return nil
	}
	return &saHeader{index: c.saIndex, total: c.saTotal, parity: c.saParity}
}

// saBits returns the number of bits the Structured Append header occupies, zero
// when no header is present.
func (c *config) saBits() int {
	if c.saTotal <= 0 {
		return 0
	}
	return saHeaderBits
}

// normalizeAndValidate clamps the version range into [1,40] and rejects an
// invalid level, version range or mask, in place. It is the single place that
// enforces these constraints, shared by build and Structured.
func (c *config) normalizeAndValidate() error {
	if c.level > High {
		return ErrBadLevel
	}
	if c.minVersion < 1 {
		c.minVersion = 1
	}
	if c.maxVersion > 40 {
		c.maxVersion = 40
	}
	if c.minVersion > c.maxVersion || c.maxVersion < 1 || c.minVersion > 40 {
		return ErrBadVersion
	}
	if c.mask > 7 {
		return ErrBadMask
	}
	if c.eci < -1 || c.eci > 999999 {
		return ErrBadECI
	}
	return nil
}

// An Option configures an encoding.
type Option func(*config)

// WithLevel sets the error-correction level. Default is Medium.
func WithLevel(l Level) Option { return func(c *config) { c.level = l } }

// WithMinVersion sets the smallest version considered for automatic sizing.
func WithMinVersion(v int) Option { return func(c *config) { c.minVersion = v } }

// WithMaxVersion sets the largest version considered for automatic sizing.
func WithMaxVersion(v int) Option { return func(c *config) { c.maxVersion = v } }

// WithVersion forces a specific version, disabling automatic sizing.
func WithVersion(v int) Option {
	return func(c *config) { c.minVersion, c.maxVersion = v, v }
}

// WithMask forces a data mask pattern (0-7) instead of choosing the best one.
func WithMask(m int) Option { return func(c *config) { c.mask = m } }

// WithBoostECL controls whether the encoder raises the error-correction level
// when the data still fits at the chosen version. Default is true.
func WithBoostECL(b bool) Option { return func(c *config) { c.boostECL = b } }

// WithECI selects an Extended Channel Interpretation designator, e.g. 26 for
// UTF-8. The designator must be in [0, 999999]; an out-of-range value makes New
// return ErrBadECI. Default is none.
func WithECI(designator int) Option { return func(c *config) { c.eci = designator } }

// WithSegments supplies explicit segments, bypassing automatic segmentation.
func WithSegments(segs ...Segment) Option {
	return func(c *config) {
		c.explicitSeg = true
		c.segments = make([]segment, len(segs))
		for i := range segs {
			c.segments[i] = segs[i].s
		}
	}
}

// New encodes data and returns the resulting symbol.
func New(data []byte, opts ...Option) (*QR, error) { return newQR(data, opts) }

// NewString encodes text as a byte-mode string. Unlike string content passed to
// New, the bytes are those of the UTF-8 string.
func NewString(text string, opts ...Option) (*QR, error) {
	return newQR([]byte(text), opts)
}

// Encode is a convenience wrapper that encodes text at the given level.
func Encode(text string, level Level) (*QR, error) {
	return newQR([]byte(text), []Option{WithLevel(level)})
}

func newQR(data []byte, opts []Option) (*QR, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return build(data, &cfg)
}

// build runs the encoding pipeline on an already-resolved configuration.
func build(data []byte, cfg *config) (*QR, error) {
	if err := cfg.normalizeAndValidate(); err != nil {
		return nil, err
	}
	if cfg.saTotal != 0 {
		if cfg.saTotal < 1 || cfg.saTotal > 16 || cfg.saIndex < 0 || cfg.saIndex >= cfg.saTotal ||
			cfg.saParity < 0 || cfg.saParity > 0xFF {
			return nil, ErrStructuredAppend
		}
	}

	segs, version, err := layout(data, cfg)
	if err != nil {
		return nil, err
	}
	// A higher level may fit at the same version after ECL boosting.
	level := cfg.level
	if cfg.boostECL {
		for l := High; l > level; l-- {
			if requiredBits(segs, version, cfg.eci)+cfg.saBits() <= dataBits(version, l) {
				level = l
				break
			}
		}
	}

	dataCW, err := assembleCodewords(segs, version, level, cfg.eci, cfg.saHeader())
	if err != nil {
		return nil, err
	}

	b := newBuilder(version, level)
	b.drawFunctionPatterns()
	b.placeCodewords(dataCW)

	mask := cfg.mask
	if mask < 0 {
		mask = b.selectMask()
	}
	b.applyMask(mask)
	b.drawFormatBits(mask)

	return &QR{
		version: version,
		level:   level,
		mask:    mask,
		size:    b.size,
		modules: b.modules,
	}, nil
}

const maxInt = math.MaxInt32
