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
	if _, err := p.consumeNewlines(); err != nil {
		return nil, err
	}
	var layoutRows []int
	flushLayout := func() {
		addVerticalLayoutConstraints(diagram, layoutRows)
		layoutRows = nil
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
		if statement.Connected {
			flushLayout()
		} else {
			addHorizontalLayoutConstraints(&statement)
		}
		diagram.Statements = append(diagram.Statements, statement)
		if !statement.Connected {
			layoutRows = append(layoutRows, len(diagram.Statements)-1)
		}
		if p.current.Type == end {
			break
		}
		if p.current.Type == TokenEOF && end == TokenRightBracket {
			return nil, p.errorf(p.current, "expected ] before end of input")
		}
		if p.current.Type != TokenNewline {
			return nil, p.errorf(p.current, "expected newline or %s, got %s", end, p.current.Type)
		}
		newlines, err := p.consumeNewlines()
		if err != nil {
			return nil, err
		}
		if newlines > 1 {
			flushLayout()
		}
	}
	flushLayout()
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
	for !statement.Connected && p.current.Type == TokenIdentifier {
		box, err := p.parseBox()
		if err != nil {
			return Statement{}, err
		}
		statement.Boxes = append(statement.Boxes, box)
	}
	return statement, nil
}

func addHorizontalLayoutConstraints(statement *Statement) {
	for i := 1; i < len(statement.Boxes); i++ {
		previous := &statement.Boxes[i-1]
		if previous.Identifier == statement.Boxes[i].Identifier {
			continue
		}
		previous.Constraints = append(previous.Constraints, LayoutConstraint{
			Relation: ConstraintThen,
			Target:   statement.Boxes[i].Identifier,
			Position: previous.Position,
		})
	}
}

func addVerticalLayoutConstraints(diagram *Diagram, rows []int) {
	for i := 1; i < len(rows); i++ {
		above := &diagram.Statements[rows[i-1]]
		below := &diagram.Statements[rows[i]]
		columns := min(len(above.Boxes), len(below.Boxes))
		for column := 0; column < columns; column++ {
			box := &above.Boxes[column]
			if box.Identifier == below.Boxes[column].Identifier {
				continue
			}
			box.Constraints = append(box.Constraints, LayoutConstraint{
				Relation: ConstraintAbove,
				Target:   below.Boxes[column].Identifier,
				Position: box.Position,
			})
		}
	}
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
		if p.current.Type == TokenString {
			text := p.current.Value
			box.Text = &text
			if err := p.advance(); err != nil {
				return BoxExpression{}, err
			}
			if p.current.Type != TokenRightBracket {
				return BoxExpression{}, p.errorf(p.current, "expected ] after quoted text, got %s", p.current.Type)
			}
			if err := p.advance(); err != nil {
				return BoxExpression{}, err
			}
		} else {
			diagram, err := p.parseDiagram(TokenRightBracket)
			if err != nil {
				return BoxExpression{}, err
			}
			box.Diagram = diagram
			if err := p.advance(); err != nil {
				return BoxExpression{}, err
			}
		}
	}

	if p.current.Type == TokenAt {
		constraints, err := p.parseConstraints()
		if err != nil {
			return BoxExpression{}, err
		}
		box.Constraints = constraints
	}
	return box, nil
}

func (p *Parser) parseConstraints() ([]LayoutConstraint, error) {
	if err := p.advance(); err != nil {
		return nil, err
	}
	var constraints []LayoutConstraint
	for {
		if p.current.Type != TokenIdentifier {
			return nil, p.errorf(p.current, "expected layout constraint after @ or comma, got %s", p.current.Type)
		}
		relationToken := p.current
		relation, ok := parseConstraintRelation(relationToken.Value)
		if !ok {
			return nil, p.errorf(relationToken, "unknown layout constraint %q", relationToken.Value)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.current.Type != TokenIdentifier {
			return nil, p.errorf(p.current, "expected identifier after %s, got %s", relation, p.current.Type)
		}
		constraints = append(constraints, LayoutConstraint{
			Relation: relation, Target: p.current.Value, Position: relationToken.Position,
		})
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.current.Type != TokenComma {
			return constraints, nil
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
	}
}

func parseConstraintRelation(value string) (ConstraintRelation, bool) {
	relation := ConstraintRelation(value)
	switch relation {
	case ConstraintThen, ConstraintLeftOf, ConstraintRightOf, ConstraintAbove, ConstraintBelow:
		return relation, true
	default:
		return "", false
	}
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
