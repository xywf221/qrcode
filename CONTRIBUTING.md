# Contributing

Thanks for your interest in improving this library.

## Development

Requirements: Go 1.24 or newer.

```sh
go build ./...
go vet ./...
go test ./...
go test -race ./...
go test -run '^$' -bench . -benchmem
```

Linting uses [golangci-lint](https://golangci-lint.run):

```sh
golangci-lint run
```

## Guidelines

- Keep the library dependency-free. New runtime dependencies are not accepted;
  test-only or code-generation-only dependencies may be considered.
- Every behaviour change must come with a test. Bug fixes should include a test
  that reproduces the original failure.
- Run `gofmt` (or `go fmt ./...`) before committing. CI fails on unformatted
  code.
- Prefer correctness and clarity over micro-optimisation. If you optimise, keep
  it measured and covered by tests.
- Update `CHANGELOG.md` under `[Unreleased]` for any user-visible change.

## Commit messages

Use the imperative mood and a short subject line. Reference issues where
relevant.

## Regenerating the Shift-JIS table

`sjis_table.go` is generated. To regenerate it (requires network access to fetch
`golang.org/x/text`):

```sh
cd internal/sjisgen
go run . > ../../sjis_table.go
gofmt -w ../../sjis_table.go
```

`internal/sjisgen` is a nested module with its own `go.mod`, so its
`golang.org/x/text` dependency never reaches the library build.
