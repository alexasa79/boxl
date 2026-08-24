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

	ranks, err := layoutRanks(children, m.connections(scope), scope)
	if err != nil {
		return nil, err
	}
	maxRank := 0
	for _, rank := range ranks {
		maxRank = max(maxRank, rank)
	}
	columnWidths := make([]int, maxRank+1)
	for _, box := range children {
		columnWidths[ranks[box]] = max(columnWidths[ranks[box]], boxCanvases[box].width)
	}
	columnX := make([]int, len(columnWidths))
	for rank := 1; rank < len(columnX); rank++ {
		columnX[rank] = columnX[rank-1] + columnWidths[rank-1] + r.options.HorizontalGap
	}

	placements := make(map[*declaredBox]placement, len(children))
	columnY := make([]int, len(columnWidths))
	width := 0
	height := 0
	for _, box := range children {
		rank := ranks[box]
		item := placement{x: columnX[rank], y: columnY[rank], canvas: boxCanvases[box]}
		placements[box] = item
		columnY[rank] += item.canvas.height + r.options.VerticalGap
		width = max(width, item.x+item.canvas.width)
		height = max(height, item.y+item.canvas.height)
	}

	result := newCanvas(width, height)
	for _, edge := range m.connections(scope) {
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
	fromX := from.x + from.canvas.width - 1
	fromY := from.y + from.canvas.height/2
	toX := to.x
	toY := to.y + to.canvas.height/2
	startX := fromX + 1
	endX := toX - 1
	if endX-startX >= 2 {
		startX++
		endX--
	}
	if startX > endX {
		return
	}
	if fromY == toY {
		for x := startX; x < endX; x++ {
			target.connector(x, fromY, '-')
		}
		target.connector(endX, toY, '>')
		return
	}

	middleX := (startX + endX) / 2
	for x := startX; x <= middleX; x++ {
		target.connector(x, fromY, '-')
	}
	minY, maxY := fromY, toY
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	for y := minY; y <= maxY; y++ {
		target.connector(middleX, y, '|')
	}
	for x := middleX; x < endX; x++ {
		target.connector(x, toY, '-')
	}
	target.connector(endX, toY, '>')
}

func layoutRanks(children []*declaredBox, edges []connection, scope *declaredBox) (map[*declaredBox]int, error) {
	ranks := make(map[*declaredBox]int, len(children))
	indegree := make(map[*declaredBox]int, len(children))
	adjacency := make(map[*declaredBox][]*declaredBox, len(children))
	for _, box := range children {
		indegree[box] = 0
	}
	for _, edge := range edges {
		adjacency[edge.from] = append(adjacency[edge.from], edge.to)
		indegree[edge.to]++
	}
	queue := make([]*declaredBox, 0, len(children))
	for _, box := range children {
		if indegree[box] == 0 {
			queue = append(queue, box)
		}
	}
	processed := 0
	for len(queue) > 0 {
		box := queue[0]
		queue = queue[1:]
		processed++
		for _, next := range adjacency[box] {
			ranks[next] = max(ranks[next], ranks[box]+1)
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if processed != len(children) {
		where := "top-level diagram"
		if scope != nil {
			where = fmt.Sprintf("container %q", scope.name)
		}
		return nil, fmt.Errorf("cannot lay out connection cycle in %s", where)
	}
	return ranks, nil
}

func labelLines(text *string) []string {
	if text == nil {
		return []string{""}
	}
	return strings.Split(strings.ReplaceAll(*text, "\t", "    "), "\n")
}
