# Lume

<div align="center">
  <img src="../images/logo.svg" alt="Lume Language Logo" width="200" />
</div>

Lume is an AI-first backend language. It is immutable by default, designed for concise LLM-generated code, and currently implemented as an experimental compiler that transpiles `.lm` files to Go before invoking `go build`.

> Status: `v0.1.0-experimental`. Lume is not production-ready. The language reference includes target-language ideas that are not implemented in the current compiler yet.

## What Works Today

- Native CLI: `build`, `run`, `tokens`, `ast`, `gen`, and experimental `kb`.
- Go transpiler backend.
- Literals and types: `int`, `dec`, `str`, `bool`, `none`, `list[T]`, `obj`, and class instances.
- Functions, typed parameters, optional return annotations, and final-expression returns.
- Bindings, rebinding, local `let` expressions, and initial semantic checks.
- Arithmetic, comparison, logical operators, and string interpolation.
- `if/elsif/else`, `switch/case/default`, list/range `for` loops.
- Classes with named constructors, field access, `.with()`, and `.keys`.
- Basic `match` with literal, wildcard, and capture patterns.
- VS Code syntax highlighting and snippets.

## Install From Source

Requirements:

- Go matching the version declared in `go.mod`.

```powershell
git clone https://github.com/mavboas/lume.git
cd lume
go test ./...
go build -o lume ./cmd/lume
.\lume version
```

On Unix-like systems:

```bash
go test ./...
go build -o lume ./cmd/lume
./lume version
```

## Try It

```powershell
go run ./cmd/lume run examples/hello.lm
go run ./cmd/lume gen examples/math.lm
go run ./cmd/lume ast examples/match.lm
```

Compile an example to a native binary:

```powershell
go run ./cmd/lume build examples/hello.lm -o hello.exe
.\hello.exe
```

## AI Context KB

Lume includes an experimental local knowledge-base layer for reducing prompt context:

```powershell
go run ./cmd/lume kb build
go run ./cmd/lume kb pack "implement pipe" --ai --max-tokens 1200
go run ./cmd/lume kb lint
go run ./cmd/lume kb stats
```

Generated KB output is intentionally ignored by Git.

## Repository Layout

```text
cmd/lume/       CLI entry point
internal/       compiler, semantic checker, codegen, and KB internals
examples/       small Lume programs that compile today
docs/           language reference, compiler notes, roadmap, and design notes
vscode/         VS Code syntax and snippets source
images/         logo assets
```

## Documentation

- [Language reference](docs/language.md)
- [Compiler architecture](docs/compiler.md)
- [Design notes](docs/design.md)
- [Roadmap](docs/roadmap.md)
- [Changelog](CHANGELOG.md)

## License

MIT.
