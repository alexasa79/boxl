package boxl

import (
	"image/color"
	"reflect"
	"sort"
	"strings"
)

type palettePair struct {
	foreground, background color.NRGBA
}

type colorPalette [16]palettePair

func opaqueColor(rgb uint32) color.NRGBA {
	return color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 255}
}

func opaquePair(foreground, background uint32) palettePair {
	return palettePair{foreground: opaqueColor(foreground), background: opaqueColor(background)}
}

var pastelPalette = colorPalette{
	opaquePair(0x326b53, 0xd8f2e4), // Seafoam.
	opaquePair(0x3e6b3a, 0xdef2d8), // Mint.
	opaquePair(0x8d4f40, 0xf9dfd8), // Coral.
	opaquePair(0x805080, 0xf3e1f4), // Lilac.
	opaquePair(0x66508c, 0xebe2f7), // Lavender.
	opaquePair(0x365f85, 0xdeeffa), // Sky.
	opaquePair(0x756020, 0xfaf1c8), // Butter.
	opaquePair(0x8a4b70, 0xf9e1ee), // Pink.
	opaquePair(0x4b6670, 0xe1eaed), // Mist.
	opaquePair(0x2f6a65, 0xd9f1ee), // Aqua.
	opaquePair(0x665c75, 0xeae6ef), // Gray violet.
	opaquePair(0x8a3f52, 0xfbe4e8), // Rose.
	opaquePair(0x536b45, 0xe5ebdd), // Sage.
	opaquePair(0x8a512f, 0xfce8d5), // Peach.
	opaquePair(0x72613e, 0xefe8d5), // Sand.
	opaquePair(0x485a91, 0xe2e8fc), // Periwinkle.
}

var earthPalette = colorPalette{
	opaquePair(0x4f6338, 0xe6ebd8), // Moss.
	opaquePair(0x92563f, 0xf2dfd3), // Clay.
	opaquePair(0x4b6063, 0xdfe8e7), // Slate.
	opaquePair(0x85661f, 0xf0e4ba), // Ochre.
	opaquePair(0x6e5569, 0xebdddf), // Heather.
	opaquePair(0x62704b, 0xe7ead8), // Sage.
	opaquePair(0x9a5141, 0xf3dcd1), // Terracotta.
	opaquePair(0x7c684a, 0xeee4d2), // Sandstone.
	opaquePair(0x3d654d, 0xdce9dc), // Pine.
	opaquePair(0x70523a, 0xeadccc), // Walnut.
	opaquePair(0x6b6a36, 0xecebcf), // Olive.
	opaquePair(0x686159, 0xe8e3da), // Stone.
	opaquePair(0x83513f, 0xeedad2), // Chestnut.
	opaquePair(0x60714d, 0xe2e9d6), // Lichen.
	opaquePair(0x526c78, 0xdfe8ed), // Dusty blue.
	opaquePair(0x775265, 0xebdce4), // Mulberry.
}

var oceanPalette = colorPalette{
	opaquePair(0x285e87, 0xdcecf8), // Blue.
	opaquePair(0x216d70, 0xd6f0ec), // Lagoon.
	opaquePair(0x465a91, 0xe0e7fa), // Indigo.
	opaquePair(0x3d7665, 0xddf1e7), // Sea glass.
	opaquePair(0x315e73, 0xdcecf0), // Deep sea.
	opaquePair(0x5d5c8f, 0xe8e5f7), // Twilight.
	opaquePair(0x24787a, 0xd7f2f0), // Aqua.
	opaquePair(0x536e7a, 0xe1eaee), // Storm.
	opaquePair(0x3c6f99, 0xe0effb), // Surf.
	opaquePair(0x48705a, 0xe1eedf), // Kelp.
	opaquePair(0x3c5d9b, 0xe2eafb), // Cobalt.
	opaquePair(0x2d746d, 0xd9efe9), // Tidepool.
	opaquePair(0x5a7380, 0xe7eff1), // Ocean mist.
	opaquePair(0x365783, 0xdde6f5), // Pelagic.
	opaquePair(0x426d65, 0xe4f0e9), // Coast.
	opaquePair(0x50657c, 0xe3eaf2), // Fjord.
}

var availablePalettes = map[string]*colorPalette{
	"pastel": &pastelPalette,
	"earth":  &earthPalette,
	"ocean":  &oceanPalette,
}

func lookupPalette(name string) *colorPalette {
	return availablePalettes[strings.ToLower(name)]
}

func paletteNames() []string {
	names := make([]string, 0, len(availablePalettes))
	for name := range availablePalettes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type paletteState struct {
	palette *colorPalette
	index   int
}

type resolvedBoxColors struct {
	background, foreground color.NRGBA
}

func (p palettePair) colors() resolvedBoxColors {
	return resolvedBoxColors{background: p.background, foreground: p.foreground}
}

func directiveExpression(directive *ColorDirective) Expression {
	if directive.Expression != nil {
		return directive.Expression
	}
	return ColorLiteral{Value: directive.Color}
}

func sameColorDirective(a, b *ColorDirective) bool {
	first, firstErr := evaluateColorExpression(directiveExpression(a), nil)
	second, secondErr := evaluateColorExpression(directiveExpression(b), nil)
	if firstErr == nil && secondErr == nil {
		return first == second
	}
	return reflect.DeepEqual(directiveExpression(a), directiveExpression(b))
}

func applyColorStyle(style BoxStyle, colors resolvedBoxColors, state *paletteState) (resolvedBoxColors, error) {
	type assignment struct {
		kind      string
		directive *ColorDirective
	}
	var assignments []assignment
	for _, item := range []assignment{{"bg", style.Background}, {"fg", style.Foreground}, {"pair", style.Pair}} {
		if item.directive != nil {
			assignments = append(assignments, item)
		}
	}
	sort.SliceStable(assignments, func(i, j int) bool {
		return assignments[i].directive.Position.Offset < assignments[j].directive.Position.Offset
	})
	for _, item := range assignments {
		value, err := evaluateColorExpression(directiveExpression(item.directive), state)
		if err != nil {
			return colors, modelError(item.directive.Position, "%v", err)
		}
		switch item.kind {
		case "pair":
			if value.kind != palettePairValue {
				return colors, modelError(item.directive.Position, "expected a palette pair")
			}
			colors = value.pair.colors()
			state.index = value.index
		case "bg", "fg":
			var selected color.NRGBA
			switch value.kind {
			case palettePairValue:
				selected = value.pair.background
				if item.kind == "fg" {
					selected = value.pair.foreground
				}
			case literalColorValue:
				selected = value.color
			default:
				return colors, modelError(item.directive.Position, "expected a color")
			}
			if item.kind == "bg" {
				colors.background = selected
			} else {
				colors.foreground = selected
			}
		}
	}
	return colors, nil
}

func (m *model) resolveColorStyles() error {
	m.resolvedStyles = make(map[*declaredBox]resolvedBoxColors)
	root := resolvedBoxColors{foreground: opaqueColor(0xffffff)}
	state := paletteState{index: -1}
	if directive := m.styles[nil].Palette; directive != nil {
		state.palette = lookupPalette(directive.Name)
		if state.palette == nil {
			return modelError(directive.Position, "unknown palette %q", directive.Name)
		}
		root.foreground = state.palette[0].foreground
	}
	var err error
	root, err = applyColorStyle(m.styles[nil], root, &state)
	if err != nil {
		return err
	}
	m.resolvedStyles[nil] = root
	return m.resolveChildColors(nil, state, root)
}

func (m *model) resolveChildColors(scope *declaredBox, parent paletteState, parentColors resolvedBoxColors) error {
	level := parent
	current := resolvedBoxColors{foreground: parentColors.foreground}
	if level.palette != nil {
		level.index = (level.index + 1) % len(level.palette)
		current = level.palette[level.index].colors()
	}
	for _, box := range m.children(scope) {
		state, colors := level, current
		style := m.styles[box]
		if directive := style.Palette; directive != nil {
			state = paletteState{palette: lookupPalette(directive.Name), index: 0}
			if state.palette == nil {
				return modelError(directive.Position, "unknown palette %q", directive.Name)
			}
			colors = state.palette[0].colors()
		}
		var err error
		colors, err = applyColorStyle(style, colors, &state)
		if err != nil {
			return err
		}
		m.resolvedStyles[box] = colors
		if level.palette != nil && state.palette == level.palette {
			// Child evaluation receives a copy, so deeper levels cannot advance this one.
			current, level.index = colors, state.index
		}
		if err := m.resolveChildColors(box, state, colors); err != nil {
			return err
		}
	}
	return nil
}
