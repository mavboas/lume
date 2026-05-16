# Contributing

Lume is experimental. Contributions should keep the implemented compiler subset clear and tested.

## Development

```powershell
go test ./...
go build -o lume ./cmd/lume
go run ./cmd/lume run examples/hello.lm
```

## Guidelines

- Keep public repository text in English.
- Add or update examples when changing language behavior.
- Add focused Go tests for parser, semantic checker, driver, or KB behavior.
- Do not commit generated binaries, cache directories, `.vsix` packages, or generated KB output.
- Keep target-language ideas clearly marked as not implemented until the compiler supports them.
