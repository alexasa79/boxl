package boxl

import "image/color"

// Position identifies a byte offset and its one-based line and column.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Diagram contains box declarations, connection chains, and an optional layout.
type Diagram struct {
	Statements []Statement
	Layout     *LayoutDirective
	Style      BoxStyle
}

// BoxStyle sets a box's fill and its inheritable foreground color.
type BoxStyle struct {
	Background *ColorDirective
	Foreground *ColorDirective
}

// ColorDirective stores a straight-alpha color and its source position.
type ColorDirective struct {
	Color    color.NRGBA
	Position Position
}

// LayoutDirection selects how a diagram arranges its immediate children.
type LayoutDirection string

const (
	LayoutRow      LayoutDirection = "row"
	LayoutColumn   LayoutDirection = "column"
	LayoutGrid     LayoutDirection = "grid"
	LayoutDiagonal LayoutDirection = "diagonal"
)

// LayoutDirective configures one diagram. Grid dimensions are rows by columns.
// A nil directive means row layout.
type LayoutDirective struct {
	Direction LayoutDirection
	Rows      int
	Columns   int
	Position  Position
}

// Statement declares a box or a connection chain.
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
	Style      BoxStyle
	Position   Position
}
