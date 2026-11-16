package boxl

import (
	"math"
	"sort"
)

type imageRouteBounds struct {
	minX, minY, maxX, maxY float64
}

func (a imageRouteBounds) intersects(b imageRouteBounds) bool {
	return a.minX <= b.maxX && b.minX <= a.maxX && a.minY <= b.maxY && b.minY <= a.maxY
}

type imageRouteSegment struct {
	start, end imagePoint
	bounds     imageRouteBounds
}

type imageRouteOption struct {
	route    imageConnectionRoute
	segments []imageRouteSegment
	bounds   imageRouteBounds
	length   float64
	head     [3]imagePoint
	id       int
}

func makeImageRouteOption(route imageConnectionRoute, id int) imageRouteOption {
	option := imageRouteOption{route: route, id: id, length: connectionLength(route.points)}
	option.bounds = imageRouteBounds{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for i, point := range route.points {
		option.bounds.minX = math.Min(option.bounds.minX, point.x)
		option.bounds.minY = math.Min(option.bounds.minY, point.y)
		option.bounds.maxX = math.Max(option.bounds.maxX, point.x)
		option.bounds.maxY = math.Max(option.bounds.maxY, point.y)
		if i > 0 {
			previous := route.points[i-1]
			option.segments = append(option.segments, imageRouteSegment{
				start: previous, end: point,
				bounds: imageRouteBounds{math.Min(previous.x, point.x), math.Min(previous.y, point.y), math.Max(previous.x, point.x), math.Max(previous.y, point.y)},
			})
		}
	}
	return option
}

// Continuous overlaps are excluded first, then crossings and arrowhead contacts
// are minimized. Curve count and length break ties to keep simple links straight.
type imageRoutingScore struct {
	overlaps, crossings, contacts, curves int
	length                                float64
}

func (a imageRoutingScore) add(b imageRoutingScore) imageRoutingScore {
	return imageRoutingScore{a.overlaps + b.overlaps, a.crossings + b.crossings, a.contacts + b.contacts, a.curves + b.curves, a.length + b.length}
}

func (a imageRoutingScore) less(b imageRoutingScore) bool {
	if a.overlaps != b.overlaps {
		return a.overlaps < b.overlaps
	}
	if a.crossings != b.crossings {
		return a.crossings < b.crossings
	}
	if a.contacts != b.contacts {
		return a.contacts < b.contacts
	}
	if a.curves != b.curves {
		return a.curves < b.curves
	}
	return a.length < b.length-1e-6
}

func (a imageRoutingScore) conflictFree() bool {
	return a.overlaps == 0 && a.crossings == 0 && a.contacts == 0
}

func imageRouteIntersections(a, b imageRouteOption) imageRoutingScore {
	if !a.bounds.intersects(b.bounds) {
		return imageRoutingScore{}
	}
	var intersections []imagePoint
	overlap := false
	for _, first := range a.segments {
		for _, second := range b.segments {
			if !first.bounds.intersects(second.bounds) {
				continue
			}
			point, intersects, sharedSegment := imageSegmentIntersection(first.start, first.end, second.start, second.end)
			if sharedSegment {
				overlap = true
				continue
			}
			if !intersects {
				continue
			}
			duplicate := false
			for _, previous := range intersections {
				if point.sub(previous).length() < 0.25 {
					duplicate = true
					break
				}
			}
			if !duplicate {
				intersections = append(intersections, point)
			}
		}
	}
	score := imageRoutingScore{crossings: len(intersections)}
	if overlap {
		score.overlaps = 1
	}
	return score
}

func imageSegmentIntersection(a, b, c, d imagePoint) (imagePoint, bool, bool) {
	cross := func(p, q imagePoint) float64 { return p.x*q.y - p.y*q.x }
	u, v, delta := b.sub(a), d.sub(c), c.sub(a)
	denominator := cross(u, v)
	const epsilon = 1e-8
	if math.Abs(denominator) < epsilon {
		lengthSquared := u.x*u.x + u.y*u.y
		if lengthSquared < epsilon || math.Abs(cross(delta, u)) > epsilon {
			return imagePoint{}, false, false
		}
		first := (delta.x*u.x + delta.y*u.y) / lengthSquared
		lastDelta := d.sub(a)
		last := (lastDelta.x*u.x + lastDelta.y*u.y) / lengthSquared
		low, high := math.Max(0, math.Min(first, last)), math.Min(1, math.Max(first, last))
		if low > high+epsilon {
			return imagePoint{}, false, false
		}
		return a.add(u.mul((low + high) / 2)), true, (high-low)*math.Sqrt(lengthSquared) > epsilon
	}
	t, s := cross(delta, v)/denominator, cross(delta, u)/denominator
	if t < -epsilon || t > 1+epsilon || s < -epsilon || s > 1+epsilon {
		return imagePoint{}, false, false
	}
	return a.add(u.mul(t)), true, false
}

func (r *imageRenderer) chooseConnectionRoutes(candidates [][]imageConnectionRoute) []imageConnectionRoute {
	if len(candidates) == 0 {
		return nil
	}
	options := make([][]imageRouteOption, len(candidates))
	id := 0
	for i, routes := range candidates {
		for _, route := range routes {
			option := makeImageRouteOption(route, id)
			option.head = r.connectionArrowhead(route)
			options[i] = append(options[i], option)
			id++
		}
		sort.SliceStable(options[i], func(a, b int) bool { return options[i][a].length < options[i][b].length })
	}
	pairs := make(map[[2]int]imageRoutingScore)
	pairScore := func(a, b imageRouteOption) imageRoutingScore {
		key := [2]int{min(a.id, b.id), max(a.id, b.id)}
		if cached, ok := pairs[key]; ok {
			return cached
		}
		score := imageRouteIntersections(a, b)
		if imageRouteTouchesHead(a, b.head, float64(r.lineWidth)/2+0.5) || imageRouteTouchesHead(b, a.head, float64(r.lineWidth)/2+0.5) {
			score.contacts = 1
		}
		pairs[key] = score
		return score
	}
	optionScore := func(option imageRouteOption) imageRoutingScore {
		score := imageRoutingScore{length: option.length}
		if len(option.route.points) > 2 {
			score.curves = 1
		}
		return score
	}
	choice := make([]int, len(options))
	fullScore := func() imageRoutingScore {
		var score imageRoutingScore
		for i, selected := range choice {
			option := options[i][selected]
			score = score.add(optionScore(option))
			for j := 0; j < i; j++ {
				score = score.add(pairScore(option, options[j][choice[j]]))
			}
		}
		return score
	}
	// Coordinate descent provides an incumbent for branch-and-bound and a useful
	// result for large charts when the search budget is exhausted.
	bestScore := fullScore()
	for sweep := 0; sweep < 8; sweep++ {
		improved := false
		for i := range options {
			selected := choice[i]
			for candidate := range options[i] {
				choice[i] = candidate
				if score := fullScore(); score.less(bestScore) {
					selected, bestScore, improved = candidate, score, true
				}
			}
			choice[i] = selected
		}
		if !improved {
			break
		}
	}
	best := append([]int(nil), choice...)
	order := make([]int, len(options))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return len(options[order[a]]) < len(options[order[b]]) })
	const searchBudget = 200000
	visited := 0
	var search func(int, imageRoutingScore)
	search = func(depth int, score imageRoutingScore) {
		if visited >= searchBudget || bestScore.conflictFree() {
			return
		}
		visited++
		if depth == len(order) {
			if score.less(bestScore) {
				bestScore = score
				copy(best, choice)
			}
			return
		}
		// Ignore conflicts between unassigned routes for an optimistic lower bound.
		lower := score
		for next := depth; next < len(order); next++ {
			var minimum imageRoutingScore
			for candidate, option := range options[order[next]] {
				cost := optionScore(option)
				for previous := 0; previous < depth; previous++ {
					index := order[previous]
					cost = cost.add(pairScore(option, options[index][choice[index]]))
				}
				if candidate == 0 || cost.less(minimum) {
					minimum = cost
				}
			}
			lower = lower.add(minimum)
		}
		if !lower.less(bestScore) {
			return
		}
		index := order[depth]
		for candidate, option := range options[index] {
			cost := score.add(optionScore(option))
			for previous := 0; previous < depth; previous++ {
				other := order[previous]
				cost = cost.add(pairScore(option, options[other][choice[other]]))
			}
			if cost.less(bestScore) {
				choice[index] = candidate
				search(depth+1, cost)
			}
		}
	}
	// Zero conflicts is the absolute lower bound; do not spend the search budget
	// proving a cosmetic length tie-break when that objective is already reached.
	if !bestScore.conflictFree() {
		search(0, imageRoutingScore{})
	}
	routes := make([]imageConnectionRoute, len(options))
	for i, selected := range best {
		routes[i] = options[i][selected].route
	}
	return routes
}

func imageRouteTouchesHead(route imageRouteOption, head [3]imagePoint, clearance float64) bool {
	for i, point := range head {
		next := head[(i+1)%len(head)]
		bounds := imageRouteBounds{
			math.Min(point.x, next.x) - clearance, math.Min(point.y, next.y) - clearance,
			math.Max(point.x, next.x) + clearance, math.Max(point.y, next.y) + clearance,
		}
		for _, segment := range route.segments {
			if bounds.intersects(segment.bounds) && imageSegmentsNear(point, next, segment.start, segment.end, clearance) {
				return true
			}
		}
		for j, other := range route.head {
			if imageSegmentsNear(point, next, other, route.head[(j+1)%len(route.head)], 0.5) {
				return true
			}
		}
	}
	return false
}

func imageSegmentsNear(a, b, c, d imagePoint, clearance float64) bool {
	if _, intersects, _ := imageSegmentIntersection(a, b, c, d); intersects {
		return true
	}
	distance := func(point, start, end imagePoint) float64 {
		delta := end.sub(start)
		lengthSquared := delta.x*delta.x + delta.y*delta.y
		if lengthSquared == 0 {
			return point.sub(start).length()
		}
		offset := point.sub(start)
		t := math.Max(0, math.Min(1, (offset.x*delta.x+offset.y*delta.y)/lengthSquared))
		return point.sub(start.add(delta.mul(t))).length()
	}
	return math.Min(math.Min(distance(a, c, d), distance(b, c, d)), math.Min(distance(c, a, b), distance(d, a, b))) <= clearance
}
