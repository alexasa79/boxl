package boxl

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType identifies a lexical token.
type TokenType int

const (
	TokenIdentifier TokenType = iota
	TokenString
	TokenArrow
	TokenLeftBracket
	TokenRightBracket
	TokenDirective
	TokenNewline
	TokenEOF
)

func (t TokenType) String() string {
	switch t {
	case TokenIdentifier:
		return "identifier"
	case TokenString:
		return "quoted text"
	case TokenArrow:
		return "->"
	case TokenLeftBracket:
		return "["
	case TokenRightBracket:
		return "]"
	case TokenDirective:
		return "directive"
	case TokenNewline:
		return "newline"
	case TokenEOF:
		return "end of input"
	default:
		return "unknown token"
	}
}

// Token is one item produced by Tokenizer. Lexeme contains the source spelling;
// Value contains decoded quoted text, an identifier name, or directive text
// without its leading #.
type Token struct {
	Type     TokenType
	Lexeme   string
	Value    string
	Position Position
}

// Tokenizer converts box language source into tokens. Newlines delimit statements.
type Tokenizer struct {
	input  string
	offset int
	line   int
	column int
}

// NewTokenizer creates a tokenizer for source.
func NewTokenizer(source string) *Tokenizer {
	return &Tokenizer{input: source, line: 1, column: 1}
}

// Tokenize consumes all tokens, including the final EOF token.
func Tokenize(source string) ([]Token, error) {
	tokenizer := NewTokenizer(source)
	var tokens []Token
	for {
		token, err := tokenizer.Next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
		if token.Type == TokenEOF {
			return tokens, nil
		}
	}
}

// Next returns the next token.
func (t *Tokenizer) Next() (Token, error) {
	for t.offset < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.offset:])
		if r != ' ' && r != '\t' && r != '\f' && r != '\v' {
			break
		}
		t.advanceRune(r, size)
	}

	start := t.position()
	if t.offset >= len(t.input) {
		return Token{Type: TokenEOF, Position: start}, nil
	}

	if t.input[t.offset] == '\r' || t.input[t.offset] == '\n' {
		startOffset := t.offset
		if t.input[t.offset] == '\r' {
			t.offset++
			if t.offset < len(t.input) && t.input[t.offset] == '\n' {
				t.offset++
			}
		} else {
			t.offset++
		}
		t.line++
		t.column = 1
		return Token{Type: TokenNewline, Lexeme: t.input[startOffset:t.offset], Position: start}, nil
	}

	switch t.input[t.offset] {
	case '#':
		return t.scanDirective(), nil
	case '[':
		t.offset++
		t.column++
		return Token{Type: TokenLeftBracket, Lexeme: "[", Position: start}, nil
	case ']':
		t.offset++
		t.column++
		return Token{Type: TokenRightBracket, Lexeme: "]", Position: start}, nil
	case '-':
		if t.offset+1 < len(t.input) && t.input[t.offset+1] == '>' {
			t.offset += 2
			t.column += 2
			return Token{Type: TokenArrow, Lexeme: "->", Position: start}, nil
		}
	case '"', '\'':
		return t.scanString(rune(t.input[t.offset]))
	}

	r, _ := utf8.DecodeRuneInString(t.input[t.offset:])
	if isIdentifierStart(r) {
		return t.scanIdentifier(), nil
	}
	return Token{}, lexicalError(start, "unexpected character %q", r)
}

func (t *Tokenizer) scanDirective() Token {
	start := t.position()
	startOffset := t.offset
	parentheses := 0
	for t.offset < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.offset:])
		if r == '\r' || r == '\n' || r == '[' || r == ']' || (unicode.IsSpace(r) && parentheses == 0) {
			break
		}
		if r == '(' {
			parentheses++
		} else if r == ')' && parentheses > 0 {
			parentheses--
		}
		t.advanceRune(r, size)
	}
	lexeme := t.input[startOffset:t.offset]
	return Token{Type: TokenDirective, Lexeme: lexeme, Value: lexeme[1:], Position: start}
}

func (t *Tokenizer) scanIdentifier() Token {
	start := t.position()
	startOffset := t.offset
	for t.offset < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.offset:])
		if r == '-' && t.offset+1 < len(t.input) && t.input[t.offset+1] == '>' {
			break
		}
		if !isIdentifierContinue(r) {
			break
		}
		t.advanceRune(r, size)
	}
	value := t.input[startOffset:t.offset]
	return Token{Type: TokenIdentifier, Lexeme: value, Value: value, Position: start}
}

func (t *Tokenizer) scanString(quote rune) (Token, error) {
	start := t.position()
	startOffset := t.offset
	t.offset++
	t.column++
	escaped := false
	for t.offset < len(t.input) {
		r, size := utf8.DecodeRuneInString(t.input[t.offset:])
		if r == '\n' || r == '\r' {
			t.advanceNewline()
			escaped = false
			continue
		}
		t.advanceRune(r, size)
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == quote {
			lexeme := t.input[startOffset:t.offset]
			value, err := decodeQuotedText(lexeme[1:len(lexeme)-1], quote)
			if err != nil {
				return Token{}, lexicalError(start, "invalid quoted text: %v", err)
			}
			return Token{Type: TokenString, Lexeme: lexeme, Value: value, Position: start}, nil
		}
	}
	return Token{}, lexicalError(start, "unterminated quoted text")
}

func (t *Tokenizer) advanceNewline() {
	if t.input[t.offset] == '\r' {
		t.offset++
		if t.offset < len(t.input) && t.input[t.offset] == '\n' {
			t.offset++
		}
	} else {
		t.offset++
	}
	t.line++
	t.column = 1
}

func decodeQuotedText(content string, quote rune) (string, error) {
	var decoded strings.Builder
	for content != "" {
		if content[0] == '\r' || content[0] == '\n' {
			if content[0] == '\r' && len(content) > 1 && content[1] == '\n' {
				content = content[2:]
			} else {
				content = content[1:]
			}
			decoded.WriteByte('\n')
			continue
		}
		value, _, remainder, err := strconv.UnquoteChar(content, byte(quote))
		if err != nil {
			return "", err
		}
		decoded.WriteRune(value)
		content = remainder
	}
	return decoded.String(), nil
}

func (t *Tokenizer) advanceRune(r rune, size int) {
	t.offset += size
	t.column++
}

func (t *Tokenizer) position() Position {
	return Position{Offset: t.offset, Line: t.line, Column: t.column}
}

func isIdentifierStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentifierContinue(r rune) bool {
	return isIdentifierStart(r) || unicode.IsDigit(r) || r == '-' || r == '.'
}

func lexicalError(position Position, format string, args ...any) error {
	return fmt.Errorf("line %d, column %d: %s", position.Line, position.Column, fmt.Sprintf(format, args...))
}
