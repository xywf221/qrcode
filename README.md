# qrcode

[![ci](https://github.com/xywf221/qrcode/actions/workflows/ci.yml/badge.svg)](https://github.com/xywf221/qrcode/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/xywf221/qrcode.svg)](https://pkg.go.dev/github.com/xywf221/qrcode)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A modern, zero-dependency QR code generator for Go. It implements ISO/IEC 18004
end to end — versions 1–40, all four error-correction levels, every data mode,
automatic optimal segmentation, and the full eight-mask penalty search.

```go
q, err := qrcode.NewString("https://example.com", qrcode.WithLevel(qrcode.Quartile))
if err != nil {
	log.Fatal(err)
}
if err := q.WritePNG(w); err != nil {
	log.Fatal(err)
}
```

## Features

- **All versions and levels.** Versions 1–40, error correction L/M/Q/H, with
  automatic sizing (or an exact version forced by the caller).
- **Every mode.** Numeric, alphanumeric, byte, and Kanji, plus ECI designators
  (`WithECI(26)` selects UTF-8). `NewKanjiString` converts UTF-8 to Shift-JIS
  for you.
- **Optimal segmentation.** An O(n) dynamic program splits the input across
  modes to minimise the encoded bit length, instead of encoding everything as
  bytes. It is verified against an O(n²) brute-force optimum in the test suite.
- **Full mask search.** All eight data-mask patterns are scored with the four
  ISO penalty rules, and the lowest-penalty mask is chosen. A specific mask can
  be forced with `WithMask`.
- **ECL boosting.** When the data still fits, the encoder raises the
  error-correction level for extra robustness.
- **Structured Append.** Messages larger than one symbol are split across up to
  16 symbols with the standard sequence header and parity, via `Structured`.
- **Renderers.** PNG (`image/png`), compact SVG, half-block terminal text, and a
  raw module-matrix API.
- **No dependencies** beyond the standard library. Encoded `*QR` values are
  immutable and safe for concurrent use.

## Install

```sh
go get github.com/xywf221/qrcode
```

Requires Go 1.24 or newer.

## Usage

### Encoding

`NewString` encodes a UTF-8 string as byte data; `New` takes raw bytes.

```go
q, err := qrcode.NewString("HELLO", qrcode.WithLevel(qrcode.Low), qrcode.WithBoostECL(false))
fmt.Println(q.Version(), q.Size(), q.Mask()) // 1 21 2
```

### Options

| Option | Effect |
| --- | --- |
| `WithLevel(Level)` | Error-correction level: `Low`, `Medium` (default), `Quartile`, `High`. |
| `WithVersion(v)` | Force version `v` (1–40), disabling automatic sizing. |
| `WithMinVersion(v)` / `WithMaxVersion(v)` | Bound the versions considered for automatic sizing. |
| `WithMask(m)` | Force data mask `m` (0–7) instead of the penalty-optimal one. |
| `WithBoostECL(bool)` | Raise the EC level when the data still fits (default `true`). |
| `WithECI(designator)` | Emit an ECI designator, e.g. `26` for UTF-8. |
| `WithSegments(...)` | Supply explicit pre-typed segments (see below). |
| `WithStructuredAppend(index, total, parity)` | Mark the symbol as one part of a Structured Append sequence. |

### Building a QR from segments

For full control over how data is split, build segments directly and pass them
with `WithSegments`:

```go
q, err := qrcode.New(nil, qrcode.WithSegments(
	qrcode.NewNumeric("1234567890"),
	qrcode.NewAlphanumeric("HELLO WORLD"),
	qrcode.NewText("café"),        // byte mode, UTF-8
))
```

Constructors: `NewNumeric`, `NewAlphanumeric`, `NewBytes`, `NewText`,
`NewKanji`, and `NewKanjiString`. Numeric and alphanumeric segments are
validated against their alphabets; `NewKanji` takes Shift-JIS with an even
length, while `NewKanjiString` accepts UTF-8 and converts it for you.

### Structured Append

To encode a message larger than one symbol, let `Structured` split it across up
to 16 symbols, adding the standard sequence header and parity:

```go
symbols, err := qrcode.StructuredString(message, qrcode.WithLevel(qrcode.Quartile))
if err != nil {
	log.Fatal(err)
}
for i, sym := range symbols {
	if err := sym.WritePNG(w); err != nil { // render each part
		log.Fatal(err)
	}
	_ = i
}
```

All symbols share one version and level. `ErrTooManySymbols` is returned if the
message needs more than 16 symbols at any version in range.

### Inspecting the symbol

```go
q.Size()          // width in modules (4*version + 17)
q.Version()       // 1–40
q.Level()         // effective level (may be higher than requested after boosting)
q.Mask()          // data mask in use, 0–7
q.Module(x, y)    // is the module at column x, row y dark?
q.Bitmap()        // row-major size*size bytes, 1 = dark (aliases the receiver)
```

### Rendering

All renderers accept the same options: `WithScale`, `WithQuietZone`,
`WithColor`, and `WithInvert`.

```go
png, err := q.PNG(qrcode.WithScale(8))           // []byte
err = q.WritePNG(w, qrcode.WithScale(8))         // to any io.Writer
svg := q.SVG()                                    // string
err = q.WriteSVG(w)
fmt.Print(q.Terminal(qrcode.WithQuietZone(2)))    // half-block text
```

| Render option | Effect |
| --- | --- |
| `WithScale(px)` | Pixel size of one module (PNG/SVG). Default `4`. |
| `WithQuietZone(modules)` | Light margin around the symbol. Default `4`. |
| `WithColor(fg, bg)` | Foreground (dark) and background colours. |
| `WithInvert(bool)` | Swap dark and light; useful on dark terminals. |

## Errors

Encode failures return one of the sentinel errors `ErrTooLong`, `ErrBadVersion`,
`ErrBadLevel`, `ErrBadMask`, `ErrInvalidSegment`, `ErrEmptySegments`,
`ErrStructuredAppend`, or `ErrTooManySymbols` (match with `errors.Is`).

## Testing and benchmarks

```sh
go test ./...
go test -race ./...
go test -run '^$' -fuzz FuzzEncode
go test -run '^$' -bench . -benchmem
```

Verification is layered:

- **Round-trip.** A decoder written independently from the specification
  (function patterns, format information, mask conditions, de-interleaving, and
  Reed-Solomon syndromes) reads every encoded symbol back, and the payload must
  match the input. The fuzzer runs the same check on arbitrary data.
- **Golden hashes.** SHA-256 hashes of representative matrices, produced by an
  encoder cross-checked byte-for-byte against `rsc.io/qr` and
  `boombuler/barcode`, lock the output against regressions.
- **Planner.** The O(n) segmentation is compared against an O(n²) brute-force
  optimum over thousands of random inputs.
- **Penalty.** The optimised scorer is compared against a naive reading of the
  four rules.

Indicative timings on an Intel Core i7 (Go 1.24, Windows). Laptop timings vary
widely between runs, so treat these as rough magnitudes:

| Benchmark | ns/op |
| --- | --- |
| `EncodeForcedMask` (url, v4) | ~8 µs |
| `Encode/url` (v4, 8-mask search) | ~110 µs |
| `Encode/numeric` (v21) | ~245 µs |
| `Encode/alnum` (v13) | ~625 µs |
| `RenderPNG` (scale 4) | ~300 µs |
| `RenderSVG` | ~10 µs |
| `RenderTerminal` | ~5 µs |

## License

MIT — see [LICENSE](LICENSE).
