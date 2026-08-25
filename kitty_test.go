package boxl

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"strconv"
	"strings"
	"testing"
)

func TestRenderPNG(t *testing.T) {
	data, err := RenderPNG("a[\"hello\"] -> b[\"world\"]")
	if err != nil {
		t.Fatal(err)
	}
	image, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode PNG: %v", err)
	}
	if got, want := image.Bounds().Dx(), 300; got != want {
		t.Errorf("PNG width = %d, want %d", got, want)
	}
	if got, want := image.Bounds().Dy(), 64; got != want {
		t.Errorf("PNG height = %d, want %d", got, want)
	}
	_, _, _, alpha := image.At(0, 0).RGBA()
	if alpha != 0 {
		t.Fatalf("PNG background alpha = %#x, want transparent", alpha)
	}
	for x := imageOuterMargin; x < imageOuterMargin+imageLeafMinWidth; x++ {
		red, green, blue, alpha := image.At(x, imageOuterMargin).RGBA()
		if red < 0xe000 || green < 0xe000 || blue < 0xe000 || alpha < 0xe000 {
			t.Fatalf("box border is not continuous at (%d, %d)", x, imageOuterMargin)
		}
	}
	centerY := imageOuterMargin + imageLeafMinHeight/2
	firstRight := imageOuterMargin + imageLeafMinWidth - 1
	secondLeft := firstRight + imageHorizontalGap + 1
	arrowTip := secondLeft - imageArrowBoxGap
	arrowBase := arrowTip - imageArrowLength
	for x := firstRight; x <= arrowBase; x++ {
		red, green, blue, alpha := image.At(x, centerY).RGBA()
		if red < 0xe000 || green < 0xe000 || blue < 0xe000 || alpha < 0xe000 {
			t.Fatalf("connection is broken at (%d, %d)", x, centerY)
		}
	}
	for x := arrowTip + 1; x < secondLeft; x++ {
		_, _, _, alpha := image.At(x, centerY).RGBA()
		if alpha != 0 {
			t.Fatalf("arrow does not leave a gap at (%d, %d)", x, centerY)
		}
	}
	hasAntialiasing := false
	for y := centerY - imageArrowHalfWidth; y <= centerY+imageArrowHalfWidth; y++ {
		for x := arrowBase; x <= arrowTip; x++ {
			_, _, _, alpha := image.At(x, y).RGBA()
			if alpha > 0x4000 && alpha < 0xe000 {
				hasAntialiasing = true
			}
		}
	}
	if !hasAntialiasing {
		t.Fatal("arrowhead diagonal edges are not anti-aliased")
	}
}

func TestRenderPNGUsesCompactEmptyBoxes(t *testing.T) {
	for _, source := range []string{"empty", "empty[\"\"]", "empty[]"} {
		data, err := RenderPNG(source)
		if err != nil {
			t.Fatal(err)
		}
		rendered, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		want := imageEmptyBoxSize + 2*imageOuterMargin
		if rendered.Bounds().Dx() != want || rendered.Bounds().Dy() != want {
			t.Errorf("RenderPNG(%q) size = %v, want %dx%d", source, rendered.Bounds(), want, want)
		}
	}
}

func TestRenderPNGExpandsForMultilineText(t *testing.T) {
	singleData, err := RenderPNG("box['first line']")
	if err != nil {
		t.Fatal(err)
	}
	multilineData, err := RenderPNG("box['first line\nsecond line']")
	if err != nil {
		t.Fatal(err)
	}
	single, err := png.Decode(bytes.NewReader(singleData))
	if err != nil {
		t.Fatal(err)
	}
	multiline, err := png.Decode(bytes.NewReader(multilineData))
	if err != nil {
		t.Fatal(err)
	}
	if multiline.Bounds().Dy() <= single.Bounds().Dy() {
		t.Fatalf("multiline height = %d, single-line height = %d", multiline.Bounds().Dy(), single.Bounds().Dy())
	}
}

func TestRenderPNGUsesWhitespaceLayout(t *testing.T) {
	data, err := RenderPNG("a b\nc d")
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	wantWidth := 2*imageEmptyBoxSize + imageHorizontalGap + 2*imageOuterMargin
	wantHeight := 2*imageEmptyBoxSize + imageVerticalGap + 2*imageOuterMargin
	if rendered.Bounds().Dx() != wantWidth || rendered.Bounds().Dy() != wantHeight {
		t.Fatalf("PNG size = %v, want %dx%d", rendered.Bounds(), wantWidth, wantHeight)
	}
}

func TestRenderPNGRoutesConstrainedConnections(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		gapX       int
		gapY       int
		tipX       int
		tipY       int
	}{
		{"right", "left-of", imageOuterMargin + imageEmptyBoxSize + imageHorizontalGap - 1, imageOuterMargin + imageEmptyBoxSize/2, imageOuterMargin + imageEmptyBoxSize + imageHorizontalGap - 2, imageOuterMargin + imageEmptyBoxSize/2},
		{"left", "right-of", imageOuterMargin + imageEmptyBoxSize, imageOuterMargin + imageEmptyBoxSize/2, imageOuterMargin + imageEmptyBoxSize + 1, imageOuterMargin + imageEmptyBoxSize/2},
		{"down", "above", imageOuterMargin + imageEmptyBoxSize/2, imageOuterMargin + imageEmptyBoxSize + imageVerticalGap - 1, imageOuterMargin + imageEmptyBoxSize/2, imageOuterMargin + imageEmptyBoxSize + imageVerticalGap - 2},
		{"up", "below", imageOuterMargin + imageEmptyBoxSize/2, imageOuterMargin + imageEmptyBoxSize, imageOuterMargin + imageEmptyBoxSize/2, imageOuterMargin + imageEmptyBoxSize + 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := RenderPNG("a -> b\na@" + test.constraint + " b")
			if err != nil {
				t.Fatal(err)
			}
			rendered, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			_, _, _, gapAlpha := rendered.At(test.gapX, test.gapY).RGBA()
			if gapAlpha != 0 {
				t.Errorf("pixel next to target has alpha %#x, want transparent", gapAlpha)
			}
			_, _, _, tipAlpha := rendered.At(test.tipX, test.tipY).RGBA()
			if tipAlpha == 0 {
				t.Errorf("arrow tip at (%d, %d) is transparent", test.tipX, test.tipY)
			}
		})
	}
}

func TestRenderPNGNestedBoxes(t *testing.T) {
	data, err := RenderPNG("outer[inner]")
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	outerTop := imageOuterMargin
	innerTop := imageOuterMargin + imageContainerPad
	innerLeft := imageOuterMargin + imageContainerPad
	for _, point := range [][2]int{{imageOuterMargin, outerTop}, {innerLeft, innerTop}} {
		red, green, blue, alpha := rendered.At(point[0], point[1]).RGBA()
		if red < 0xe000 || green < 0xe000 || blue < 0xe000 || alpha < 0xe000 {
			t.Fatalf("expected geometric border at (%d, %d)", point[0], point[1])
		}
	}
}

func TestTransformKittyTransmitsPNG(t *testing.T) {
	var output bytes.Buffer
	if err := TransformKitty(strings.NewReader("a[\"hello\"] -> b[\"world\"]"), &output); err != nil {
		t.Fatal(err)
	}
	commands := parseKittyCommands(t, output.Bytes())
	if len(commands) == 0 {
		t.Fatal("no Kitty commands")
	}
	if !strings.HasPrefix(commands[0].control, "a=T,f=100,q=2,m=") {
		t.Fatalf("first control data = %q", commands[0].control)
	}
	if strings.Contains(commands[0].control, "c=") || strings.Contains(commands[0].control, "r=") {
		t.Fatalf("native-pixel fallback forces scaling: %q", commands[0].control)
	}
	pngData := decodeKittyPayload(t, commands)
	rendered, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("payload is not a PNG: %v", err)
	}
	if rendered.Bounds().Dx() != 300 || rendered.Bounds().Dy() != 64 {
		t.Fatalf("native PNG size = %v", rendered.Bounds())
	}
}

func TestTransformKittyAtCellSizeMatchesPlacementPixels(t *testing.T) {
	cellSize := KittyCellSize{Width: 10, Height: 20}
	var output bytes.Buffer
	if err := TransformKittyAtCellSize(strings.NewReader("a[\"hello\"] -> b[\"world\"]"), &output, cellSize); err != nil {
		t.Fatal(err)
	}
	commands := parseKittyCommands(t, output.Bytes())
	columns := kittyControlInteger(t, commands[0].control, "c")
	rows := kittyControlInteger(t, commands[0].control, "r")
	pngData := decodeKittyPayload(t, commands)
	rendered, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := rendered.Bounds().Dx(), columns*cellSize.Width; got != want {
		t.Errorf("PNG width = %d, placement width = %d", got, want)
	}
	if got, want := rendered.Bounds().Dy(), rows*cellSize.Height; got != want {
		t.Errorf("PNG height = %d, placement height = %d", got, want)
	}
}

func TestTransformKittyAtCellSizeRejectsInvalidSize(t *testing.T) {
	err := TransformKittyAtCellSize(strings.NewReader("a"), &bytes.Buffer{}, KittyCellSize{})
	if err == nil {
		t.Fatal("accepted an invalid cell size")
	}
}

func decodeKittyPayload(t *testing.T, commands []kittyCommand) []byte {
	t.Helper()
	var encoded strings.Builder
	for i, command := range commands {
		if len(command.payload) > kittyChunkSize {
			t.Errorf("chunk %d has %d bytes", i, len(command.payload))
		}
		if i < len(commands)-1 && len(command.payload)%4 != 0 {
			t.Errorf("non-final chunk %d is not a multiple of four", i)
		}
		encoded.Write(command.payload)
	}
	pngData, err := base64.StdEncoding.DecodeString(encoded.String())
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return pngData
}

func kittyControlInteger(t *testing.T, control, key string) int {
	t.Helper()
	for _, field := range strings.Split(control, ",") {
		name, value, found := strings.Cut(field, "=")
		if found && name == key {
			integer, err := strconv.Atoi(value)
			if err != nil {
				t.Fatalf("invalid %s value in %q", key, control)
			}
			return integer
		}
	}
	t.Fatalf("missing %s value in %q", key, control)
	return 0
}

func TestKittyTransmissionChunksLargeImages(t *testing.T) {
	data := bytes.Repeat([]byte("boxl"), 3000)
	var output bytes.Buffer
	if err := writeKittyImage(&output, data, 10, 4); err != nil {
		t.Fatal(err)
	}
	commands := parseKittyCommands(t, output.Bytes())
	if len(commands) < 2 {
		t.Fatalf("got %d commands, want multiple chunks", len(commands))
	}
	for i, command := range commands {
		want := "m=1"
		if i == len(commands)-1 {
			want = "m=0"
		}
		if !strings.Contains(command.control, want) {
			t.Errorf("command %d control = %q, want %s", i, command.control, want)
		}
	}
}

func TestTransformKittyEmptyDiagram(t *testing.T) {
	var output bytes.Buffer
	if err := TransformKitty(strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("output = %q", output.Bytes())
	}
}

type kittyCommand struct {
	control string
	payload []byte
}

func parseKittyCommands(t *testing.T, stream []byte) []kittyCommand {
	t.Helper()
	const prefix = "\x1b_G"
	const suffix = "\x1b\\"
	var commands []kittyCommand
	for len(stream) > 0 {
		if !bytes.HasPrefix(stream, []byte(prefix)) {
			t.Fatalf("stream does not start with Kitty APC: %q", stream)
		}
		end := bytes.Index(stream, []byte(suffix))
		if end < 0 {
			t.Fatal("unterminated Kitty APC")
		}
		body := stream[len(prefix):end]
		separator := bytes.IndexByte(body, ';')
		if separator < 0 {
			t.Fatal("Kitty APC has no payload separator")
		}
		commands = append(commands, kittyCommand{
			control: string(body[:separator]), payload: bytes.Clone(body[separator+1:]),
		})
		stream = stream[end+len(suffix):]
	}
	return commands
}
