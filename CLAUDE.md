# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A zero-dependency QR code generator implementing ISO/IEC 18004 (versions 1–40,
levels L/M/Q/H, numeric/alphanumeric/byte/Kanji modes, ECI, Structured Append,
optimal segmentation, all eight data masks). The module path is
`github.com/xywf221/qrcode`. The library has **no runtime dependencies** — keep it
that way.

## Commands

```sh
go build ./...
go vet ./...
go test ./...                      # full suite
go test -race ./...                # race detector
go test -run TestName ./...        # a single test
go test -run '^$' -bench . -benchmem
go test -run '^$' -fuzz FuzzEncode -fuzztime 30s .   # fuzz (encode + decode round-trip)
go test -run '^$' -fuzz FuzzPenalty -fuzztime 15s .  # fuzz (penalty vs naive)
golangci-lint run                  # config: .golangci.yml; must stay at 0 issues
gofmt -l .                         # CI fails on unformatted code
```

Regenerate the Shift-JIS table (needs the generator's own module):

```sh
cd internal/sjisgen && go run . > ../../sjis_table.go && gofmt -w ../../sjis_table.go
```

## Architecture

The encode pipeline is a single pass through these files, in order:

`qrcode.go` (public API, options, validation) → `planner.go` (`layout` picks the
version, `planSegments` does segmentation, `assembleCodewords` builds the data +
interleaved EC codewords) → `matrix.go` (`builder` draws function patterns,
places codewords, selects and applies a mask) → the `QR` struct.

Data transformations, in sequence:

1. **Segmentation.** `planSegments` (`planner.go`) is an exact O(n) dynamic
   program over the state `(mode, run-length mod period)`; it minimises total
   bits across numeric/alphanumeric/byte. It is validated against an O(n²)
   brute-force optimum in `planner_test.go` — if you change it, that test is the
   contract. Character-count widths depend on the version range (`verRange`), so
   segmentation is planned per range (1–9, 10–26, 27–40).
2. **Bit assembly.** `bitBuffer` (`bitbuffer.go`) accumulates bits; segment
   writers live in `segment.go`. A Structured Append header and an ECI
   designator are written *ahead* of the segments, in that order.
3. **Error correction.** `gf256.go` holds GF(2⁸) tables (`0x11D`) and
   `rsEncode`; `addECC` (`planner.go`) splits data into blocks and interleaves
   data and EC codewords.
4. **Placement + masking.** `placeCodewords` (`matrix.go`) walks two-module
   columns right-to-left in an up/down zigzag, skipping `isFunc` modules.
   `applyMask` has one specialised branch per mask, hoisted out of the x-loop
   for speed; `selectMask` tries all eight and keeps the lowest `penalty()`.

### Invariants that span files

- **Structured Append size accounting.** The 20-bit header (`saHeaderBits`,
  `structured.go`) is *not* a `segment`. Every place that compares required bits
  against `dataBits` must add `cfg.saBits()` — see `layout` and `build`
  (`qrcode.go`). Miss one and a symbol overflows undetected.
- **The penalty scorer must match the reference.** `penalty()` (`mask.go`) is a
  single fused pass; `penalty_test.go` asserts it equals `naivePenalty` on random
  matrices. The rule-3 11-bit window is masked with `& 0x7ff` — an un-masked
  window silently undercounts.
- **Mask 4 is `(y/2 + x/3) % 2`.** All eight mask conditions are defined in
  `applyMask` (`matrix.go`) and mirrored in the decoder's `maskCondition`.
- **Spec constants live in `tables.go`** (EC block counts, alignment positions,
  `numRawDataModules`, format/version BCH). These were transcribed from the spec
  and cross-checked against independent implementations; treat changes as
  high-risk and back them with the golden tests.

### Verification model (keep it intact)

`decode_test.go` contains a **self-contained decoder** (test-only, not shipped)
that re-derives function patterns, format info, masks, de-interleaving, and
Reed-Solomon syndromes from the spec independently of the encoder. The round-trip
(`TestDecodeRoundtrip`, `FuzzEncode`) is the primary proof that symbols are
correct. `golden_test.go` locks output matrices with SHA-256 hashes; there is no
auto-generator — if output legitimately changes, update the hashes deliberately.

### Dependency isolation

`internal/sjisgen/` is a **separate nested module** (its own `go.mod`) that
depends on `golang.org/x/text` to generate `sjis_table.go`. The parent build and
`go mod tidy` ignore it, which is what keeps the library dependency-free — do not
fold it into the root module.

## Conventions

- Renderers (`png.go`, `svg.go`, `ascii.go`) share `render.go`'s `renderConfig`
  and `RenderOption`s; add new render options there.
- New user-visible changes get a `CHANGELOG.md` entry under `[Unreleased]`.
- Fuzz the encoder for any change to segmentation, placement, or masking.
