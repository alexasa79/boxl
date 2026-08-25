package boxl

import (
	"strings"
	"testing"
)

func TestParseConnectionsAndNesting(t *testing.T) {
	source := "a[\n  b[c -> d]\n  b -> e\n]\na -> f[\"done\"]\n"
	diagram, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagram.Statements) != 2 {
		t.Fatalf("statements = %d", len(diagram.Statements))
	}
	outer := diagram.Statements[0].Boxes[0]
	if outer.Identifier != "a" || outer.Diagram == nil {
		t.Fatalf("outer box = %+v", outer)
	}
	if len(outer.Diagram.Statements) != 2 {
		t.Fatalf("nested statements = %d", len(outer.Diagram.Statements))
	}
	inner := outer.Diagram.Statements[0].Boxes[0]
	if inner.Identifier != "b" || inner.Diagram == nil {
		t.Fatalf("inner box = %+v", inner)
	}
	if got := inner.Diagram.Statements[0].Boxes; len(got) != 2 || got[0].Identifier != "c" || got[1].Identifier != "d" {
		t.Fatalf("deep connection = %+v", got)
	}
	label := diagram.Statements[1].Boxes[1]
	if label.Text == nil || *label.Text != "done" {
		t.Fatalf("label = %+v", label.Text)
	}
}

func TestParseEmptyDiagramAndContainer(t *testing.T) {
	diagram, err := Parse("empty[]")
	if err != nil {
		t.Fatal(err)
	}
	box := diagram.Statements[0].Boxes[0]
	if box.Diagram == nil || len(box.Diagram.Statements) != 0 {
		t.Fatalf("empty container = %+v", box)
	}
}

func TestParseSingleQuotedText(t *testing.T) {
	diagram, err := Parse("message['hello world']")
	if err != nil {
		t.Fatal(err)
	}
	text := diagram.Statements[0].Boxes[0].Text
	if text == nil || *text != "hello world" {
		t.Fatalf("text = %v", text)
	}
}

func TestParseMultilineQuotedText(t *testing.T) {
	diagram, err := Parse("message[\"first line\nsecond line\"]")
	if err != nil {
		t.Fatal(err)
	}
	text := diagram.Statements[0].Boxes[0].Text
	if text == nil || *text != "first line\nsecond line" {
		t.Fatalf("text = %v", text)
	}
}

func TestParseWhitespaceLayoutConstraints(t *testing.T) {
	diagram, err := Parse("a b\nc d")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagram.Statements) != 2 || diagram.Statements[0].Connected || diagram.Statements[1].Connected {
		t.Fatalf("statements = %+v", diagram.Statements)
	}
	want := [][]LayoutConstraint{
		{
			{Relation: ConstraintThen, Target: "b", Position: Position{Line: 1, Column: 1}},
			{Relation: ConstraintAbove, Target: "c", Position: Position{Line: 1, Column: 1}},
		},
		{{Relation: ConstraintAbove, Target: "d", Position: Position{Offset: 2, Line: 1, Column: 3}}},
		{{Relation: ConstraintThen, Target: "d", Position: Position{Offset: 4, Line: 2, Column: 1}}},
	}
	got := []BoxExpression{
		diagram.Statements[0].Boxes[0],
		diagram.Statements[0].Boxes[1],
		diagram.Statements[1].Boxes[0],
	}
	for i := range want {
		if len(got[i].Constraints) != len(want[i]) {
			t.Fatalf("box %s constraints = %+v", got[i].Identifier, got[i].Constraints)
		}
		for j := range want[i] {
			if got[i].Constraints[j] != want[i][j] {
				t.Errorf("box %s constraint %d = %+v, want %+v", got[i].Identifier, j, got[i].Constraints[j], want[i][j])
			}
		}
	}
}

func TestBlankLineTerminatesWhitespaceLayout(t *testing.T) {
	diagram, err := Parse("a\n\nb")
	if err != nil {
		t.Fatal(err)
	}
	if constraints := diagram.Statements[0].Boxes[0].Constraints; len(constraints) != 0 {
		t.Fatalf("a constraints = %+v", constraints)
	}
}

func TestEOFCompletesVerticalWhitespaceLayout(t *testing.T) {
	diagram, err := Parse("a\nb")
	if err != nil {
		t.Fatal(err)
	}
	constraints := diagram.Statements[0].Boxes[0].Constraints
	if len(constraints) != 1 || constraints[0].Relation != ConstraintAbove || constraints[0].Target != "b" {
		t.Fatalf("a constraints = %+v", constraints)
	}
}

func TestArrowStatementsDoNotCreateWhitespaceConstraints(t *testing.T) {
	diagram, err := Parse("a -> b\nc -> d")
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range diagram.Statements {
		if !statement.Connected {
			t.Fatalf("statement = %+v", statement)
		}
		for _, box := range statement.Boxes {
			if len(box.Constraints) != 0 {
				t.Fatalf("box %s constraints = %+v", box.Identifier, box.Constraints)
			}
		}
	}
}

func TestParserReportsMissingBracket(t *testing.T) {
	_, err := Parse("a[\n b -> c")
	if err == nil || !strings.Contains(err.Error(), "expected ]") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseLayoutConstraints(t *testing.T) {
	diagram, err := Parse("b['box']@then d,left-of e,right-of c,above f,below a")
	if err != nil {
		t.Fatal(err)
	}
	box := diagram.Statements[0].Boxes[0]
	if box.Text == nil || *box.Text != "box" {
		t.Fatalf("text = %v", box.Text)
	}
	wantRelations := []ConstraintRelation{
		ConstraintThen, ConstraintLeftOf, ConstraintRightOf, ConstraintAbove, ConstraintBelow,
	}
	if len(box.Constraints) != len(wantRelations) {
		t.Fatalf("constraints = %+v", box.Constraints)
	}
	for i, want := range wantRelations {
		if box.Constraints[i].Relation != want {
			t.Errorf("constraint %d relation = %q, want %q", i, box.Constraints[i].Relation, want)
		}
	}
}

func TestParseRejectsInvalidLayoutConstraints(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"a@near b", "unknown layout constraint"},
		{"a@above", "expected identifier after above"},
		{"a@above b,", "expected layout constraint"},
	}
	for _, test := range tests {
		_, err := Parse(test.source)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("Parse(%q) error = %v, want text %q", test.source, err, test.want)
		}
	}
}
