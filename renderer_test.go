package boxl

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderUsesDeclarationsCollectedInFirstPass(t *testing.T) {
	got, err := Render("a -> b\na[\"hello\"]")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"+-------+     +-------+",
		"| hello | --> |       |",
		"+-------+     +-------+",
	}, "\n")
	if got != want {
		t.Fatalf("rendered diagram:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderBranchingHorizontalLayout(t *testing.T) {
	got, err := Render("a[\"hello world\"]\na -> b\na -> c")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"+-------------+     +-------------+",
		"| hello world | -+> |             |",
		"+-------------+  |  +-------------+",
		"                 |",
		"                 |  +-------------+",
		"                 +> |             |",
		"                    +-------------+",
	}, "\n")
	if got != want {
		t.Fatalf("rendered diagram:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderNestedContainer(t *testing.T) {
	got, err := Render("a[b[\"one\"] -> c[\"two\"]]")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"+---------------------+",
		"| +-----+     +-----+ |",
		"| | one | --> | two | |",
		"| +-----+     +-----+ |",
		"+---------------------+",
	}, "\n")
	if got != want {
		t.Fatalf("rendered diagram:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderRejectsAmbiguousDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"conflicting text", "a[\"one\"]\na[\"two\"]", "conflicting text"},
		{"text and children", "a[\"one\"]\na[b]", "both text and nested"},
		{"two parents", "a[x]\nb[x]", "belongs to both"},
		{"boundary crossing", "a[x]\nx -> b", "crosses the boundary"},
		{"connection cycle", "a -> b\nb -> a", "connection cycle"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Render(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestTransform(t *testing.T) {
	var output bytes.Buffer
	if err := Transform(strings.NewReader("a -> b"), &output); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.HasSuffix(got, "\n") || !strings.Contains(got, "-->") {
		t.Fatalf("output = %q", got)
	}
}

func TestNewRendererValidatesOptions(t *testing.T) {
	_, err := NewRenderer(RenderOptions{HorizontalGap: 2})
	if err == nil {
		t.Fatal("expected invalid horizontal gap")
	}
}
