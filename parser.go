package boxl

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser turns tokenizer output into a Diagram.
type Parser struct {
	tokenizer *Tokenizer
	current   Token
	started   bool
}

// NewParser creates a parser for source.
func NewParser(source string) *Parser {
	return &Parser{tokenizer: NewTokenizer(source)}
}

// Parse parses a complete source string.
func Parse(source string) (*Diagram, error) {
	return NewParser(source).Parse()
}

// Parse parses all input owned by p.
func (p *Parser) Parse() (*Diagram, error) {
	if err := p.start(); err != nil {
		return nil, err
	}
	diagram, err := p.parseDiagram(TokenEOF, nil)
	if err != nil {
		return nil, err
	}
	return diagram, nil
}

func (p *Parser) start() error {
	if p.started {
		return nil
	}
	p.started = true
	return p.advance()
}

func (p *Parser) parseDiagram(end TokenType, text **string) (*Diagram, error) {
	diagram := &Diagram{}
	if _, err := p.consumeNewlines(); err != nil {
		return nil, err
	}
	for p.current.Type != end {
		if p.current.Type == TokenEOF {
			if end == TokenRightBracket {
				return nil, p.errorf(p.current, "expected ] before end of input")
			}
			break
		}
		if p.current.Type == TokenDirective {
			if err := p.parseDirective(diagram); err != nil {
				return nil, err
			}
			if _, err := p.consumeNewlines(); err != nil {
				return nil, err
			}
			continue
		}
		if p.current.Type == TokenString && text != nil {
			if *text != nil || len(diagram.Statements) != 0 {
				return nil, p.errorf(p.current, "a box may contain one quoted label or nested boxes")
			}
			label := p.current.Value
			*text = &label
			if err := p.advance(); err != nil {
				return nil, err
			}
			if _, err := p.consumeNewlines(); err != nil {
				return nil, err
			}
			continue
		}
		if text != nil && *text != nil {
			return nil, p.errorf(p.current, "a box cannot have both quoted text and nested boxes")
		}
		statement, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		diagram.Statements = append(diagram.Statements, statement)
		if p.current.Type == end {
			break
		}
		if p.current.Type == TokenEOF && end == TokenRightBracket {
			return nil, p.errorf(p.current, "expected ] before end of input")
		}
		if p.current.Type == TokenDirective || p.current.Type == TokenIdentifier || (p.current.Type == TokenString && text != nil) {
			continue
		}
		if p.current.Type != TokenNewline {
			return nil, p.errorf(p.current, "expected box, directive, newline or %s, got %s", end, p.current.Type)
		}
		if _, err := p.consumeNewlines(); err != nil {
			return nil, err
		}
	}
	return diagram, nil
}

func (p *Parser) parseStatement() (Statement, error) {
	if p.current.Type != TokenIdentifier {
		return Statement{}, p.errorf(p.current, "expected identifier, got %s", p.current.Type)
	}
	statement := Statement{Position: p.current.Position}
	box, err := p.parseBox()
	if err != nil {
		return Statement{}, err
	}
	statement.Boxes = append(statement.Boxes, box)
	if p.current.Type == TokenArrow {
		statement.Connected = true
	}
	for statement.Connected && p.current.Type == TokenArrow {
		if err := p.advance(); err != nil {
			return Statement{}, err
		}
		if p.current.Type != TokenIdentifier {
			return Statement{}, p.errorf(p.current, "expected identifier after ->, got %s", p.current.Type)
		}
		box, err := p.parseBox()
		if err != nil {
			return Statement{}, err
		}
		statement.Boxes = append(statement.Boxes, box)
	}
	return statement, nil
}

func (p *Parser) parseLayoutDirective() (*LayoutDirective, error) {
	token := p.current
	parts := strings.Split(token.Value, ":")
	switch parts[0] {
	case "dir", "direction", "layout":
	default:
		return nil, p.errorf(token, "unknown directive %q", parts[0])
	}
	if len(parts) < 2 {
		return nil, p.errorf(token, "expected layout direction after #%s:", parts[0])
	}
	layout := &LayoutDirective{Position: token.Position}
	switch parts[1] {
	case "row":
		layout.Direction = LayoutRow
	case "col", "column":
		layout.Direction = LayoutColumn
	case "grid":
		layout.Direction = LayoutGrid
	case "diag", "diagonal":
		layout.Direction = LayoutDiagonal
	default:
		return nil, p.errorf(token, "unknown layout direction %q; expected row, col, column, grid, diag, or diagonal", parts[1])
	}
	if layout.Direction == LayoutGrid {
		if len(parts) != 3 {
			return nil, p.errorf(token, "grid requires dimensions as rows x columns, for example #dir:grid:3x3")
		}
		rows, columns, found := strings.Cut(parts[2], "x")
		layout.Rows = positiveDimension(rows)
		layout.Columns = positiveDimension(columns)
		if !found || layout.Rows == 0 || layout.Columns == 0 {
			return nil, p.errorf(token, "grid dimensions must be positive integers in rows x columns form")
		}
	} else if len(parts) != 2 {
		return nil, p.errorf(token, "%s layout does not accept dimensions", parts[1])
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	return layout, nil
}

func positiveDimension(value string) int {
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0
		}
	}
	dimension, err := strconv.Atoi(value)
	if err != nil || dimension <= 0 {
		return 0
	}
	return dimension
}

func (p *Parser) parseBox() (BoxExpression, error) {
	box := BoxExpression{Identifier: p.current.Value, Position: p.current.Position}
	if err := p.advance(); err != nil {
		return BoxExpression{}, err
	}
	if p.current.Type == TokenLeftBracket {
		if err := p.advance(); err != nil {
			return BoxExpression{}, err
		}
		diagram, err := p.parseDiagram(TokenRightBracket, &box.Text)
		if err != nil {
			return BoxExpression{}, err
		}
		box.Style = diagram.Style
		diagram.Style = BoxStyle{}
		if box.Text == nil {
			box.Diagram = diagram
		} else if diagram.Layout != nil {
			return BoxExpression{}, modelError(diagram.Layout.Position, "layout directives require nested boxes")
		}
		if err := p.advance(); err != nil {
			return BoxExpression{}, err
		}
	}

	return box, nil
}

func (p *Parser) consumeNewlines() (int, error) {
	count := 0
	for p.current.Type == TokenNewline {
		count++
		if err := p.advance(); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (p *Parser) advance() error {
	token, err := p.tokenizer.Next()
	if err != nil {
		return err
	}
	p.current = token
	return nil
}

func (p *Parser) errorf(token Token, format string, args ...any) error {
	return fmt.Errorf("line %d, column %d: %s", token.Position.Line, token.Position.Column, fmt.Sprintf(format, args...))
}
