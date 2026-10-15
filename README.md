# qrcode

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
  (`WithECI(26)` selects UTF-8).
- **Optimal segmentation.** An O(n) dynamic program splits the input across
  modes to minimise the encoded bit length, instead of encoding everything as
  bytes. It is verified against an O(n²) brute-force optimum in the test suite.
- **Full mask search.** All eight data-mask patterns are scored with the four
  ISO penalty rules, and the lowest-penalty mask is chosen. A specific mask can
  be forced with `WithMask`.
- **ECL boosting.** When the data still fits, the encoder raises the
  error-correction level for extra robustness.
- **Renderers.** PNG (`image/png`), compact SVG, half-block terminal text, and a
  raw module-matrix API.
- **No dependencies** beyond the standard library. Encoded `*QR` values are
  immutable and safe for concurrent use.

## Install

```sh
go get qrcode
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
`NewKanji`. Numeric and alphanumeric segments are validated against their
alphabets; Kanji segments must be Shift-JIS with an even length.

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
`ErrBadLevel`, `ErrBadMask`, `ErrInvalidSegment`, or `ErrEmptySegments` (match
with `errors.Is`).

## Testing and benchmarks

```sh
go test ./...
go test -run '^$' -bench . -benchmem
```

Selected results on a 12th-generation Intel Core i7 (Go 1.24, Windows):

| Benchmark | ns/op |
| --- | --- |
| `EncodeForcedMask` (url, v4) | 6 388 |
| `Encode/url` (v4, 8-mask search) | 111 479 |
| `Encode/numeric` (v21) | 245 160 |
| `Encode/alnum` (v13) | 625 339 |
| `RenderTerminal` | 4 365 |
| `RenderSVG` | 10 480 |
| `RenderPNG` (scale 4) | 235 488 |

The core encoder is validated against two independent implementations
(`rsc.io/qr` and `boombuler/barcode`) across every mode and level, byte-for-byte
at the matching mask.

## License

MIT — see [LICENSE](LICENSE).
