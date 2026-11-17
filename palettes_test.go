package boxl

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"strings"
	"testing"
)

func paletteTestModel(t *testing.T, source string) *model {
	t.Helper()
	diagram, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	m, err := declarationPass(diagram)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func assertPalettePair(t *testing.T, m *model, name string, number int) {
	t.Helper()
	assertThemePair(t, m, name, "pastel", number)
}

func assertThemePair(t *testing.T, m *model, name, theme string, number int) {
	t.Helper()
	box := m.boxes[name]
	if box == nil {
		t.Fatalf("box %q is missing", name)
	}
	if got, want := m.resolvedStyles[box], lookupPalette(theme)[number-1].colors(); got != want {
		t.Errorf("box %s colors = %+v, want %s pair %d: %+v", name, got, theme, number, want)
	}
}

func TestBuiltInThemes(t *testing.T) {
	for _, theme := range []string{"pastel", "earth", "ocean"} {
		t.Run(theme, func(t *testing.T) {
			palette := lookupPalette(theme)
			if palette == nil {
				t.Fatal("palette is missing")
			}
			seen := make(map[palettePair]bool)
			for i, pair := range palette {
				if pair.foreground.A != 255 || pair.background.A != 255 || pair.foreground == pair.background || seen[pair] {
					t.Fatalf("pair %d is invalid or repeated: %+v", i+1, pair)
				}
				seen[pair] = true
			}
			for _, alias := range []string{"palette", "theme", "colors"} {
				m := paletteTestModel(t, "#"+alias+":"+strings.ToUpper(theme)+" A[B C] D")
				assertThemePair(t, m, "A", theme, 1)
				assertThemePair(t, m, "D", theme, 1)
				assertThemePair(t, m, "B", theme, 2)
				assertThemePair(t, m, "C", theme, 2)
			}
		})
	}
}

func paletteTestHue(value color.NRGBA) float64 {
	r, g, b := float64(value.R), float64(value.G), float64(value.B)
	high, low := max(r, g, b), min(r, g, b)
	if high == low {
		return 0
	}
	var hue float64
	switch high {
	case r:
		hue = (g - b) / (high - low)
	case g:
		hue = 2 + (b-r)/(high-low)
	case b:
		hue = 4 + (r-g)/(high-low)
	}
	return math.Mod(hue*60+360, 360)
}

func TestPaletteNeighborsHaveDistinctColors(t *testing.T) {
	for _, theme := range paletteNames() {
		t.Run(theme, func(t *testing.T) {
			palette := lookupPalette(theme)
			for i, pair := range palette {
				next := palette[(i+1)%len(palette)]
				for _, channel := range []struct {
					name       string
					first      color.NRGBA
					second     color.NRGBA
					separation float64
				}{
					{"foreground", pair.foreground, next.foreground, 50},
					{"background", pair.background, next.background, 30},
				} {
					gap := math.Abs(paletteTestHue(channel.first) - paletteTestHue(channel.second))
					gap = math.Min(gap, 360-gap)
					r := float64(channel.first.R) - float64(channel.second.R)
					g := float64(channel.first.G) - float64(channel.second.G)
					b := float64(channel.first.B) - float64(channel.second.B)
					if gap < 55 || math.Sqrt(r*r+g*g+b*b) < channel.separation {
						t.Errorf("pairs %d and %d have similar %s colors", i+1, (i+1)%len(palette)+1, channel.name)
					}
				}
			}
			for i := 0; i < 3; i++ {
				for _, value := range []color.NRGBA{palette[i].foreground, palette[i].background} {
					hue := paletteTestHue(value)
					matches := (i == 0 && hue >= 180 && hue <= 250) || (i == 1 && hue >= 80 && hue <= 170) || (i == 2 && (hue <= 30 || hue >= 330))
					if !matches {
						t.Errorf("pair %d does not follow the blue, green, red opening", i+1)
					}
				}
			}
		})
	}
}

func TestOceanFillsAreDistinctFromPastel(t *testing.T) {
	for i, pair := range oceanPalette {
		fill := pair.background
		high := float64(max(fill.R, fill.G, fill.B))
		low := float64(min(fill.R, fill.G, fill.B))
		if high == 0 || (high-low)/high < 0.27 {
			t.Errorf("ocean pair %d has a washed-out fill", i+1)
		}
		for _, pastel := range pastelPalette {
			r := float64(fill.R) - float64(pastel.background.R)
			g := float64(fill.G) - float64(pastel.background.G)
			b := float64(fill.B) - float64(pastel.background.B)
			if math.Sqrt(r*r+g*g+b*b) < 60 {
				t.Errorf("ocean pair %d is too close to a pastel fill", i+1)
			}
		}
	}
}

func TestThemeVariablesFunctionsAndRendering(t *testing.T) {
	for _, theme := range []string{"earth", "ocean"} {
		t.Run(theme, func(t *testing.T) {
			source := "#theme:" + theme + ` a["A"] b["B" #colors:$color16] c["C" #colors:nextcolors()]`
			m := paletteTestModel(t, source)
			assertThemePair(t, m, "a", theme, 1)
			assertThemePair(t, m, "b", theme, 16)
			assertThemePair(t, m, "c", theme, 1)
			rendered := renderStyleTest(t, source)
			palette := lookupPalette(theme)
			for i, number := range []int{1, 16, 1} {
				x := 6 + i*(imageLeafMinWidth+imageHorizontalGap)
				assertImageColor(t, rendered, x, 6, color.RGBAModel.Convert(palette[number-1].foreground).(color.RGBA))
				assertImageColor(t, rendered, x+4, 10, color.RGBAModel.Convert(palette[number-1].background).(color.RGBA))
			}
			m = paletteTestModel(t, "#palette:"+theme+` a[#fg:$color3 #bg:$color7] b`)
			for _, name := range []string{"a", "b"} {
				got := m.resolvedStyles[m.boxes[name]]
				if got.foreground != palette[2].foreground || got.background != palette[6].background {
					t.Errorf("box %s uses the wrong theme components: %+v", name, got)
				}
			}
		})
	}
}

func TestNestedThemeDoesNotChangeParentSequence(t *testing.T) {
	m := paletteTestModel(t, `#palette:ocean A[#theme:earth #colors:$color5 B[#colors:nextcolors()] C] D[E] F[#colors:nextcolors()]`)
	assertThemePair(t, m, "A", "earth", 5)
	assertThemePair(t, m, "B", "earth", 7)
	assertThemePair(t, m, "C", "earth", 7)
	assertThemePair(t, m, "D", "ocean", 1)
	assertThemePair(t, m, "E", "ocean", 2)
	assertThemePair(t, m, "F", "ocean", 2)
}

func TestUnknownPaletteListsAvailableThemes(t *testing.T) {
	_, err := Parse("#theme:unknown")
	if err == nil || !strings.Contains(err.Error(), "earth, ocean, pastel") {
		t.Fatalf("palette error = %v", err)
	}
}

func TestPaletteAliasesAndHierarchy(t *testing.T) {
	for _, alias := range []string{"palette", "theme", "colors", "colorpair"} {
		t.Run(alias, func(t *testing.T) {
			m := paletteTestModel(t, "#"+alias+`:pastel A[B["B"] C["C"]] D[E["E"] F["F"]]`)
			for _, name := range []string{"A", "D"} {
				assertPalettePair(t, m, name, 1)
			}
			for _, name := range []string{"B", "C", "E", "F"} {
				assertPalettePair(t, m, name, 2)
			}
		})
	}
}

func TestPaletteOverridesCarryToLaterSiblings(t *testing.T) {
	m := paletteTestModel(t, `#palette:pastel A["A"] B["B" #colors:$color5] C["C"] D[E["E"]]`)
	assertPalettePair(t, m, "A", 1)
	for _, name := range []string{"B", "C", "D"} {
		assertPalettePair(t, m, name, 5)
	}
	assertPalettePair(t, m, "E", 6)

	m = paletteTestModel(t, `#palette:pastel A[B["B" #colors:$color5] C["C"]] D[E["E"] F["F"]]`)
	for _, name := range []string{"A", "D"} {
		assertPalettePair(t, m, name, 1)
	}
	for _, name := range []string{"B", "C"} {
		assertPalettePair(t, m, name, 5)
	}
	for _, name := range []string{"E", "F"} {
		assertPalettePair(t, m, name, 2)
	}
}

func TestPaletteComponentOverrides(t *testing.T) {
	m := paletteTestModel(t, `#palette:pastel A B[#bg:red] C D[#fg:blue] E`)
	assertPalettePair(t, m, "A", 1)
	for _, name := range []string{"B", "C", "D", "E"} {
		got := m.resolvedStyles[m.boxes[name]]
		if got.background != (color.NRGBA{255, 0, 0, 255}) {
			t.Errorf("box %s did not retain its predecessor's background: %+v", name, got)
		}
	}
	for _, name := range []string{"D", "E"} {
		if got := m.resolvedStyles[m.boxes[name]].foreground; got != (color.NRGBA{0, 0, 255, 255}) {
			t.Errorf("box %s foreground = %+v, want blue", name, got)
		}
	}

	m = paletteTestModel(t, `#theme:pastel A[#fg:$color4 #bg:$color9] B`)
	for _, name := range []string{"A", "B"} {
		got := m.resolvedStyles[m.boxes[name]]
		if got.foreground != pastelPalette[3].foreground || got.background != pastelPalette[8].background {
			t.Errorf("box %s did not select variable components: %+v", name, got)
		}
	}
}

func TestNextColorsAdvancesInSourceOrder(t *testing.T) {
	m := paletteTestModel(t, `#colors:pastel A[#fg:nextcolors() #bg:nextcolors()] B[#colors:nextcolors()] C`)
	first := m.resolvedStyles[m.boxes["A"]]
	if first.foreground != pastelPalette[1].foreground || first.background != pastelPalette[2].background {
		t.Fatalf("calls did not advance left to right: %+v", first)
	}
	assertPalettePair(t, m, "B", 4)
	assertPalettePair(t, m, "C", 4)

	m = paletteTestModel(t, `#palette:pastel A[#bg:nextcolors() #fg:nextcolors()] B`)
	first = m.resolvedStyles[m.boxes["A"]]
	if first.background != pastelPalette[1].background || first.foreground != pastelPalette[2].foreground {
		t.Fatalf("reversed call order was ignored: %+v", first)
	}

	m = paletteTestModel(t, `#theme:pastel A[#colors:$color5] B[#colorpair:nextcolors()] C`)
	assertPalettePair(t, m, "A", 5)
	assertPalettePair(t, m, "B", 6)
	assertPalettePair(t, m, "C", 6)
}

func TestNextColorsIsLocalToSiblingGroup(t *testing.T) {
	m := paletteTestModel(t, `#palette:pastel A[B[#colors:nextcolors()] C[#colors:nextcolors()]] D[#colors:nextcolors() E]`)
	assertPalettePair(t, m, "A", 1)
	assertPalettePair(t, m, "B", 3)
	assertPalettePair(t, m, "C", 4)
	assertPalettePair(t, m, "D", 2)
	assertPalettePair(t, m, "E", 3)
}

func TestPaletteLookupsAndAssignmentOrder(t *testing.T) {
	m := paletteTestModel(t, `#palette:pastel A[#fg:$color16 #bg:$color8] B[#colors:nextcolors()]`)
	assertPalettePair(t, m, "B", 2)
	m = paletteTestModel(t, `#palette:pastel A[#bg:white #colors:$color5 #fg:blue] B C[#colors:nextcolors()]`)
	for _, name := range []string{"A", "B"} {
		got := m.resolvedStyles[m.boxes[name]]
		if got.background != pastelPalette[4].background || got.foreground != opaqueColor(0x0000ff) {
			t.Errorf("box %s ignored assignment order: %+v", name, got)
		}
	}
	assertPalettePair(t, m, "C", 6)
}

func TestPaletteWrapsAtSixteen(t *testing.T) {
	m := paletteTestModel(t, `#palette:pastel A[#colors:$color16] B[#colors:nextcolors()] C[#colors:nextcolors()]`)
	assertPalettePair(t, m, "A", 16)
	assertPalettePair(t, m, "B", 1)
	assertPalettePair(t, m, "C", 2)
	var source strings.Builder
	source.WriteString("#palette:pastel ")
	for i := 1; i <= 18; i++ {
		fmt.Fprintf(&source, "n%d[", i)
	}
	source.WriteString(strings.Repeat("]", 18))
	m = paletteTestModel(t, source.String())
	for i := 1; i <= 18; i++ {
		assertPalettePair(t, m, fmt.Sprintf("n%d", i), (i-1)%16+1)
	}
}

func TestPaletteScopeAndForwardReferences(t *testing.T) {
	m := paletteTestModel(t, `A[#palette:pastel B C] D`)
	assertPalettePair(t, m, "A", 1)
	assertPalettePair(t, m, "B", 2)
	assertPalettePair(t, m, "C", 2)
	if got := m.resolvedStyles[m.boxes["D"]]; got.background.A != 0 || got.foreground != opaqueColor(0xffffff) {
		t.Fatalf("local palette leaked to sibling: %+v", got)
	}
	m = paletteTestModel(t, `#palette:pastel A -> B B["B" #colors:$color4] C`)
	assertPalettePair(t, m, "A", 1)
	assertPalettePair(t, m, "B", 4)
	assertPalettePair(t, m, "C", 4)
	m = paletteTestModel(t, `#palette:pastel A[#colors:nextcolors()] A -> B`)
	assertPalettePair(t, m, "A", 2)
	assertPalettePair(t, m, "B", 2)
}

func TestPaletteExpressionsAreRealSyntax(t *testing.T) {
	diagram, err := Parse(`A[#bg:rgb(1, 2, 3) #fg:$color4] B[#colors:nextcolors()]`)
	if err != nil {
		t.Fatal(err)
	}
	style := diagram.Statements[0].Boxes[0].Style
	call, ok := style.Background.Expression.(CallExpression)
	if !ok || call.Name != "rgb" || len(call.Arguments) != 3 {
		t.Fatalf("rgb expression = %#v", style.Background.Expression)
	}
	if variable, ok := style.Foreground.Expression.(VariableExpression); !ok || variable.Name != "color4" {
		t.Fatalf("variable expression = %#v", style.Foreground.Expression)
	}
	call, ok = diagram.Statements[1].Boxes[0].Style.Pair.Expression.(CallExpression)
	if !ok || call.Name != "nextcolors" || len(call.Arguments) != 0 {
		t.Fatalf("nextcolors expression = %#v", call)
	}
}

func TestPaletteExpressionErrors(t *testing.T) {
	for _, test := range []struct {
		source string
		want   string
	}{
		{"#palette:unknown A", "unknown palette"},
		{"#palette: A", "unknown palette"},
		{"#palette:pastel #theme:pastel A", "only one palette directive"},
		{"#palette:pastel A[#colors:$color1 #colorpair:$color2]", "only one palette pair directive"},
		{"A[#bg:$color1]", "active palette is required"},
		{"A[#colors:nextcolors()]", "active palette is required"},
		{"#palette:pastel A[#fg:$color0]", "unknown variable"},
		{"#palette:pastel A[#fg:$color17]", "unknown variable"},
		{"#palette:pastel A[#fg:$missing]", "unknown variable"},
		{"#palette:pastel A[#bg:nextcolors(1)]", "takes no arguments"},
		{"#palette:pastel A[#colors:nextcolor()]", "unknown function"},
		{"#palette:pastel A[#bg:missing()]", "unknown function"},
		{"#palette:pastel A[#bg:rgb($color1,0,0)]", "three integers"},
		{"#palette:pastel A[#colors:rgb(1,2,3)]", "expected a palette pair"},
		{"#palette:pastel A[#bg:$color2junk]", "unknown variable"},
	} {
		t.Run(test.source, func(t *testing.T) {
			_, err := RenderPNG(test.source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPaletteRenderingAndVariables(t *testing.T) {
	rendered := renderStyleTest(t, `#palette:pastel A["A"] B["B" #colors:$color3] C["C"]`)
	assertImageColor(t, rendered, 0, 0, color.RGBA{})
	assertImageColor(t, rendered, 10, 10, color.RGBAModel.Convert(pastelPalette[0].background).(color.RGBA))
	assertImageColor(t, rendered, 6, 6, color.RGBAModel.Convert(pastelPalette[0].foreground).(color.RGBA))
	for _, x := range []int{182, 358} {
		assertImageColor(t, rendered, x, 6, color.RGBAModel.Convert(pastelPalette[2].foreground).(color.RGBA))
		assertImageColor(t, rendered, x+4, 10, color.RGBAModel.Convert(pastelPalette[2].background).(color.RGBA))
	}
	const source = `#palette:pastel A["A" #colors:nextcolors()] B["B" #colors:nextcolors()] A -> B`
	first, err := RenderPNG(source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderPNG(source)
	if err != nil {
		t.Fatal(err)
	}
	var kitty bytes.Buffer
	if err := TransformKitty(strings.NewReader(source), &kitty); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || !bytes.Equal(first, decodeKittyPayload(t, parseKittyCommands(t, kitty.Bytes()))) {
		t.Fatal("color functions changed across renders or between PNG and Kitty")
	}
}

func TestPairDirectiveAliasesRenderIdentically(t *testing.T) {
	const source = `#palette:pastel A["A" #colors:$color3] B["B" #colors:nextcolors()]`
	first, err := RenderPNG(source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderPNG(strings.ReplaceAll(source, "#colors:", "#colorpair:"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("pair selection aliases render differently")
	}
}
