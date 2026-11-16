package boxl

import (
	"image"
	"testing"
)

func testStraightRoute(points ...imagePoint) imageConnectionRoute {
	return imageConnectionRoute{points: points, tangent: points[len(points)-1].sub(points[len(points)-2]).unit()}
}

func testRoutingConflicts(routes []imageConnectionRoute) imageRoutingScore {
	var score imageRoutingScore
	for i, route := range routes {
		for j := 0; j < i; j++ {
			score = score.add(imageRouteIntersections(makeImageRouteOption(route, i), makeImageRouteOption(routes[j], j)))
		}
	}
	return score
}

func TestRouteIntersectionCounts(t *testing.T) {
	for _, test := range []struct {
		name      string
		first     imageConnectionRoute
		second    imageConnectionRoute
		crossings int
		overlaps  int
	}{
		{"crossing", testStraightRoute(imagePoint{0, 0}, imagePoint{10, 10}), testStraightRoute(imagePoint{0, 10}, imagePoint{10, 0}), 1, 0},
		{"sampled crossing", testStraightRoute(imagePoint{0, 0}, imagePoint{5, 5}, imagePoint{10, 10}), testStraightRoute(imagePoint{0, 10}, imagePoint{5, 5}, imagePoint{10, 0}), 1, 0},
		{"separate", testStraightRoute(imagePoint{0, 0}, imagePoint{10, 0}), testStraightRoute(imagePoint{0, 1}, imagePoint{10, 1}), 0, 0},
		{"overlap", testStraightRoute(imagePoint{0, 0}, imagePoint{10, 0}), testStraightRoute(imagePoint{8, 0}, imagePoint{2, 0}), 0, 1},
		{"touching endpoints", testStraightRoute(imagePoint{0, 0}, imagePoint{10, 0}), testStraightRoute(imagePoint{10, 0}, imagePoint{10, 10}), 1, 0},
		{"multiple crossings", testStraightRoute(imagePoint{0, 5}, imagePoint{30, 5}), testStraightRoute(imagePoint{0, 0}, imagePoint{10, 10}, imagePoint{20, 0}, imagePoint{30, 10}), 3, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			score := imageRouteIntersections(makeImageRouteOption(test.first, 0), makeImageRouteOption(test.second, 1))
			if score.crossings != test.crossings || score.overlaps != test.overlaps {
				t.Fatalf("conflicts = %+v, want %d crossings and %d overlaps", score, test.crossings, test.overlaps)
			}
		})
	}
}

func TestDiagonalConnectionsSpreadWithoutCrossings(t *testing.T) {
	for _, returns := range []string{"c -> a b -> a a -> c", "a -> c b -> a c -> a", "b -> a a -> c c -> a"} {
		source := `a["A"] -> b["B"] -> c["C"] #dir:diag ` + returns
		for _, size := range [][2]int{{imageEmptyBoxSize, imageEmptyBoxSize}, {imageLeafMinWidth, imageLeafMinHeight}} {
			r, routes, boxes := testConnectionRoutesWithBoxSize(t, source, size[0], size[1])
			if score := testRoutingConflicts(routes); score.crossings != 0 || score.overlaps != 0 {
				t.Fatalf("avoidable intersections for boxes %v, %q: %+v", size, returns, score)
			}
			if len(routes[0].points) != 2 || len(routes[1].points) != 2 {
				t.Fatal("primary diagonal connections should stay straight")
			}
			assertRenderedRoutesClearBoxes(t, r, routes, boxes)
			assertRenderedRoutesDoNotOverlap(t, r, routes)
		}
	}
}

func assertRenderedRoutesDoNotOverlap(t *testing.T, r *imageRenderer, routes []imageConnectionRoute) {
	t.Helper()
	var bounds image.Rectangle
	for _, route := range routes {
		bounds = bounds.Union(r.connectionBounds(route))
	}
	owners := make([]bool, bounds.Dx()*bounds.Dy())
	for i, route := range routes {
		canvas := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		r.drawConnectionRoute(canvas, route, imagePoint{float64(-bounds.Min.X), float64(-bounds.Min.Y)})
		for y := 0; y < canvas.Bounds().Dy(); y++ {
			for x := 0; x < canvas.Bounds().Dx(); x++ {
				if canvas.RGBAAt(x, y).A <= 32 {
					continue
				}
				index := y*bounds.Dx() + x
				if owners[index] {
					t.Fatalf("route %d overlaps an earlier arrow at (%d, %d)", i, x, y)
				}
				owners[index] = true
			}
		}
	}
}

func TestRouteSelectionEscapesGreedyMinimum(t *testing.T) {
	r, _, _ := testConnectionRoutes(t, "a")
	candidates := [][]imageConnectionRoute{
		{testStraightRoute(imagePoint{0, 0}, imagePoint{10, 0}), testStraightRoute(imagePoint{0, 20}, imagePoint{10, 20})},
		{testStraightRoute(imagePoint{5, -5}, imagePoint{5, 5}), testStraightRoute(imagePoint{15, -5}, imagePoint{15, 5})},
		{testStraightRoute(imagePoint{0, 25}, imagePoint{20, -5}), testStraightRoute(imagePoint{50, -20}, imagePoint{50, 20})},
	}
	// No single change reduces the initial crossing count. Joint changes can.
	initial := []imageConnectionRoute{candidates[0][0], candidates[1][0], candidates[2][0]}
	if score := testRoutingConflicts(initial); score.crossings != 1 {
		t.Fatalf("initial crossings = %d, want 1", score.crossings)
	}
	routes := r.chooseConnectionRoutes(candidates)
	if score := testRoutingConflicts(routes); score.crossings != 0 || score.overlaps != 0 {
		t.Fatalf("joint route selection did not remove crossings: %+v", score)
	}
}

func TestRouteSelectionMinimizesUnavoidableCrossings(t *testing.T) {
	r, _, _ := testConnectionRoutes(t, "a")
	candidates := [][]imageConnectionRoute{
		{testStraightRoute(imagePoint{0, 0}, imagePoint{10, 10})},
		{testStraightRoute(imagePoint{0, 10}, imagePoint{10, 0})},
		{testStraightRoute(imagePoint{0, 5}, imagePoint{10, 5}), testStraightRoute(imagePoint{20, 5}, imagePoint{30, 5})},
	}
	if score := testRoutingConflicts(r.chooseConnectionRoutes(candidates)); score.crossings != 1 || score.overlaps != 0 {
		t.Fatalf("conflicts = %+v, want exactly one unavoidable crossing", score)
	}
}
