package boxl

type layoutPosition struct {
	column int
	row    int
}

func layoutBoxes(children []*declaredBox, constraints []declaredConstraint, edges []connection, scope *declaredBox) (map[*declaredBox]layoutPosition, error) {
	if len(constraints) > 0 {
		return layoutGrid(children, constraints, scope)
	}
	return layoutConnectionDefaults(children, edges), nil
}

func layoutConnectionDefaults(children []*declaredBox, edges []connection) map[*declaredBox]layoutPosition {
	columns := make(map[*declaredBox]int, len(children))
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
			columns[next] = max(columns[next], columns[box]+1)
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if processed != len(children) {
		positions := make(map[*declaredBox]layoutPosition, len(children))
		for column, box := range children {
			positions[box] = layoutPosition{column: column}
		}
		return positions
	}

	rowInColumn := make(map[int]int)
	positions := make(map[*declaredBox]layoutPosition, len(children))
	for _, box := range children {
		column := columns[box]
		positions[box] = layoutPosition{column: column, row: rowInColumn[column]}
		rowInColumn[column]++
	}
	return positions
}

type boxUnion struct {
	parent map[*declaredBox]*declaredBox
}

func newBoxUnion(boxes []*declaredBox) *boxUnion {
	parent := make(map[*declaredBox]*declaredBox, len(boxes))
	for _, box := range boxes {
		parent[box] = box
	}
	return &boxUnion{parent: parent}
}

func (u *boxUnion) find(box *declaredBox) *declaredBox {
	root := box
	for u.parent[root] != root {
		root = u.parent[root]
	}
	for u.parent[box] != box {
		next := u.parent[box]
		u.parent[box] = root
		box = next
	}
	return root
}

func (u *boxUnion) join(first, second *declaredBox) {
	firstRoot := u.find(first)
	secondRoot := u.find(second)
	if firstRoot != secondRoot {
		u.parent[secondRoot] = firstRoot
	}
}

func layoutGrid(children []*declaredBox, constraints []declaredConstraint, scope *declaredBox) (map[*declaredBox]layoutPosition, error) {
	positions := make(map[*declaredBox]layoutPosition, len(children))
	if len(children) == 0 {
		return positions, nil
	}

	components := newBoxUnion(children)
	for _, constraint := range constraints {
		components.join(constraint.box, constraint.target)
	}

	componentBoxes := make(map[*declaredBox][]*declaredBox)
	var componentOrder []*declaredBox
	for _, box := range children {
		root := components.find(box)
		if _, exists := componentBoxes[root]; !exists {
			componentOrder = append(componentOrder, root)
		}
		componentBoxes[root] = append(componentBoxes[root], box)
	}
	componentConstraints := make(map[*declaredBox][]declaredConstraint)
	for _, constraint := range constraints {
		root := components.find(constraint.box)
		componentConstraints[root] = append(componentConstraints[root], constraint)
	}

	columnOffset := 0
	for _, root := range componentOrder {
		local, columns, err := layoutConstraintComponent(
			componentBoxes[root], componentConstraints[root], scope,
		)
		if err != nil {
			return nil, err
		}
		for box, position := range local {
			position.column += columnOffset
			positions[box] = position
		}
		columnOffset += columns
	}
	return positions, nil
}

type layoutArc struct {
	from     *declaredBox
	to       *declaredBox
	position Position
}

func layoutConstraintComponent(boxes []*declaredBox, constraints []declaredConstraint, scope *declaredBox) (map[*declaredBox]layoutPosition, int, error) {
	rows := newBoxUnion(boxes)
	columns := newBoxUnion(boxes)
	for _, constraint := range constraints {
		switch constraint.relation {
		case ConstraintThen, ConstraintLeftOf, ConstraintRightOf:
			rows.join(constraint.box, constraint.target)
		case ConstraintAbove, ConstraintBelow:
			columns.join(constraint.box, constraint.target)
		}
	}

	var horizontal, vertical []layoutArc
	for _, constraint := range constraints {
		switch constraint.relation {
		case ConstraintThen, ConstraintLeftOf:
			horizontal = append(horizontal, layoutArc{
				from: columns.find(constraint.box), to: columns.find(constraint.target), position: constraint.position,
			})
		case ConstraintRightOf:
			horizontal = append(horizontal, layoutArc{
				from: columns.find(constraint.target), to: columns.find(constraint.box), position: constraint.position,
			})
		case ConstraintAbove:
			vertical = append(vertical, layoutArc{
				from: rows.find(constraint.box), to: rows.find(constraint.target), position: constraint.position,
			})
		case ConstraintBelow:
			vertical = append(vertical, layoutArc{
				from: rows.find(constraint.target), to: rows.find(constraint.box), position: constraint.position,
			})
		}
	}

	columnGroups := unionGroupsInOrder(boxes, columns)
	rowGroups := unionGroupsInOrder(boxes, rows)
	columnRanks, err := rankLayoutGroups(columnGroups, horizontal, "horizontal", scope)
	if err != nil {
		return nil, 0, err
	}
	rowRanks, err := rankLayoutGroups(rowGroups, vertical, "vertical", scope)
	if err != nil {
		return nil, 0, err
	}

	positions := make(map[*declaredBox]layoutPosition, len(boxes))
	for _, box := range boxes {
		positions[box] = layoutPosition{
			column: columnRanks[columns.find(box)],
			row:    rowRanks[rows.find(box)],
		}
	}
	return positions, len(columnGroups), nil
}

func unionGroupsInOrder(boxes []*declaredBox, union *boxUnion) []*declaredBox {
	groups := make([]*declaredBox, 0, len(boxes))
	seen := make(map[*declaredBox]bool, len(boxes))
	for _, box := range boxes {
		root := union.find(box)
		if !seen[root] {
			seen[root] = true
			groups = append(groups, root)
		}
	}
	return groups
}

func rankLayoutGroups(groups []*declaredBox, arcs []layoutArc, axis string, scope *declaredBox) (map[*declaredBox]int, error) {
	indegree := make(map[*declaredBox]int, len(groups))
	adjacency := make(map[*declaredBox][]*declaredBox, len(groups))
	for _, group := range groups {
		indegree[group] = 0
	}
	seen := make(map[[2]*declaredBox]bool, len(arcs))
	for _, arc := range arcs {
		if arc.from == arc.to {
			return nil, modelError(arc.position, "%s layout constraint conflicts with a required alignment", axis)
		}
		key := [2]*declaredBox{arc.from, arc.to}
		if seen[key] {
			continue
		}
		seen[key] = true
		adjacency[arc.from] = append(adjacency[arc.from], arc.to)
		indegree[arc.to]++
	}

	ranks := make(map[*declaredBox]int, len(groups))
	processed := make(map[*declaredBox]bool, len(groups))
	for rank := 0; rank < len(groups); rank++ {
		var next *declaredBox
		for _, group := range groups {
			if !processed[group] && indegree[group] == 0 {
				next = group
				break
			}
		}
		if next == nil {
			position := Position{Line: 1, Column: 1}
			if len(arcs) > 0 {
				position = arcs[0].position
			}
			return nil, modelError(position, "cannot satisfy %s layout constraints in %s", axis, scopeLabel(scope))
		}
		processed[next] = true
		ranks[next] = rank
		for _, target := range adjacency[next] {
			indegree[target]--
		}
	}
	return ranks, nil
}
