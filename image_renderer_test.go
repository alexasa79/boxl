package boxl

import (
	"bytes"
	"image"
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
		{"diagonal", "#dir:diagonal\na b c", 260, 188},
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
	for _, directive := range []string{"", "#dir:row\n", "#dir:col\n", "#dir:grid:2x2\n", "#dir:diag\n"} {
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

func TestRenderPNGDiagonalArrows(t *testing.T) {
	data, err := RenderPNG("#dir:diag\na -> b")
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	// The shaft and rotated arrowhead follow a straight diagonal.
	for _, point := range []image.Point{image.Pt(65, 51), image.Pt(78, 60), image.Pt(97, 77)} {
		if _, _, _, alpha := rendered.At(point.X, point.Y).RGBA(); alpha == 0 {
			t.Errorf("diagonal arrow missing at %v", point)
		}
	}
	// Keep the gaps, box interiors, and unused elbow routes transparent.
	for _, point := range []image.Point{
		image.Pt(76, 26), image.Pt(76, 94), image.Pt(47, 40), image.Pt(109, 80),
		image.Pt(40, 35), image.Pt(115, 84),
	} {
		if _, _, _, alpha := rendered.At(point.X, point.Y).RGBA(); alpha != 0 {
			t.Errorf("unexpected arrow pixel at %v", point)
		}
	}
}

func TestRenderPNGDiagonalAliases(t *testing.T) {
	for _, source := range []string{
		"#dir:diag\na['A'] -> b['B'] -> c['C']",
		"outer[#dir:diag a['A'] -> b -> c['C']]",
		"#dir:diag\nouter[a b] -> next",
	} {
		short, err := RenderPNG(source)
		if err != nil {
			t.Fatal(err)
		}
		long, err := RenderPNG(strings.ReplaceAll(source, ":diag", ":diagonal"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(short, long) {
			t.Errorf("aliases render differently for %q", source)
		}
	}
}

func TestRenderPNGContainerIncludesReturnCurve(t *testing.T) {
	decode := func(source string) image.Image {
		t.Helper()
		data, err := RenderPNG(source)
		if err != nil {
			t.Fatal(err)
		}
		rendered, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		return rendered
	}
	straight := decode("outer[a['A'] -> b['B'] -> c['C']]")
	cycle := decode("outer[a['A'] -> b['B'] -> c['C']\nc -> a]")
	if cycle.Bounds().Dy() <= straight.Bounds().Dy() {
		t.Fatalf("containing box did not grow for return curve: straight %v, cycle %v", straight.Bounds(), cycle.Bounds())
	}
	for x := imageOuterMargin; x < cycle.Bounds().Dx()-imageOuterMargin; x++ {
		for _, y := range []int{imageOuterMargin, cycle.Bounds().Dy() - imageOuterMargin - 1} {
			if _, _, _, alpha := cycle.At(x, y).RGBA(); alpha == 0 {
				t.Fatalf("containing box border is broken at (%d, %d)", x, y)
			}
		}
	}
}
