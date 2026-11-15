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
	Layout     *LayoutDirective
}

// LayoutDirection selects how a diagram arranges its immediate children.
type LayoutDirection string

const (
	LayoutRow    LayoutDirection = "row"
	LayoutColumn LayoutDirection = "column"
	LayoutGrid   LayoutDirection = "grid"
)

// LayoutDirective configures one diagram. Grid dimensions are rows by columns.
// A nil directive means row layout.
type LayoutDirective struct {
	Direction LayoutDirection
	Rows      int
	Columns   int
	Position  Position
}

// Statement declares boxes or a connection chain.
type Statement struct {
	Boxes     []BoxExpression
	Connected bool
	Position  Position
}

// BoxExpression is one occurrence of a box identifier. Text and Diagram are
// mutually exclusive in the grammar.
type BoxExpression struct {
	Identifier string
	Text       *string
	Diagram    *Diagram
	Position   Position
}
