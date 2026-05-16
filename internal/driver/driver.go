// Package driver wires the compiler stages together: source file -> tokens ->
// AST -> Go source -> native binary. It also owns temp-dir bookkeeping so the
// CLI doesn't have to.
package driver

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mavboas/lume/internal/ast"
	"github.com/mavboas/lume/internal/codegen"
	"github.com/mavboas/lume/internal/lexer"
	"github.com/mavboas/lume/internal/parser"
	"github.com/mavboas/lume/internal/sema"
)

// CompileResult bundles the artifacts of a compilation step.
type CompileResult struct {
	Source     string // the original .lm source
	Tokens     []lexer.Token
	AST        *ast.File
	GoSource   string // generated Go program
	BinaryPath string // populated after Build
}

// CompileFile runs lex + parse + codegen on srcPath and returns the result.
// It does NOT invoke `go build`; call Build for that.
func CompileFile(srcPath string) (*CompileResult, error) {
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", srcPath, err)
	}
	src := string(srcBytes)

	tokens, lexErrs := lexer.New(src).Tokenize()
	if len(lexErrs) > 0 {
		return nil, errors.Join(lexErrs...)
	}

	tree, parseErrs := parser.New(tokens).Parse()
	if len(parseErrs) > 0 {
		return nil, errors.Join(parseErrs...)
	}

	if err := sema.Check(tree); err != nil {
		return &CompileResult{Source: src, Tokens: tokens, AST: tree}, err
	}

	gen := codegen.New()
	goSrc, err := gen.File(tree)
	if err != nil {
		// Show the raw Go output to help debug a codegen bug.
		return &CompileResult{Source: src, Tokens: tokens, AST: tree, GoSource: goSrc},
			fmt.Errorf("codegen: %w", err)
	}

	return &CompileResult{
		Source:   src,
		Tokens:   tokens,
		AST:      tree,
		GoSource: goSrc,
	}, nil
}

// Build performs CompileFile + invokes `go build` on the generated Go source.
// outBinary is the path the final native executable should be written to.
// If outBinary is empty, it derives one from srcPath (drop .lm, add .exe on win).
func Build(srcPath, outBinary string) (*CompileResult, error) {
	res, err := CompileFile(srcPath)
	if err != nil {
		return res, err
	}

	tmpDir, err := os.MkdirTemp("", "lume-build-*")
	if err != nil {
		return res, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(res.GoSource), 0o644); err != nil {
		return res, fmt.Errorf("write generated go: %w", err)
	}

	if outBinary == "" {
		outBinary = defaultOutputName(srcPath)
	}
	absOut, err := filepath.Abs(outBinary)
	if err != nil {
		return res, err
	}

	cmd := exec.Command("go", "build", "-o", absOut, goFile)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return res, fmt.Errorf("go build: %w", err)
	}

	res.BinaryPath = absOut
	return res, nil
}

// Run builds and immediately executes the program. Output flows to the
// caller's stdout/stderr. Returns the exit code from the program.
func Run(srcPath string, args []string) (int, *CompileResult, error) {
	tmpPattern := "lume-run-*"
	if runtime.GOOS == "windows" {
		tmpPattern = "lume-run-*.exe"
	}
	tmpBin, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return 1, nil, err
	}
	tmpBin.Close()
	binPath := tmpBin.Name()
	defer os.Remove(binPath)

	res, err := Build(srcPath, binPath)
	if err != nil {
		return 1, res, err
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), res, nil
		}
		return 1, res, err
	}
	return 0, res, nil
}

func defaultOutputName(srcPath string) string {
	base := filepath.Base(srcPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return name
}
