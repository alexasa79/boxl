package boxl

import "fmt"

// model is built during rendering pass one. A single declaredBox is shared by
// every occurrence of the same identifier.
type model struct {
	boxes          map[string]*declaredBox
	order          []*declaredBox
	edges          []connection
	layouts        map[*declaredBox]*LayoutDirective
	styles         map[*declaredBox]BoxStyle
	resolvedStyles map[*declaredBox]resolvedBoxColors
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
	m := &model{
		boxes:   make(map[string]*declaredBox),
		layouts: make(map[*declaredBox]*LayoutDirective),
		styles:  make(map[*declaredBox]BoxStyle),
	}
	if err := m.declareDiagram(diagram, nil); err != nil {
		return nil, err
	}
	if err := m.validateContainment(); err != nil {
		return nil, err
	}
	if err := m.validateConnections(); err != nil {
		return nil, err
	}
	if err := m.resolveColorStyles(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *model) declareDiagram(diagram *Diagram, scope *declaredBox) error {
	if err := m.declareStyle(scope, diagram.Style); err != nil {
		return err
	}
	if layout := diagram.Layout; layout != nil {
		if previous := m.layouts[scope]; previous != nil &&
			(previous.Direction != layout.Direction || previous.Rows != layout.Rows || previous.Columns != layout.Columns) {
			return modelError(layout.Position, "conflicting layout directives in %s", scopeLabel(scope))
		}
		m.layouts[scope] = layout
	}
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
		if statement.Connected {
			for j := 1; j < len(boxes); j++ {
				m.edges = append(m.edges, connection{
					from: boxes[j-1], to: boxes[j], scope: scope, position: statement.Position,
				})
			}
		}
	}
	return nil
}

func (m *model) declareBox(expression *BoxExpression, scope *declaredBox) (*declaredBox, error) {
	box, err := m.declareReference(expression.Identifier, expression.Position, scope)
	if err != nil {
		return nil, err
	}
	if err := m.declareStyle(box, expression.Style); err != nil {
		return nil, err
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

func (m *model) declareStyle(box *declaredBox, style BoxStyle) error {
	previous := m.styles[box]
	merge := func(target **ColorDirective, value *ColorDirective, name string) error {
		if value == nil {
			return nil
		}
		if *target != nil && !sameColorDirective(*target, value) {
			targetName := "chart"
			if box != nil {
				targetName = fmt.Sprintf("box %q", box.name)
			}
			return modelError(value.Position, "conflicting #%s colors for %s", name, targetName)
		}
		if *target == nil {
			*target = value
		}
		return nil
	}
	if err := merge(&previous.Background, style.Background, "bg"); err != nil {
		return err
	}
	if err := merge(&previous.Foreground, style.Foreground, "fg"); err != nil {
		return err
	}
	if err := merge(&previous.Pair, style.Pair, "colors"); err != nil {
		return err
	}
	if palette := style.Palette; palette != nil {
		if previous.Palette != nil && previous.Palette.Name != palette.Name {
			return modelError(palette.Position, "conflicting palette declarations in %s", scopeLabel(box))
		}
		if previous.Palette == nil {
			previous.Palette = palette
		}
	}
	m.styles[box] = previous
	return nil
}

func (m *model) declareReference(name string, position Position, scope *declaredBox) (*declaredBox, error) {
	box := m.boxes[name]
	if box == nil {
		box = &declaredBox{name: name, position: position}
		m.boxes[name] = box
		m.order = append(m.order, box)
	}
	if scope != nil {
		if box.parentSet && box.parent != scope {
			return nil, modelError(position, "box %q belongs to both %s and %s",
				box.name, scopeLabel(box.parent), scopeLabel(scope))
		}
		if !box.parentSet {
			box.parent = scope
			box.parentSet = true
			box.parentSetAt = position
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

func scopeLabel(scope *declaredBox) string {
	if scope == nil {
		return "the top level"
	}
	return fmt.Sprintf("container %q", scope.name)
}

func modelError(position Position, format string, args ...any) error {
	return fmt.Errorf("line %d, column %d: %s", position.Line, position.Column, fmt.Sprintf(format, args...))
}
