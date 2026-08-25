package boxl

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// RenderOptions controls the spacing in the ASCII diagram.
type RenderOptions struct {
	HorizontalGap int
	VerticalGap   int
	PaddingX      int
	PaddingY      int
}

// DefaultRenderOptions returns the canonical boxl layout values.
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{HorizontalGap: 5, VerticalGap: 1, PaddingX: 1, PaddingY: 0}
}

// Renderer performs the declaration pass followed by the rendering pass.
type Renderer struct {
	options RenderOptions
}

// NewRenderer creates a renderer. HorizontalGap must leave room for an arrow.
func NewRenderer(options RenderOptions) (*Renderer, error) {
	if options.HorizontalGap < 3 {
		return nil, fmt.Errorf("horizontal gap must be at least 3")
	}
	if options.VerticalGap < 0 || options.PaddingX < 0 || options.PaddingY < 0 {
		return nil, fmt.Errorf("vertical gap and padding cannot be negative")
	}
	return &Renderer{options: options}, nil
}

// Render validates declarations in pass one and lays out ASCII boxes in pass two.
func (r *Renderer) Render(diagram *Diagram) (string, error) {
	m, err := declarationPass(diagram)
	if err != nil {
		return "", err
	}
	metrics := r.measureLeaves(m)
	result, err := r.renderScope(m, nil, metrics)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

type leafMetrics struct {
	innerWidth  int
	innerHeight int
}

type placement struct {
	x      int
	y      int
	canvas *canvas
}

func (r *Renderer) measureLeaves(m *model) leafMetrics {
	metrics := leafMetrics{innerWidth: 3, innerHeight: 1}
	for _, box := range m.order {
		if box.container {
			continue
		}
		for _, line := range labelLines(box.text) {
			metrics.innerWidth = max(metrics.innerWidth, utf8.RuneCountInString(line))
		}
		metrics.innerHeight = max(metrics.innerHeight, len(labelLines(box.text)))
	}
	return metrics
}

func (r *Renderer) renderScope(m *model, scope *declaredBox, metrics leafMetrics) (*canvas, error) {
	children := m.children(scope)
	if len(children) == 0 {
		return newCanvas(0, 0), nil
	}

	boxCanvases := make(map[*declaredBox]*canvas, len(children))
	for _, box := range children {
		var rendered *canvas
		if box.container {
			inside, err := r.renderScope(m, box, metrics)
			if err != nil {
				return nil, err
			}
			rendered = r.containerCanvas(inside)
		} else {
			rendered = r.leafCanvas(box, metrics)
		}
		boxCanvases[box] = rendered
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
		columnWidths[position.column] = max(columnWidths[position.column], boxCanvases[box].width)
		rowHeights[position.row] = max(rowHeights[position.row], boxCanvases[box].height)
	}
	columnX := make([]int, len(columnWidths))
	for column := 1; column < len(columnX); column++ {
		columnX[column] = columnX[column-1] + columnWidths[column-1] + r.options.HorizontalGap
	}
	rowY := make([]int, len(rowHeights))
	for row := 1; row < len(rowY); row++ {
		rowY[row] = rowY[row-1] + rowHeights[row-1] + r.options.VerticalGap
	}

	placements := make(map[*declaredBox]placement, len(children))
	width := 0
	height := 0
	for _, box := range children {
		position := grid[box]
		boxCanvas := boxCanvases[box]
		item := placement{
			x:      columnX[position.column] + (columnWidths[position.column]-boxCanvas.width)/2,
			y:      rowY[position.row] + (rowHeights[position.row]-boxCanvas.height)/2,
			canvas: boxCanvas,
		}
		placements[box] = item
		width = max(width, item.x+item.canvas.width)
		height = max(height, item.y+item.canvas.height)
	}

	result := newCanvas(width, height)
	for _, edge := range edges {
		r.drawConnection(result, placements[edge.from], placements[edge.to])
	}
	for _, box := range children {
		item := placements[box]
		result.overlay(item.canvas, item.x, item.y)
	}
	return result, nil
}

func (r *Renderer) leafCanvas(box *declaredBox, metrics leafMetrics) *canvas {
	width := metrics.innerWidth + 2*r.options.PaddingX + 2
	height := metrics.innerHeight + 2*r.options.PaddingY + 2
	c := newCanvas(width, height)
	drawBorder(c)
	lines := labelLines(box.text)
	startY := 1 + r.options.PaddingY + (metrics.innerHeight-len(lines))/2
	for i, line := range lines {
		runes := []rune(line)
		startX := 1 + r.options.PaddingX + (metrics.innerWidth-len(runes))/2
		for j, value := range runes {
			c.set(startX+j, startY+i, value)
		}
	}
	return c
}

func (r *Renderer) containerCanvas(inside *canvas) *canvas {
	insideWidth := max(inside.width, 3)
	insideHeight := max(inside.height, 1)
	c := newCanvas(insideWidth+2*r.options.PaddingX+2, insideHeight+2*r.options.PaddingY+2)
	drawBorder(c)
	c.overlay(inside, 1+r.options.PaddingX, 1+r.options.PaddingY)
	return c
}

func (r *Renderer) drawConnection(target *canvas, from, to placement) {
	fromCenterX := from.x + from.canvas.width/2
	fromCenterY := from.y + from.canvas.height/2
	toCenterX := to.x + to.canvas.width/2
	toCenterY := to.y + to.canvas.height/2
	if abs(toCenterX-fromCenterX) >= abs(toCenterY-fromCenterY) {
		if toCenterX >= fromCenterX {
			r.drawHorizontalConnection(target,
				from.x+from.canvas.width-1, fromCenterY, to.x, toCenterY, 1, '>')
		} else {
			r.drawHorizontalConnection(target,
				from.x, fromCenterY, to.x+to.canvas.width-1, toCenterY, -1, '<')
		}
		return
	}
	if toCenterY >= fromCenterY {
		r.drawVerticalConnection(target,
			fromCenterX, from.y+from.canvas.height-1, toCenterX, to.y, 1, 'v')
	} else {
		r.drawVerticalConnection(target,
			fromCenterX, from.y, toCenterX, to.y+to.canvas.height-1, -1, '^')
	}
}

func (r *Renderer) drawHorizontalConnection(target *canvas, fromX, fromY, toX, toY, direction int, arrow rune) {
	startX := fromX + direction
	endX := toX - direction
	if (endX-startX)*direction >= 2 {
		startX += direction
		endX -= direction
	}
	if (endX-startX)*direction < 0 {
		return
	}
	if fromY == toY {
		drawCanvasHorizontal(target, startX, endX, fromY)
		target.connector(endX, toY, arrow)
		return
	}
	middleX := (startX + endX) / 2
	drawCanvasHorizontal(target, startX, middleX, fromY)
	drawCanvasVertical(target, middleX, fromY, toY)
	drawCanvasHorizontal(target, middleX, endX, toY)
	target.connector(endX, toY, arrow)
}

func (r *Renderer) drawVerticalConnection(target *canvas, fromX, fromY, toX, toY, direction int, arrow rune) {
	startY := fromY + direction
	endY := toY - direction
	if (endY-startY)*direction >= 2 {
		startY += direction
		endY -= direction
	}
	if (endY-startY)*direction < 0 {
		return
	}
	if fromX == toX {
		drawCanvasVertical(target, fromX, startY, endY)
		target.connector(toX, endY, arrow)
		return
	}
	middleY := (startY + endY) / 2
	drawCanvasVertical(target, fromX, startY, middleY)
	drawCanvasHorizontal(target, fromX, toX, middleY)
	drawCanvasVertical(target, toX, middleY, endY)
	target.connector(toX, endY, arrow)
}

func drawCanvasHorizontal(target *canvas, x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		target.connector(x, y, '-')
	}
}

func drawCanvasVertical(target *canvas, x, y1, y2 int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		target.connector(x, y, '|')
	}
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
