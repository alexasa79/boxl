package boxl

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/vector"
)

type imagePoint struct {
	x, y float64
}

func (p imagePoint) add(q imagePoint) imagePoint { return imagePoint{p.x + q.x, p.y + q.y} }
func (p imagePoint) sub(q imagePoint) imagePoint { return imagePoint{p.x - q.x, p.y - q.y} }
func (p imagePoint) mul(s float64) imagePoint    { return imagePoint{p.x * s, p.y * s} }
func (p imagePoint) length() float64             { return math.Hypot(p.x, p.y) }
func (p imagePoint) unit() imagePoint {
	if length := p.length(); length > 0 {
		return p.mul(1 / length)
	}
	return imagePoint{}
}

type imageConnectionRoute struct {
	points  []imagePoint
	tangent imagePoint
}

func imageBoxCenter(box imagePlacement) imagePoint {
	return imagePoint{
		float64(box.x) + float64(box.image.Bounds().Dx()-1)/2,
		float64(box.y) + float64(box.image.Bounds().Dy()-1)/2,
	}
}

func (r *imageRenderer) connectionAnchor(box imagePlacement, toward imagePoint) imagePoint {
	center := imageBoxCenter(box)
	direction := toward.sub(center).unit()
	radius := math.Min(
		float64(box.image.Bounds().Dx()-1)/(2*math.Abs(direction.x)),
		float64(box.image.Bounds().Dy()-1)/(2*math.Abs(direction.y)),
	)
	gap := math.Hypot(direction.x*float64(r.arrowGapX), direction.y*float64(r.arrowGapY))
	return center.add(direction.mul(radius + gap))
}

func (r *imageRenderer) routeConnections(children []*declaredBox, edges []connection, placements map[*declaredBox]imagePlacement) ([]imageConnectionRoute, error) {
	order := make(map[*declaredBox]int, len(children))
	boxes := make([]imagePlacement, 0, len(children))
	for i, child := range children {
		order[child] = i
		boxes = append(boxes, placements[child])
	}
	candidates := make([][]imageConnectionRoute, 0, len(edges))
	for _, edge := range edges {
		from, to := placements[edge.from], placements[edge.to]
		var options []imageConnectionRoute
		if order[edge.to] > order[edge.from] {
			start := r.connectionAnchor(from, imageBoxCenter(to))
			end := r.connectionAnchor(to, imageBoxCenter(from))
			straight := imageConnectionRoute{points: []imagePoint{start, end}, tangent: end.sub(start).unit()}
			if r.connectionIsClear(straight, boxes) {
				options = append(options, straight)
			}
		}
		options = append(options, r.curvedConnections(from, to, boxes)...)
		if len(options) == 0 {
			return nil, modelError(edge.position, "cannot route connection %q -> %q without crossing a box", edge.from.name, edge.to.name)
		}
		candidates = append(candidates, options)
	}
	return r.chooseConnectionRoutes(candidates), nil
}

func (r *imageRenderer) curvedConnections(from, to imagePlacement, boxes []imagePlacement) []imageConnectionRoute {
	startCenter, endCenter := imageBoxCenter(from), imageBoxCenter(to)
	delta := endCenter.sub(startCenter)
	direction := delta.unit()
	normal := imagePoint{-direction.y, direction.x}
	extent := func(box imagePlacement) float64 {
		return (math.Abs(normal.x)*float64(box.image.Bounds().Dx()) + math.Abs(normal.y)*float64(box.image.Bounds().Dy())) / 2
	}
	spacing := float64(max(r.arrowGapX, r.arrowGapY) * 4)
	bend := math.Max(float64(r.horizontalGap), math.Max(extent(from), extent(to))+spacing)
	var routes []imageConnectionRoute
	clearLevels := 0
	for attempt := 0; attempt < 24; attempt++ {
		found := false
		for _, side := range []float64{1, -1} {
			var start, c1, c2, end imagePoint
			if delta.length() == 0 {
				// A self-loop uses two distinct ports on the same box.
				halfWidth := float64(from.image.Bounds().Dx()-1) / 2
				halfHeight := float64(from.image.Bounds().Dy()-1) / 2
				start = startCenter.add(imagePoint{side * (halfWidth + float64(r.arrowGapX)), 0})
				end = startCenter.add(imagePoint{0, -side * (halfHeight + float64(r.arrowGapY))})
				c1 = start.add(imagePoint{side * bend, 0})
				c2 = end.add(imagePoint{0, -side * bend})
			} else {
				offset := normal.mul(side * bend)
				c1 = startCenter.add(delta.mul(1.0 / 3)).add(offset)
				c2 = endCenter.sub(delta.mul(1.0 / 3)).add(offset)
				start = r.connectionAnchor(from, c1)
				end = r.connectionAnchor(to, c2)
			}
			route := cubicConnection(start, c1, c2, end)
			if r.connectionIsClear(route, boxes) {
				routes = append(routes, route)
				found = true
			}
		}
		if found {
			clearLevels++
			if clearLevels == 6 {
				break
			}
		}
		bend *= 1.35
	}
	return routes
}

func cubicConnection(start, c1, c2, end imagePoint) imageConnectionRoute {
	length := c1.sub(start).length() + c2.sub(c1).length() + end.sub(c2).length()
	steps := min(512, max(24, int(math.Ceil(length/4))))
	points := make([]imagePoint, steps+1)
	for i := range points {
		t := float64(i) / float64(steps)
		s := 1 - t
		points[i] = start.mul(s * s * s).add(c1.mul(3 * s * s * t)).add(c2.mul(3 * s * t * t)).add(end.mul(t * t * t))
	}
	return imageConnectionRoute{points: points, tangent: end.sub(c2).unit()}
}

func connectionLength(points []imagePoint) float64 {
	var length float64
	for i := 1; i < len(points); i++ {
		length += points[i].sub(points[i-1]).length()
	}
	return length
}

func (r *imageRenderer) connectionArrowhead(route imageConnectionRoute) [3]imagePoint {
	tip := route.points[len(route.points)-1]
	direction := route.tangent
	length := math.Min(math.Hypot(direction.x*float64(r.arrowLengthX), direction.y*float64(r.arrowLengthY)), connectionLength(route.points)/2)
	halfWidth := math.Hypot(direction.y*float64(r.arrowHalfWidthX), direction.x*float64(r.arrowHalfWidthY))
	base := tip.sub(direction.mul(length))
	normal := imagePoint{-direction.y, direction.x}.mul(halfWidth)
	return [3]imagePoint{tip, base.add(normal), base.sub(normal)}
}

func (r *imageRenderer) connectionIsClear(route imageConnectionRoute, boxes []imagePlacement) bool {
	head := r.connectionArrowhead(route)
	for _, box := range boxes {
		bounds := box.image.Bounds().Add(image.Pt(box.x, box.y))
		for i := 1; i < len(route.points); i++ {
			if imageSegmentHitsBox(route.points[i-1], route.points[i], bounds, float64(r.lineWidth)/2+1) {
				return false
			}
		}
		for i := range head {
			if imageSegmentHitsBox(head[i], head[(i+1)%len(head)], bounds, 1) {
				return false
			}
		}
	}
	return true
}

func imageSegmentHitsBox(start, end imagePoint, bounds image.Rectangle, padding float64) bool {
	low, high := 0.0, 1.0
	for _, axis := range [][4]float64{
		{start.x, end.x - start.x, float64(bounds.Min.X) - padding, float64(bounds.Max.X-1) + padding},
		{start.y, end.y - start.y, float64(bounds.Min.Y) - padding, float64(bounds.Max.Y-1) + padding},
	} {
		origin, delta, minimum, maximum := axis[0], axis[1], axis[2], axis[3]
		if delta == 0 {
			if origin < minimum || origin > maximum {
				return false
			}
			continue
		}
		first, last := (minimum-origin)/delta, (maximum-origin)/delta
		if first > last {
			first, last = last, first
		}
		low, high = math.Max(low, first), math.Min(high, last)
		if low > high {
			return false
		}
	}
	return true
}

func (r *imageRenderer) connectionBounds(route imageConnectionRoute) image.Rectangle {
	padding := float64(max(r.arrowHalfWidthX, r.arrowHalfWidthY, r.lineWidth)) + 1
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, point := range route.points {
		minX, minY = math.Min(minX, point.x-padding), math.Min(minY, point.y-padding)
		maxX, maxY = math.Max(maxX, point.x+padding), math.Max(maxY, point.y+padding)
	}
	return image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1)
}

func (r *imageRenderer) drawConnectionRoute(target *image.RGBA, route imageConnectionRoute, offset imagePoint, foreground color.Color) {
	head := r.connectionArrowhead(route)
	base := head[1].add(head[2]).mul(0.5)
	remaining := connectionLength(route.points) - head[0].sub(base).length()
	shaft := []imagePoint{route.points[0]}
	for i := 1; i < len(route.points); i++ {
		segment := route.points[i].sub(route.points[i-1]).length()
		if segment >= remaining {
			break
		}
		shaft = append(shaft, route.points[i])
		remaining -= segment
	}
	shaft = append(shaft, base)
	left, right := make([]imagePoint, len(shaft)), make([]imagePoint, len(shaft))
	for i, point := range shaft {
		direction := shaft[min(i+1, len(shaft)-1)].sub(shaft[max(i-1, 0)]).unit()
		normal := imagePoint{-direction.y, direction.x}.mul(float64(r.lineWidth) / 2)
		left[i], right[i] = point.add(normal), point.sub(normal)
	}
	outline := append([]imagePoint{}, left...)
	for i := len(right) - 1; i >= 0; i-- {
		outline = append(outline, right[i])
	}
	drawConnectionPolygon(target, outline, offset, foreground)
	drawConnectionPolygon(target, head[:], offset, foreground)
}

func drawConnectionPolygon(target *image.RGBA, points []imagePoint, offset imagePoint, foreground color.Color) {
	rasterizer := vector.NewRasterizer(target.Bounds().Dx(), target.Bounds().Dy())
	for i, point := range points {
		x, y := float32(point.x+offset.x)+0.5, float32(point.y+offset.y)+0.5
		if i == 0 {
			rasterizer.MoveTo(x, y)
		} else {
			rasterizer.LineTo(x, y)
		}
	}
	rasterizer.ClosePath()
	rasterizer.Draw(target, target.Bounds(), image.NewUniform(foreground), image.Point{})
}
