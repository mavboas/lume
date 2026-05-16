package kb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesStructure(t *testing.T) {
	root := fixtureProject(t)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".lume/kb", "kb/AGENTS.md", "kb/language", "kb/compiler", "kb/examples", "kb/errors", "kb/decisions"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestBuildIndexesExamplesAndErrors(t *testing.T) {
	root := fixtureProject(t)
	stats, err := Build(Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Examples == 0 || stats.Errors == 0 {
		t.Fatalf("expected examples and errors, got %+v", stats)
	}
	index := read(t, filepath.Join(root, "kb", "index.md"))
	if !strings.Contains(index, "[[language/let]]") {
		t.Fatalf("expected let concept in index:\n%s", index)
	}
	if !strings.Contains(index, "[[examples/let.lm]]") {
		t.Fatalf("expected let example in index:\n%s", index)
	}
	if !strings.Contains(read(t, filepath.Join(root, "kb", "errors", "E2805.md")), "fix_hint") {
		t.Fatal("expected AI-readable error metadata")
	}
}

func TestPackIncludesRelevantLetContext(t *testing.T) {
	root := fixtureProject(t)
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatal(err)
	}
	pack, err := PackContext(Options{Root: root, Query: "como usar let", MaxTokens: 900, AI: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pack.Body, "# Lume AI Context Pack") {
		t.Fatal("expected AI pack header")
	}
	if !strings.Contains(pack.Body, "[[language/let]]") {
		t.Fatalf("expected let concept:\n%s", pack.Body)
	}
	if !strings.Contains(pack.Body, "let.lm") {
		t.Fatalf("expected let example:\n%s", pack.Body)
	}
	if pack.Tokens > 900 {
		t.Fatalf("expected token budget respected, got %d", pack.Tokens)
	}
}

func TestPackIncludesErrorContext(t *testing.T) {
	root := fixtureProject(t)
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatal(err)
	}
	pack, err := PackContext(Options{Root: root, Query: "erro match E2805", MaxTokens: 1200, AI: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pack.Body, "errors/E2805") && !strings.Contains(pack.Body, "# E2805") {
		t.Fatalf("expected E2805 context:\n%s", pack.Body)
	}
}

func TestLintRejectsBrokenWikilink(t *testing.T) {
	root := fixtureProject(t)
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "kb", "language", "let.md")
	body := read(t, path) + "\n[[missing/page]]\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	errs := Lint(root)
	if len(errs) == 0 {
		t.Fatal("expected lint error")
	}
}

func fixtureProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module example.test/lume\n")
	write(t, filepath.Join(root, "cmd", "lume", ".keep"), "")
	write(t, filepath.Join(root, "internal", ".keep"), "")
	write(t, filepath.Join(root, "docs", "language.md"), "# Language\n\nlet creates local bindings.\nmatch checks patterns.\nclass values use cl.\nswitch supports literal cases.\nfor iterates lists.\n")
	write(t, filepath.Join(root, "docs", "compiler.md"), "# Compiler\n\nlet, match, class, switch and for are implemented in the current subset.\n")
	write(t, filepath.Join(root, "examples", "let.lm"), "fn main(){\n    print(let(x = 1){ x })\n}\n")
	write(t, filepath.Join(root, "examples", "match.lm"), "fn main(){\n    match(true){ case(true){ print(\"yes\") } case(false){ print(\"no\") } }\n}\n")
	write(t, filepath.Join(root, "examples", "classes.lm"), "cl User { name: str }\nfn main(){ print(User(name= \"a\").name) }\n")
	write(t, filepath.Join(root, "examples", "switch.lm"), "fn main(){ switch(1){ case(1){ print(\"one\") } } }\n")
	write(t, filepath.Join(root, "examples", "for.lm"), "fn main(){ for(i in 0..2){ print(i) } }\n")
	return root
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
