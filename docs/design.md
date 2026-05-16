# Design Notes

Lume explores a language designed with AI agents as first-class users.

## Principles

- Tokens are a real cost. Syntax should be concise without becoming cryptic.
- Immutability is the default. Rebinding creates a new value; field mutation is rejected.
- Errors should become values, not hidden control flow.
- The language should prefer one canonical way to express common backend tasks.
- The compiler should provide structured diagnostics that an AI agent can repair against.

## Current Strategy

The first implementation is intentionally pragmatic: Lume transpiles to Go. This keeps the runtime, scheduler, garbage collector, and native binary story simple while the language surface is still changing.

The long-term design can revisit a native backend after the syntax and semantics have proved useful in real programs.

## AI-Focused Tooling

The experimental `lume kb` commands compile local project knowledge into small, source-backed Markdown pages. The goal is to avoid sending the same language reference, examples, diagnostics, and compiler notes to an LLM on every interaction.
