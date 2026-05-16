# Compiler Architecture

The current Lume compiler is a small Go module that transpiles Lume source to Go and then invokes `go build` for native binaries.

## Pipeline

```text
.lm source
  -> lexer
  -> parser
  -> AST
  -> semantic checker
  -> Go code generator
  -> go build
```

## Main Components

- `cmd/lume`: command-line interface and subcommand dispatch.
- `internal/lexer`: tokenizes source, including comments and interpolated strings.
- `internal/parser`: recursive descent plus Pratt parsing for expressions.
- `internal/ast`: AST node definitions.
- `internal/sema`: current semantic and type checks.
- `internal/codegen`: emits Go source.
- `internal/driver`: wires the pipeline and invokes `go build`.
- `internal/kb`: experimental local knowledge-base commands for AI context packing.

## Current Boundaries

The compiler implements a practical v0 subset. It does not yet implement full Hindley-Milner inference, ADTs, modules, imports, lambdas, `Result`/`Option`, the pipe operator, refinements, effects, or a standard library.

Use `lume gen <file.lm>` to inspect generated Go when debugging compiler behavior.
