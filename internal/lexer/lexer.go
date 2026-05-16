package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

// Lexer turns source text into tokens.
type Lexer struct {
	src    []rune
	pos    int
	line   int
	col    int
	tokens []Token
	errs   []error
}

// New creates a Lexer for the given source.
func New(source string) *Lexer {
	return &Lexer{
		src:  []rune(source),
		pos:  0,
		line: 1,
		col:  1,
	}
}

// Tokenize consumes the entire source and returns the token stream.
// On lex errors, it still returns whatever it managed to scan plus the errors.
func (l *Lexer) Tokenize() ([]Token, []error) {
	for l.pos < len(l.src) {
		l.skipWhitespaceAndComments()
		if l.pos >= len(l.src) {
			break
		}
		l.scanToken()
	}
	l.emit(EOF, "", l.position())
	return l.tokens, l.errs
}

// peek returns the rune at offset n from current position, or 0 if out of bounds.
func (l *Lexer) peek(n int) rune {
	if l.pos+n >= len(l.src) {
		return 0
	}
	return l.src[l.pos+n]
}

func (l *Lexer) cur() rune { return l.peek(0) }

// advance moves forward one rune, tracking line/col.
func (l *Lexer) advance() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r := l.src[l.pos]
	l.pos++
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

func (l *Lexer) position() Position {
	return Position{Line: l.line, Col: l.col}
}

func (l *Lexer) emit(k Kind, val string, pos Position) {
	l.tokens = append(l.tokens, Token{Kind: k, Value: val, Pos: pos})
}

func (l *Lexer) errf(pos Position, format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf("lex error at %s: %s", pos, fmt.Sprintf(format, args...)))
}

// skipWhitespaceAndComments advances past spaces, tabs, newlines, // line comments
// and /* block comments */ (not nested).
func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		r := l.cur()
		switch {
		case unicode.IsSpace(r):
			l.advance()
		case r == '/' && l.peek(1) == '/':
			for l.pos < len(l.src) && l.cur() != '\n' {
				l.advance()
			}
		case r == '/' && l.peek(1) == '*':
			start := l.position()
			l.advance() // /
			l.advance() // *
			closed := false
			for l.pos < len(l.src) {
				if l.cur() == '*' && l.peek(1) == '/' {
					l.advance()
					l.advance()
					closed = true
					break
				}
				l.advance()
			}
			if !closed {
				l.errf(start, "unterminated block comment")
			}
		default:
			return
		}
	}
}

// scanToken reads one token and appends it.
func (l *Lexer) scanToken() {
	pos := l.position()
	r := l.cur()

	switch {
	case isIdentStart(r):
		l.scanIdent(pos)
	case unicode.IsDigit(r):
		l.scanNumber(pos)
	case r == '"':
		l.scanString(pos)
	default:
		l.scanPunct(pos)
	}
}

func isIdentStart(r rune) bool { return unicode.IsLetter(r) || r == '_' }
func isIdentPart(r rune) bool  { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }

func (l *Lexer) scanIdent(pos Position) {
	var sb strings.Builder
	for l.pos < len(l.src) && isIdentPart(l.cur()) {
		sb.WriteRune(l.advance())
	}
	val := sb.String()
	l.emit(LookupKeyword(val), val, pos)
}

func (l *Lexer) scanNumber(pos Position) {
	var sb strings.Builder
	for l.pos < len(l.src) && unicode.IsDigit(l.cur()) {
		sb.WriteRune(l.advance())
	}
	// decimal? `..` is range, so only treat as decimal when next-after-dot is a digit
	if l.cur() == '.' && unicode.IsDigit(l.peek(1)) {
		sb.WriteRune(l.advance()) // consume .
		for l.pos < len(l.src) && unicode.IsDigit(l.cur()) {
			sb.WriteRune(l.advance())
		}
		l.emit(Dec, sb.String(), pos)
		return
	}
	l.emit(Int, sb.String(), pos)
}

// scanString picks between a plain string (single Str token) and an
// interpolated string (multi-token sequence) based on whether ${ appears
// before the closing ".
func (l *Lexer) scanString(pos Position) {
	if l.stringHasInterpolation() {
		l.scanInterpolatedString(pos)
		return
	}
	l.scanPlainString(pos)
}

// stringHasInterpolation does a read-only scan from the current " to find
// either a closing " (no interp) or a ${ (has interp).
func (l *Lexer) stringHasInterpolation() bool {
	for i := l.pos + 1; i < len(l.src); i++ {
		r := l.src[i]
		if r == '\\' {
			i++ // skip the next char
			continue
		}
		if r == '"' {
			return false
		}
		if r == '$' && i+1 < len(l.src) && l.src[i+1] == '{' {
			return true
		}
	}
	return false
}

// readEscape handles \n \t \\ \" \r \0 from the current position (cursor on the
// backslash). It writes the decoded rune to sb and advances past the escape.
func (l *Lexer) readEscape(sb *strings.Builder) {
	l.advance() // backslash
	switch l.cur() {
	case 'n':
		sb.WriteRune('\n')
	case 't':
		sb.WriteRune('\t')
	case 'r':
		sb.WriteRune('\r')
	case '\\':
		sb.WriteRune('\\')
	case '"':
		sb.WriteRune('"')
	case '0':
		sb.WriteRune(0)
	default:
		l.errf(l.position(), "unknown escape \\%c", l.cur())
		sb.WriteRune(l.cur())
	}
	l.advance()
}

func (l *Lexer) scanPlainString(pos Position) {
	l.advance() // opening "
	var sb strings.Builder
	for l.pos < len(l.src) && l.cur() != '"' {
		if l.cur() == '\\' {
			l.readEscape(&sb)
			continue
		}
		if l.cur() == '\n' {
			l.errf(pos, "newline inside string literal")
			break
		}
		sb.WriteRune(l.advance())
	}
	if l.cur() != '"' {
		l.errf(pos, "unterminated string literal")
		return
	}
	l.advance() // closing "
	l.emit(Str, sb.String(), pos)
}

// scanInterpolatedString emits the multi-token form for strings containing
// ${expr}. Inside an interpolation we tokenize normally, tracking brace depth
// so a literal { in the embedded expression doesn't terminate it.
func (l *Lexer) scanInterpolatedString(pos Position) {
	l.advance() // opening "
	l.emit(StrStart, "", pos)

	var lit strings.Builder
	flushChunk := func() {
		if lit.Len() == 0 {
			return
		}
		l.emit(StrChunk, lit.String(), pos)
		lit.Reset()
	}

	for l.pos < len(l.src) && l.cur() != '"' {
		if l.cur() == '\\' {
			l.readEscape(&lit)
			continue
		}
		if l.cur() == '\n' {
			l.errf(pos, "newline inside interpolated string")
			break
		}
		if l.cur() == '$' && l.peek(1) == '{' {
			flushChunk()
			interpPos := l.position()
			l.advance() // $
			l.advance() // {
			l.emit(StrInterpStart, "${", interpPos)
			l.scanInterpolationBody()
			continue
		}
		lit.WriteRune(l.advance())
	}
	flushChunk()

	if l.cur() != '"' {
		l.errf(pos, "unterminated interpolated string")
		return
	}
	endPos := l.position()
	l.advance() // closing "
	l.emit(StrEnd, "", endPos)
}

// scanInterpolationBody tokenizes the contents of `${...}`. Stops when it
// sees the matching closing `}` (depth-aware, so `${ {x: 1}.x }` works).
// Emits StrInterpEnd for the closing `}`.
func (l *Lexer) scanInterpolationBody() {
	depth := 1
	for depth > 0 && l.pos < len(l.src) {
		l.skipWhitespaceAndComments()
		if l.pos >= len(l.src) {
			break
		}
		r := l.cur()
		if r == '}' {
			if depth == 1 {
				endPos := l.position()
				l.advance()
				l.emit(StrInterpEnd, "}", endPos)
				return
			}
			depth--
			l.scanToken() // emits RBrace
			continue
		}
		if r == '{' {
			depth++
		}
		l.scanToken()
	}
	l.errf(l.position(), "unterminated ${...} interpolation")
}

// scanPunct handles single- and multi-char punctuation/operators.
func (l *Lexer) scanPunct(pos Position) {
	r := l.advance()
	switch r {
	case '(':
		l.emit(LParen, "(", pos)
	case ')':
		l.emit(RParen, ")", pos)
	case '{':
		l.emit(LBrace, "{", pos)
	case '}':
		l.emit(RBrace, "}", pos)
	case '[':
		l.emit(LBracket, "[", pos)
	case ']':
		l.emit(RBracket, "]", pos)
	case ',':
		l.emit(Comma, ",", pos)
	case ':':
		l.emit(Colon, ":", pos)
	case ';':
		l.emit(Semi, ";", pos)
	case '?':
		l.emit(Quest, "?", pos)
	case '+':
		l.emit(Plus, "+", pos)
	case '*':
		l.emit(Star, "*", pos)
	case '/':
		l.emit(Slash, "/", pos)
	case '%':
		l.emit(Percent, "%", pos)
	case '.':
		if l.cur() == '.' && l.peek(1) == '.' {
			l.advance()
			l.advance()
			l.emit(Spread, "...", pos)
		} else if l.cur() == '.' {
			l.advance()
			l.emit(Range, "..", pos)
		} else {
			l.emit(Dot, ".", pos)
		}
	case '-':
		if l.cur() == '>' {
			l.advance()
			l.emit(Arrow, "->", pos)
		} else {
			l.emit(Minus, "-", pos)
		}
	case '=':
		if l.cur() == '=' {
			l.advance()
			l.emit(Eq, "==", pos)
		} else {
			l.emit(Assign, "=", pos)
		}
	case '!':
		if l.cur() == '=' {
			l.advance()
			l.emit(Neq, "!=", pos)
		} else {
			l.emit(Bang, "!", pos)
		}
	case '<':
		if l.cur() == '=' {
			l.advance()
			l.emit(Lte, "<=", pos)
		} else {
			l.emit(Lt, "<", pos)
		}
	case '>':
		if l.cur() == '=' {
			l.advance()
			l.emit(Gte, ">=", pos)
		} else {
			l.emit(Gt, ">", pos)
		}
	case '&':
		if l.cur() == '&' {
			l.advance()
			l.emit(AndAnd, "&&", pos)
		} else {
			l.errf(pos, "expected && (got single &)")
			l.emit(Illegal, "&", pos)
		}
	case '|':
		if l.cur() == '|' {
			l.advance()
			l.emit(OrOr, "||", pos)
		} else if l.cur() == '>' {
			l.advance()
			l.emit(Pipe, "|>", pos)
		} else {
			l.errf(pos, "expected || or |> (got single |)")
			l.emit(Illegal, "|", pos)
		}
	default:
		l.errf(pos, "unexpected character %q", r)
		l.emit(Illegal, string(r), pos)
	}
}
