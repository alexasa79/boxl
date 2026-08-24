package boxl

// Position identifies a byte offset and its one-based line and column.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Diagram is a sequence of newline-separated statements.
type Diagram struct {
	Statements []Statement
}

// Statement declares one or more boxes and optional connections between them.
type Statement struct {
	Boxes    []BoxExpression
	Position Position
}

// BoxExpression is one occurrence of a box identifier. Text and Diagram are
// mutually exclusive in the grammar.
type BoxExpression struct {
	Identifier string
	Text       *string
	Diagram    *Diagram
	Position   Position
}
