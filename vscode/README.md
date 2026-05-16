# Lume for VS Code

Editor support for `.lm` files.

## Features

- Syntax highlighting for Lume keywords, builtins, strings, interpolation, numbers, comments, operators, and declarations.
- Snippets for common language forms such as `fn`, `cl`, `if`, `switch`, `match`, `for`, `let`, `main`, and tests.
- Bracket matching, auto-closing pairs, and comment toggling.
- File icons for `.lm` files.

## Not Included Yet

This extension does not include a language server. Completion, go-to-definition, inline compiler diagnostics, rename, and formatting are future work.

## Local Development

Open this repository in VS Code and press `F5` from the extension folder to launch an Extension Development Host.

To package manually:

```bash
npm install -g @vscode/vsce
cd vscode
vsce package
```

The generated `.vsix` file is intentionally ignored by Git.
