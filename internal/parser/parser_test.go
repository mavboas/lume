package parser

import (
	"strings"
	"testing"

	"github.com/mavboas/lume/internal/ast"
	"github.com/mavboas/lume/internal/lexer"
)

func parseExprStmt(t *testing.T, src string) ast.Expr {
	t.Helper()
	toks, lexErrs := lexer.New("fn main(){\n" + src + "\n}").Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	fn := tree.Decls[0].(*ast.FnDecl)
	stmt := fn.Body.Stmts[0].(*ast.ExprStmt)
	return stmt.Expr
}

func TestParseDecimalLiteral(t *testing.T) {
	expr := parseExprStmt(t, "1.5")
	lit, ok := expr.(*ast.DecLit)
	if !ok {
		t.Fatalf("expected DecLit, got %T", expr)
	}
	if lit.Value != 1.5 {
		t.Fatalf("expected 1.5, got %v", lit.Value)
	}
}

func TestParseNoneLiteral(t *testing.T) {
	expr := parseExprStmt(t, "none")
	if _, ok := expr.(*ast.NoneLit); !ok {
		t.Fatalf("expected NoneLit, got %T", expr)
	}
}

func TestParseAnnotatedListBinding(t *testing.T) {
	toks, lexErrs := lexer.New("fn main(){\nxs: list[int] = [1, 2]\n}").Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	fn := tree.Decls[0].(*ast.FnDecl)
	stmt := fn.Body.Stmts[0].(*ast.LetStmt)
	if stmt.Type == nil || stmt.Type.Name != "list" || len(stmt.Type.Args) != 1 || stmt.Type.Args[0].Name != "int" {
		t.Fatalf("expected list[int] annotation, got %#v", stmt.Type)
	}
	list, ok := stmt.Value.(*ast.ListLit)
	if !ok {
		t.Fatalf("expected ListLit, got %T", stmt.Value)
	}
	if len(list.Elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(list.Elems))
	}
}

func parseDecl(t *testing.T, src string) []ast.Decl {
	t.Helper()
	toks, lexErrs := lexer.New(src).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	return tree.Decls
}

func TestParseClassDecl(t *testing.T) {
	decls := parseDecl(t, "cl Account\n{\n    id: str\n    balance: int\n}")
	if len(decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(decls))
	}
	cl, ok := decls[0].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("expected ClassDecl, got %T", decls[0])
	}
	if cl.Name != "Account" {
		t.Fatalf("expected name Account, got %q", cl.Name)
	}
	if len(cl.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(cl.Fields))
	}
	if cl.Fields[0].Name != "id" || cl.Fields[0].Type.Name != "str" {
		t.Fatalf("unexpected field 0: %+v", cl.Fields[0])
	}
	if cl.Fields[1].Name != "balance" || cl.Fields[1].Type.Name != "int" {
		t.Fatalf("unexpected field 1: %+v", cl.Fields[1])
	}
}

func TestParsePubClassDecl(t *testing.T) {
	decls := parseDecl(t, "pub cl User\n{\n    name: str\n}")
	cl, ok := decls[0].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("expected ClassDecl, got %T", decls[0])
	}
	if !cl.Public {
		t.Fatalf("expected Public=true")
	}
}

func TestParseConstructorCall(t *testing.T) {
	expr := parseExprStmt(t, `Account(id= "1", balance= 100)`)
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", expr)
	}
	if len(call.Args) != 0 {
		t.Fatalf("expected 0 positional args, got %d", len(call.Args))
	}
	if len(call.NamedArgs) != 2 {
		t.Fatalf("expected 2 named args, got %d", len(call.NamedArgs))
	}
	if call.NamedArgs[0].Name != "id" {
		t.Fatalf("expected first named arg id, got %q", call.NamedArgs[0].Name)
	}
	if call.NamedArgs[1].Name != "balance" {
		t.Fatalf("expected second named arg balance, got %q", call.NamedArgs[1].Name)
	}
}

func TestParseObjectFieldAccess(t *testing.T) {
	expr := parseExprStmt(t, `{name: "Jane", age: 30}.name`)
	field, ok := expr.(*ast.FieldExpr)
	if !ok {
		t.Fatalf("expected FieldExpr, got %T", expr)
	}
	if field.Name != "name" {
		t.Fatalf("expected field name, got %q", field.Name)
	}
	obj, ok := field.Target.(*ast.ObjLit)
	if !ok {
		t.Fatalf("expected ObjLit target, got %T", field.Target)
	}
	if len(obj.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(obj.Fields))
	}
}

func TestParseSwitchStmt(t *testing.T) {
	toks, lexErrs := lexer.New(`
fn main()
{
    switch("2"){
        case("1"){
            print("one")
        }
        case("2"){
            print("two")
        }
        default(){
            print("other")
        }
    }
}
`).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	fn := tree.Decls[0].(*ast.FnDecl)
	sw, ok := fn.Body.Stmts[0].(*ast.SwitchStmt)
	if !ok {
		t.Fatalf("expected SwitchStmt, got %T", fn.Body.Stmts[0])
	}
	if len(sw.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(sw.Cases))
	}
	if sw.Default == nil {
		t.Fatalf("expected default branch")
	}
	if _, ok := sw.Cases[0].Value.(*ast.StrLit); !ok {
		t.Fatalf("expected string case value, got %T", sw.Cases[0].Value)
	}
}

func TestParseSwitchWithoutDefault(t *testing.T) {
	toks, lexErrs := lexer.New(`
fn main()
{
    switch(true){
        case(true){
            print("yes")
        }
    }
}
`).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	fn := tree.Decls[0].(*ast.FnDecl)
	sw := fn.Body.Stmts[0].(*ast.SwitchStmt)
	if sw.Default != nil {
		t.Fatalf("expected no default branch")
	}
}

func TestParseForListStmt(t *testing.T) {
	toks, lexErrs := lexer.New(`
fn main()
{
    xs = [1, 2]
    for(x in xs){
        print(x)
    }
}
`).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	fn := tree.Decls[0].(*ast.FnDecl)
	stmt, ok := fn.Body.Stmts[1].(*ast.ForStmt)
	if !ok {
		t.Fatalf("expected ForStmt, got %T", fn.Body.Stmts[1])
	}
	if stmt.Var != "x" {
		t.Fatalf("expected loop var x, got %q", stmt.Var)
	}
	if _, ok := stmt.Target.(*ast.Ident); !ok {
		t.Fatalf("expected ident target, got %T", stmt.Target)
	}
}

func TestParseForRangeStmt(t *testing.T) {
	toks, lexErrs := lexer.New(`
fn main()
{
    for(i in 0..10){
        print(i)
    }
}
`).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	fn := tree.Decls[0].(*ast.FnDecl)
	stmt := fn.Body.Stmts[0].(*ast.ForStmt)
	rng, ok := stmt.Target.(*ast.RangeExpr)
	if !ok {
		t.Fatalf("expected RangeExpr target, got %T", stmt.Target)
	}
	from := rng.From.(*ast.IntLit)
	to := rng.To.(*ast.IntLit)
	if from.Value != 0 || to.Value != 10 {
		t.Fatalf("expected 0..10, got %d..%d", from.Value, to.Value)
	}
}

func TestParseRejectsMalformedForHeader(t *testing.T) {
	errs := parseErrs(t, "fn main(){ for(x xs){ print(x) } }")
	if len(errs) == 0 {
		t.Fatal("expected parse error for malformed for header")
	}
}

func TestParseRejectsDuplicateSwitchDefault(t *testing.T) {
	toks, lexErrs := lexer.New(`
fn main()
{
    switch("x"){
        default(){ print("a") }
        default(){ print("b") }
    }
}
`).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	_, parseErrs := New(toks).Parse()
	if len(parseErrs) == 0 {
		t.Fatalf("expected duplicate default parse error")
	}
}

func parseErrs(t *testing.T, src string) []error {
	t.Helper()
	toks, lexErrs := lexer.New(src).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	_, errs := New(toks).Parse()
	return errs
}

func TestParseRejectsFieldAssignment(t *testing.T) {
	errs := parseErrs(t, "fn main(){\n    t = 1\n    t.x = 2\n}")
	if len(errs) == 0 {
		t.Fatal("expected parse error for field assignment, got none")
	}
	if !strings.Contains(errs[0].Error(), "immutable") {
		t.Fatalf("expected immutability message, got: %v", errs[0])
	}
}

func TestParseFieldAssignmentMentionsWithSyntax(t *testing.T) {
	errs := parseErrs(t, "fn main(){\n    t = 1\n    t.x = 99\n}")
	if len(errs) == 0 {
		t.Fatal("expected parse error for field assignment, got none")
	}
	if !strings.Contains(errs[0].Error(), ".with(") {
		t.Fatalf("expected error to mention .with(), got: %v", errs[0])
	}
}

func TestParseFieldAssignmentDoesNotCascade(t *testing.T) {
	// The rest of the block after the bad assignment should still parse cleanly
	// (no spurious second error from leftover tokens).
	errs := parseErrs(t, "fn main(){\n    t = 1\n    t.x = 2\n    print(t)\n}")
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 parse error (no cascade), got %d: %v", len(errs), errs)
	}
}

func TestParseLetExpr(t *testing.T) {
	expr := parseExprStmt(t, `let(x = 10, y = x + 1){ y }`)
	let, ok := expr.(*ast.LetExpr)
	if !ok {
		t.Fatalf("expected LetExpr, got %T", expr)
	}
	if len(let.Bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(let.Bindings))
	}
	if let.Bindings[0].Name != "x" || let.Bindings[1].Name != "y" {
		t.Fatalf("unexpected bindings: %+v", let.Bindings)
	}
}

func TestParseRejectsLetDestructuring(t *testing.T) {
	errs := parseErrs(t, "fn main(){ let([x] = [1]){ x } }")
	if len(errs) == 0 {
		t.Fatal("expected parse error for let destructuring")
	}
	if !strings.Contains(errs[0].Error(), "Destructuring") && !strings.Contains(errs[0].Error(), "destructuring") {
		t.Fatalf("expected destructuring error, got: %v", errs[0])
	}
}

func TestParseMatchExpr(t *testing.T) {
	expr := parseExprStmt(t, `match(x){ case(1){ "one" } case(_){ "other" } }`)
	m, ok := expr.(*ast.MatchExpr)
	if !ok {
		t.Fatalf("expected MatchExpr, got %T", expr)
	}
	if len(m.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(m.Cases))
	}
	if _, ok := m.Cases[0].Pattern.(*ast.LiteralPattern); !ok {
		t.Fatalf("expected literal pattern, got %T", m.Cases[0].Pattern)
	}
	if _, ok := m.Cases[1].Pattern.(*ast.WildcardPattern); !ok {
		t.Fatalf("expected wildcard pattern, got %T", m.Cases[1].Pattern)
	}
}

func TestParseMatchCapturePattern(t *testing.T) {
	expr := parseExprStmt(t, `match(x){ case(value){ value } }`)
	m := expr.(*ast.MatchExpr)
	cap, ok := m.Cases[0].Pattern.(*ast.CapturePattern)
	if !ok {
		t.Fatalf("expected capture pattern, got %T", m.Cases[0].Pattern)
	}
	if cap.Name != "value" {
		t.Fatalf("expected capture name value, got %q", cap.Name)
	}
}
