package boxl

import (
	"image"
	"math"
	"testing"
)

func testConnectionRoutes(t *testing.T, source string) (*imageRenderer, []imageConnectionRoute, []imagePlacement) {
	return testConnectionRoutesWithBoxSize(t, source, imageEmptyBoxSize, imageEmptyBoxSize)
}

func testConnectionRoutesWithBoxSize(t *testing.T, source string, width, height int) (*imageRenderer, []imageConnectionRoute, []imagePlacement) {
	t.Helper()
	diagram, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	m, err := declarationPass(diagram)
	if err != nil {
		t.Fatal(err)
	}
	r, err := newImageRenderer(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.face.Close() })
	children := m.children(nil)
	grid, err := layoutBoxes(children, m.layouts[nil], nil)
	if err != nil {
		t.Fatal(err)
	}
	placements := make(map[*declaredBox]imagePlacement, len(children))
	boxes := make([]imagePlacement, 0, len(children))
	for _, child := range children {
		position := grid[child]
		box := imagePlacement{
			x:     position.column * (width + imageHorizontalGap),
			y:     position.row * (height + imageVerticalGap),
			image: image.NewRGBA(image.Rect(0, 0, width, height)),
		}
		placements[child] = box
		boxes = append(boxes, box)
	}
	routes, err := r.routeConnections(children, m.edges, placements)
	if err != nil {
		t.Fatal(err)
	}
	return r, routes, boxes
}

func TestConnectionsFollowOrderAndAvoidBoxes(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		curved []bool
	}{
		{"diagonal cycle", "#dir:diag\na -> b -> c\nc -> a", []bool{false, false, true}},
		{"row cycle", "a -> b -> c\nc -> a", []bool{false, false, true}},
		{"column cycle", "#dir:col\na -> b -> c\nc -> a", []bool{false, false, true}},
		{"diagonal skip", "#dir:diag\na b c\na -> c", []bool{true}},
		{"row skip", "a b c\na -> c", []bool{true}},
		{"clear grid diagonal", "#dir:grid:2x2\na b c d\na -> d", []bool{false}},
		{"blocked grid diagonal", "#dir:grid:3x3\na b c d e f g h i\na -> i", []bool{true}},
		{"grid interior return", "#dir:grid:3x3\na b c d e f g h i\nh -> b", []bool{true}},
		{"grid corner return", "#dir:grid:3x3\na b c d e f g h i\ne -> a", []bool{true}},
		{"bidirectional", "a -> b\nb -> a", []bool{false, true}},
		{"self loop", "a -> a", []bool{true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, routes, boxes := testConnectionRoutes(t, test.source)
			if len(routes) != len(test.curved) {
				t.Fatalf("routes = %d, want %d", len(routes), len(test.curved))
			}
			for i, route := range routes {
				if curved := len(route.points) > 2; curved != test.curved[i] {
					t.Errorf("route %d curved = %v, want %v", i, curved, test.curved[i])
				}
				// Every sampled point must lie outside every box, with a visible gap.
				for _, point := range route.points {
					for _, box := range boxes {
						dx := math.Max(0, math.Max(float64(box.x)-point.x, point.x-float64(box.x+imageEmptyBoxSize-1)))
						dy := math.Max(0, math.Max(float64(box.y)-point.y, point.y-float64(box.y+imageEmptyBoxSize-1)))
						if distance := math.Hypot(dx, dy); distance < 3 {
							t.Fatalf("route %d approaches a box too closely at %+v: distance %.2f", i, point, distance)
						}
					}
				}
			}
			assertRenderedRoutesClearBoxes(t, r, routes, boxes)
		})
	}
}

func assertRenderedRoutesClearBoxes(t *testing.T, r *imageRenderer, routes []imageConnectionRoute, boxes []imagePlacement) {
	t.Helper()
	var bounds image.Rectangle
	for _, box := range boxes {
		bounds = bounds.Union(box.image.Bounds().Add(image.Pt(box.x, box.y)))
	}
	for _, route := range routes {
		bounds = bounds.Union(r.connectionBounds(route))
	}
	// Extra canvas around the measured bounds reveals strokes that would be clipped.
	const margin = 8
	canvas := image.NewRGBA(image.Rect(0, 0, bounds.Dx()+2*margin, bounds.Dy()+2*margin))
	offset := imagePoint{float64(margin - bounds.Min.X), float64(margin - bounds.Min.Y)}
	for _, route := range routes {
		r.drawConnectionRoute(canvas, route, offset, imageStroke)
	}
	for _, box := range boxes {
		for y := 0; y < box.image.Bounds().Dy(); y++ {
			for x := 0; x < box.image.Bounds().Dx(); x++ {
				pixel := canvas.RGBAAt(box.x+x+int(offset.x), box.y+y+int(offset.y))
				if pixel.A != 0 {
					t.Fatalf("arrow crosses a box at (%d, %d)", box.x+x, box.y+y)
				}
			}
		}
	}
	for y := 0; y < canvas.Bounds().Dy(); y++ {
		for x := 0; x < canvas.Bounds().Dx(); x++ {
			if x >= margin && x < canvas.Bounds().Dx()-margin && y >= margin && y < canvas.Bounds().Dy()-margin {
				continue
			}
			if canvas.RGBAAt(x, y).A != 0 {
				t.Fatalf("arrow extends beyond its measured bounds at (%d, %d)", x, y)
			}
		}
	}
}

func TestReturnCurvesUseSeparateLanes(t *testing.T) {
	_, routes, _ := testConnectionRoutes(t, "#dir:diag\na -> b -> c\nc -> a\nc -> a")
	first, second := routes[2], routes[3]
	firstMiddle := first.points[len(first.points)/2]
	secondMiddle := second.points[len(second.points)/2]
	if distance := secondMiddle.sub(firstMiddle).length(); distance < 8 {
		t.Fatalf("return curves overlap: midpoint separation %.2f", distance)
	}
}

func TestCurveChoosesClearSide(t *testing.T) {
	r, _, boxes := testConnectionRoutes(t, "a b c")
	obstacle := imagePlacement{x: 80, y: -90, image: image.NewRGBA(image.Rect(0, 0, 90, 80))}
	routes := r.curvedConnections(boxes[2], boxes[0], append(boxes, obstacle))
	if len(routes) == 0 {
		t.Fatal("no route around obstacle")
	}
	route := r.chooseConnectionRoutes([][]imageConnectionRoute{routes})[0]
	if middle := route.points[len(route.points)/2]; middle.y <= imageEmptyBoxSize {
		t.Fatalf("curve did not choose the clear lower side: %+v", middle)
	}
}

func TestCurvedArrowheadFollowsEndpointTangent(t *testing.T) {
	r, routes, _ := testConnectionRoutes(t, "#dir:diag\na -> b -> c\nc -> a")
	route := routes[2]
	head := r.connectionArrowhead(route)
	base := head[1].add(head[2]).mul(0.5)
	direction := head[0].sub(base).unit()
	if alignment := direction.x*route.tangent.x + direction.y*route.tangent.y; alignment < 0.999 {
		t.Fatalf("arrowhead is not aligned with endpoint tangent: %.3f", alignment)
	}
	if math.Abs(direction.x) < 0.01 && math.Abs(direction.y) < 0.01 {
		t.Fatal("arrowhead has no direction")
	}
}
