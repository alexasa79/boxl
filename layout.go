package boxl

type layoutPosition struct {
	column int
	row    int
}

func layoutBoxes(children []*declaredBox, layout *LayoutDirective, scope *declaredBox) (map[*declaredBox]layoutPosition, error) {
	if layout == nil {
		layout = &LayoutDirective{Direction: LayoutRow}
	}
	if layout.Direction == LayoutGrid {
		if layout.Rows <= 0 || layout.Columns <= 0 {
			return nil, modelError(layout.Position, "grid dimensions must be positive")
		}
		if len(children) > 0 && (len(children)-1)/layout.Columns >= layout.Rows {
			return nil, modelError(layout.Position, "%dx%d grid in %s cannot fit %d boxes", layout.Rows, layout.Columns, scopeLabel(scope), len(children))
		}
	}
	positions := make(map[*declaredBox]layoutPosition, len(children))
	for i, box := range children {
		var position layoutPosition
		switch layout.Direction {
		case LayoutRow:
			position.column = i
		case LayoutColumn:
			position.row = i
		case LayoutDiagonal:
			position.column = i
			position.row = i
		case LayoutGrid:
			position.column = i % layout.Columns
			position.row = i / layout.Columns
		default:
			return nil, modelError(layout.Position, "unknown layout direction %q", layout.Direction)
		}
		positions[box] = position
	}
	return positions, nil
}
