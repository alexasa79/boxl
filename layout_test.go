package boxl

import (
	"strings"
	"testing"
)

func TestLayoutConstraints(t *testing.T) {
	diagram, err := Parse(strings.Join([]string{
		"b@then d,below a",
		"b@left-of e",
		"b@right-of c",
		"b@above f",
	}, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := declarationPass(diagram)
	if err != nil {
		t.Fatal(err)
	}
	positions, err := layoutBoxes(model.children(nil), model.layoutConstraints(nil), model.connections(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	position := func(name string) layoutPosition { return positions[model.boxes[name]] }
	b := position("b")
	for _, name := range []string{"d", "e"} {
		got := position(name)
		if got.column <= b.column || got.row != b.row {
			t.Errorf("%s position = %+v, want right of b at %+v", name, got, b)
		}
	}
	c := position("c")
	if c.column >= b.column || c.row != b.row {
		t.Errorf("c position = %+v, want left of b at %+v", c, b)
	}
	a := position("a")
	if a.row >= b.row || a.column != b.column {
		t.Errorf("a position = %+v, want above b at %+v", a, b)
	}
	f := position("f")
	if f.row <= b.row || f.column != b.column {
		t.Errorf("f position = %+v, want below b at %+v", f, b)
	}
}

func TestLayoutRejectsContradictoryConstraints(t *testing.T) {
	tests := []string{
		"a@left-of b\nb@left-of a",
		"a@above b\nb@above a",
		"a@left-of b\nb@above a",
	}
	for _, source := range tests {
		_, err := Render(source)
		if err == nil || !strings.Contains(err.Error(), "layout constraint") {
			t.Errorf("Render(%q) error = %v", source, err)
		}
	}
}

func TestLayoutConstraintCannotCrossContainerBoundary(t *testing.T) {
	_, err := Render("outer[a]\nc@below a")
	if err == nil || !strings.Contains(err.Error(), "crosses the boundary") {
		t.Fatalf("error = %v", err)
	}
}
