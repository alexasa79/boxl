package boxl

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func renderStyleTest(t *testing.T, source string) image.Image {
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

func assertImageColor(t *testing.T, rendered image.Image, x, y int, want color.RGBA) {
	t.Helper()
	got := color.RGBAModel.Convert(rendered.At(x, y)).(color.RGBA)
	if got != want {
		t.Errorf("pixel (%d, %d) = %+v, want %+v", x, y, got, want)
	}
}

func TestUnderscoreContainersHideBorders(t *testing.T) {
	visible := renderStyleTest(t, "outer[a]")
	hidden := renderStyleTest(t, "_outer[a]")
	if hidden.Bounds() != visible.Bounds() {
		t.Fatalf("borderless container changed layout: %v, want %v", hidden.Bounds(), visible.Bounds())
	}
	assertImageColor(t, hidden, imageOuterMargin, imageOuterMargin, color.RGBA{})
	assertImageColor(t, visible, imageOuterMargin, imageOuterMargin, color.RGBA{255, 255, 255, 255})
	assertImageColor(t, hidden, imageOuterMargin+imageContainerPad, imageOuterMargin+imageContainerPad, color.RGBA{255, 255, 255, 255})
	empty := renderStyleTest(t, "_empty[]")
	for y := 0; y < empty.Bounds().Dy(); y++ {
		for x := 0; x < empty.Bounds().Dx(); x++ {
			if _, _, _, alpha := empty.At(x, y).RGBA(); alpha != 0 {
				t.Fatalf("empty invisible container paints pixel (%d, %d)", x, y)
			}
		}
	}
}

func TestBoxBackgroundForegroundAndText(t *testing.T) {
	rendered := renderStyleTest(t, `#bg:white #fg:black a["A" #bg:rgb(255, 0, 0)] -> b["B" #fg:#00f]`)
	assertImageColor(t, rendered, 0, 0, color.RGBA{255, 255, 255, 255})
	assertImageColor(t, rendered, 6, 6, color.RGBA{0, 0, 0, 255})
	assertImageColor(t, rendered, 10, 10, color.RGBA{255, 0, 0, 255})
	assertImageColor(t, rendered, 145, 32, color.RGBA{0, 0, 0, 255})
	assertImageColor(t, rendered, 182, 6, color.RGBA{0, 0, 255, 255})
	assertImageColor(t, rendered, 186, 10, color.RGBA{255, 255, 255, 255})
	blueText := false
	for y := 18; y < 46; y++ {
		for x := 198; x < 278; x++ {
			pixel := color.RGBAModel.Convert(rendered.At(x, y)).(color.RGBA)
			if pixel.B > 128 && pixel.R < 128 && pixel.G < 128 {
				blueText = true
			}
		}
	}
	if !blueText {
		t.Fatal("foreground did not color the label")
	}
}

func TestForegroundInheritsWithoutLeakingToSiblings(t *testing.T) {
	rendered := renderStyleTest(t, `#fg:red outer[#fg:blue a["A" #fg:green] -> b["B"]] next["N"]`)
	assertImageColor(t, rendered, 6, 6, color.RGBA{0, 0, 255, 255})
	assertImageColor(t, rendered, 30, 30, color.RGBA{0, 128, 0, 255})
	assertImageColor(t, rendered, 170, 56, color.RGBA{0, 0, 255, 255})
	assertImageColor(t, rendered, 206, 30, color.RGBA{0, 0, 255, 255})
	assertImageColor(t, rendered, 406, 30, color.RGBA{255, 0, 0, 255})
}

func TestInvisibleContainerStillPaintsExplicitBackground(t *testing.T) {
	rendered := renderStyleTest(t, `_root[#bg:white #fg:black a["A"] -> b["B"]]`)
	assertImageColor(t, rendered, 0, 0, color.RGBA{})
	assertImageColor(t, rendered, 6, 6, color.RGBA{255, 255, 255, 255})
	assertImageColor(t, rendered, 30, 30, color.RGBA{0, 0, 0, 255})
	assertImageColor(t, rendered, 170, 56, color.RGBA{0, 0, 0, 255})
}

func TestColorAlphaBlending(t *testing.T) {
	transparent := renderStyleTest(t, "a[#bg:#ff000080]")
	assertImageColor(t, transparent, 10, 10, color.RGBA{128, 0, 0, 128})
	opaque := renderStyleTest(t, "#bg:white a[#bg:#ff000080]")
	assertImageColor(t, opaque, 10, 10, color.RGBA{255, 127, 127, 255})
	nested := renderStyleTest(t, "outer[#bg:#ff000080 a[]]")
	assertImageColor(t, nested, 35, 35, color.RGBA{128, 0, 0, 128})
	foreground := renderStyleTest(t, "#fg:#00ff0080 a -> b")
	assertImageColor(t, foreground, 6, 6, color.RGBA{0, 128, 0, 128})
	assertImageColor(t, foreground, 65, 26, color.RGBA{0, 128, 0, 128})
}

func TestKittyPaddingPreservesChartBackground(t *testing.T) {
	for _, test := range []struct {
		background string
		want       color.RGBA
	}{
		{"white", color.RGBA{255, 255, 255, 255}},
		{"#ff000080", color.RGBA{128, 0, 0, 128}},
	} {
		t.Run(test.background, func(t *testing.T) {
			var output bytes.Buffer
			source := "#bg:" + test.background + " #fg:black a -> b"
			if err := TransformKittyAtCellSize(strings.NewReader(source), &output, KittyCellSize{Width: 10, Height: 24}); err != nil {
				t.Fatal(err)
			}
			data := decodeKittyPayload(t, parseKittyCommands(t, output.Bytes()))
			rendered, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			assertImageColor(t, rendered, 0, 0, test.want)
			assertImageColor(t, rendered, rendered.Bounds().Dx()-1, rendered.Bounds().Dy()-1, test.want)
			// The central image must retain its original alpha when copied into padding.
			assertImageColor(t, rendered, 30, 40, test.want)
		})
	}
}
