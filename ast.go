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

// Statement declares a connection chain or one row of a whitespace layout.
type Statement struct {
	Boxes     []BoxExpression
	Connected bool
	Position  Position
}

// ConstraintRelation identifies a relative placement constraint.
type ConstraintRelation string

const (
	ConstraintThen    ConstraintRelation = "then"
	ConstraintLeftOf  ConstraintRelation = "left-of"
	ConstraintRightOf ConstraintRelation = "right-of"
	ConstraintAbove   ConstraintRelation = "above"
	ConstraintBelow   ConstraintRelation = "below"
)

// LayoutConstraint places the containing box relative to Target. Then is an
// alias for placing Target to the right of the containing box.
type LayoutConstraint struct {
	Relation ConstraintRelation
	Target   string
	Position Position
}

// BoxExpression is one occurrence of a box identifier. Text and Diagram are
// mutually exclusive in the grammar.
type BoxExpression struct {
	Identifier  string
	Text        *string
	Diagram     *Diagram
	Constraints []LayoutConstraint
	Position    Position
}
