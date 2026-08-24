package boxl

import "fmt"

// model is built during rendering pass one. A single declaredBox is shared by
// every occurrence of the same identifier.
type model struct {
	boxes map[string]*declaredBox
	order []*declaredBox
	edges []connection
}

type declaredBox struct {
	name        string
	position    Position
	text        *string
	textAt      Position
	container   bool
	parent      *declaredBox
	parentSet   bool
	parentSetAt Position
}

type connection struct {
	from     *declaredBox
	to       *declaredBox
	scope    *declaredBox
	position Position
}

func declarationPass(diagram *Diagram) (*model, error) {
	if diagram == nil {
		return nil, fmt.Errorf("diagram is nil")
	}
	m := &model{boxes: make(map[string]*declaredBox)}
	if err := m.declareDiagram(diagram, nil); err != nil {
		return nil, err
	}
	if err := m.validateContainment(); err != nil {
		return nil, err
	}
	if err := m.validateConnections(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *model) declareDiagram(diagram *Diagram, scope *declaredBox) error {
	for i := range diagram.Statements {
		statement := &diagram.Statements[i]
		boxes := make([]*declaredBox, len(statement.Boxes))
		for j := range statement.Boxes {
			box, err := m.declareBox(&statement.Boxes[j], scope)
			if err != nil {
				return err
			}
			boxes[j] = box
		}
		for j := 1; j < len(boxes); j++ {
			m.edges = append(m.edges, connection{
				from: boxes[j-1], to: boxes[j], scope: scope, position: statement.Position,
			})
		}
	}
	return nil
}

func (m *model) declareBox(expression *BoxExpression, scope *declaredBox) (*declaredBox, error) {
	box := m.boxes[expression.Identifier]
	if box == nil {
		box = &declaredBox{name: expression.Identifier, position: expression.Position}
		m.boxes[box.name] = box
		m.order = append(m.order, box)
	}

	if scope != nil {
		if box.parentSet && box.parent != scope {
			return nil, modelError(expression.Position,
				"box %q belongs to both %q and %q", box.name, box.parent.name, scope.name)
		}
		if !box.parentSet {
			box.parent = scope
			box.parentSet = true
			box.parentSetAt = expression.Position
		}
	}

	if expression.Text != nil {
		if box.container {
			return nil, modelError(expression.Position, "box %q cannot have both text and nested boxes", box.name)
		}
		if box.text != nil && *box.text != *expression.Text {
			return nil, modelError(expression.Position, "box %q has conflicting text declarations", box.name)
		}
		if box.text == nil {
			text := *expression.Text
			box.text = &text
			box.textAt = expression.Position
		}
	}

	if expression.Diagram != nil {
		if box.text != nil {
			return nil, modelError(expression.Position, "box %q cannot have both text and nested boxes", box.name)
		}
		box.container = true
		if err := m.declareDiagram(expression.Diagram, box); err != nil {
			return nil, err
		}
	}
	return box, nil
}

func (m *model) validateContainment() error {
	state := make(map[*declaredBox]uint8, len(m.order))
	var visit func(*declaredBox) error
	visit = func(box *declaredBox) error {
		switch state[box] {
		case 1:
			return modelError(box.parentSetAt, "box %q creates a containment cycle", box.name)
		case 2:
			return nil
		}
		state[box] = 1
		if box.parent != nil {
			if err := visit(box.parent); err != nil {
				return err
			}
		}
		state[box] = 2
		return nil
	}
	for _, box := range m.order {
		if err := visit(box); err != nil {
			return err
		}
	}
	return nil
}

func (m *model) validateConnections() error {
	for _, edge := range m.edges {
		if edge.from.parent != edge.scope || edge.to.parent != edge.scope {
			scopeName := "the top level"
			if edge.scope != nil {
				scopeName = fmt.Sprintf("container %q", edge.scope.name)
			}
			return modelError(edge.position,
				"connection %q -> %q crosses the boundary of %s", edge.from.name, edge.to.name, scopeName)
		}
	}
	return nil
}

func (m *model) children(scope *declaredBox) []*declaredBox {
	children := make([]*declaredBox, 0)
	for _, box := range m.order {
		if box.parent == scope {
			children = append(children, box)
		}
	}
	return children
}

func (m *model) connections(scope *declaredBox) []connection {
	edges := make([]connection, 0)
	for _, edge := range m.edges {
		if edge.scope == scope {
			edges = append(edges, edge)
		}
	}
	return edges
}

func modelError(position Position, format string, args ...any) error {
	return fmt.Errorf("line %d, column %d: %s", position.Line, position.Column, fmt.Sprintf(format, args...))
}
