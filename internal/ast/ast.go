// Package ast defines the abstract syntax tree for Lume programs.
package ast

import "github.com/mavboas/lume/internal/lexer"

// Node is any AST node. Position points to the start in the source.
type Node interface {
	Position() lexer.Position
}

// File is the root of one source file.
type File struct {
	Decls []Decl
	Pos   lexer.Position
}

func (f *File) Position() lexer.Position { return f.Pos }

// ---- Declarations ----

type Decl interface {
	Node
	declNode()
}

type FnDecl struct {
	Name   string
	Params []Param
	Return *TypeExpr // nil if no return type declared
	Doc    string    // optional documentation string after signature
	Body   *Block
	Public bool
	Pos    lexer.Position
}

func (f *FnDecl) Position() lexer.Position { return f.Pos }
func (f *FnDecl) declNode()                {}

// ClassDecl is a `cl Name { field: Type ... }` top-level declaration.
type ClassDecl struct {
	Name   string
	Fields []ClassField
	Public bool
	Pos    lexer.Position
}

func (c *ClassDecl) Position() lexer.Position { return c.Pos }
func (c *ClassDecl) declNode()                {}

type ClassField struct {
	Name string
	Type *TypeExpr
	Pos  lexer.Position
}

type Param struct {
	Name string
	Type *TypeExpr
	Pos  lexer.Position
}

// TypeExpr represents a type reference. Args holds generic type arguments,
// such as the `int` in `list[int]`.
type TypeExpr struct {
	Name string
	Args []*TypeExpr
	Pos  lexer.Position
}

func (t *TypeExpr) Position() lexer.Position { return t.Pos }

// ---- Statements ----

type Stmt interface {
	Node
	stmtNode()
}

// LetStmt is a binding: `name = expr`. The language is immutable so a re-binding
// of the same name shadows the previous value within the same scope.
type LetStmt struct {
	Name  string
	Type  *TypeExpr
	Value Expr
	Pos   lexer.Position
}

func (l *LetStmt) Position() lexer.Position { return l.Pos }
func (l *LetStmt) stmtNode()                {}

// ExprStmt wraps an expression used in statement position.
type ExprStmt struct {
	Expr Expr
	Pos  lexer.Position
}

func (e *ExprStmt) Position() lexer.Position { return e.Pos }
func (e *ExprStmt) stmtNode()                {}

// IfStmt is `if(cond){body} elsif(cond){body}* else{body}?`. When the IfStmt
// is the last statement of a function with a return type, the codegen
// propagates the "return the last expression" rule into each branch's body.
type IfStmt struct {
	Cond   Expr
	Body   *Block
	Elsifs []ElsifBranch
	Else   *Block // nil if absent
	Pos    lexer.Position
}

func (i *IfStmt) Position() lexer.Position { return i.Pos }
func (i *IfStmt) stmtNode()                {}

// ElsifBranch is one `elsif(cond){body}` clause.
type ElsifBranch struct {
	Cond Expr
	Body *Block
	Pos  lexer.Position
}

// SwitchStmt is `switch(value){ case(literal){body}* default(){body}? }`.
// Like IfStmt, codegen propagates final-expression returns into each branch
// when the switch is the final statement of a value-returning function.
type SwitchStmt struct {
	Value   Expr
	Cases   []CaseBranch
	Default *Block // nil if absent
	Pos     lexer.Position
}

func (s *SwitchStmt) Position() lexer.Position { return s.Pos }
func (s *SwitchStmt) stmtNode()                {}

// CaseBranch is one `case(literal){body}` clause.
type CaseBranch struct {
	Value Expr
	Body  *Block
	Pos   lexer.Position
}

// ForStmt is `for(Var in target){ body }`. Target is either a list expression
// or a RangeExpr. The loop variable is bound to each element (list) or each
// integer in the half-open interval [From, To) (range).
type ForStmt struct {
	Var    string
	Target Expr // list Expr or *RangeExpr
	Body   *Block
	Pos    lexer.Position
}

func (f *ForStmt) Position() lexer.Position { return f.Pos }
func (f *ForStmt) stmtNode()                {}

// Block is a sequence of statements wrapped in `{ }`. The value of a block is
// the value of its last expression statement.
type Block struct {
	Stmts []Stmt
	Pos   lexer.Position
}

func (b *Block) Position() lexer.Position { return b.Pos }

// ---- Expressions ----

type Expr interface {
	Node
	exprNode()
}

type IntLit struct {
	Value int64
	Pos   lexer.Position
}

func (l *IntLit) Position() lexer.Position { return l.Pos }
func (l *IntLit) exprNode()                {}

type DecLit struct {
	Value float64
	Pos   lexer.Position
}

func (l *DecLit) Position() lexer.Position { return l.Pos }
func (l *DecLit) exprNode()                {}

type StrLit struct {
	Value string
	Pos   lexer.Position
}

func (l *StrLit) Position() lexer.Position { return l.Pos }
func (l *StrLit) exprNode()                {}

// InterpolatedStr is a string with embedded ${expr} parts. The Parts slice
// alternates between literal text (IsExpr=false) and embedded expressions
// (IsExpr=true). The final code generation typically emits fmt.Sprintf.
type InterpolatedStr struct {
	Parts []InterpPart
	Pos   lexer.Position
}

func (i *InterpolatedStr) Position() lexer.Position { return i.Pos }
func (i *InterpolatedStr) exprNode()                {}

// InterpPart is one segment of an interpolated string. Either Lit holds the
// literal text or Expr holds the embedded expression.
type InterpPart struct {
	IsExpr bool
	Lit    string
	Expr   Expr
}

type BoolLit struct {
	Value bool
	Pos   lexer.Position
}

func (l *BoolLit) Position() lexer.Position { return l.Pos }
func (l *BoolLit) exprNode()                {}

type NoneLit struct {
	Pos lexer.Position
}

func (l *NoneLit) Position() lexer.Position { return l.Pos }
func (l *NoneLit) exprNode()                {}

type ListLit struct {
	Elems []Expr
	Pos   lexer.Position
}

func (l *ListLit) Position() lexer.Position { return l.Pos }
func (l *ListLit) exprNode()                {}

type ObjLit struct {
	Fields []ObjField
	Pos    lexer.Position
}

func (o *ObjLit) Position() lexer.Position { return o.Pos }
func (o *ObjLit) exprNode()                {}

type ObjField struct {
	Name  string
	Value Expr
	Pos   lexer.Position
}

type Ident struct {
	Name string
	Pos  lexer.Position
}

func (i *Ident) Position() lexer.Position { return i.Pos }
func (i *Ident) exprNode()                {}

// NamedArg is a `field= value` argument used in class constructor calls and .with().
type NamedArg struct {
	Name  string
	Value Expr
	Pos   lexer.Position
}

// CallExpr is a function call. Exactly one of Args or NamedArgs is non-nil.
// NamedArgs is used for class constructor calls: MyClass(field= val).
type CallExpr struct {
	Fn        Expr
	Args      []Expr
	NamedArgs []NamedArg
	Pos       lexer.Position
}

func (c *CallExpr) Position() lexer.Position { return c.Pos }
func (c *CallExpr) exprNode()                {}

// FieldExpr is a field access `target.name`. TargetIsClass is set by sema
// when the target is a class instance (`target.Field` in Go) vs obj
// (`target["field"]` in Go). ResolvedClass holds the class name for
// codegen to look up field order (e.g., for .with() and .keys).
type FieldExpr struct {
	Target        Expr
	Name          string
	TargetIsClass bool
	ResolvedClass string // set by sema: the class name when TargetIsClass is true
	Pos           lexer.Position
}

func (f *FieldExpr) Position() lexer.Position { return f.Pos }
func (f *FieldExpr) exprNode()                {}

type BinaryExpr struct {
	Op    string
	Left  Expr
	Right Expr
	Pos   lexer.Position
}

func (b *BinaryExpr) Position() lexer.Position { return b.Pos }
func (b *BinaryExpr) exprNode()                {}

// RangeExpr is a half-open integer range `From..To`, valid as a for-loop target.
type RangeExpr struct {
	From Expr
	To   Expr
	Pos  lexer.Position
}

func (r *RangeExpr) Position() lexer.Position { return r.Pos }
func (r *RangeExpr) exprNode()                {}

type UnaryExpr struct {
	Op  string
	Sub Expr
	Pos lexer.Position
}

func (u *UnaryExpr) Position() lexer.Position { return u.Pos }
func (u *UnaryExpr) exprNode()                {}

// LetExpr is an expression-local binding form:
//
//	let(x = expr, y = x + 1){ y }
//
// Bindings are sequential and visible only in Body. ResolvedType is set by
// semantic analysis for Go IIFE code generation.
type LetExpr struct {
	Bindings     []LetBinding
	Body         *Block
	ResolvedType string
	Pos          lexer.Position
}

func (l *LetExpr) Position() lexer.Position { return l.Pos }
func (l *LetExpr) exprNode()                {}

type LetBinding struct {
	Name  string
	Value Expr
	Pos   lexer.Position
}

// MatchExpr is a pattern-oriented expression/statement form over literals,
// wildcard, and identifier capture patterns.
type MatchExpr struct {
	Value        Expr
	Cases        []MatchCase
	ResolvedType string
	Pos          lexer.Position
}

func (m *MatchExpr) Position() lexer.Position { return m.Pos }
func (m *MatchExpr) exprNode()                {}

type MatchCase struct {
	Pattern Pattern
	Body    *Block
	Pos     lexer.Position
}

type Pattern interface {
	Node
	patternNode()
}

type LiteralPattern struct {
	Value Expr
	Pos   lexer.Position
}

func (p *LiteralPattern) Position() lexer.Position { return p.Pos }
func (p *LiteralPattern) patternNode()             {}

type WildcardPattern struct {
	Pos lexer.Position
}

func (p *WildcardPattern) Position() lexer.Position { return p.Pos }
func (p *WildcardPattern) patternNode()             {}

type CapturePattern struct {
	Name string
	Pos  lexer.Position
}

func (p *CapturePattern) Position() lexer.Position { return p.Pos }
func (p *CapturePattern) patternNode()             {}
