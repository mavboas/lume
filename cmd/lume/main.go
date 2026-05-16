// Command lume is the Lume language toolchain: build, run, test, and a few
// debug subcommands (tokens, ast) for inspecting the compiler pipeline.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mavboas/lume/internal/driver"
	"github.com/mavboas/lume/internal/kb"
	"github.com/mavboas/lume/internal/lexer"
)

const version = "0.0.1"

const usage = `lume - Lume language toolchain

usage:
  lume build <file.lm> [-o output]       compile to a native binary
  lume run   <file.lm> [args...]         compile and run
  lume test  [dir]                       run tests (placeholder)
  lume tokens <file.lm>                  print the token stream (debug)
  lume ast    <file.lm>                  print the AST as JSON (debug)
  lume gen    <file.lm>                  print generated Go code (debug)
  lume kb init                           create the local KB structure
  lume kb build [--source docs|compiler|all]
  lume kb pack "<question>" [--ai] [--max-tokens N]
  lume kb lint                           validate links and basic coverage
  lume kb stats                          show token and index statistics
  lume version                           show version
  lume help                              show this message
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	rest := os.Args[2:]

	switch cmd {
	case "build":
		os.Exit(cmdBuild(rest))
	case "run":
		os.Exit(cmdRun(rest))
	case "test":
		os.Exit(cmdTest(rest))
	case "tokens":
		os.Exit(cmdTokens(rest))
	case "ast":
		os.Exit(cmdAST(rest))
	case "gen":
		os.Exit(cmdGen(rest))
	case "kb":
		os.Exit(cmdKB(rest))
	case "version", "--version", "-v":
		fmt.Println("lume", version)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "lume: unknown subcommand %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
}

func cmdKB(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: lume kb <init|build|pack|lint|stats>")
		return 2
	}
	switch args[0] {
	case "init":
		if err := kb.Init(""); err != nil {
			fmt.Fprintln(os.Stderr, "lume kb init failed:", err)
			return 1
		}
		fmt.Println("lume kb: structure created in .lume/kb and kb/")
		return 0
	case "build":
		source, ok := parseKBSource(args[1:])
		if !ok {
			fmt.Fprintln(os.Stderr, "usage: lume kb build [--source docs|compiler|all]")
			return 2
		}
		stats, err := kb.Build(kb.Options{Source: source})
		if err != nil {
			fmt.Fprintln(os.Stderr, "lume kb build failed:", err)
			return 1
		}
		fmt.Printf("lume kb: %d pages, %d examples, %d errors; raw context ~%d tokens\n", stats.Pages, stats.Examples, stats.Errors, stats.EstimatedIn)
		return 0
	case "pack":
		opts, ok := parseKBPackArgs(args[1:])
		if !ok {
			fmt.Fprintln(os.Stderr, `usage: lume kb pack "<question or task>" [--ai] [--max-tokens N]`)
			return 2
		}
		pack, err := kb.PackContext(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "lume kb pack failed:", err)
			return 1
		}
		fmt.Println(pack.Body)
		return 0
	case "lint":
		errs := kb.Lint("")
		if len(errs) > 0 {
			for _, err := range errs {
				fmt.Fprintln(os.Stderr, "kb lint:", err)
			}
			return 1
		}
		fmt.Println("lume kb lint: ok")
		return 0
	case "stats":
		stats, err := kb.Stats("")
		if err != nil {
			fmt.Fprintln(os.Stderr, "lume kb stats failed:", err)
			return 1
		}
		saved := stats.EstimatedIn - stats.EstimatedPack
		if saved < 0 {
			saved = 0
		}
		fmt.Printf("pages: %d\nexamples: %d\nerrors: %d\nestimated raw tokens: %d\nestimated KB tokens: %d\nestimated savings: %d\n",
			stats.Pages, stats.Examples, stats.Errors, stats.EstimatedIn, stats.EstimatedPack, saved)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "lume kb: unknown subcommand %q\n", args[0])
		return 2
	}
}

func cmdBuild(args []string) int {
	src, out, ok := parseBuildArgs(args)
	if !ok {
		fmt.Fprintln(os.Stderr, "usage: lume build <file.lm> [-o output]")
		return 2
	}
	res, err := driver.Build(src, out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lume build failed:", err)
		return 1
	}
	fmt.Printf("lume: %s -> %s\n", src, res.BinaryPath)
	return 0
}

func cmdRun(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: lume run <file.lm> [args...]")
		return 2
	}
	src := args[0]
	progArgs := args[1:]
	code, _, err := driver.Run(src, progArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lume run failed:", err)
		return 1
	}
	return code
}

func cmdTest(args []string) int {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}
	fmt.Printf("lume test: not implemented yet (target: %s)\n", target)
	return 0
}

func cmdTokens(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: lume tokens <file.lm>")
		return 2
	}
	srcBytes, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	toks, errs := lexer.New(string(srcBytes)).Tokenize()
	for _, t := range toks {
		fmt.Println(t)
	}
	for _, e := range errs {
		fmt.Fprintln(os.Stderr, e)
	}
	if len(errs) > 0 {
		return 1
	}
	return 0
}

func cmdAST(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: lume ast <file.lm>")
		return 2
	}
	res, err := driver.CompileFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res.AST); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cmdGen(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: lume gen <file.lm>")
		return 2
	}
	res, err := driver.CompileFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if res != nil {
			fmt.Println(res.GoSource)
		}
		return 1
	}
	fmt.Print(res.GoSource)
	return 0
}

// parseBuildArgs supports `lume build foo.lm` and `lume build foo.lm -o bar`.
func parseBuildArgs(args []string) (src, out string, ok bool) {
	i := 0
	for i < len(args) {
		a := args[i]
		switch a {
		case "-o", "--output":
			if i+1 >= len(args) {
				return "", "", false
			}
			out = args[i+1]
			i += 2
		default:
			if src != "" {
				return "", "", false
			}
			src = a
			i++
		}
	}
	if src == "" {
		return "", "", false
	}
	return src, out, true
}

func parseKBSource(args []string) (string, bool) {
	source := "all"
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				return "", false
			}
			source = args[i+1]
			if source != "docs" && source != "compiler" && source != "all" {
				return "", false
			}
			i += 2
		default:
			return "", false
		}
	}
	return source, true
}

func parseKBPackArgs(args []string) (kb.Options, bool) {
	opts := kb.Options{MaxTokens: 1800}
	var queryParts []string
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--ai":
			opts.AI = true
			i++
		case "--max-tokens":
			if i+1 >= len(args) {
				return opts, false
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				return opts, false
			}
			opts.MaxTokens = n
			i += 2
		default:
			queryParts = append(queryParts, args[i])
			i++
		}
	}
	opts.Query = strings.TrimSpace(strings.Join(queryParts, " "))
	return opts, opts.Query != ""
}
