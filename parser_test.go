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

func TestParseAdjacentConnectionChains(t *testing.T) {
	for _, test := range []struct {
		source string
		want   string
	}{
		{"a -> b c -> d", "a->b c->d"},
		{"a b -> c", "a b->c"},
		{"a['A'] b['B'] -> c['C'] d['D'] -> a", "a b->c d->a"},
		{"a -> b -> c c -> a b -> a", "a->b->c c->a b->a"},
	} {
		t.Run(test.source, func(t *testing.T) {
			diagram, err := Parse(test.source)
			if err != nil {
				t.Fatal(err)
			}
			var chains []string
			for _, statement := range diagram.Statements {
				var identifiers []string
				for _, box := range statement.Boxes {
					identifiers = append(identifiers, box.Identifier)
				}
				chains = append(chains, strings.Join(identifiers, "->"))
			}
			if got := strings.Join(chains, " "); got != test.want {
				t.Fatalf("statements = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseAdjacentChainsInsideContainer(t *testing.T) {
	diagram, err := Parse(`_[a["A"] -> b["B"]-> c["C"] #dir:diag c -> a b -> a]`)
	if err != nil {
		t.Fatal(err)
	}
	inner := diagram.Statements[0].Boxes[0].Diagram
	if inner.Layout == nil || inner.Layout.Direction != LayoutDiagonal {
		t.Fatalf("layout = %+v", inner.Layout)
	}
	m, err := declarationPass(diagram)
	if err != nil {
		t.Fatal(err)
	}
	var edges []string
	for _, edge := range m.edges {
		edges = append(edges, edge.from.name+"->"+edge.to.name)
	}
	if got, want := strings.Join(edges, " "), "a->b b->c c->a b->a"; got != want {
		t.Fatalf("connections = %q, want %q", got, want)
	}
}

func TestParseRejectsIncompleteChains(t *testing.T) {
	for _, source := range []string{"a ->", "a -> -> b", "a -> b c ->", "outer[a -> b c ->]"} {
		if _, err := Parse(source); err == nil || !strings.Contains(err.Error(), "expected identifier after ->") {
			t.Errorf("Parse(%q) error = %v, want missing arrow target", source, err)
		}
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

func TestParseLayoutDirectives(t *testing.T) {
	for _, name := range []string{"dir", "layout", "direction"} {
		for _, mode := range []struct {
			value string
			want  LayoutDirection
		}{
			{"row", LayoutRow}, {"col", LayoutColumn},
			{"column", LayoutColumn}, {"grid:2x3", LayoutGrid},
			{"diag", LayoutDiagonal}, {"diagonal", LayoutDiagonal},
		} {
			t.Run(name+":"+mode.value, func(t *testing.T) {
				diagram, err := Parse("#" + name + ":" + mode.value + "\na b")
				if err != nil {
					t.Fatal(err)
				}
				layout := diagram.Layout
				if layout == nil || layout.Direction != mode.want {
					t.Fatalf("layout = %+v, want %s", layout, mode.want)
				}
				if mode.want == LayoutGrid && (layout.Rows != 2 || layout.Columns != 3) {
					t.Fatalf("grid = %+v, want 2 rows and 3 columns", layout)
				}
			})
		}
	}
}

func TestParseNestedLayoutDirectives(t *testing.T) {
	diagram, err := Parse("#dir:col\nouter[#layout:grid:2x3 a b c]\nother[x y]")
	if err != nil {
		t.Fatal(err)
	}
	outer := diagram.Statements[0].Boxes[0].Diagram
	other := diagram.Statements[1].Boxes[0].Diagram
	if diagram.Layout.Direction != LayoutColumn || outer.Layout.Direction != LayoutGrid || other.Layout != nil {
		t.Fatalf("layouts: top=%+v, outer=%+v, other=%+v", diagram.Layout, outer.Layout, other.Layout)
	}
}

func TestParseRejectsInvalidLayoutDirectives(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"#unknown:row", "unknown directive"},
		{"#dir", "expected layout direction"},
		{"#dir:", "unknown layout direction"},
		{"#dir:circle", "unknown layout direction"},
		{"#dir:diag:2x3", "does not accept dimensions"},
		{"#dir:row:2x3", "does not accept dimensions"},
		{"#dir:col:2", "does not accept dimensions"},
		{"#dir:grid", "grid requires dimensions"},
		{"#dir:grid:2x3:4", "grid requires dimensions"},
		{"#dir:grid:0x3", "positive integers"},
		{"#dir:grid:2x0", "positive integers"},
		{"#dir:grid:-2x3", "positive integers"},
		{"#dir:grid:+2x3", "positive integers"},
		{"#dir:grid:2", "positive integers"},
		{"#dir:grid:2x", "positive integers"},
		{"#dir:grid:2x3x4", "positive integers"},
		{"#dir:grid:2xthree", "positive integers"},
		{"#dir:grid:999999999999999999999999x2", "positive integers"},
		{"#dir:row\n#layout:col", "only one layout directive"},
		{"outer[#dir:row #dir:row]", "only one layout directive"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			_, err := Parse(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	_, err := Parse("outer[\n  #dir:grid:0x2\n]")
	if err == nil || !strings.Contains(err.Error(), "line 2, column 3") {
		t.Fatalf("directive error position = %v", err)
	}
}

func TestParserReportsMissingBracket(t *testing.T) {
	_, err := Parse("a[\n b -> c")
	if err == nil || !strings.Contains(err.Error(), "expected ]") {
		t.Fatalf("error = %v", err)
	}
}

func TestParserRejectsUnsupportedPunctuation(t *testing.T) {
	for _, source := range []string{"a@above b", "a,b", "outer[a@then b]"} {
		if _, err := Parse(source); err == nil || !strings.Contains(err.Error(), "unexpected character") {
			t.Errorf("Parse(%q) error = %v, want unexpected character", source, err)
		}
	}
}
