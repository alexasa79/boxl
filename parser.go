package boxl

import "fmt"

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
	diagram, err := p.parseDiagram(TokenEOF)
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

func (p *Parser) parseDiagram(end TokenType) (*Diagram, error) {
	diagram := &Diagram{}
	if err := p.skipNewlines(); err != nil {
		return nil, err
	}
	for p.current.Type != end {
		if p.current.Type == TokenEOF {
			if end == TokenRightBracket {
				return nil, p.errorf(p.current, "expected ] before end of input")
			}
			break
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
		if p.current.Type != TokenNewline {
			return nil, p.errorf(p.current, "expected newline or %s, got %s", end, p.current.Type)
		}
		if err := p.skipNewlines(); err != nil {
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
	for p.current.Type == TokenArrow {
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

func (p *Parser) parseBox() (BoxExpression, error) {
	box := BoxExpression{Identifier: p.current.Value, Position: p.current.Position}
	if err := p.advance(); err != nil {
		return BoxExpression{}, err
	}
	if p.current.Type != TokenLeftBracket {
		return box, nil
	}
	if err := p.advance(); err != nil {
		return BoxExpression{}, err
	}
	if p.current.Type == TokenString {
		text := p.current.Value
		box.Text = &text
		if err := p.advance(); err != nil {
			return BoxExpression{}, err
		}
		if p.current.Type != TokenRightBracket {
			return BoxExpression{}, p.errorf(p.current, "expected ] after quoted text, got %s", p.current.Type)
		}
		return box, p.advance()
	}

	diagram, err := p.parseDiagram(TokenRightBracket)
	if err != nil {
		return BoxExpression{}, err
	}
	box.Diagram = diagram
	return box, p.advance()
}

func (p *Parser) skipNewlines() error {
	for p.current.Type == TokenNewline {
		if err := p.advance(); err != nil {
			return err
		}
	}
	return nil
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
