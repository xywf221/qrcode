# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- PNG rendering now emits a hand-rolled indexed-PNG document through a pooled
  zlib writer, an order of magnitude faster and far lighter on allocations than
  the standard `image/png` encoder (which rebuilds a ~1.3 MB flate compressor on
  every call).
- PNG rendering reuses its uncompressed scanline buffer through a pool, cutting
  the render path's allocations.
- Mask selection no longer writes each of the eight candidate masks into the
  symbol matrix: the penalty search now scores a reused masked copy, roughly 9%
  faster per encode, and the penalty pass sizes its scan scratch once instead of
  on every candidate.

### Fixed

- `Structured` no longer rejects numeric or alphanumeric messages that exceed a
  byte-mode estimate of sixteen symbols' capacity; the precheck now uses a
  mode-aware lower bound. It also reports `ErrStructuredOption` when passed
  `WithSegments` or `WithStructuredAppend`, instead of silently ignoring them.
- PNG and SVG rendering preserve the alpha channel of `WithColor` values: PNG
  emits a `tRNS` chunk and SVG a `fill-opacity` attribute, rather than
  premultiplying translucent colours into opaque ones.
- Out-of-range options are rejected instead of silently clamped or ignored:
  `WithMask` below -1 returns `ErrBadMask`, versions outside `[1,40]` return
  `ErrBadVersion`, and `WithStructuredAppend` with total 0 returns
  `ErrStructuredAppend`.
- Kanji segments now reject Shift-JIS pairs whose low byte is below `0x40`
  (e.g. `0x9330`). Such codes pass the JIS X 0208 range test but are not
  reversible through the 13-bit Kanji encoding and previously decoded back as a
  different value; `NewKanji` and `NewKanjiString` now report
  `ErrInvalidSegment` instead.
- `WithECI` rejects designators outside `[0, 999999]` with `ErrBadECI` rather
  than emitting a malformed designator.
- The Reed-Solomon generator polynomials are built during package init, removing
  a data race that could occur when encoding from multiple goroutines.
- `WithInvert` no longer flips the quiet zone in SVG and terminal output; the
  quiet zone is always light, matching the PNG renderer.

## [1.0.0] - 2026-10-15

### Added

- ISO/IEC 18004 QR Code generation for versions 1 through 40 at error-correction
  levels L, M, Q, and H.
- Numeric, alphanumeric, byte, and Kanji segment modes, plus ECI designators.
- Optimal segmentation via an O(n) dynamic program, validated against a
  brute-force optimum.
- Full eight-mask penalty evaluation with all four ISO penalty rules.
- Error-correction level boosting.
- Structured Append (mode 0x3) with sequence header, total count, and parity,
  including automatic multi-symbol splitting.
- UTF-8 to Shift-JIS conversion for Kanji mode, using a generated, embedded
  mapping table (no runtime dependency).
- Renderers: PNG (stdlib `image/png`), SVG, half-block terminal text, and a raw
  module-matrix API.
- Cross-implementation equivalence tests against `rsc.io/qr` and
  `boombuler/barcode`, plus a full decoder round-trip test.
- Fuzz tests, benchmarks, and a CI pipeline (build, vet, race, lint, fuzz).

[Unreleased]: https://github.com/xywf221/qrcode/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/xywf221/qrcode/releases/tag/v1.0.0
