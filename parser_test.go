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

func TestParserRequiresNewlineBetweenStatements(t *testing.T) {
	_, err := Parse("a b")
	if err == nil || !strings.Contains(err.Error(), "expected newline") {
		t.Fatalf("error = %v", err)
	}
}

func TestParserReportsMissingBracket(t *testing.T) {
	_, err := Parse("a[\n b -> c")
	if err == nil || !strings.Contains(err.Error(), "expected ]") {
		t.Fatalf("error = %v", err)
	}
}
