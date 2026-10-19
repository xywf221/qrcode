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
go test -run '^$' -fuzz FuzzEncode -fuzztime 30s .   # fuzz (byte encode + decode round-trip)
go test -run '^$' -fuzz FuzzPenalty -fuzztime 15s .  # fuzz (penalty vs naive)
# fuzz_test.go also has FuzzExplicitSegments, FuzzECI, FuzzKanjiString,
# FuzzStructured and FuzzMask; each does an encode/decode round-trip.
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
   `maskMatrix` holds the eight data-mask conditions (one branch each, hoisted
   out of the x-loop for speed) and is their single definition; `applyMask`
   applies one in place once the winner is known. `selectMask` scores all eight
   through `penaltyMask`, which builds a masked *copy* into a reused scratch
   buffer so the symbol matrix is never mutated while candidates are compared.

### Invariants that span files

- **Structured Append size accounting.** The 20-bit header (`saHeaderBits`,
  `structured.go`) is *not* a `segment`. Every place that compares required bits
  against `dataBits` must add `cfg.saBits()` — see `layout` and `build`
  (`qrcode.go`). Miss one and a symbol overflows undetected.
- **The penalty scorer must match the reference.** `penalty()` / `penaltyOf()`
  (`mask.go`) are a single fused pass over a module matrix; `penalty_test.go`
  asserts they equal `naivePenalty` on random matrices, and the masked-copy
  variant `penaltyMask` (used by `selectMask`) must agree with applying the mask
  first. The rule-3 11-bit window is masked with `& 0x7ff` — an un-masked window
  silently undercounts.
- **Mask 4 is `(y/2 + x/3) % 2`.** All eight mask conditions are defined in
  `maskMatrix` (`matrix.go`) and mirrored in the decoder's `maskCondition`.
- **Spec constants live in `tables.go`** (EC block counts, alignment positions,
  `numRawDataModules`, format/version BCH). These were transcribed from the spec
  and cross-checked against independent implementations; treat changes as
  high-risk and back them with the golden tests. `tables_oracle_test.go` pins
  them to hard-coded spec tables, so a transcription slip is caught independently
  of the encoder.
- **Kanji codes must be reversible.** `validKanjiCode` (`segment.go`) requires
  both the JIS X 0208 range *and* a low byte of at least `0x40`: the 13-bit
  encoding is `high*0xC0 + low` of the value after subtracting the range base,
  which only inverts when that low byte is below `0xC0`. A code like `0x9330`
  passes the range test but decodes back as `0x9370`, so both `NewKanji` and the
  Shift-JIS converter (`sjis.go`) reject it rather than corrupting the data.

### Verification model (keep it intact)

`decode_test.go` contains a **self-contained decoder** (test-only, not shipped)
that re-derives function patterns, format info, masks, de-interleaving, and
Reed-Solomon syndromes from the spec independently of the encoder. The round-trip
(`TestDecodeRoundtrip`, `FuzzEncode`) is the primary proof that symbols are
correct. `golden_test.go` locks output matrices with SHA-256 hashes; there is no
auto-generator — if output legitimately changes, update the hashes deliberately.
`TestGoldenExtended` additionally locks high versions (>9), Kanji, ECI, Structured
Append and a forced mask, and decodes each case round-trip so the hash is backed
by independent verification. `tables_oracle_test.go` pins the spec tables to
hard-coded values.

### Dependency isolation

`internal/sjisgen/` is a **separate nested module** (its own `go.mod`) that
depends on `golang.org/x/text` to generate `sjis_table.go`. The parent build and
`go mod tidy` ignore it, which is what keeps the library dependency-free — do not
fold it into the root module.

## Conventions

- Renderers (`png.go`, `svg.go`, `ascii.go`) share `render.go`'s `renderConfig`
  and `RenderOption`s; add new render options there.
- `png.go` hand-rolls the indexed-PNG document and deflates through a
  `sync.Pool`ed `zlib.Writer` instead of `image/png`, whose per-call flate
  compressor (~1.3 MB) dominates rendering cost. `TestPNGMatchesStdlib` pins our
  output pixel-for-pixel to the standard-library encoder; keep it green.
- New user-visible changes get a `CHANGELOG.md` entry under `[Unreleased]`.
- Fuzz the encoder for any change to segmentation, placement, or masking.
