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

### Fixed

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
