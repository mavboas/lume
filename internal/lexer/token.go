// Package lexer turns Lume source code into a stream of tokens.
package lexer

import "fmt"

// Kind identifies the category of a token.
type Kind int

const (
	EOF Kind = iota
	Illegal

	// Literals & identifiers
	Ident
	Int
	Dec
	Str // simple string literal "abc"

	// Interpolated string fragments. Emitted only when a string contains ${...}.
	// Sequence is: StrStart, (StrChunk | StrInterpStart ... StrInterpEnd)*, StrEnd
	StrStart       // opening " of an interpolated string
	StrChunk       // literal text piece between interpolations
	StrInterpStart // ${
	StrInterpEnd   // }  closing an interpolation
	StrEnd         // closing "

	// Keywords
	KwFn
	KwLet
	KwCl
	KwIf
	KwElsif
	KwElse
	KwSwitch
	KwCase
	KwDefault
	KwFor
	KwIn
	KwMatch
	KwTrue
	KwFalse
	KwNone
	KwPub
	KwImport
	KwAs
	KwType
	KwSpec
	KwPre
	KwPos
	KwInvariant
	KwErro
	KwTry
	KwCatch
	KwFinally
	KwWhere
	KwAnd
	KwOr
	KwNot

	// Punctuation
	LParen   // (
	RParen   // )
	LBrace   // {
	RBrace   // }
	LBracket // [
	RBracket // ]
	Comma    // ,
	Colon    // :
	Semi     // ;
	Dot      // .
	Arrow    // ->
	Range    // ..
	Spread   // ...

	// Operators
	Assign  // =
	Eq      // ==
	Neq     // !=
	Lt      // <
	Lte     // <=
	Gt      // >
	Gte     // >=
	Plus    // +
	Minus   // -
	Star    // *
	Slash   // /
	Percent // %
	AndAnd  // &&
	OrOr    // ||
	Bang    // !
	Pipe    // |>
	Quest   // ?
)

// Position is a 1-indexed source location.
type Position struct {
	Line int
	Col  int
}

func (p Position) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// Token is a lexed unit of source code.
type Token struct {
	Kind  Kind
	Value string
	Pos   Position
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q)@%s", kindName(t.Kind), t.Value, t.Pos)
}

func kindName(k Kind) string {
	switch k {
	case EOF:
		return "EOF"
	case Illegal:
		return "ILLEGAL"
	case Ident:
		return "IDENT"
	case Int:
		return "INT"
	case Dec:
		return "DEC"
	case Str:
		return "STR"
	case StrStart:
		return "STR_START"
	case StrChunk:
		return "STR_CHUNK"
	case StrInterpStart:
		return "STR_INTERP_START"
	case StrInterpEnd:
		return "STR_INTERP_END"
	case StrEnd:
		return "STR_END"
	case KwFn:
		return "FN"
	case KwLet:
		return "LET"
	case KwCl:
		return "CL"
	case KwIf:
		return "IF"
	case KwElsif:
		return "ELSIF"
	case KwElse:
		return "ELSE"
	case KwSwitch:
		return "SWITCH"
	case KwCase:
		return "CASE"
	case KwDefault:
		return "DEFAULT"
	case KwFor:
		return "FOR"
	case KwIn:
		return "IN"
	case KwMatch:
		return "MATCH"
	case KwTrue:
		return "TRUE"
	case KwFalse:
		return "FALSE"
	case KwNone:
		return "NONE"
	case KwPub:
		return "PUB"
	case KwImport:
		return "IMPORT"
	case KwAs:
		return "AS"
	case KwType:
		return "TYPE"
	case KwSpec:
		return "SPEC"
	case KwPre:
		return "PRE"
	case KwPos:
		return "POS"
	case KwInvariant:
		return "INVARIANT"
	case KwErro:
		return "ERRO"
	case KwTry:
		return "TRY"
	case KwCatch:
		return "CATCH"
	case KwFinally:
		return "FINALLY"
	case KwWhere:
		return "WHERE"
	case KwAnd:
		return "AND"
	case KwOr:
		return "OR"
	case KwNot:
		return "NOT"
	case LParen:
		return "LPAREN"
	case RParen:
		return "RPAREN"
	case LBrace:
		return "LBRACE"
	case RBrace:
		return "RBRACE"
	case LBracket:
		return "LBRACKET"
	case RBracket:
		return "RBRACKET"
	case Comma:
		return "COMMA"
	case Colon:
		return "COLON"
	case Semi:
		return "SEMI"
	case Dot:
		return "DOT"
	case Arrow:
		return "ARROW"
	case Range:
		return "RANGE"
	case Spread:
		return "SPREAD"
	case Assign:
		return "ASSIGN"
	case Eq:
		return "EQ"
	case Neq:
		return "NEQ"
	case Lt:
		return "LT"
	case Lte:
		return "LTE"
	case Gt:
		return "GT"
	case Gte:
		return "GTE"
	case Plus:
		return "PLUS"
	case Minus:
		return "MINUS"
	case Star:
		return "STAR"
	case Slash:
		return "SLASH"
	case Percent:
		return "PERCENT"
	case AndAnd:
		return "AND_AND"
	case OrOr:
		return "OR_OR"
	case Bang:
		return "BANG"
	case Pipe:
		return "PIPE"
	case Quest:
		return "QUEST"
	}
	return "UNKNOWN"
}

// keywords maps source text to its keyword Kind.
var keywords = map[string]Kind{
	"fn":        KwFn,
	"let":       KwLet,
	"cl":        KwCl,
	"if":        KwIf,
	"elsif":     KwElsif,
	"else":      KwElse,
	"switch":    KwSwitch,
	"case":      KwCase,
	"default":   KwDefault,
	"for":       KwFor,
	"in":        KwIn,
	"match":     KwMatch,
	"true":      KwTrue,
	"false":     KwFalse,
	"none":      KwNone,
	"pub":       KwPub,
	"import":    KwImport,
	"as":        KwAs,
	"type":      KwType,
	"spec":      KwSpec,
	"pre":       KwPre,
	"pos":       KwPos,
	"invariant": KwInvariant,
	"erro":      KwErro,
	"try":       KwTry,
	"catch":     KwCatch,
	"finally":   KwFinally,
	"where":     KwWhere,
	"and":       KwAnd,
	"or":        KwOr,
	"not":       KwNot,
}

// LookupKeyword returns the Kind for an identifier if it matches a keyword,
// or Ident if it is just a regular identifier.
func LookupKeyword(s string) Kind {
	if k, ok := keywords[s]; ok {
		return k
	}
	return Ident
}
