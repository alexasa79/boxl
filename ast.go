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

// BoxStyle contains explicit colors, a palette, and an optional pair selection.
type BoxStyle struct {
	Background *ColorDirective
	Foreground *ColorDirective
	Palette    *PaletteDirective
	Pair       *ColorDirective
}

// PaletteDirective selects the palette inherited by a box's descendants.
type PaletteDirective struct {
	Name     string
	Position Position
}

// ColorDirective stores a color expression and its source position. Color is
// also populated for expressions that can be evaluated without a palette.
type ColorDirective struct {
	Color      color.NRGBA
	Expression Expression
	Position   Position
}

// Expression is a value used by a directive.
type Expression interface{ expression() }

// ColorLiteral is a color independent of the current palette.
type ColorLiteral struct{ Value color.NRGBA }

// IntegerLiteral is a numeric function argument.
type IntegerLiteral struct{ Value int }

// VariableExpression names a value without its leading $.
type VariableExpression struct{ Name string }

// CallExpression records a function and its unevaluated arguments.
type CallExpression struct {
	Name      string
	Arguments []Expression
}

func (ColorLiteral) expression()       {}
func (IntegerLiteral) expression()     {}
func (VariableExpression) expression() {}
func (CallExpression) expression()     {}

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
