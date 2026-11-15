package boxl

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
)

func TestRenderPNGUsesDeclarationsCollectedInFirstPass(t *testing.T) {
	got, err := RenderPNG("a -> b\na[\"hello\"]")
	if err != nil {
		t.Fatal(err)
	}
	want, err := RenderPNG("a[\"hello\"] -> b")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("forward declaration produces a different diagram")
	}
}

func TestRenderPNGRejectsAmbiguousDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"conflicting text", "a[\"one\"]\na[\"two\"]", "conflicting text"},
		{"text and children", "a[\"one\"]\na[b]", "both text and nested"},
		{"two parents", "a[x]\nb[x]", "belongs to both"},
		{"boundary crossing", "a[x]\nx -> b", "crosses the boundary"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := RenderPNG(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestRenderPNGAllowsConnectionCycles(t *testing.T) {
	if _, err := RenderPNG("a -> b\nb -> a"); err != nil {
		t.Fatal(err)
	}
}

func TestRenderPNGLayoutDimensions(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		width  int
		height int
	}{
		{"default row", "a\nb\n\nc d e", 468, 52},
		{"column", "#direction:column\na b c d e", 52, 324},
		{"grid", "#layout:grid:2x3\na b c d e", 260, 120},
		{"nested layouts", "#dir:col\nouter[#dir:grid:2x2 a b c d]\nother[x y]", 204, 284},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := RenderPNG(test.source)
			if err != nil {
				t.Fatal(err)
			}
			rendered, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if rendered.Bounds().Dx() != test.width || rendered.Bounds().Dy() != test.height {
				t.Fatalf("image bounds = %v, want %dx%d", rendered.Bounds(), test.width, test.height)
			}
		})
	}
}

func TestRenderPNGLayoutIgnoresLineBreaks(t *testing.T) {
	for _, directive := range []string{"", "#dir:row\n", "#dir:col\n", "#dir:grid:2x2\n"} {
		want, err := RenderPNG(directive + "a b c d")
		if err != nil {
			t.Fatal(err)
		}
		got, err := RenderPNG(directive + "a\n b\n\n c d")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("line breaks change layout for %q", directive)
		}
	}
}
