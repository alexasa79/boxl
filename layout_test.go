package boxl

import (
	"strings"
	"testing"
)

func TestLayoutDirectivePositions(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []layoutPosition
	}{
		{"default row", "a b\nc\n\nd", []layoutPosition{{column: 0}, {column: 1}, {column: 2}, {column: 3}}},
		{"explicit row", "#dir:row\na b c", []layoutPosition{{column: 0}, {column: 1}, {column: 2}}},
		{"col alias", "#dir:col\na b c", []layoutPosition{{row: 0}, {row: 1}, {row: 2}}},
		{"column", "#dir:column\na\nb c", []layoutPosition{{row: 0}, {row: 1}, {row: 2}}},
		{"wide grid", "#dir:grid:2x3\na b c d e", []layoutPosition{{}, {column: 1}, {column: 2}, {row: 1}, {column: 1, row: 1}}},
		{"tall grid", "#dir:grid:3x2\na b c d e", []layoutPosition{{}, {column: 1}, {row: 1}, {column: 1, row: 1}, {row: 2}}},
		{"connections do not branch layout", "a -> b\na -> c", []layoutPosition{{column: 0}, {column: 1}, {column: 2}}},
		{"first appearance order", "b\na -> b\nc", []layoutPosition{{column: 0}, {column: 1}, {column: 2}}},
		{"repeated boxes occupy one cell", "#dir:grid:1x2\na -> b\na b", []layoutPosition{{}, {column: 1}}},
		{"directive after declarations", "a b\n#direction:column", []layoutPosition{{}, {row: 1}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagram, err := Parse(test.source)
			if err != nil {
				t.Fatal(err)
			}
			m, err := declarationPass(diagram)
			if err != nil {
				t.Fatal(err)
			}
			children := m.children(nil)
			positions, err := layoutBoxes(children, m.layouts[nil], nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(children) != len(test.want) {
				t.Fatalf("children = %d, want %d", len(children), len(test.want))
			}
			for i, child := range children {
				if got := positions[child]; got != test.want[i] {
					t.Errorf("%s position = %+v, want %+v", child.name, got, test.want[i])
				}
			}
		})
	}
}

func TestLayoutDirectivesAreScoped(t *testing.T) {
	diagram, err := Parse("#dir:col\nouter[#dir:grid:1x2 a b]\nother[c d]")
	if err != nil {
		t.Fatal(err)
	}
	m, err := declarationPass(diagram)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		scope *declaredBox
		first string
		last  string
		want  layoutPosition
	}{
		{nil, "outer", "other", layoutPosition{row: 1}},
		{m.boxes["outer"], "a", "b", layoutPosition{column: 1}},
		{m.boxes["other"], "c", "d", layoutPosition{column: 1}},
	} {
		positions, err := layoutBoxes(m.children(test.scope), m.layouts[test.scope], test.scope)
		if err != nil {
			t.Fatal(err)
		}
		if positions[m.boxes[test.first]] != (layoutPosition{}) || positions[m.boxes[test.last]] != test.want {
			t.Errorf("%s positions = %+v", scopeLabel(test.scope), positions)
		}
	}
}

func TestLayoutDirectiveValidation(t *testing.T) {
	for _, test := range []struct {
		source string
		want   string
	}{
		{"#dir:grid:1x2\na b c", "cannot fit 3 boxes"},
		{"outer[#dir:grid:1x1 a b]", "grid in container \"outer\" cannot fit"},
		{"outer[#dir:row a]\nouter[#dir:col b]", "conflicting layout directives"},
	} {
		t.Run(test.source, func(t *testing.T) {
			_, err := RenderPNG(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	for _, source := range []string{
		"outer[#dir:col a]\nouter[#direction:column b]",
		"#dir:grid:9223372036854775807x9223372036854775807\na b",
	} {
		if _, err := RenderPNG(source); err != nil {
			t.Errorf("RenderPNG(%q): %v", source, err)
		}
	}
}
