// Package kb builds and queries a compact local knowledge base for Lume.
package kb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/mavboas/lume/internal/driver"
	"github.com/mavboas/lume/internal/sema"
)

const (
	kbDirName   = "kb"
	metaDirName = ".lume"
)

var conceptFeatures = []string{
	"let", "match", "class", "switch", "for", "if", "functions", "types",
	"lists", "objects", "scopes", "operators", "calls", "codegen", "parser",
	"sema", "driver", "cli", "pipe",
}

type Options struct {
	Root      string
	Source    string
	Query     string
	MaxTokens int
	AI        bool
}

type BuildStats struct {
	Root          string
	Pages         int
	Examples      int
	Errors        int
	EstimatedIn   int
	EstimatedPack int
}

type Page struct {
	Path     string
	Title    string
	Feature  string
	Body     string
	Tokens   int
	Examples []string
	Sources  []string
}

type Pack struct {
	Query      string
	Budget     int
	Concepts   []string
	Examples   []string
	Errors     []string
	SourceRefs []string
	Body       string
	Tokens     int
}

func Init(root string) error {
	root, err := projectRoot(root)
	if err != nil {
		return err
	}
	dirs := []string{
		filepath.Join(root, metaDirName, kbDirName),
		filepath.Join(root, kbDirName),
		filepath.Join(root, kbDirName, "language"),
		filepath.Join(root, kbDirName, "compiler"),
		filepath.Join(root, kbDirName, "examples"),
		filepath.Join(root, kbDirName, "errors"),
		filepath.Join(root, kbDirName, "decisions"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	agents := filepath.Join(root, kbDirName, "AGENTS.md")
	if _, err := os.Stat(agents); errors.Is(err, os.ErrNotExist) {
		body := "# Lume KB Agents\n\nUse `kb/index.md` first. Cite only existing wikilinks and ask for more context by source ref when needed.\n"
		if err := os.WriteFile(agents, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func Build(opts Options) (BuildStats, error) {
	root, err := projectRoot(opts.Root)
	if err != nil {
		return BuildStats{}, err
	}
	if err := Init(root); err != nil {
		return BuildStats{}, err
	}
	pages, examples, err := collectPages(root)
	if err != nil {
		return BuildStats{}, err
	}
	for _, page := range pages {
		if err := writePage(root, page); err != nil {
			return BuildStats{}, err
		}
	}
	if err := writeExamples(root, examples); err != nil {
		return BuildStats{}, err
	}
	if err := writeErrors(root); err != nil {
		return BuildStats{}, err
	}
	if err := writeCompilerMetadata(root); err != nil {
		return BuildStats{}, err
	}
	index := renderIndex(pages, examples)
	if err := os.WriteFile(filepath.Join(root, kbDirName, "index.md"), []byte(index), 0o644); err != nil {
		return BuildStats{}, err
	}
	stats := BuildStats{
		Root:          root,
		Pages:         len(pages) + len(examples) + len(sema.DiagnosticCatalog()) + 1,
		Examples:      len(examples),
		Errors:        len(sema.DiagnosticCatalog()),
		EstimatedIn:   estimateProjectTokens(root),
		EstimatedPack: tokenEstimate(index),
	}
	return stats, nil
}

func PackContext(opts Options) (Pack, error) {
	root, err := projectRoot(opts.Root)
	if err != nil {
		return Pack{}, err
	}
	if _, err := os.Stat(filepath.Join(root, kbDirName, "index.md")); err != nil {
		if _, buildErr := Build(Options{Root: root}); buildErr != nil {
			return Pack{}, buildErr
		}
	}
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 1800
	}
	pages, err := readPages(root)
	if err != nil {
		return Pack{}, err
	}
	indexPath := filepath.Join(root, kbDirName, "index.md")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		return Pack{}, err
	}
	queryTerms := terms(opts.Query)
	indexBody := compactIndex(string(indexBytes), queryTerms)
	indexPage := Page{Path: "index.md", Title: "KB Index", Body: indexBody, Tokens: tokenEstimate(indexBody)}
	sort.SliceStable(pages, func(i, j int) bool {
		return scorePage(pages[i], queryTerms) > scorePage(pages[j], queryTerms)
	})
	pack := Pack{Query: opts.Query, Budget: opts.MaxTokens}
	var chunks []string
	addChunk := func(label, body string) bool {
		body = cleanMarkdown(body)
		next := tokenEstimate(strings.Join(append(chunks, "## "+label+"\n\n"+body), "\n\n"))
		if next > opts.MaxTokens && len(chunks) > 0 {
			return false
		}
		chunks = append(chunks, "## "+label+"\n\n"+body)
		pack.Tokens = next
		return true
	}
	addChunk("index", indexPage.Body)
	for _, page := range pages {
		if scorePage(page, queryTerms) == 0 && len(pack.Concepts) >= 5 {
			continue
		}
		if !addChunk(page.Path, page.Body) {
			break
		}
		if strings.HasPrefix(page.Path, "language/") || strings.HasPrefix(page.Path, "compiler/") {
			pack.Concepts = appendUnique(pack.Concepts, "[["+strings.TrimSuffix(page.Path, ".md")+"]]")
		}
		if strings.HasPrefix(page.Path, "examples/") {
			pack.Examples = appendUnique(pack.Examples, strings.TrimPrefix(strings.TrimSuffix(page.Path, ".md"), "examples/"))
		}
		if strings.HasPrefix(page.Path, "errors/") {
			pack.Errors = appendUnique(pack.Errors, strings.TrimSuffix(strings.TrimPrefix(page.Path, "errors/"), ".md"))
		}
		pack.SourceRefs = appendUniqueMany(pack.SourceRefs, page.Sources)
	}
	pack.Body = renderPack(pack, chunks, opts.AI)
	return pack, nil
}

func Lint(root string) []error {
	root, err := projectRoot(root)
	if err != nil {
		return []error{err}
	}
	var errs []error
	if _, err := os.Stat(filepath.Join(root, kbDirName, "index.md")); err != nil {
		errs = append(errs, fmt.Errorf("kb/index.md missing; run lume kb build"))
		return errs
	}
	pages, err := readPages(root)
	if err != nil {
		return []error{err}
	}
	existing := map[string]bool{"index": true, "index.md": true}
	for _, p := range pages {
		key := strings.TrimSuffix(p.Path, ".md")
		existing[key] = true
	}
	linkRE := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	for _, p := range pages {
		for _, m := range linkRE.FindAllStringSubmatch(p.Body, -1) {
			if !existing[m[1]] {
				errs = append(errs, fmt.Errorf("%s links missing page [[%s]]", p.Path, m[1]))
			}
		}
	}
	examples, _ := filepath.Glob(filepath.Join(root, "examples", "*.lm"))
	for _, ex := range examples {
		page := "examples/" + filepath.Base(ex)
		if !existing[page] {
			errs = append(errs, fmt.Errorf("example %s is not indexed", filepath.ToSlash(filepath.Join("examples", filepath.Base(ex)))))
		}
	}
	for _, feature := range []string{"let", "match", "class", "switch", "for"} {
		found := false
		for _, ex := range examples {
			b, _ := os.ReadFile(ex)
			if strings.Contains(strings.ToLower(string(b)), feature) || strings.Contains(strings.ToLower(filepath.Base(ex)), feature) {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("documented feature %s has no example", feature))
		}
	}
	return errs
}

func Stats(root string) (BuildStats, error) {
	root, err := projectRoot(root)
	if err != nil {
		return BuildStats{}, err
	}
	pages, err := readPages(root)
	if err != nil {
		return BuildStats{}, err
	}
	examples := 0
	errorsCount := 0
	for _, p := range pages {
		if strings.HasPrefix(p.Path, "examples/") {
			examples++
		}
		if strings.HasPrefix(p.Path, "errors/") {
			errorsCount++
		}
	}
	packTokens := tokenEstimate(readOptional(filepath.Join(root, kbDirName, "index.md")))
	return BuildStats{Root: root, Pages: len(pages), Examples: examples, Errors: errorsCount, EstimatedIn: estimateProjectTokens(root), EstimatedPack: packTokens}, nil
}

func collectPages(root string) ([]Page, []Page, error) {
	language := readOptional(filepath.Join(root, "docs", "language.md"))
	readme := readOptional(filepath.Join(root, "docs", "compiler.md"))
	examples, err := collectExamples(root)
	if err != nil {
		return nil, nil, err
	}
	var pages []Page
	for _, feature := range conceptFeatures {
		dir := "language"
		if isCompilerFeature(feature) {
			dir = "compiler"
		}
		body := renderConcept(feature, language, readme, examples)
		pages = append(pages, Page{
			Path:     filepath.ToSlash(filepath.Join(dir, feature+".md")),
			Title:    title(feature),
			Feature:  feature,
			Body:     body,
			Tokens:   tokenEstimate(body),
			Examples: relatedExamples(feature, examples),
			Sources:  sourcesForFeature(feature),
		})
	}
	return pages, examples, nil
}

func collectExamples(root string) ([]Page, error) {
	var pages []Page
	exampleDir := filepath.Join(root, "examples")
	files, err := filepath.Glob(filepath.Join(exampleDir, "*.lm"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rel := filepath.ToSlash(filepath.Join("examples", filepath.Base(path)))
		features := detectFeatures(filepath.Base(path), string(b))
		body := "# Example: " + filepath.Base(path) + "\n\n" +
			"source: `" + rel + "`\n\n" +
			"features: " + strings.Join(features, ", ") + "\n\n" +
			"```lume\n" + strings.TrimSpace(string(b)) + "\n```\n"
		pages = append(pages, Page{
			Path:     filepath.ToSlash(filepath.Join("examples", filepath.Base(path)+".md")),
			Title:    filepath.Base(path),
			Feature:  strings.Join(features, " "),
			Body:     body,
			Tokens:   tokenEstimate(body),
			Examples: []string{rel},
			Sources:  []string{rel},
		})
	}
	return pages, nil
}

func writePage(root string, page Page) error {
	out := filepath.Join(root, kbDirName, filepath.FromSlash(page.Path))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, []byte(page.Body), 0o644)
}

func writeExamples(root string, pages []Page) error {
	for _, p := range pages {
		if err := writePage(root, p); err != nil {
			return err
		}
	}
	return nil
}

func writeErrors(root string) error {
	for _, d := range sema.DiagnosticCatalog() {
		body := fmt.Sprintf("# %s\n\nfeature: [[language/%s]]\n\nmessage: %s\n\nfix_hint: %s\n\n```json\n%s\n```\n",
			d.Code, d.Feature, d.Message, d.FixHint, mustJSON(d))
		if err := writePage(root, Page{Path: filepath.ToSlash(filepath.Join("errors", d.Code+".md")), Body: body}); err != nil {
			return err
		}
	}
	return nil
}

func writeCompilerMetadata(root string) error {
	meta := map[string]any{
		"types":       []string{"int", "dec", "str", "bool", "none", "obj", "range", "list[T]", "class"},
		"statements":  []string{"let binding", "expression", "if", "switch", "for"},
		"expressions": []string{"literal", "identifier", "call", "field", "unary", "binary", "range", "let", "match", "list", "object"},
		"builtins":    []string{"print"},
		"errors":      sema.DiagnosticCatalog(),
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	out := filepath.Join(root, metaDirName, kbDirName, "compiler-metadata.json")
	return os.WriteFile(out, append(b, '\n'), 0o644)
}

func renderConcept(feature, language, readme string, examples []Page) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title(feature))
	fmt.Fprintf(&b, "concept: `%s`\n\n", feature)
	fmt.Fprintf(&b, "summary: %s\n\n", summaryFor(feature))
	refs := sourcesForFeature(feature)
	if len(refs) > 0 {
		b.WriteString("source_refs:\n")
		for _, ref := range refs {
			fmt.Fprintf(&b, "- %s\n", ref)
		}
		b.WriteString("\n")
	}
	related := relatedExamples(feature, examples)
	if len(related) > 0 {
		b.WriteString("examples:\n")
		for _, ex := range related {
			fmt.Fprintf(&b, "- %s\n", ex)
		}
		b.WriteString("\n")
	}
	if s := excerpt(language, feature); s != "" {
		b.WriteString("language_notes:\n\n")
		b.WriteString(s)
		b.WriteString("\n\n")
	}
	if s := excerpt(readme, feature); s != "" {
		b.WriteString("compiler_subset:\n\n")
		b.WriteString(s)
		b.WriteString("\n\n")
	}
	for _, d := range sema.DiagnosticCatalog() {
		if d.Feature == feature {
			fmt.Fprintf(&b, "- [[errors/%s]]: %s. fix: %s\n", d.Code, d.Message, d.FixHint)
		}
	}
	return b.String()
}

func renderIndex(pages, examples []Page) string {
	var b strings.Builder
	b.WriteString("# Lume KB Index\n\n")
	b.WriteString("Start here for compact context. Pages are small and source-backed.\n\n")
	b.WriteString("## Language\n\n")
	for _, p := range pages {
		if strings.HasPrefix(p.Path, "language/") {
			fmt.Fprintf(&b, "- [[%s]] - %s\n", strings.TrimSuffix(p.Path, ".md"), summaryFor(p.Feature))
		}
	}
	b.WriteString("\n## Compiler\n\n")
	for _, p := range pages {
		if strings.HasPrefix(p.Path, "compiler/") {
			fmt.Fprintf(&b, "- [[%s]] - %s\n", strings.TrimSuffix(p.Path, ".md"), summaryFor(p.Feature))
		}
	}
	b.WriteString("\n## Examples\n\n")
	for _, p := range examples {
		fmt.Fprintf(&b, "- [[%s]] - %s\n", strings.TrimSuffix(p.Path, ".md"), strings.Join(p.Examples, ", "))
	}
	b.WriteString("\n## Errors\n\n")
	for _, d := range sema.DiagnosticCatalog() {
		fmt.Fprintf(&b, "- [[errors/%s]] %s: %s\n", d.Code, d.Feature, d.FixHint)
	}
	return b.String()
}

func renderPack(pack Pack, chunks []string, ai bool) string {
	var b strings.Builder
	if ai {
		b.WriteString("# Lume AI Context Pack\n")
	} else {
		b.WriteString("# Lume Context Pack\n")
	}
	fmt.Fprintf(&b, "query: %s\n", pack.Query)
	fmt.Fprintf(&b, "budget: %d\n", pack.Budget)
	b.WriteString("concepts:\n")
	for _, c := range pack.Concepts {
		fmt.Fprintf(&b, "- %s\n", c)
	}
	b.WriteString("examples:\n")
	for _, e := range pack.Examples {
		fmt.Fprintf(&b, "- %s\n", e)
	}
	b.WriteString("errors:\n")
	for _, e := range pack.Errors {
		fmt.Fprintf(&b, "- %s\n", e)
	}
	b.WriteString("source_refs:\n")
	for _, r := range pack.SourceRefs {
		fmt.Fprintf(&b, "- %s\n", r)
	}
	b.WriteString("\n")
	b.WriteString(strings.Join(chunks, "\n\n"))
	return b.String()
}

func readPages(root string) ([]Page, error) {
	var pages []Page
	base := filepath.Join(root, kbDirName)
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" || filepath.Base(path) == "AGENTS.md" || filepath.Base(path) == "index.md" {
			return nil
		}
		rel, _ := filepath.Rel(base, path)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(b)
		pages = append(pages, Page{Path: filepath.ToSlash(rel), Title: filepath.Base(path), Feature: strings.TrimSuffix(filepath.Base(path), ".md"), Body: body, Tokens: tokenEstimate(body), Sources: extractSources(body)})
		return nil
	})
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })
	return pages, err
}

func projectRoot(start string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(abs, "go.mod")) && dirExists(filepath.Join(abs, "cmd", "lume")) && dirExists(filepath.Join(abs, "internal")) {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			break
		}
		abs = parent
	}
	return "", fmt.Errorf("could not find Lume project root from %s", start)
}

func detectFeatures(name, src string) []string {
	text := strings.ToLower(name + "\n" + src)
	var out []string
	for _, f := range conceptFeatures {
		if isCompilerFeature(f) {
			continue
		}
		if strings.Contains(text, f) || (f == "class" && strings.Contains(text, "cl ")) || (f == "functions" && strings.Contains(text, "fn ")) {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		out = append(out, "functions")
	}
	return out
}

func relatedExamples(feature string, examples []Page) []string {
	var out []string
	for _, ex := range examples {
		if strings.Contains(ex.Feature, feature) || strings.Contains(strings.ToLower(ex.Path), feature) {
			out = append(out, strings.TrimPrefix(strings.TrimSuffix(ex.Path, ".md"), "examples/"))
		}
	}
	return out
}

func sourcesForFeature(feature string) []string {
	switch feature {
	case "parser":
		return []string{"internal/parser/parser.go", "internal/ast/ast.go"}
	case "sema", "types", "scopes", "calls", "operators":
		return []string{"internal/sema/checker.go", "internal/sema/diagnostics.go"}
	case "codegen":
		return []string{"internal/codegen/golang.go"}
	case "driver", "cli":
		return []string{"internal/driver/driver.go", "cmd/lume/main.go"}
	default:
		return []string{"docs/language.md", "docs/compiler.md"}
	}
}

func summaryFor(feature string) string {
	switch feature {
	case "let":
		return "local binding as statement or expression; expression bindings are sequential and scoped to the block"
	case "match":
		return "literal, wildcard, and capture pattern matching with exhaustiveness checks for value contexts"
	case "class":
		return "class declarations, named constructors, field access, .with(), and .keys in the current subset"
	case "switch":
		return "literal case dispatch with optional default and branch type checks"
	case "for":
		return "iteration over lists and half-open int ranges"
	case "pipe":
		return "planned feature, not implemented in the current compiler subset"
	case "sema":
		return "semantic checker for names, types, calls, branches, lists, objects, classes, and match"
	case "parser":
		return "recursive descent plus Pratt parser that builds the AST"
	case "codegen":
		return "Go backend that emits package main and native builds through go build"
	case "cli":
		return "toolchain commands including build, run, debug views, and kb"
	default:
		return "current language/compiler concept with source references and examples when available"
	}
}

func excerpt(text, feature string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	var hits []string
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(feature)) {
			start := max(0, i-1)
			end := min(len(lines), i+3)
			hits = append(hits, strings.TrimSpace(strings.Join(lines[start:end], "\n")))
			if len(hits) >= 2 {
				break
			}
		}
	}
	return strings.Join(hits, "\n\n")
}

func cleanMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blank := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "<!--") || strings.HasPrefix(trim, "//") {
			continue
		}
		if trim == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func compactIndex(s string, query map[string]bool) string {
	lines := strings.Split(cleanMarkdown(s), "\n")
	var out []string
	section := ""
	keptInSection := 0
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "# ") {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trim, "## ") {
			section = trim
			keptInSection = 0
			continue
		}
		if trim == "" {
			continue
		}
		lower := strings.ToLower(trim)
		relevant := false
		for term := range query {
			if strings.Contains(lower, term) {
				relevant = true
				break
			}
		}
		if relevant || keptInSection < 2 {
			if section != "" {
				out = append(out, "", section)
				section = ""
			}
			out = append(out, line)
			keptInSection++
		}
	}
	return strings.Join(out, "\n")
}

func scorePage(p Page, query map[string]bool) int {
	score := 0
	text := strings.ToLower(p.Path + "\n" + p.Title + "\n" + p.Feature + "\n" + p.Body)
	for term := range query {
		if strings.Contains(strings.ToLower(p.Path), term) {
			score += 8
		}
		if strings.Contains(text, term) {
			score += 2
		}
	}
	for _, d := range sema.DiagnosticCatalog() {
		if query[strings.ToLower(d.Code)] && strings.Contains(p.Path, d.Code) {
			score += 20
		}
	}
	return score
}

func terms(q string) map[string]bool {
	out := map[string]bool{}
	var b strings.Builder
	flush := func() {
		if b.Len() >= 2 {
			out[strings.ToLower(b.String())] = true
		}
		b.Reset()
	}
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(unicode.ToLower(r))
		} else {
			flush()
		}
	}
	flush()
	return out
}

func extractSources(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if isSourceRef(line) || line == "docs/language.md" || line == "docs/compiler.md" {
			out = appendUnique(out, strings.Trim(line, "`"))
		}
		if strings.HasPrefix(line, "source: `") {
			out = appendUnique(out, strings.TrimSuffix(strings.TrimPrefix(line, "source: `"), "`"))
		}
	}
	return out
}

func isSourceRef(line string) bool {
	if !strings.Contains(line, "/") {
		return false
	}
	ext := filepath.Ext(line)
	return ext == ".go" || ext == ".md" || ext == ".lm"
}

func estimateProjectTokens(root string) int {
	total := 0
	for _, path := range []string{filepath.Join(root, "docs", "language.md"), filepath.Join(root, "docs", "compiler.md")} {
		total += tokenEstimate(readOptional(path))
	}
	filepath.WalkDir(filepath.Join(root, "examples"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".lm" {
			total += tokenEstimate(readOptional(path))
		}
		return nil
	})
	return total
}

func tokenEstimate(s string) int {
	words := len(strings.Fields(s))
	if words == 0 {
		return 0
	}
	return (words*4 + 2) / 3
}

func mustJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func readOptional(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func isCompilerFeature(feature string) bool {
	switch feature {
	case "parser", "sema", "codegen", "driver", "cli":
		return true
	default:
		return false
	}
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func appendUnique(in []string, s string) []string {
	if s == "" {
		return in
	}
	for _, x := range in {
		if x == s {
			return in
		}
	}
	return append(in, s)
}

func appendUniqueMany(in []string, values []string) []string {
	for _, v := range values {
		in = appendUnique(in, v)
	}
	return in
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ = driver.CompileFile
