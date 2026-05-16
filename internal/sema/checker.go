// Package sema validates Lume programs before code generation.
package sema

import (
	"errors"
	"fmt"

	"github.com/mavboas/lume/internal/ast"
	"github.com/mavboas/lume/internal/lexer"
)

type TypeKind int

const (
	KindInvalid TypeKind = iota
	KindUnknown
	KindNone
	KindInt
	KindDec
	KindStr
	KindBool
	KindObj
	KindRange
	KindList
	KindClass
)

type Type struct {
	Kind  TypeKind
	Elem  *Type
	Class string
}

var (
	Invalid = Type{Kind: KindInvalid}
	Unknown = Type{Kind: KindUnknown}
	None    = Type{Kind: KindNone}
	Int     = Type{Kind: KindInt}
	Dec     = Type{Kind: KindDec}
	Str     = Type{Kind: KindStr}
	Bool    = Type{Kind: KindBool}
	Obj     = Type{Kind: KindObj}
	Range   = Type{Kind: KindRange}
)

func (t Type) String() string {
	switch t.Kind {
	case KindInvalid:
		return "<invalid>"
	case KindUnknown:
		return "<unknown>"
	case KindNone:
		return "none"
	case KindInt:
		return "int"
	case KindDec:
		return "dec"
	case KindStr:
		return "str"
	case KindBool:
		return "bool"
	case KindObj:
		return "obj"
	case KindRange:
		return "range"
	case KindList:
		if t.Elem == nil {
			return "list[<invalid>]"
		}
		return "list[" + t.Elem.String() + "]"
	case KindClass:
		return t.Class
	}
	return "<invalid>"
}

// Error is a structured compiler diagnostic.
type Error struct {
	Code    string
	Pos     lexer.Position
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s at %s: %s", e.Code, e.Pos, e.Message)
}

type fnSig struct {
	name   string
	params []Type
	ret    Type
	pos    lexer.Position
}

type classInfo struct {
	fields []classField
	pos    lexer.Position
}

type classField struct {
	name string
	typ  Type
}

type Checker struct {
	fns     map[string]fnSig
	classes map[string]classInfo
	errs    []error
}

func Check(file *ast.File) error {
	c := &Checker{
		fns:     map[string]fnSig{},
		classes: map[string]classInfo{},
	}
	c.collect(file)
	c.check(file)
	if len(c.errs) > 0 {
		return errors.Join(c.errs...)
	}
	return nil
}

func (c *Checker) collect(file *ast.File) {
	// First pass: collect class declarations so they're available for type resolution.
	for _, d := range file.Decls {
		cl, ok := d.(*ast.ClassDecl)
		if !ok {
			continue
		}
		if _, exists := c.classes[cl.Name]; exists {
			c.error(cl.Pos, "E1002", "class %q is already declared", cl.Name)
			continue
		}
		info := classInfo{pos: cl.Pos}
		for _, f := range cl.Fields {
			info.fields = append(info.fields, classField{name: f.Name, typ: c.typeExpr(f.Type)})
		}
		c.classes[cl.Name] = info
	}
	// Second pass: collect function signatures.
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FnDecl)
		if !ok {
			continue
		}
		if _, exists := c.fns[fn.Name]; exists {
			c.error(fn.Pos, "E1001", "function %q is already declared", fn.Name)
			continue
		}
		sig := fnSig{name: fn.Name, ret: None, pos: fn.Pos}
		if fn.Return != nil {
			sig.ret = c.typeExpr(fn.Return)
		}
		for _, p := range fn.Params {
			if p.Type == nil {
				sig.params = append(sig.params, Unknown)
				continue
			}
			sig.params = append(sig.params, c.typeExpr(p.Type))
		}
		c.fns[fn.Name] = sig
	}
}

func (c *Checker) check(file *ast.File) {
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FnDecl)
		if !ok {
			continue
		}
		env := map[string]Type{}
		for _, p := range fn.Params {
			if p.Type == nil {
				env[p.Name] = Unknown
				continue
			}
			env[p.Name] = c.typeExpr(p.Type)
		}
		got := c.block(fn.Body, env)
		want := None
		if fn.Return != nil {
			want = c.typeExpr(fn.Return)
		}
		if fn.Return != nil && !assignable(want, got) {
			c.error(fn.Pos, "E2001", "function %q returns %s, expected %s", fn.Name, got, want)
		}
	}
}

func (c *Checker) block(b *ast.Block, env map[string]Type) Type {
	if b == nil || len(b.Stmts) == 0 {
		return None
	}
	var out Type = None
	last := len(b.Stmts) - 1
	for i, s := range b.Stmts {
		out = c.stmt(s, env, i == last)
	}
	return out
}

func (c *Checker) stmt(s ast.Stmt, env map[string]Type, final bool) Type {
	switch ss := s.(type) {
	case *ast.LetStmt:
		if lit, ok := ss.Value.(*ast.ListLit); ok && len(lit.Elems) == 0 && ss.Type == nil {
			c.error(ss.Pos, "E2601", "empty list literal needs an explicit type annotation")
		}
		t := c.expr(ss.Value, env)
		if ss.Type != nil {
			want := c.typeExpr(ss.Type)
			if !assignable(want, t) {
				c.error(ss.Pos, "E2002", "binding %q has type %s, expected %s", ss.Name, t, want)
			}
			t = want
		}
		env[ss.Name] = t
		return None
	case *ast.ExprStmt:
		if m, ok := ss.Expr.(*ast.MatchExpr); ok {
			return c.matchExpr(m, env, final)
		}
		return c.expr(ss.Expr, env)
	case *ast.IfStmt:
		return c.ifStmt(ss, env)
	case *ast.SwitchStmt:
		return c.switchStmt(ss, env)
	case *ast.ForStmt:
		return c.forStmt(ss, env)
	}
	return Invalid
}

func (c *Checker) ifStmt(s *ast.IfStmt, env map[string]Type) Type {
	if got := c.expr(s.Cond, env); !assignable(Bool, got) {
		c.error(s.Cond.Position(), "E2101", "if condition must be bool, got %s", got)
	}
	branchTypes := []Type{c.block(s.Body, cloneEnv(env))}
	for _, e := range s.Elsifs {
		if got := c.expr(e.Cond, env); !assignable(Bool, got) {
			c.error(e.Cond.Position(), "E2101", "elsif condition must be bool, got %s", got)
		}
		branchTypes = append(branchTypes, c.block(e.Body, cloneEnv(env)))
	}
	if s.Else == nil {
		return None
	}
	branchTypes = append(branchTypes, c.block(s.Else, cloneEnv(env)))
	first := branchTypes[0]
	for _, t := range branchTypes[1:] {
		if !assignable(first, t) {
			c.error(s.Position(), "E2102", "if branches return incompatible types %s and %s", first, t)
			return Invalid
		}
	}
	return first
}

func (c *Checker) switchStmt(s *ast.SwitchStmt, env map[string]Type) Type {
	valueType := c.expr(s.Value, env)
	branchTypes := make([]Type, 0, len(s.Cases)+1)
	for _, branch := range s.Cases {
		if !isSwitchCaseLiteral(branch.Value) {
			c.error(branch.Value.Position(), "E2103", "switch case value must be a literal")
		}
		caseType := c.expr(branch.Value, env)
		if !assignable(valueType, caseType) {
			c.error(branch.Value.Position(), "E2104", "switch case has type %s, expected %s", caseType, valueType)
		}
		branchTypes = append(branchTypes, c.block(branch.Body, cloneEnv(env)))
	}
	if s.Default == nil {
		return None
	}
	branchTypes = append(branchTypes, c.block(s.Default, cloneEnv(env)))
	if len(branchTypes) == 0 {
		return None
	}
	first := branchTypes[0]
	for _, t := range branchTypes[1:] {
		if !assignable(first, t) {
			c.error(s.Position(), "E2105", "switch branches return incompatible types %s and %s", first, t)
			return Invalid
		}
	}
	return first
}

func (c *Checker) forStmt(s *ast.ForStmt, env map[string]Type) Type {
	target := c.expr(s.Target, env)
	var elem Type
	switch target {
	case Range:
		elem = Int
	default:
		elemType, ok := listElem(target)
		if !ok && target != Unknown && target != Invalid {
			c.error(s.Target.Position(), "E2106", "for target must be list or int range, got %s", target)
		}
		elem = elemType
	}
	loopEnv := cloneEnv(env)
	loopEnv[s.Var] = elem
	c.block(s.Body, loopEnv)
	return None
}

func isSwitchCaseLiteral(e ast.Expr) bool {
	switch e.(type) {
	case *ast.IntLit, *ast.DecLit, *ast.StrLit, *ast.BoolLit, *ast.NoneLit:
		return true
	}
	return false
}

func (c *Checker) expr(e ast.Expr, env map[string]Type) Type {
	switch ee := e.(type) {
	case *ast.IntLit:
		return Int
	case *ast.DecLit:
		return Dec
	case *ast.StrLit:
		return Str
	case *ast.InterpolatedStr:
		for _, p := range ee.Parts {
			if p.IsExpr {
				c.expr(p.Expr, env)
			}
		}
		return Str
	case *ast.BoolLit:
		return Bool
	case *ast.NoneLit:
		return None
	case *ast.ListLit:
		return c.listLit(ee, env)
	case *ast.ObjLit:
		return c.objLit(ee, env)
	case *ast.Ident:
		t, ok := env[ee.Name]
		if !ok {
			c.error(ee.Pos, "E2201", "unknown identifier %q", ee.Name)
			return Invalid
		}
		return t
	case *ast.UnaryExpr:
		return c.unary(ee, env)
	case *ast.BinaryExpr:
		return c.binary(ee, env)
	case *ast.RangeExpr:
		return c.rangeExpr(ee, env)
	case *ast.CallExpr:
		return c.call(ee, env)
	case *ast.FieldExpr:
		return c.field(ee, env)
	case *ast.LetExpr:
		return c.letExpr(ee, env)
	case *ast.MatchExpr:
		return c.matchExpr(ee, env, true)
	}
	return Invalid
}

func (c *Checker) letExpr(e *ast.LetExpr, env map[string]Type) Type {
	letEnv := cloneEnv(env)
	for _, b := range e.Bindings {
		if lit, ok := b.Value.(*ast.ListLit); ok && len(lit.Elems) == 0 {
			c.error(b.Pos, "E2601", "empty list literal needs an explicit type annotation")
		}
		t := c.expr(b.Value, letEnv)
		letEnv[b.Name] = t
	}
	out := c.block(e.Body, letEnv)
	e.ResolvedType = out.String()
	return out
}

func (c *Checker) matchExpr(e *ast.MatchExpr, env map[string]Type, expression bool) Type {
	valueType := c.expr(e.Value, env)
	branchTypes := make([]Type, 0, len(e.Cases))
	seenDefaultPattern := false
	boolLits := map[bool]bool{}

	for i, branch := range e.Cases {
		if seenDefaultPattern {
			c.error(branch.Pos, "E2803", "cases after wildcard or capture are unreachable")
		}
		caseEnv := cloneEnv(env)
		switch p := branch.Pattern.(type) {
		case *ast.LiteralPattern:
			pt := c.expr(p.Value, env)
			if !assignable(valueType, pt) {
				c.error(p.Pos, "E2801", "match pattern has type %s, expected %s", pt, valueType)
			}
			if lit, ok := p.Value.(*ast.BoolLit); ok {
				boolLits[lit.Value] = true
			}
		case *ast.WildcardPattern:
			seenDefaultPattern = true
			if i != len(e.Cases)-1 {
				c.error(p.Pos, "E2803", "wildcard pattern must be the last match case")
			}
		case *ast.CapturePattern:
			caseEnv[p.Name] = valueType
			seenDefaultPattern = true
			if i != len(e.Cases)-1 {
				c.error(p.Pos, "E2803", "capture pattern must be the last match case")
			}
		default:
			c.error(branch.Pos, "E2802", "unsupported match pattern")
		}
		branchTypes = append(branchTypes, c.block(branch.Body, caseEnv))
	}

	if len(branchTypes) == 0 {
		e.ResolvedType = None.String()
		return None
	}
	first := branchTypes[0]
	for _, t := range branchTypes[1:] {
		if !assignable(first, t) {
			c.error(e.Pos, "E2804", "match branches return incompatible types %s and %s", first, t)
			e.ResolvedType = Invalid.String()
			return Invalid
		}
	}
	exhaustiveBool := valueType == Bool && boolLits[true] && boolLits[false]
	if expression && !seenDefaultPattern && !exhaustiveBool {
		c.error(e.Pos, "E2805", "match expression is not exhaustive")
	}
	e.ResolvedType = first.String()
	return first
}

func (c *Checker) rangeExpr(e *ast.RangeExpr, env map[string]Type) Type {
	from := c.expr(e.From, env)
	to := c.expr(e.To, env)
	if !assignable(Int, from) || !assignable(Int, to) {
		c.error(e.Pos, "E2107", "range bounds must be int, got %s and %s", from, to)
		return Invalid
	}
	return Range
}

func (c *Checker) listLit(e *ast.ListLit, env map[string]Type) Type {
	if len(e.Elems) == 0 {
		return listOf(Unknown)
	}
	elemType := c.expr(e.Elems[0], env)
	for _, elem := range e.Elems[1:] {
		got := c.expr(elem, env)
		if !assignable(elemType, got) {
			c.error(elem.Position(), "E2602", "list elements must have one type, got %s and %s", elemType, got)
		}
	}
	return listOf(elemType)
}

func (c *Checker) objLit(e *ast.ObjLit, env map[string]Type) Type {
	seen := map[string]bool{}
	for _, field := range e.Fields {
		if seen[field.Name] {
			c.error(field.Pos, "E2701", "object field %q is already defined", field.Name)
		}
		seen[field.Name] = true
		c.expr(field.Value, env)
	}
	return Obj
}

func (c *Checker) field(e *ast.FieldExpr, env map[string]Type) Type {
	target := c.expr(e.Target, env)
	// Class instance field access.
	if target.Kind == KindClass {
		info, isClass := c.classes[target.Class]
		if !isClass {
			c.error(e.Pos, "E2702", "unknown class %q", target.Class)
			return Invalid
		}
		e.TargetIsClass = true
		e.ResolvedClass = target.Class
		if e.Name == "keys" {
			return listOf(Str)
		}
		for _, f := range info.fields {
			if f.name == e.Name {
				return f.typ
			}
		}
		c.error(e.Pos, "E2703", "class %q has no field %q", target, e.Name)
		return Invalid
	}
	if target != Obj && target != Unknown && target != Invalid {
		c.error(e.Pos, "E2702", "field access expects obj or class instance, got %s", target)
		return Invalid
	}
	return Unknown
}

func (c *Checker) unary(e *ast.UnaryExpr, env map[string]Type) Type {
	sub := c.expr(e.Sub, env)
	switch e.Op {
	case "-":
		if sub == Int || sub == Dec || sub == Unknown {
			return sub
		}
		c.error(e.Pos, "E2301", "operator - expects int or dec, got %s", sub)
	case "!":
		if assignable(Bool, sub) {
			return Bool
		}
		c.error(e.Pos, "E2302", "operator ! expects bool, got %s", sub)
	}
	return Invalid
}

func (c *Checker) binary(e *ast.BinaryExpr, env map[string]Type) Type {
	left := c.expr(e.Left, env)
	right := c.expr(e.Right, env)
	switch e.Op {
	case "+", "-", "*", "/", "%":
		if e.Op == "+" && assignable(Str, left) && assignable(Str, right) {
			return Str
		}
		if isNumeric(left) && assignable(left, right) {
			return left
		}
		c.error(e.Pos, "E2401", "operator %s cannot be used with %s and %s", e.Op, left, right)
		return Invalid
	case "==", "!=":
		if assignable(left, right) {
			return Bool
		}
		c.error(e.Pos, "E2402", "comparison requires matching types, got %s and %s", left, right)
		return Invalid
	case "<", "<=", ">", ">=":
		if isNumeric(left) && assignable(left, right) {
			return Bool
		}
		c.error(e.Pos, "E2403", "ordered comparison requires matching numeric types, got %s and %s", left, right)
		return Invalid
	case "&&", "||":
		if assignable(Bool, left) && assignable(Bool, right) {
			return Bool
		}
		c.error(e.Pos, "E2404", "logical operator %s expects bool operands, got %s and %s", e.Op, left, right)
		return Invalid
	}
	return Invalid
}

func (c *Checker) call(e *ast.CallExpr, env map[string]Type) Type {
	// Method call: target.method(args), e.g. instance.with(field= val).
	if fe, ok := e.Fn.(*ast.FieldExpr); ok {
		targetType := c.expr(fe.Target, env)
		if targetType.Kind == KindClass {
			info, isClass := c.classes[targetType.Class]
			if !isClass {
				c.error(e.Pos, "E2501", "unknown class %q", targetType.Class)
				return Invalid
			}
			fe.TargetIsClass = true
			fe.ResolvedClass = targetType.Class
			if fe.Name == "with" {
				return c.checkWithCall(e, targetType, info, env)
			}
		}
		c.error(e.Pos, "E2501", "method calls other than .with() are not yet supported")
		return Invalid
	}

	id, ok := e.Fn.(*ast.Ident)
	if !ok {
		c.error(e.Pos, "E2501", "only direct function calls are supported in v0")
		return Invalid
	}

	// Built-in: print.
	if id.Name == "print" {
		for _, arg := range e.Args {
			c.expr(arg, env)
		}
		return None
	}

	// Class constructor call with named args.
	if info, isClass := c.classes[id.Name]; isClass {
		return c.checkConstructorCall(e, id.Name, info, env)
	}

	// Named-arg call on a non-class is an error.
	if len(e.NamedArgs) > 0 {
		c.error(e.Pos, "E2505", "named arguments are only supported for class constructors")
		return Invalid
	}

	sig, ok := c.fns[id.Name]
	if !ok {
		c.error(id.Pos, "E2502", "unknown function %q", id.Name)
		return Invalid
	}
	if len(e.Args) != len(sig.params) {
		c.error(e.Pos, "E2503", "function %q expects %d args, got %d", id.Name, len(sig.params), len(e.Args))
		return sig.ret
	}
	for i, arg := range e.Args {
		got := c.expr(arg, env)
		if !assignable(sig.params[i], got) {
			c.error(arg.Position(), "E2504", "argument %d to %q has type %s, expected %s", i+1, id.Name, got, sig.params[i])
		}
	}
	return sig.ret
}

func (c *Checker) checkConstructorCall(e *ast.CallExpr, className string, info classInfo, env map[string]Type) Type {
	seen := map[string]bool{}
	for _, na := range e.NamedArgs {
		if seen[na.Name] {
			c.error(na.Pos, "E2507", "field %q is already specified in constructor", na.Name)
			continue
		}
		seen[na.Name] = true
		fieldType, found := c.classFieldType(info, na.Name)
		if !found {
			c.error(na.Pos, "E2505", "class %q has no field %q", className, na.Name)
			continue
		}
		got := c.expr(na.Value, env)
		if !assignable(fieldType, got) {
			c.error(na.Pos, "E2504", "field %q expects %s, got %s", na.Name, fieldType, got)
		}
	}
	for _, f := range info.fields {
		if !seen[f.name] {
			c.error(e.Pos, "E2506", "constructor for %q is missing field %q", className, f.name)
		}
	}
	return classType(className)
}

func (c *Checker) checkWithCall(e *ast.CallExpr, targetType Type, info classInfo, env map[string]Type) Type {
	seen := map[string]bool{}
	for _, na := range e.NamedArgs {
		if seen[na.Name] {
			c.error(na.Pos, "E2507", "field %q is already specified in .with()", na.Name)
			continue
		}
		seen[na.Name] = true
		fieldType, found := c.classFieldType(info, na.Name)
		if !found {
			c.error(na.Pos, "E2505", "class %q has no field %q", targetType, na.Name)
			continue
		}
		got := c.expr(na.Value, env)
		if !assignable(fieldType, got) {
			c.error(na.Pos, "E2504", "field %q expects %s, got %s", na.Name, fieldType, got)
		}
	}
	return targetType
}

func (c *Checker) classFieldType(info classInfo, name string) (Type, bool) {
	for _, f := range info.fields {
		if f.name == name {
			return f.typ, true
		}
	}
	return Invalid, false
}

func (c *Checker) typeExpr(t *ast.TypeExpr) Type {
	if t == nil {
		return Unknown
	}
	switch t.Name {
	case "int":
		return Int
	case "dec":
		return Dec
	case "str":
		return Str
	case "bool":
		return Bool
	case "none":
		return None
	case "obj":
		return Obj
	case "list":
		if len(t.Args) != 1 {
			c.error(t.Pos, "E1102", "type list expects exactly one type argument")
			return Invalid
		}
		return listOf(c.typeExpr(t.Args[0]))
	default:
		if _, isClass := c.classes[t.Name]; isClass {
			return classType(t.Name)
		}
		c.error(t.Pos, "E1101", "unknown type %q", t.Name)
		return Invalid
	}
}

func (c *Checker) error(pos lexer.Position, code, format string, args ...any) {
	c.errs = append(c.errs, Error{Code: code, Pos: pos, Message: fmt.Sprintf(format, args...)})
}

func assignable(want, got Type) bool {
	if want == Unknown || got == Unknown || want == got {
		return true
	}
	wantElem, wantList := listElem(want)
	gotElem, gotList := listElem(got)
	return wantList && gotList && assignable(wantElem, gotElem)
}

func isNumeric(t Type) bool {
	return t == Int || t == Dec || t == Unknown
}

func cloneEnv(in map[string]Type) map[string]Type {
	out := make(map[string]Type, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func listOf(elem Type) Type {
	return Type{Kind: KindList, Elem: &elem}
}

func listElem(t Type) (Type, bool) {
	if t.Kind != KindList || t.Elem == nil {
		return Invalid, false
	}
	return *t.Elem, true
}

func classType(name string) Type {
	return Type{Kind: KindClass, Class: name}
}
