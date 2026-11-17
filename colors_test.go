package boxl

import (
	"image/color"
	"strings"
	"testing"
)

func TestParseColors(t *testing.T) {
	for _, test := range []struct {
		value string
		want  color.NRGBA
	}{
		{"white", color.NRGBA{255, 255, 255, 255}},
		{"RED", color.NRGBA{255, 0, 0, 255}},
		{"darkslateblue", color.NRGBA{72, 61, 139, 255}},
		{"transparent", color.NRGBA{}},
		{"rgb(1, 128, 255)", color.NRGBA{1, 128, 255, 255}},
		{"rgb(0,0,0)", color.NRGBA{0, 0, 0, 255}},
		{"#aF0", color.NRGBA{170, 255, 0, 255}},
		{"#123456", color.NRGBA{18, 52, 86, 255}},
		{"#12345680", color.NRGBA{18, 52, 86, 128}},
		{"#ff000000", color.NRGBA{255, 0, 0, 0}},
	} {
		t.Run(test.value, func(t *testing.T) {
			got, err := parseColor(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("color = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestParseRejectsInvalidColors(t *testing.T) {
	for _, value := range []string{"", "not-a-color", "#12", "#1234", "#1234567", "#gg0000ff", "rgb()", "rgb(1,2)", "rgb(1,2,3,4)", "rgb(256,0,0)", "rgb(-1,0,0)", "rgb(1.5,2,3)", "rgb(1,2,3"} {
		t.Run(value, func(t *testing.T) {
			if _, err := parseColor(value); err == nil {
				t.Fatalf("parseColor(%q) succeeded", value)
			}
		})
	}
}

func TestParseBoxColorDirectives(t *testing.T) {
	diagram, err := Parse(`#dir:col #bg:white #fg:black outer[#bg:#12345680 a[#bg:rgb(1, 2, 3) "A" #fg:#f08]]`)
	if err != nil {
		t.Fatal(err)
	}
	if diagram.Style.Background.Color != (color.NRGBA{255, 255, 255, 255}) || diagram.Style.Foreground.Color != (color.NRGBA{0, 0, 0, 255}) {
		t.Fatalf("chart style = %+v", diagram.Style)
	}
	outer := diagram.Statements[0].Boxes[0]
	if outer.Style.Background.Color != (color.NRGBA{18, 52, 86, 128}) {
		t.Fatalf("outer background = %+v", outer.Style.Background)
	}
	leaf := outer.Diagram.Statements[0].Boxes[0]
	if leaf.Text == nil || *leaf.Text != "A" || leaf.Diagram != nil {
		t.Fatalf("leaf = %+v", leaf)
	}
	if leaf.Style.Background.Color != (color.NRGBA{1, 2, 3, 255}) || leaf.Style.Foreground.Color != (color.NRGBA{255, 0, 136, 255}) {
		t.Fatalf("leaf style = %+v", leaf.Style)
	}
}

func TestColorDirectiveErrors(t *testing.T) {
	for _, test := range []struct {
		source string
		want   string
	}{
		{"#bg:", "invalid #bg color"},
		{"#fg:unknown", "invalid #fg color"},
		{"#bg:white #bg:black", "only one #bg directive"},
		{"a[#fg:red 'A' #fg:blue]", "only one #fg directive"},
		{"a['A' #dir:row]", "layout directives require nested boxes"},
		{"a[#bg:white b 'A']", "one quoted label or nested boxes"},
		{"a['A' #bg:white b]", "both quoted text and nested boxes"},
		{"a['A' 'B']", "one quoted label or nested boxes"},
		{"#dir:row\n  #bg:rgb(300, 0, 0)", "line 2, column 3"},
	} {
		t.Run(test.source, func(t *testing.T) {
			_, err := Parse(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRepeatedBoxColors(t *testing.T) {
	for _, test := range []struct {
		source string
		want   string
	}{
		{`a["A" #bg:red] a["A" #bg:#f00]`, ""},
		{`a["A" #fg:#000] a["A" #fg:black]`, ""},
		{`outer[#bg:white a] outer[#bg:white b]`, ""},
		{`a["A" #bg:red] a["A" #bg:blue]`, "conflicting #bg colors"},
		{`a["A" #fg:red] a["A" #fg:blue]`, "conflicting #fg colors"},
	} {
		_, err := RenderPNG(test.source)
		if test.want == "" {
			if err != nil {
				t.Errorf("RenderPNG(%q): %v", test.source, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("RenderPNG(%q) error = %v, want %q", test.source, err, test.want)
		}
	}
}
