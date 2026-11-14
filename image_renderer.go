package boxl

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

const (
	imageFontSize       = 14
	imageEmptyBoxSize   = 40
	imageLeafMinWidth   = 112
	imageLeafMinHeight  = 52
	imageTextPaddingX   = 16
	imageTextPaddingY   = 12
	imageContainerPad   = 24
	imageHorizontalGap  = 64
	imageVerticalGap    = 28
	imageLineWidth      = 2
	imageArrowLength    = 11
	imageArrowHalfWidth = 6
	imageArrowBoxGap    = 2
	imageOuterMargin    = 6
	kittyCellWidth      = 8
	kittyCellHeight     = 16
	kittyImageScale     = 1.0
)

var (
	imageBackground    = color.RGBA{}
	imageBoxFill       = color.RGBA{}
	imageContainerFill = color.RGBA{}
	imageStroke        = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

type imageRenderer struct {
	face              font.Face
	ascent            int
	lineHeight        int
	emptyBoxWidth     int
	emptyBoxHeight    int
	leafMinWidth      int
	leafMinHeight     int
	textPaddingX      int
	textPaddingY      int
	containerPaddingX int
	containerPaddingY int
	horizontalGap     int
	verticalGap       int
	lineWidth         int
	arrowLengthX      int
	arrowLengthY      int
	arrowHalfWidthX   int
	arrowHalfWidthY   int
	outerMarginX      int
	outerMarginY      int
}

type imageLeafMetrics struct {
	width  int
	height int
}

type imagePlacement struct {
	x     int
	y     int
	image *image.RGBA
}

func renderPNGSource(source string) ([]byte, error) {
	rendered, err := renderImageSource(source, 1, 1)
	if err != nil || rendered == nil {
		return nil, err
	}
	return encodePNG(rendered)
}

func renderKittyPNGSource(source string, cellSize KittyCellSize) ([]byte, int, int, error) {
	scaleX, scaleY := kittyImageScale, kittyImageScale
	if cellSize.valid() {
		scaleX *= float64(cellSize.Width) / kittyCellWidth
		scaleY *= float64(cellSize.Height) / kittyCellHeight
	}
	rendered, err := renderImageSource(source, scaleX, scaleY)
	if err != nil || rendered == nil {
		return nil, 0, 0, err
	}
	if !cellSize.valid() {
		data, err := encodePNG(rendered)
		return data, 0, 0, err
	}

	columns := (rendered.Bounds().Dx() + cellSize.Width - 1) / cellSize.Width
	rows := (rendered.Bounds().Dy() + cellSize.Height - 1) / cellSize.Height
	targetWidth := columns * cellSize.Width
	targetHeight := rows * cellSize.Height
	if targetWidth != rendered.Bounds().Dx() || targetHeight != rendered.Bounds().Dy() {
		padded := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
		fillImage(padded, imageBackground)
		left := (targetWidth - rendered.Bounds().Dx()) / 2
		top := (targetHeight - rendered.Bounds().Dy()) / 2
		draw.Draw(padded, rendered.Bounds().Add(image.Pt(left, top)), rendered, image.Point{}, draw.Over)
		rendered = padded
	}
	data, err := encodePNG(rendered)
	return data, columns, rows, err
}

func renderImageSource(source string, scaleX, scaleY float64) (*image.RGBA, error) {
	diagram, err := Parse(source)
	if err != nil {
		return nil, err
	}
	m, err := declarationPass(diagram)
	if err != nil {
		return nil, err
	}
	if len(m.order) == 0 {
		return nil, nil
	}
	renderer, err := newImageRenderer(scaleX, scaleY)
	if err != nil {
		return nil, err
	}
	defer renderer.face.Close()

	diagramImage, err := renderer.renderScope(m, nil, renderer.measureLeaves(m))
	if err != nil {
		return nil, err
	}
	width := diagramImage.Bounds().Dx() + 2*renderer.outerMarginX
	height := diagramImage.Bounds().Dy() + 2*renderer.outerMarginY
	output := image.NewRGBA(image.Rect(0, 0, width, height))
	fillImage(output, imageBackground)
	draw.Draw(output, diagramImage.Bounds().Add(image.Pt(renderer.outerMarginX, renderer.outerMarginY)), diagramImage, image.Point{}, draw.Over)
	return output, nil
}

func encodePNG(rendered *image.RGBA) ([]byte, error) {
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&encoded, rendered); err != nil {
		return nil, fmt.Errorf("encode rendered diagram as PNG: %w", err)
	}
	return encoded.Bytes(), nil
}

func newImageRenderer(scaleX, scaleY float64) (*imageRenderer, error) {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse embedded diagram font: %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size: imageFontSize * scaleY, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("create diagram font: %w", err)
	}
	metrics := face.Metrics()
	strokeScale := min(scaleX, scaleY)
	return &imageRenderer{
		face: face, ascent: metrics.Ascent.Ceil(), lineHeight: metrics.Height.Ceil(),
		emptyBoxWidth:     scaledPixel(imageEmptyBoxSize, scaleX),
		emptyBoxHeight:    scaledPixel(imageEmptyBoxSize, scaleY),
		leafMinWidth:      scaledPixel(imageLeafMinWidth, scaleX),
		leafMinHeight:     scaledPixel(imageLeafMinHeight, scaleY),
		textPaddingX:      scaledPixel(imageTextPaddingX, scaleX),
		textPaddingY:      scaledPixel(imageTextPaddingY, scaleY),
		containerPaddingX: scaledPixel(imageContainerPad, scaleX),
		containerPaddingY: scaledPixel(imageContainerPad, scaleY),
		horizontalGap:     scaledPixel(imageHorizontalGap, scaleX),
		verticalGap:       scaledPixel(imageVerticalGap, scaleY),
		lineWidth:         scaledPixel(imageLineWidth, strokeScale),
		arrowLengthX:      scaledPixel(imageArrowLength, scaleX),
		arrowLengthY:      scaledPixel(imageArrowLength, scaleY),
		arrowHalfWidthX:   scaledPixel(imageArrowHalfWidth, scaleX),
		arrowHalfWidthY:   scaledPixel(imageArrowHalfWidth, scaleY),
		outerMarginX:      scaledPixel(imageOuterMargin, scaleX),
		outerMarginY:      scaledPixel(imageOuterMargin, scaleY),
	}, nil
}

func scaledPixel(value int, scale float64) int {
	return max(1, int(float64(value)*scale))
}

func (r *imageRenderer) measureLeaves(m *model) imageLeafMetrics {
	metrics := imageLeafMetrics{width: r.leafMinWidth, height: r.leafMinHeight}
	for _, box := range m.order {
		if box.container || isEmptyImageBox(box) {
			continue
		}
		lines := labelLines(box.text)
		for _, line := range lines {
			metrics.width = max(metrics.width, font.MeasureString(r.face, line).Ceil()+2*r.textPaddingX)
		}
		metrics.height = max(metrics.height, len(lines)*r.lineHeight+2*r.textPaddingY)
	}
	return metrics
}

func (r *imageRenderer) renderScope(m *model, scope *declaredBox, metrics imageLeafMetrics) (*image.RGBA, error) {
	children := m.children(scope)
	if len(children) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0)), nil
	}

	boxImages := make(map[*declaredBox]*image.RGBA, len(children))
	for _, box := range children {
		if box.container {
			inside, err := r.renderScope(m, box, metrics)
			if err != nil {
				return nil, err
			}
			boxImages[box] = r.containerImage(inside)
		} else {
			boxImages[box] = r.leafImage(box, metrics)
		}
	}

	edges := m.connections(scope)
	grid, err := layoutBoxes(children, m.layoutConstraints(scope), edges, scope)
	if err != nil {
		return nil, err
	}
	maxColumn, maxRow := 0, 0
	for _, position := range grid {
		maxColumn = max(maxColumn, position.column)
		maxRow = max(maxRow, position.row)
	}
	columnWidths := make([]int, maxColumn+1)
	rowHeights := make([]int, maxRow+1)
	for _, box := range children {
		position := grid[box]
		boxBounds := boxImages[box].Bounds()
		columnWidths[position.column] = max(columnWidths[position.column], boxBounds.Dx())
		rowHeights[position.row] = max(rowHeights[position.row], boxBounds.Dy())
	}
	columnX := make([]int, len(columnWidths))
	for column := 1; column < len(columnX); column++ {
		columnX[column] = columnX[column-1] + columnWidths[column-1] + r.horizontalGap
	}
	rowY := make([]int, len(rowHeights))
	for row := 1; row < len(rowY); row++ {
		rowY[row] = rowY[row-1] + rowHeights[row-1] + r.verticalGap
	}

	placements := make(map[*declaredBox]imagePlacement, len(children))
	width, height := 0, 0
	for _, box := range children {
		position := grid[box]
		boxImage := boxImages[box]
		placement := imagePlacement{
			x:     columnX[position.column] + (columnWidths[position.column]-boxImage.Bounds().Dx())/2,
			y:     rowY[position.row] + (rowHeights[position.row]-boxImage.Bounds().Dy())/2,
			image: boxImage,
		}
		placements[box] = placement
		width = max(width, placement.x+boxImage.Bounds().Dx())
		height = max(height, placement.y+boxImage.Bounds().Dy())
	}

	result := image.NewRGBA(image.Rect(0, 0, width, height))
	for _, edge := range edges {
		r.drawImageConnection(result, placements[edge.from], placements[edge.to])
	}
	for _, box := range children {
		placement := placements[box]
		draw.Draw(result, placement.image.Bounds().Add(image.Pt(placement.x, placement.y)), placement.image, image.Point{}, draw.Over)
	}
	return result, nil
}

func (r *imageRenderer) leafImage(box *declaredBox, metrics imageLeafMetrics) *image.RGBA {
	if isEmptyImageBox(box) {
		result := image.NewRGBA(image.Rect(0, 0, r.emptyBoxWidth, r.emptyBoxHeight))
		r.drawImageBox(result, imageBoxFill)
		return result
	}
	result := image.NewRGBA(image.Rect(0, 0, metrics.width, metrics.height))
	r.drawImageBox(result, imageBoxFill)
	lines := labelLines(box.text)
	textHeight := len(lines) * r.lineHeight
	firstBaseline := (metrics.height-textHeight)/2 + r.ascent
	drawer := font.Drawer{Dst: result, Src: image.NewUniform(imageStroke), Face: r.face}
	for i, line := range lines {
		textWidth := font.MeasureString(r.face, line).Ceil()
		drawer.Dot = fixed.P((metrics.width-textWidth)/2, firstBaseline+i*r.lineHeight)
		drawer.DrawString(line)
	}
	return result
}

func (r *imageRenderer) containerImage(inside *image.RGBA) *image.RGBA {
	if inside.Bounds().Empty() {
		result := image.NewRGBA(image.Rect(0, 0, r.emptyBoxWidth, r.emptyBoxHeight))
		r.drawImageBox(result, imageContainerFill)
		return result
	}
	width := inside.Bounds().Dx() + 2*r.containerPaddingX
	height := inside.Bounds().Dy() + 2*r.containerPaddingY
	result := image.NewRGBA(image.Rect(0, 0, width, height))
	r.drawImageBox(result, imageContainerFill)
	left := (width - inside.Bounds().Dx()) / 2
	top := (height - inside.Bounds().Dy()) / 2
	draw.Draw(result, inside.Bounds().Add(image.Pt(left, top)), inside, image.Point{}, draw.Over)
	return result
}

func isEmptyImageBox(box *declaredBox) bool {
	return box.text == nil || *box.text == ""
}

func (r *imageRenderer) drawImageBox(target *image.RGBA, fill color.Color) {
	fillImage(target, fill)
	bounds := target.Bounds()
	fillRectangle(target, image.Rect(0, 0, bounds.Dx(), r.lineWidth), imageStroke)
	fillRectangle(target, image.Rect(0, bounds.Dy()-r.lineWidth, bounds.Dx(), bounds.Dy()), imageStroke)
	fillRectangle(target, image.Rect(0, 0, r.lineWidth, bounds.Dy()), imageStroke)
	fillRectangle(target, image.Rect(bounds.Dx()-r.lineWidth, 0, bounds.Dx(), bounds.Dy()), imageStroke)
}

func (r *imageRenderer) drawImageConnection(target *image.RGBA, from, to imagePlacement) {
	fromCenterX := from.x + from.image.Bounds().Dx()/2
	fromCenterY := from.y + from.image.Bounds().Dy()/2
	toCenterX := to.x + to.image.Bounds().Dx()/2
	toCenterY := to.y + to.image.Bounds().Dy()/2
	if abs(toCenterX-fromCenterX) >= abs(toCenterY-fromCenterY) {
		direction := 1
		fromX := from.x + from.image.Bounds().Dx() - 1
		toX := to.x
		if toCenterX < fromCenterX {
			direction = -1
			fromX = from.x
			toX = to.x + to.image.Bounds().Dx() - 1
		}
		tipX := toX - direction*imageArrowBoxGap
		baseX := tipX - direction*r.arrowLengthX
		middleX := (fromX + baseX) / 2
		r.drawHorizontalImageLine(target, fromX, middleX, fromCenterY)
		r.drawVerticalImageLine(target, middleX, fromCenterY, toCenterY)
		r.drawHorizontalImageLine(target, middleX, baseX, toCenterY)
		r.drawImageArrowhead(target, tipX, toCenterY, direction, 0, r.arrowLengthX, r.arrowHalfWidthY)
		return
	}

	direction := 1
	fromY := from.y + from.image.Bounds().Dy() - 1
	toY := to.y
	if toCenterY < fromCenterY {
		direction = -1
		fromY = from.y
		toY = to.y + to.image.Bounds().Dy() - 1
	}
	tipY := toY - direction*imageArrowBoxGap
	baseY := tipY - direction*r.arrowLengthY
	middleY := (fromY + baseY) / 2
	r.drawVerticalImageLine(target, fromCenterX, fromY, middleY)
	r.drawHorizontalImageLine(target, fromCenterX, toCenterX, middleY)
	r.drawVerticalImageLine(target, toCenterX, middleY, baseY)
	r.drawImageArrowhead(target, toCenterX, tipY, 0, direction, r.arrowLengthY, r.arrowHalfWidthX)
}

func (r *imageRenderer) drawHorizontalImageLine(target *image.RGBA, x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	fillRectangle(target, image.Rect(x1, y-r.lineWidth/2, x2+1, y+(r.lineWidth+1)/2), imageStroke)
}

func (r *imageRenderer) drawVerticalImageLine(target *image.RGBA, x, y1, y2 int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	fillRectangle(target, image.Rect(x-r.lineWidth/2, y1, x+(r.lineWidth+1)/2, y2+1), imageStroke)
}

func (r *imageRenderer) drawImageArrowhead(target *image.RGBA, tipX, tipY, directionX, directionY, length, halfWidth int) {
	baseX := tipX - directionX*length
	baseY := tipY - directionY*length
	perpendicularX := -directionY * halfWidth
	perpendicularY := directionX * halfWidth
	rasterizer := vector.NewRasterizer(target.Bounds().Dx(), target.Bounds().Dy())
	rasterizer.MoveTo(float32(tipX)+0.5, float32(tipY)+0.5)
	rasterizer.LineTo(float32(baseX+perpendicularX)+0.5, float32(baseY+perpendicularY)+0.5)
	rasterizer.LineTo(float32(baseX-perpendicularX)+0.5, float32(baseY-perpendicularY)+0.5)
	rasterizer.ClosePath()
	rasterizer.Draw(target, target.Bounds(), image.NewUniform(imageStroke), image.Point{})
}

func fillImage(target *image.RGBA, fill color.Color) {
	draw.Draw(target, target.Bounds(), image.NewUniform(fill), image.Point{}, draw.Src)
}

func fillRectangle(target *image.RGBA, rectangle image.Rectangle, fill color.Color) {
	draw.Draw(target, rectangle.Intersect(target.Bounds()), image.NewUniform(fill), image.Point{}, draw.Src)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func labelLines(text *string) []string {
	if text == nil {
		return []string{""}
	}
	return strings.Split(strings.ReplaceAll(*text, "\t", "    "), "\n")
}
