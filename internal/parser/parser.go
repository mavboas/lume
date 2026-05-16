// Package parser turns a token stream into an AST.
package parser

import (
	"fmt"
	"strconv"

	"github.com/mavboas/lume/internal/ast"
	"github.com/mavboas/lume/internal/lexer"
)

// Parser builds an ast.File from a token stream.
type Parser struct {
	tokens []lexer.Token
	pos    int
	errs   []error
}

// New creates a Parser positioned at the first token.
func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens}
}

// Parse consumes the token stream and returns the file AST plus any errors.
func (p *Parser) Parse() (*ast.File, []error) {
	file := &ast.File{Pos: p.cur().Pos}
	for !p.atEOF() {
		d := p.parseDecl()
		if d != nil {
			file.Decls = append(file.Decls, d)
		}
	}
	return file, p.errs
}

// ---- token helpers ----

func (p *Parser) cur() lexer.Token { return p.tokens[p.pos] }

func (p *Parser) peek(n int) lexer.Token {
	if p.pos+n >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1] // EOF
	}
	return p.tokens[p.pos+n]
}

func (p *Parser) advance() lexer.Token {
	t := p.tokens[p.pos]
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return t
}

func (p *Parser) atEOF() bool { return p.cur().Kind == lexer.EOF }

func (p *Parser) accept(k lexer.Kind) bool {
	if p.cur().Kind == k {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) expect(k lexer.Kind, ctx string) lexer.Token {
	t := p.cur()
	if t.Kind != k {
		p.errf(t.Pos, "expected %s but got %s (%q): %s", kindLabel(k), kindLabel(t.Kind), t.Value, ctx)
		return t
	}
	return p.advance()
}

func (p *Parser) errf(pos lexer.Position, format string, args ...any) {
	p.errs = append(p.errs, fmt.Errorf("parse error at %s: %s", pos, fmt.Sprintf(format, args...)))
}

// ---- declarations ----

func (p *Parser) parseDecl() ast.Decl {
	public := false
	if p.cur().Kind == lexer.KwPub {
		public = true
		p.advance()
	}
	switch p.cur().Kind {
	case lexer.KwFn:
		return p.parseFnDecl(public)
	case lexer.KwCl:
		return p.parseClassDecl(public)
	default:
		t := p.advance()
		p.errf(t.Pos, "unexpected token %q at top level: expected fn or cl declaration", t.Value)
		return nil
	}
}

func (p *Parser) parseClassDecl(public bool) *ast.ClassDecl {
	pos := p.cur().Pos
	p.expect(lexer.KwCl, "class declaration")
	nameTok := p.expect(lexer.Ident, "class name")
	cl := &ast.ClassDecl{Name: nameTok.Value, Public: public, Pos: pos}
	p.expect(lexer.LBrace, "class body")
	for p.cur().Kind != lexer.RBrace && !p.atEOF() {
		p.accept(lexer.Semi) // skip optional newline/semicolons between fields
		if p.cur().Kind == lexer.RBrace {
			break
		}
		fieldName := p.expect(lexer.Ident, "class field name")
		p.expect(lexer.Colon, "after class field name")
		fieldType := p.parseTypeExpr()
		cl.Fields = append(cl.Fields, ast.ClassField{
			Name: fieldName.Value,
			Type: fieldType,
			Pos:  fieldName.Pos,
		})
		p.accept(lexer.Semi)
	}
	p.expect(lexer.RBrace, "end of class body")
	return cl
}

func (p *Parser) parseFnDecl(public bool) *ast.FnDecl {
	pos := p.cur().Pos
	p.expect(lexer.KwFn, "function declaration")
	nameTok := p.expect(lexer.Ident, "function name")

	fn := &ast.FnDecl{
		Name:   nameTok.Value,
		Public: public,
		Pos:    pos,
	}

	// Parameters
	p.expect(lexer.LParen, "after function name")
	for p.cur().Kind != lexer.RParen && !p.atEOF() {
		param := p.parseParam()
		fn.Params = append(fn.Params, param)
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.RParen, "end of parameter list")

	// Optional return type: `-> Type`
	if p.accept(lexer.Arrow) {
		fn.Return = p.parseTypeExpr()
	}

	// Optional documentation string before the body
	if p.cur().Kind == lexer.Str {
		fn.Doc = p.advance().Value
	}

	fn.Body = p.parseBlock()
	return fn
}

func (p *Parser) parseParam() ast.Param {
	nameTok := p.expect(lexer.Ident, "parameter name")
	param := ast.Param{Name: nameTok.Value, Pos: nameTok.Pos}
	if p.accept(lexer.Colon) {
		param.Type = p.parseTypeExpr()
	}
	return param
}

func (p *Parser) parseTypeExpr() *ast.TypeExpr {
	t := p.expect(lexer.Ident, "type name")
	expr := &ast.TypeExpr{Name: t.Value, Pos: t.Pos}
	if p.accept(lexer.LBracket) {
		for p.cur().Kind != lexer.RBracket && !p.atEOF() {
			expr.Args = append(expr.Args, p.parseTypeExpr())
			if !p.accept(lexer.Comma) {
				break
			}
		}
		p.expect(lexer.RBracket, "end of type argument list")
	}
	return expr
}

// ---- blocks & statements ----

func (p *Parser) parseBlock() *ast.Block {
	pos := p.cur().Pos
	p.expect(lexer.LBrace, "start of block")
	block := &ast.Block{Pos: pos}
	for p.cur().Kind != lexer.RBrace && !p.atEOF() {
		s := p.parseStmt()
		if s != nil {
			block.Stmts = append(block.Stmts, s)
		}
		// Optional semicolon between statements
		p.accept(lexer.Semi)
	}
	p.expect(lexer.RBrace, "end of block")
	return block
}

func (p *Parser) parseStmt() ast.Stmt {
	switch p.cur().Kind {
	case lexer.KwIf:
		return p.parseIfStmt()
	case lexer.KwSwitch:
		return p.parseSwitchStmt()
	case lexer.KwFor:
		return p.parseForStmt()
	}
	// Look ahead: `IDENT =` or `IDENT: Type =` is a binding, anything else is
	// an expression.
	if p.cur().Kind == lexer.Ident && (p.peek(1).Kind == lexer.Assign || p.peek(1).Kind == lexer.Colon) {
		return p.parseLetStmt()
	}
	pos := p.cur().Pos
	expr := p.parseExpr()
	if expr == nil {
		return nil
	}
	// Detect attempted field/expression assignment: t.field = val.
	// Only simple-identifier bindings (handled above) are legal assignment targets.
	if p.cur().Kind == lexer.Assign {
		if fe, ok := expr.(*ast.FieldExpr); ok {
			p.errf(p.cur().Pos,
				"cannot assign to field %q: class fields are immutable; use .with(%s= newValue) to derive a new instance",
				fe.Name, fe.Name)
		} else {
			p.errf(p.cur().Pos,
				"unexpected '=': only simple bindings (name = value) are allowed as statements")
		}
		p.advance()   // consume '=' to prevent a second spurious error
		p.parseExpr() // consume RHS so the rest of the block parses cleanly
		return nil
	}
	return &ast.ExprStmt{Expr: expr, Pos: pos}
}

// parseIfStmt parses `if(cond){body} elsif(cond){body}* else{body}?`.
func (p *Parser) parseIfStmt() *ast.IfStmt {
	pos := p.cur().Pos
	p.expect(lexer.KwIf, "if statement")
	p.expect(lexer.LParen, "after if")
	cond := p.parseExpr()
	p.expect(lexer.RParen, "after if condition")
	body := p.parseBlock()

	stmt := &ast.IfStmt{Cond: cond, Body: body, Pos: pos}

	for p.cur().Kind == lexer.KwElsif {
		ePos := p.cur().Pos
		p.advance()
		p.expect(lexer.LParen, "after elsif")
		ec := p.parseExpr()
		p.expect(lexer.RParen, "after elsif condition")
		eb := p.parseBlock()
		stmt.Elsifs = append(stmt.Elsifs, ast.ElsifBranch{Cond: ec, Body: eb, Pos: ePos})
	}

	if p.cur().Kind == lexer.KwElse {
		p.advance()
		stmt.Else = p.parseBlock()
	}
	return stmt
}

// parseForStmt parses `for(name in target){body}`.
func (p *Parser) parseForStmt() *ast.ForStmt {
	pos := p.cur().Pos
	p.expect(lexer.KwFor, "for statement")
	p.expect(lexer.LParen, "after for")
	nameTok := p.expect(lexer.Ident, "for loop variable")
	p.expect(lexer.KwIn, "after for loop variable")
	target := p.parseExpr()
	p.expect(lexer.RParen, "after for target")
	body := p.parseBlock()
	return &ast.ForStmt{Var: nameTok.Value, Target: target, Body: body, Pos: pos}
}

// parseSwitchStmt parses `switch(expr){ case(expr){body}* default(){body}? }`.
func (p *Parser) parseSwitchStmt() *ast.SwitchStmt {
	pos := p.cur().Pos
	p.expect(lexer.KwSwitch, "switch statement")
	p.expect(lexer.LParen, "after switch")
	value := p.parseExpr()
	p.expect(lexer.RParen, "after switch value")

	stmt := &ast.SwitchStmt{Value: value, Pos: pos}
	p.expect(lexer.LBrace, "switch body")
	seenDefault := false
	for p.cur().Kind != lexer.RBrace && !p.atEOF() {
		p.accept(lexer.Semi)
		if p.cur().Kind == lexer.RBrace {
			break
		}
		switch p.cur().Kind {
		case lexer.KwCase:
			casePos := p.cur().Pos
			p.advance()
			p.expect(lexer.LParen, "after case")
			caseValue := p.parseExpr()
			p.expect(lexer.RParen, "after case value")
			body := p.parseBlock()
			stmt.Cases = append(stmt.Cases, ast.CaseBranch{Value: caseValue, Body: body, Pos: casePos})
		case lexer.KwDefault:
			defaultPos := p.cur().Pos
			p.advance()
			if seenDefault {
				p.errf(defaultPos, "duplicate default clause in switch")
			}
			seenDefault = true
			p.expect(lexer.LParen, "after default")
			p.expect(lexer.RParen, "default takes no value")
			stmt.Default = p.parseBlock()
		default:
			t := p.advance()
			p.errf(t.Pos, "unexpected token %q in switch body: expected case or default", t.Value)
		}
		p.accept(lexer.Semi)
	}
	p.expect(lexer.RBrace, "end of switch body")
	return stmt
}

func (p *Parser) parseLetStmt() *ast.LetStmt {
	nameTok := p.advance() // IDENT
	stmt := &ast.LetStmt{Name: nameTok.Value, Pos: nameTok.Pos}
	if p.accept(lexer.Colon) {
		stmt.Type = p.parseTypeExpr()
	}
	p.expect(lexer.Assign, "binding value")
	val := p.parseExpr()
	stmt.Value = val
	return stmt
}

// ---- expressions (Pratt parsing for operator precedence) ----

// precedence returns the binding power for a binary operator token, or 0 if not one.
func precedence(k lexer.Kind) int {
	switch k {
	case lexer.Range:
		return 1
	case lexer.OrOr:
		return 2
	case lexer.AndAnd:
		return 3
	case lexer.Eq, lexer.Neq:
		return 4
	case lexer.Lt, lexer.Lte, lexer.Gt, lexer.Gte:
		return 5
	case lexer.Plus, lexer.Minus:
		return 6
	case lexer.Star, lexer.Slash, lexer.Percent:
		return 7
	}
	return 0
}

func opString(k lexer.Kind) string {
	switch k {
	case lexer.Plus:
		return "+"
	case lexer.Minus:
		return "-"
	case lexer.Star:
		return "*"
	case lexer.Slash:
		return "/"
	case lexer.Percent:
		return "%"
	case lexer.Eq:
		return "=="
	case lexer.Neq:
		return "!="
	case lexer.Lt:
		return "<"
	case lexer.Lte:
		return "<="
	case lexer.Gt:
		return ">"
	case lexer.Gte:
		return ">="
	case lexer.AndAnd:
		return "&&"
	case lexer.OrOr:
		return "||"
	case lexer.Bang:
		return "!"
	case lexer.Range:
		return ".."
	}
	return "?"
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseBinary(1)
}

func (p *Parser) parseBinary(minPrec int) ast.Expr {
	left := p.parseUnary()
	for {
		prec := precedence(p.cur().Kind)
		if prec < minPrec {
			return left
		}
		opTok := p.advance()
		right := p.parseBinary(prec + 1) // left-associative
		if opTok.Kind == lexer.Range {
			left = &ast.RangeExpr{
				From: left,
				To:   right,
				Pos:  opTok.Pos,
			}
			continue
		}
		left = &ast.BinaryExpr{
			Op:    opString(opTok.Kind),
			Left:  left,
			Right: right,
			Pos:   opTok.Pos,
		}
	}
}

func (p *Parser) parseUnary() ast.Expr {
	switch p.cur().Kind {
	case lexer.Minus, lexer.Bang:
		opTok := p.advance()
		sub := p.parseUnary()
		return &ast.UnaryExpr{Op: opString(opTok.Kind), Sub: sub, Pos: opTok.Pos}
	}
	return p.parsePostfix()
}

// parsePostfix handles call expressions: primary followed by zero or more `(args)`.
func (p *Parser) parsePostfix() ast.Expr {
	expr := p.parsePrimary()
	for {
		if p.cur().Kind == lexer.LParen {
			expr = p.parseCall(expr)
			continue
		}
		if p.cur().Kind == lexer.Dot {
			expr = p.parseField(expr)
			continue
		}
		return expr
	}
}

func (p *Parser) parseField(target ast.Expr) ast.Expr {
	pos := p.cur().Pos
	p.advance() // .
	name := p.expect(lexer.Ident, "field name")
	return &ast.FieldExpr{Target: target, Name: name.Value, Pos: pos}
}

func (p *Parser) parseCall(callee ast.Expr) ast.Expr {
	pos := p.cur().Pos
	p.advance() // (
	call := &ast.CallExpr{Fn: callee, Pos: pos}
	if p.cur().Kind == lexer.RParen {
		p.advance()
		return call
	}
	// Named-arg detection: if first arg looks like `Ident =` it's a constructor call.
	if p.cur().Kind == lexer.Ident && p.peek(1).Kind == lexer.Assign {
		for p.cur().Kind != lexer.RParen && !p.atEOF() {
			nameTok := p.expect(lexer.Ident, "named argument name")
			p.expect(lexer.Assign, "after named argument name")
			val := p.parseExpr()
			if val != nil {
				call.NamedArgs = append(call.NamedArgs, ast.NamedArg{
					Name:  nameTok.Value,
					Value: val,
					Pos:   nameTok.Pos,
				})
			}
			if !p.accept(lexer.Comma) {
				break
			}
		}
	} else {
		for p.cur().Kind != lexer.RParen && !p.atEOF() {
			arg := p.parseExpr()
			if arg != nil {
				call.Args = append(call.Args, arg)
			}
			if !p.accept(lexer.Comma) {
				break
			}
		}
	}
	p.expect(lexer.RParen, "end of call")
	return call
}

func (p *Parser) parsePrimary() ast.Expr {
	t := p.cur()
	switch t.Kind {
	case lexer.Int:
		p.advance()
		v, err := strconv.ParseInt(t.Value, 10, 64)
		if err != nil {
			p.errf(t.Pos, "invalid integer literal %q", t.Value)
		}
		return &ast.IntLit{Value: v, Pos: t.Pos}
	case lexer.Dec:
		p.advance()
		v, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			p.errf(t.Pos, "invalid decimal literal %q", t.Value)
		}
		return &ast.DecLit{Value: v, Pos: t.Pos}
	case lexer.Str:
		p.advance()
		return &ast.StrLit{Value: t.Value, Pos: t.Pos}
	case lexer.StrStart:
		return p.parseInterpolatedStr()
	case lexer.KwTrue:
		p.advance()
		return &ast.BoolLit{Value: true, Pos: t.Pos}
	case lexer.KwFalse:
		p.advance()
		return &ast.BoolLit{Value: false, Pos: t.Pos}
	case lexer.KwNone:
		p.advance()
		return &ast.NoneLit{Pos: t.Pos}
	case lexer.KwLet:
		return p.parseLetExpr()
	case lexer.KwMatch:
		return p.parseMatchExpr()
	case lexer.Ident:
		p.advance()
		return &ast.Ident{Name: t.Value, Pos: t.Pos}
	case lexer.LParen:
		p.advance()
		expr := p.parseExpr()
		p.expect(lexer.RParen, "closing parenthesis")
		return expr
	case lexer.LBracket:
		return p.parseListLit()
	case lexer.LBrace:
		return p.parseObjLit()
	}
	p.errf(t.Pos, "unexpected token %q in expression", t.Value)
	p.advance()
	return nil
}

func (p *Parser) parseLetExpr() ast.Expr {
	pos := p.cur().Pos
	p.expect(lexer.KwLet, "let expression")
	p.expect(lexer.LParen, "after let")
	node := &ast.LetExpr{Pos: pos}
	for p.cur().Kind != lexer.RParen && !p.atEOF() {
		if p.cur().Kind == lexer.LBracket || p.cur().Kind == lexer.LBrace || p.cur().Kind == lexer.LParen {
			p.errf(p.cur().Pos, "destructuring bindings are not supported in let expressions")
			p.parseExpr()
			p.expect(lexer.Assign, "after unsupported destructuring pattern")
			p.parseExpr()
		} else {
			nameTok := p.expect(lexer.Ident, "let binding name")
			if nameTok.Value == "_" {
				p.errf(nameTok.Pos, "wildcard is not a valid let binding name")
			}
			p.expect(lexer.Assign, "after let binding name")
			value := p.parseExpr()
			if value != nil {
				node.Bindings = append(node.Bindings, ast.LetBinding{
					Name:  nameTok.Value,
					Value: value,
					Pos:   nameTok.Pos,
				})
			}
		}
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.RParen, "end of let bindings")
	node.Body = p.parseBlock()
	return node
}

func (p *Parser) parseMatchExpr() ast.Expr {
	pos := p.cur().Pos
	p.expect(lexer.KwMatch, "match expression")
	p.expect(lexer.LParen, "after match")
	value := p.parseExpr()
	p.expect(lexer.RParen, "after match value")

	node := &ast.MatchExpr{Value: value, Pos: pos}
	p.expect(lexer.LBrace, "match body")
	for p.cur().Kind != lexer.RBrace && !p.atEOF() {
		p.accept(lexer.Semi)
		if p.cur().Kind == lexer.RBrace {
			break
		}
		casePos := p.cur().Pos
		p.expect(lexer.KwCase, "match case")
		p.expect(lexer.LParen, "after case")
		pat := p.parsePattern()
		p.expect(lexer.RParen, "after case pattern")
		body := p.parseBlock()
		node.Cases = append(node.Cases, ast.MatchCase{Pattern: pat, Body: body, Pos: casePos})
		p.accept(lexer.Semi)
	}
	p.expect(lexer.RBrace, "end of match body")
	return node
}

func (p *Parser) parsePattern() ast.Pattern {
	t := p.cur()
	switch t.Kind {
	case lexer.Int, lexer.Dec, lexer.Str, lexer.KwTrue, lexer.KwFalse, lexer.KwNone:
		expr := p.parsePrimary()
		return &ast.LiteralPattern{Value: expr, Pos: t.Pos}
	case lexer.Ident:
		p.advance()
		if t.Value == "_" {
			return &ast.WildcardPattern{Pos: t.Pos}
		}
		return &ast.CapturePattern{Name: t.Value, Pos: t.Pos}
	case lexer.LBracket, lexer.LBrace:
		p.errf(t.Pos, "list and object patterns are not supported in match yet")
		p.parseExpr()
		return &ast.WildcardPattern{Pos: t.Pos}
	default:
		p.errf(t.Pos, "unsupported match pattern %q", t.Value)
		p.advance()
		return &ast.WildcardPattern{Pos: t.Pos}
	}
}

func (p *Parser) parseListLit() ast.Expr {
	pos := p.cur().Pos
	p.expect(lexer.LBracket, "list literal")
	node := &ast.ListLit{Pos: pos}
	for p.cur().Kind != lexer.RBracket && !p.atEOF() {
		elem := p.parseExpr()
		if elem != nil {
			node.Elems = append(node.Elems, elem)
		}
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.RBracket, "end of list literal")
	return node
}

func (p *Parser) parseObjLit() ast.Expr {
	pos := p.cur().Pos
	p.expect(lexer.LBrace, "object literal")
	node := &ast.ObjLit{Pos: pos}
	for p.cur().Kind != lexer.RBrace && !p.atEOF() {
		name := p.expect(lexer.Ident, "object field name")
		p.expect(lexer.Colon, "after object field name")
		value := p.parseExpr()
		if value != nil {
			node.Fields = append(node.Fields, ast.ObjField{Name: name.Value, Value: value, Pos: name.Pos})
		}
		if !p.accept(lexer.Comma) {
			break
		}
	}
	p.expect(lexer.RBrace, "end of object literal")
	return node
}

// parseInterpolatedStr consumes a multi-token interpolated string:
//
//	StrStart (StrChunk | StrInterpStart expr StrInterpEnd)* StrEnd
func (p *Parser) parseInterpolatedStr() ast.Expr {
	startTok := p.advance() // StrStart
	node := &ast.InterpolatedStr{Pos: startTok.Pos}

	for !p.atEOF() && p.cur().Kind != lexer.StrEnd {
		switch p.cur().Kind {
		case lexer.StrChunk:
			t := p.advance()
			node.Parts = append(node.Parts, ast.InterpPart{IsExpr: false, Lit: t.Value})
		case lexer.StrInterpStart:
			p.advance()
			expr := p.parseExpr()
			p.expect(lexer.StrInterpEnd, "after ${expr} interpolation")
			node.Parts = append(node.Parts, ast.InterpPart{IsExpr: true, Expr: expr})
		default:
			p.errf(p.cur().Pos, "unexpected token %q inside string", p.cur().Value)
			p.advance()
		}
	}
	p.expect(lexer.StrEnd, "end of interpolated string")
	return node
}

// kindLabel returns a friendly label for a Kind (for error messages).
func kindLabel(k lexer.Kind) string {
	switch k {
	case lexer.LParen:
		return "("
	case lexer.RParen:
		return ")"
	case lexer.LBrace:
		return "{"
	case lexer.RBrace:
		return "}"
	case lexer.Comma:
		return ","
	case lexer.Colon:
		return ":"
	case lexer.LBracket:
		return "["
	case lexer.RBracket:
		return "]"
	case lexer.Dot:
		return "."
	case lexer.Range:
		return ".."
	case lexer.Arrow:
		return "->"
	case lexer.Ident:
		return "identifier"
	case lexer.Str:
		return "string literal"
	case lexer.Int:
		return "integer literal"
	case lexer.Dec:
		return "decimal literal"
	case lexer.KwFn:
		return "fn"
	case lexer.KwCl:
		return "cl"
	case lexer.KwSwitch:
		return "switch"
	case lexer.KwCase:
		return "case"
	case lexer.KwDefault:
		return "default"
	case lexer.KwFor:
		return "for"
	case lexer.KwIn:
		return "in"
	case lexer.KwLet:
		return "let"
	case lexer.KwMatch:
		return "match"
	case lexer.Assign:
		return "="
	case lexer.EOF:
		return "end of file"
	case lexer.StrEnd:
		return "end of string"
	case lexer.StrInterpEnd:
		return "}"
	}
	return fmt.Sprintf("token(%d)", k)
}
